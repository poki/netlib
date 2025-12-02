package stores

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/koenbollen/logging"
	"github.com/pgvector/pgvector-go"
	"github.com/poki/mongodb-filter-to-postgres/filter"
	"github.com/poki/netlib/internal/util"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var isTestEnv = os.Getenv("ENV") == "test"

var topicRegexp = regexp.MustCompile(`^[a-zA-Z0-9\-]{1,76}$`)

type PostgresStore struct {
	DB *pgxpool.Pool

	mutex             sync.Mutex
	callbacks         map[string]map[uint64]SubscriptionCallback
	nextCallbackIndex uint64
	filterConverter   *filter.Converter
}

func NewPostgresStore(ctx context.Context, db *pgxpool.Pool) (*PostgresStore, error) {
	filterConverter, err := filter.NewConverter(
		filter.WithNestedJSONB("custom_data", "code", "playerCount", "createdAt", "updatedAt", "latency", "latency2"),
		filter.WithEmptyCondition("TRUE"), // No filter returns all lobbies.
	)
	if err != nil {
		return nil, err
	}

	s := &PostgresStore{
		DB:              db,
		callbacks:       make(map[string]map[uint64]SubscriptionCallback),
		filterConverter: filterConverter,
	}
	go s.run(ctx)
	return s, nil
}

func (s *PostgresStore) run(ctx context.Context) {
	logger := logging.GetLogger(ctx)

	for {
		err := s.listen(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			logger.Error("pubsub bus failed, retrying", zap.Error(err))
		}
	}
}

func (s *PostgresStore) listen(ctx context.Context) error {
	conn, err := s.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	_, err = conn.Exec(ctx, "LISTEN lobbies")
	if err != nil {
		return fmt.Errorf("failed to LISTEN to lobbies: %w", err)
	}
	defer conn.Release()

	for {
		n, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return fmt.Errorf("failed to wait for notification: %w", err)
		}
		topic, data, ok := strings.Cut(n.Payload, ":")
		if !ok {
			continue
		}
		raw := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
		l, err := base64.StdEncoding.Decode(raw, []byte(data))
		if err != nil {
			return fmt.Errorf("failed to decode payload: %w", err)
		}
		raw = raw[:l]
		s.notify(ctx, topic, raw)
	}
}

func (s *PostgresStore) notify(ctx context.Context, topic string, data []byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if callbacks, found := s.callbacks[topic]; found {
		for _, callback := range callbacks {
			go callback(ctx, data)
		}
	}
}

func (s *PostgresStore) Subscribe(ctx context.Context, callback SubscriptionCallback, game, lobby, peerID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id := s.nextCallbackIndex
	s.nextCallbackIndex += 1

	topics := []string{
		game + lobby + peerID, // Topic for a specific peer in a specific lobby.
		game + lobby,          // Topic for all peers in a specific lobby.
	}

	for _, topic := range topics {
		if _, found := s.callbacks[topic]; !found {
			s.callbacks[topic] = make(map[uint64]SubscriptionCallback)
		}

		s.callbacks[topic][id] = callback
	}

	go func() {
		defer func() {
			s.mutex.Lock()
			defer s.mutex.Unlock()

			for _, topic := range topics {
				delete(s.callbacks[topic], id)
				if len(s.callbacks[topic]) == 0 {
					delete(s.callbacks, topic)
				}
			}
		}()

		<-ctx.Done()
	}()
}

func (s *PostgresStore) Publish(ctx context.Context, topic string, data []byte) error {
	if !topicRegexp.MatchString(topic) {
		return fmt.Errorf("topic %q is invalid", topic)
	}
	totalLength := base64.StdEncoding.EncodedLen(len(data)) + len(topic) + 1
	if totalLength > 8000 {
		return fmt.Errorf("data too long for topic %q: %d", topic, totalLength)
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	payload := topic + ":" + encoded
	_, err := s.DB.Exec(ctx, `NOTIFY lobbies, '`+payload+`'`)
	if err != nil {
		return fmt.Errorf("failed to publish to lobbies: %w", err)
	}
	return nil
}

func (s *PostgresStore) CreateLobby(ctx context.Context, game, lobbyCode, peerID string, options LobbyOptions) error {
	if len(lobbyCode) > 20 {
		logger := logging.GetLogger(ctx)
		logger.Warn("lobby code too long", zap.String("lobbyCode", lobbyCode))
		return ErrInvalidLobbyCode
	}
	if len(peerID) > 20 {
		logger := logging.GetLogger(ctx)
		logger.Warn("peer id too long", zap.String("peerID", peerID))
		return ErrInvalidPeerID
	}

	var hashedPassword []byte

	if options.Password != nil && len(*options.Password) > 0 {
		var err error
		hashedPassword, err = bcrypt.GenerateFromPassword([]byte(*options.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
	}

	now := util.NowUTC(ctx)
	res, err := s.DB.Exec(ctx, `
		INSERT INTO lobbies (code, game, peers, public, custom_data, created_at, updated_at, leader, term, can_update_by, creator, password, max_players)
		VALUES ($1, $2, $3, $4, $5, $6, $6, $7, 1, $8, $7, $9, $10)
		ON CONFLICT DO NOTHING
	`, lobbyCode, game, []string{peerID}, options.Public, options.CustomData, now, peerID, options.CanUpdateBy, hashedPassword, options.MaxPlayers)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrLobbyExists
	}
	return nil
}

func (s *PostgresStore) JoinLobby(ctx context.Context, game, lobbyCode, peerID, password string) error {
	if len(peerID) > 20 {
		logger := logging.GetLogger(ctx)
		logger.Warn("peer id too long", zap.String("peerID", peerID))
		return ErrInvalidPeerID
	}

	now := util.NowUTC(ctx)

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	var peerlist []string
	var lobbyPassword []byte
	var maxPlayers int
	err = tx.QueryRow(ctx, `
		SELECT peers, password, max_players
		FROM lobbies
		WHERE code = $1
		AND game = $2
		FOR UPDATE
	`, lobbyCode, game).Scan(&peerlist, &lobbyPassword, &maxPlayers)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if lobbyPassword != nil && bcrypt.CompareHashAndPassword(lobbyPassword, []byte(password)) != nil {
		return ErrInvalidPassword
	}

	if maxPlayers > 0 && len(peerlist) >= maxPlayers {
		return ErrLobbyIsFull
	}

	if slices.Contains(peerlist, peerID) {
		return ErrAlreadyInLobby
	}

	_, err = tx.Exec(ctx, `
		UPDATE lobbies
		SET
			peers = array_append(peers, $1),
			updated_at = $2
		WHERE code = $3
		AND game = $4
	`, peerID, now, lobbyCode, game)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) LeaveLobby(ctx context.Context, game, lobbyCode, peerID string) error {
	now := util.NowUTC(ctx)

	_, err := s.DB.Exec(ctx, `
		UPDATE lobbies
		SET
			peers = array_remove(peers, $1),
			updated_at = $2
		WHERE code = $3
		AND game = $4
	`, peerID, now, lobbyCode, game)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	return nil
}

func (s *PostgresStore) GetLobby(ctx context.Context, game, lobbyCode string) (Lobby, error) {
	var lobby Lobby
	err := s.DB.QueryRow(ctx, `
		SELECT
			code,
			peers,
			COALESCE(ARRAY_LENGTH(peers, 1), 0) AS "playerCount",
			public,
			custom_data,
			created_at AS "createdAt",
			updated_at AS "updatedAt",
			leader,
			term,
			can_update_by,
			creator,
			password IS NOT NULL,
			max_players
		FROM lobbies
		WHERE code = $1
		AND game = $2
	`, lobbyCode, game).Scan(&lobby.Code, &lobby.Peers, &lobby.PlayerCount, &lobby.Public, &lobby.CustomData, &lobby.CreatedAt, &lobby.UpdatedAt, &lobby.Leader, &lobby.Term, &lobby.CanUpdateBy, &lobby.Creator, &lobby.HasPassword, &lobby.MaxPlayers)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Lobby{}, ErrNotFound
		}
		return Lobby{}, err
	}
	sort.Strings(lobby.Peers)
	return lobby, nil
}

func (s *PostgresStore) ListLobbies(ctx context.Context, game string, latency []float32, lat, lon *float64, filter, sort string, limit int) ([]Lobby, error) {
	// TODO: Remove this.
	if filter == "" {
		filter = "{}"
	}

	if limit <= 0 {
		limit = 50
	}

	var latencyVector any
	if len(latency) == 11 {
		latencyVector = pgvector.NewVector(latency)
	}

	preValues := []any{game, latencyVector, lat, lon, limit}

	where, values, err := s.filterConverter.Convert([]byte(filter), len(preValues)+1)
	if err != nil {
		logger := logging.GetLogger(ctx)
		logger.Warn("failed to convert filter", zap.String("filter", filter), zap.Error(err))
		return nil, fmt.Errorf("invalid filter: %w", err)
	}

	var order string
	if sort != "" {
		order, err = s.filterConverter.ConvertOrderBy([]byte(sort))
		if err != nil {
			logger := logging.GetLogger(ctx)
			logger.Warn("failed to convert order", zap.String("sort", sort), zap.Error(err))
			return nil, fmt.Errorf("invalid order: %w", err)
		}
	}
	if order == "" {
		order = `"createdAt" DESC, "code" ASC`
	} else {
		order += `, "createdAt" DESC, "code" ASC`
	}

	var lobbies []Lobby
	rows, err := s.DB.Query(ctx, `
		WITH game_lobbies AS (
			SELECT
				code,
				COALESCE(ARRAY_LENGTH(peers, 1), 0) AS "playerCount",
				public,
				custom_data,
				created_at AS "createdAt",
				updated_at AS "updatedAt",
				leader,
				term,
				can_update_by,
				creator,
				password IS NOT NULL,
				max_players,
				lobby_latency_estimate(
					$2,
					ARRAY(
						SELECT p.latency_vector
						FROM peers p
						WHERE p.peer = ANY (lobbies.peers)
						  AND p.latency_vector IS NOT NULL
					)
				) AS latency,
				CASE
					WHEN $3::double precision IS NULL OR $4::double precision IS NULL THEN NULL
					ELSE (
						SELECT ROUND(AVG(earth_distance(ll_to_earth($3::double precision, $4::double precision), p.geo) / 1000.0 * 0.015 + 5.0))
						FROM peers p
						WHERE p.peer = ANY (lobbies.peers)
							AND p.geo IS NOT NULL
					)
				END AS latency2
			FROM lobbies
			WHERE game = $1
			  AND public = true
		)
		SELECT *
		FROM game_lobbies
		WHERE `+where+`
		ORDER BY `+order+`
		LIMIT $`+strconv.Itoa(len(preValues))+`
	`, append(preValues, values...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var lobby Lobby
		err = rows.Scan(&lobby.Code, &lobby.PlayerCount, &lobby.Public, &lobby.CustomData, &lobby.CreatedAt, &lobby.UpdatedAt, &lobby.Leader, &lobby.Term, &lobby.CanUpdateBy, &lobby.Creator, &lobby.HasPassword, &lobby.MaxPlayers, &lobby.Latency, &lobby.Latency2)
		if err != nil {
			return nil, err
		}
		lobbies = append(lobbies, lobby)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return lobbies, nil
}

func (s *PostgresStore) CreatePeer(ctx context.Context, peerID, secret, gameID string) error {
	if len(peerID) > 20 {
		logger := logging.GetLogger(ctx)
		logger.Warn("peer id too long", zap.String("peerID", peerID))
		return ErrInvalidPeerID
	}

	now := util.NowUTC(ctx)
	_, err := s.DB.Exec(ctx, `
		INSERT INTO peers (peer, secret, game, last_seen, updated_at)
		VALUES ($1, $2, $3, $4, $4)
	`, peerID, secret, gameID, now)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) UpdatePeerLatency(ctx context.Context, peerID string, latency []float32) error {
	now := util.NowUTC(ctx)

	var vec *pgvector.Vector
	if len(latency) > 0 {
		v := pgvector.NewVector(latency)
		vec = &v
	}
	_, err := s.DB.Exec(ctx, `
		UPDATE peers
		SET
			latency_vector = $1,
			updated_at = $2
		WHERE peer = $3
	`, vec, now, peerID)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) UpdatePeerGeo(ctx context.Context, peerID string, lat, lon *float64) error {
	now := util.NowUTC(ctx)

	_, err := s.DB.Exec(ctx, `
		UPDATE peers
		SET
			lat = $1,
			lon = $2,
			geo = CASE
				WHEN $1 IS NOT NULL AND $2 IS NOT NULL THEN ll_to_earth($1, $2)
				ELSE NULL
			END,
			updated_at = $3
		WHERE peer = $4
	`, lat, lon, now, peerID)
	return err
}

func (s *PostgresStore) MarkPeerAsActive(ctx context.Context, peerID string) error {
	now := util.NowUTC(ctx)

	_, err := s.DB.Exec(ctx, `
		UPDATE peers
		SET
			disconnected = FALSE,
			last_seen = $1,
			updated_at = $1
		WHERE peer = $2
	`, now, peerID)
	return err
}

func (s *PostgresStore) MarkPeerAsDisconnected(ctx context.Context, peerID string) error {
	now := util.NowUTC(ctx)

	_, err := s.DB.Exec(ctx, `
		UPDATE peers
		SET
			disconnected = TRUE,
			updated_at = $1
		WHERE peer = $2
	`, now, peerID)
	return err
}

func (s *PostgresStore) MarkPeerAsReconnected(ctx context.Context, peerID, secret, gameID string) (bool, []string, error) {
	now := util.NowUTC(ctx)

	result, err := s.DB.Exec(ctx, `
		UPDATE peers
		SET
			disconnected = FALSE,
			last_seen = $1,
			updated_at = $1
		WHERE peer = $2
		AND secret = $3
		AND game = $4
	`, now, peerID, secret, gameID)
	if err != nil {
		return false, nil, err
	}
	if result.RowsAffected() == 0 {
		return false, nil, nil
	}

	var lobbies []string
	rows, err := s.DB.Query(ctx, `
		SELECT
			code
		FROM lobbies
		WHERE $1 = ANY(peers)
		  AND game = $2
	`, peerID, gameID)
	if err != nil {
		return false, nil, err
	}

	for rows.Next() {
		var lobby string

		if err := rows.Scan(&lobby); err != nil {
			return false, nil, err
		}

		lobbies = append(lobbies, lobby)
	}

	if err = rows.Err(); err != nil {
		return false, nil, err
	}

	return true, lobbies, nil
}

func (s *PostgresStore) ClaimNextTimedOutPeer(ctx context.Context, threshold time.Duration) (string, bool, map[string][]string, error) {
	now := util.NowUTC(ctx)

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return "", false, nil, err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	// DELETE FROM peers will lock the row for this peer in this transaction.
	// This means we can safely remove the peer from lobbies without getting a
	// race condition with DoLeaderElection.
	// It is important that both ClaimNextTimedOutPeer and DoLeaderElection always
	// lock peers first to avoid deadlocks.

	var peerID string
	var disconnected bool
	err = tx.QueryRow(ctx, `
		WITH d AS (
			SELECT peer, disconnected
			FROM peers
			WHERE last_seen < $1
			LIMIT 1
		)
		DELETE FROM peers
		USING d
		WHERE peers.peer = d.peer
		RETURNING d.peer, d.disconnected
	`, now.Add(-threshold)).Scan(&peerID, &disconnected)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil, nil
		}
		return "", false, nil, err
	}

	gameLobbies := make(map[string][]string)

	rows, err := tx.Query(ctx, `
		UPDATE lobbies
		SET
			peers = array_remove(peers, $1),
			updated_at = $2
		WHERE $1 = ANY(peers)
		RETURNING game, code
	`, peerID, now)
	if err != nil {
		return "", false, nil, err
	}

	for rows.Next() {
		var game string
		var lobby string

		err = rows.Scan(&game, &lobby)
		if err != nil {
			return "", false, nil, err
		}

		gameLobbies[game] = append(gameLobbies[game], lobby)
	}

	if err = rows.Err(); err != nil {
		return "", false, nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return "", false, nil, err
	}

	return peerID, disconnected, gameLobbies, nil
}

// ResetAllPeerLastSeen will reset all last_seen.
// This is being called when the process restarts so it doesn't matter
// how long the process was down.
func (s *PostgresStore) ResetAllPeerLastSeen(ctx context.Context) error {
	now := util.NowUTC(ctx)

	_, err := s.DB.Exec(ctx, `
		UPDATE peers
		SET
			last_seen = $1,
			updated_at = $1
	`, now)
	return err
}

func (s *PostgresStore) CleanEmptyLobbies(ctx context.Context, olderThan time.Time) error {
	_, err := s.DB.Exec(ctx, `
		DELETE FROM lobbies
		WHERE updated_at < $1
		AND peers = '{}'
	`, olderThan)
	return err
}

// DoLeaderElection attempts to elect a leader for the given lobby. If a correct leader already exists it will return nil.
// If no leader can be elected, it will return an ElectionResult with a nil leader.
func (s *PostgresStore) DoLeaderElection(ctx context.Context, gameID, lobbyCode string) (*ElectionResult, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	// We need to lock the whole table as SELECT FOR UPDATE does not lock rows that do not exist yet
	// And we can't have timed out peers being added during the election.
	_, err = tx.Exec(ctx, `
		LOCK TABLE peers IN EXCLUSIVE MODE
	`)
	if err != nil {
		return nil, err
	}

	var timedOutPeers []string
	rows, err := tx.Query(ctx, `
		SELECT peer
		FROM peers
		WHERE disconnected = TRUE
	`)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	} else {
		defer rows.Close() //nolint:errcheck

		for rows.Next() {
			var peer string
			err = rows.Scan(&peer)
			if err != nil {
				return nil, err
			}
			timedOutPeers = append(timedOutPeers, peer)
		}

		if err = rows.Err(); err != nil {
			return nil, err
		}
	}

	var currentLeader string
	var currentTerm int
	var peers []string
	err = tx.QueryRow(ctx, `
		SELECT leader, term, peers
		FROM lobbies
		WHERE game = $1
		AND code = $2
		FOR UPDATE
	`, gameID, lobbyCode).Scan(&currentLeader, &currentTerm, &peers)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	needNewLeader := currentLeader == ""

	if !needNewLeader {
		found := slices.Contains(peers, currentLeader)
		if !found {
			needNewLeader = true
		}
	}

	if !needNewLeader {
		found := slices.Contains(timedOutPeers, currentLeader)
		if found {
			needNewLeader = true
		}
	}

	if !needNewLeader {
		return nil, nil
	}

	if isTestEnv {
		// In tests we want to have a deterministic leader.
		sort.Strings(peers)
	} else {
		// Randomize the order of the peers to avoid always picking the same leader.
		rand.Shuffle(len(peers), func(i, j int) {
			peers[i], peers[j] = peers[j], peers[i]
		})
	}

	newLeader := ""
	for _, peer := range peers {
		found := slices.Contains(timedOutPeers, peer)
		if !found {
			newLeader = peer
			break
		}
	}

	newTerm := currentTerm + 1

	now := util.NowUTC(ctx)
	_, err = tx.Exec(ctx, `
		UPDATE lobbies
		SET
			leader = $1,
			term = $2,
			updated_at = $3
		WHERE game = $4
		AND code = $5
	`, newLeader, newTerm, now, gameID, lobbyCode)
	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return &ElectionResult{
		Leader: newLeader,
		Term:   newTerm,
	}, nil
}

func (s *PostgresStore) UpdateLobby(ctx context.Context, game, lobbyCode, peerID string, options LobbyOptions) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	var leader string
	var currentCanUpdateBy string
	var creator string
	err = tx.QueryRow(ctx, `
		SELECT leader, can_update_by, creator
		FROM lobbies
		WHERE game = $1
		AND code = $2
		FOR UPDATE
	`, game, lobbyCode).Scan(&leader, &currentCanUpdateBy, &creator)
	if err != nil {
		return err
	}

	switch currentCanUpdateBy {
	case CanUpdateByAnyone:
		// No restrictions.
	case CanUpdateByCreator:
		if creator != peerID {
			return errors.New("not allowed: peer is not the creator")
		}
	case CanUpdateByLeader:
		if leader != peerID {
			return errors.New("not allowed: peer is not the leader")
		}
	default:
		return fmt.Errorf("invalid can_update_by value: %q", currentCanUpdateBy)
	}

	columns := make([]string, 0, 3)
	values := []any{game, lobbyCode}

	if options.Public != nil {
		columns = append(columns, fmt.Sprintf("public = $%d", len(values)+1))
		values = append(values, *options.Public)
	}
	if options.CustomData != nil {
		columns = append(columns, fmt.Sprintf("custom_data = $%d", len(values)+1))
		values = append(values, *options.CustomData)
	}
	if options.CanUpdateBy != nil {
		columns = append(columns, fmt.Sprintf("can_update_by = $%d", len(values)+1))
		values = append(values, *options.CanUpdateBy)
	}
	if options.Password != nil {
		var hashedPassword []byte

		if len(*options.Password) > 0 {
			hashedPassword, err = bcrypt.GenerateFromPassword([]byte(*options.Password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
		}

		columns = append(columns, fmt.Sprintf("password = $%d", len(values)+1))
		values = append(values, hashedPassword)
	}
	if options.MaxPlayers != nil {
		columns = append(columns, fmt.Sprintf("max_players = $%d", len(values)+1))
		values = append(values, *options.MaxPlayers)
	}

	if len(columns) == 0 {
		return nil
	}

	_, err = tx.Exec(ctx, `
		UPDATE lobbies
		SET `+strings.Join(columns, ", ")+`
		WHERE game = $1
		AND code = $2
	`, values...)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
