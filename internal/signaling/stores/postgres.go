package stores

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/koenbollen/logging"
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
		filter.WithNestedJSONB("custom_data", "code", "playerCount", "createdAt", "updatedAt", "latency"),
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
		rawCompressed := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
		l, err := base64.StdEncoding.Decode(rawCompressed, []byte(data))
		if err != nil {
			return fmt.Errorf("failed to decode payload: %w", err)
		}
		rawCompressed = rawCompressed[:l]

		raw, err := util.GzipDecompress(rawCompressed)
		if err != nil {
			return fmt.Errorf("failed to decompress payload: %w", err)
		}

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

	compressedData, err := util.GzipCompress(data)
	if err != nil {
		return fmt.Errorf("failed to gzip data: %w", err)
	}

	totalLength := base64.StdEncoding.EncodedLen(len(compressedData)) + len(topic) + 1
	if totalLength > 8000 {
		return fmt.Errorf("data too long for topic %q: %d", topic, totalLength)
	}
	encoded := base64.StdEncoding.EncodeToString(compressedData)
	payload := topic + ":" + encoded
	_, err = s.DB.Exec(ctx, `NOTIFY lobbies, '`+payload+`'`)
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

func (s *PostgresStore) JoinLobby(ctx context.Context, game, lobbyCode, peerID, password string) ([]string, error) {
	if len(peerID) > 20 {
		logger := logging.GetLogger(ctx)
		logger.Warn("peer id too long", zap.String("peerID", peerID))
		return nil, ErrInvalidPeerID
	}

	now := util.NowUTC(ctx)

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
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
			return nil, ErrNotFound
		}
		return nil, err
	}

	if lobbyPassword != nil && bcrypt.CompareHashAndPassword(lobbyPassword, []byte(password)) != nil {
		return nil, ErrInvalidPassword
	}

	if maxPlayers > 0 && len(peerlist) >= maxPlayers {
		return nil, ErrLobbyIsFull
	}

	if slices.Contains(peerlist, peerID) {
		return nil, ErrAlreadyInLobby
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
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return peerlist, nil
}

func (s *PostgresStore) JoinOrCreateLobby(ctx context.Context, game, peerID, country, region, lobbyFilter, codeFormat string, options LobbyOptions) (string, []string, bool, error) {
	if len(peerID) > 20 {
		return "", nil, false, ErrInvalidPeerID
	}

	if lobbyFilter == "" {
		lobbyFilter = "{}"
	}
	where, values, err := s.filterConverter.Convert([]byte(lobbyFilter), 4)
	if err != nil {
		return "", nil, false, fmt.Errorf("invalid filter: %w", err)
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return "", nil, false, err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	h := fnv.New64a()
	_, _ = h.Write([]byte(game))
	hash := int64(h.Sum64())

	// Lock the game before looking for a lobby, including when none exists yet.
	//
	// The SELECT FOR UPDATE below will only lock an existing lobby row, it won't help us
	// with the race condition where 2 players both don't find a lobby and try to create one simultaneously.
	//
	// pg_advisory_xact_lock automatically releases the lock at the end of the transaction.
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, hash)
	if err != nil {
		return "", nil, false, err
	}

	var lobbyCode string
	var peerlist []string
	err = tx.QueryRow(ctx, `
		SELECT code, peers
		FROM (
			SELECT *,
				COALESCE(ARRAY_LENGTH(peers, 1), 0) AS "playerCount",
				created_at AS "createdAt",
				updated_at AS "updatedAt",
				lobby_latency_estimate(peers, $2, $3) AS latency
			FROM lobbies
			WHERE game = $1
		) AS game_lobbies
		WHERE public = true
		AND password IS NULL
		AND COALESCE(ARRAY_LENGTH(peers, 1), 0) > 0
		AND (max_players = 0 OR ARRAY_LENGTH(peers, 1) < max_players)
		AND (`+where+`)
		ORDER BY RANDOM()
		LIMIT 1
		FOR UPDATE
	`, append([]any{game, country, region}, values...)...).Scan(&lobbyCode, &peerlist)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", nil, false, err
	}

	exists := !errors.Is(err, pgx.ErrNoRows)
	now := util.NowUTC(ctx)
	if !exists {
		for attempts := range 20 {
			lobbyCode = util.GenerateLobbyCodeWithFormat(ctx, codeFormat, attempts)
			res, err := tx.Exec(ctx, `
				INSERT INTO lobbies (code, game, peers, public, created_at, updated_at, leader, term, can_update_by, creator, max_players, custom_data)
				VALUES ($1, $2, $3, true, $4, $4, $5, 1, $6, $5, $7, $8)
				ON CONFLICT DO NOTHING
			`, lobbyCode, game, []string{peerID}, now, peerID, options.CanUpdateBy, options.MaxPlayers, options.CustomData)
			if err != nil {
				return "", nil, false, err
			}
			if res.RowsAffected() > 0 {
				break
			}
			if attempts == 19 {
				return "", nil, false, ErrLobbyExists
			}
		}
	} else {
		if slices.Contains(peerlist, peerID) {
			return "", nil, false, ErrAlreadyInLobby
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
			return "", nil, false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", nil, false, err
	}

	return lobbyCode, peerlist, !exists, nil
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

func (s *PostgresStore) GetPlayerCounts(ctx context.Context, game, country, region, filter string) (PlayerCounts, error) {
	var counts PlayerCounts
	if filter == "" {
		filter = "{}"
	}
	where, values, err := s.filterConverter.Convert([]byte(filter), 4)
	if err != nil {
		return counts, fmt.Errorf("invalid filter: %w", err)
	}
	err = s.DB.QueryRow(ctx, `
		WITH game_lobbies AS (
			SELECT code, peers, custom_data,
				COALESCE(ARRAY_LENGTH(peers, 1), 0) AS "playerCount",
				created_at AS "createdAt", updated_at AS "updatedAt",
				lobby_latency_estimate(peers, $2, $3) AS latency
			FROM lobbies
			WHERE game = $1
		)
		SELECT COUNT(*), COUNT(*) FILTER (WHERE EXISTS (
			SELECT 1 FROM game_lobbies
			WHERE peers.peer = ANY(game_lobbies.peers) AND (`+where+`)
		))
		FROM peers
		WHERE game = $1 AND disconnected = FALSE
	`, append([]any{game, country, region}, values...)...).Scan(&counts.Online, &counts.InLobby)
	return counts, err
}

func (s *PostgresStore) ListLobbies(ctx context.Context, game string, country, region string, filter, sort string, limit int) ([]Lobby, error) {
	// TODO: Remove this.
	if filter == "" {
		filter = "{}"
	}

	if limit <= 0 {
		limit = 50
	}

	preValues := []any{game, country, region, limit}

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
				lobby_latency_estimate(peers, $2, $3) AS latency
			FROM lobbies
			WHERE game = $1
			  AND public = true
		)
		SELECT *
		FROM game_lobbies
		WHERE `+where+`
		ORDER BY `+order+`
		LIMIT $4
	`, append(preValues, values...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var lobby Lobby
		err = rows.Scan(&lobby.Code, &lobby.PlayerCount, &lobby.Public, &lobby.CustomData, &lobby.CreatedAt, &lobby.UpdatedAt, &lobby.Leader, &lobby.Term, &lobby.CanUpdateBy, &lobby.Creator, &lobby.HasPassword, &lobby.MaxPlayers, &lobby.Latency)
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

func (s *PostgresStore) CreatePeer(ctx context.Context, peerID, secret, gameID, connectionID string) error {
	if len(peerID) > 20 {
		logger := logging.GetLogger(ctx)
		logger.Warn("peer id too long", zap.String("peerID", peerID))
		return ErrInvalidPeerID
	}

	now := util.NowUTC(ctx)
	_, err := s.DB.Exec(ctx, `
		INSERT INTO peers (peer, secret, game, last_seen, updated_at, connection_id)
		VALUES ($1, $2, $3, $4, $4, $5)
	`, peerID, secret, gameID, now, connectionID)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) UpdatePeerGeo(ctx context.Context, peerID string, country, region string) error {
	now := util.NowUTC(ctx)

	_, err := s.DB.Exec(ctx, `
		UPDATE peers
		SET
			country = CASE
				WHEN $1 = '' OR $1 = 'XX' THEN NULL
				ELSE $1
			END,
			region = NULLIF($2, ''),
			updated_at = $3
		WHERE peer = $4
	`, country, region, now, peerID)
	return err
}

func (s *PostgresStore) MarkPeerAsActive(ctx context.Context, peerID, connectionID string) error {
	// Only an authenticated reconnect may reactivate a disconnected peer.
	// Late heartbeats must not revive it or postpone timeout cleanup.
	now := util.NowUTC(ctx)

	_, err := s.DB.Exec(ctx, `
		UPDATE peers
		SET
			last_seen = $1,
			updated_at = $1
		WHERE peer = $2 AND connection_id = $3 AND disconnected = FALSE
	`, now, peerID, connectionID)
	return err
}

// MarkPeerAsDisconnected ignores handlers superseded by an authenticated reconnect.
func (s *PostgresStore) MarkPeerAsDisconnected(ctx context.Context, peerID, connectionID string) (bool, error) {
	now := util.NowUTC(ctx)
	result, err := s.DB.Exec(ctx, `
		UPDATE peers
		SET disconnected = TRUE, updated_at = $1
		WHERE peer = $2 AND connection_id = $3
	`, now, peerID, connectionID)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

func (s *PostgresStore) MarkPeerAsReconnected(ctx context.Context, peerID, secret, gameID, connectionID string) (bool, []string, error) {
	now := util.NowUTC(ctx)

	result, err := s.DB.Exec(ctx, `
		UPDATE peers
		SET
			disconnected = FALSE,
			connection_id = $5,
			last_seen = $1,
			updated_at = $1
		WHERE peer = $2
		AND secret = $3
		AND game = $4
	`, now, peerID, secret, gameID, connectionID)
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

func (s *PostgresStore) ClaimTimedOutPeers(ctx context.Context, threshold time.Duration, limit int) ([]TimedOutPeer, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("cleanup limit must be positive")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	now := util.NowUTC(ctx)
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	rows, err := tx.Query(ctx, `
		WITH expired AS (
			SELECT peer FROM peers
			WHERE last_seen < $1
			ORDER BY last_seen, peer
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		DELETE FROM peers p USING expired e
		WHERE p.peer = e.peer
		RETURNING p.peer
	`, now.Add(-threshold), limit)
	if err != nil {
		return nil, err
	}
	var result []TimedOutPeer
	var ids []string
	indexes := make(map[string]int)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		indexes[id] = len(result)
		ids = append(ids, id)
		result = append(result, TimedOutPeer{ID: id, GameLobbies: make(map[string][]string)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}

	// Lock each affected lobby once, retaining its old membership for notifications.
	// Array overlap uses lobbies_peers; ANY(peers) cannot use that GIN index.
	rows, err = tx.Query(ctx, `
		WITH affected AS MATERIALIZED (
			SELECT game, code, peers FROM lobbies
			WHERE peers && $1::varchar(20)[]
			ORDER BY game, code
			FOR UPDATE
		)
		UPDATE lobbies l
		SET peers = ARRAY(SELECT p FROM unnest(l.peers) AS p WHERE p <> ALL($1::varchar(20)[])),
			updated_at = $2
		FROM affected a
		WHERE l.game = a.game AND l.code = a.code
		RETURNING a.game, a.code, a.peers
	`, ids, now)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var game, code string
		var members []string
		if err := rows.Scan(&game, &code, &members); err != nil {
			rows.Close()
			return nil, err
		}
		for _, id := range members {
			if i, ok := indexes[id]; ok {
				result[i].GameLobbies[game] = append(result[i].GameLobbies[game], code)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
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

	// Locking the lobby row serializes membership and leader updates. We only
	// consider peers that still exist and are connected, so a peer deleted by
	// timeout cleanup cannot remain leader even before it is removed from peers[].
	connectedPeers := make([]string, 0, len(peers))
	if len(peers) > 0 {
		rows, err := tx.Query(ctx, `
			SELECT peer
			FROM peers
			WHERE game = $1
			AND peer = ANY($2)
			AND disconnected = FALSE
		`, gameID, peers)
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
				connectedPeers = append(connectedPeers, peer)
			}

			if err = rows.Err(); err != nil {
				return nil, err
			}
		}
	}

	needNewLeader := currentLeader == "" || !slices.Contains(connectedPeers, currentLeader)

	if !needNewLeader {
		return nil, nil
	}

	if isTestEnv {
		// In tests we want to have a deterministic leader.
		sort.Strings(connectedPeers)
	} else {
		// Randomize the order of connected peers to avoid always picking the same leader.
		rand.Shuffle(len(connectedPeers), func(i, j int) {
			connectedPeers[i], connectedPeers[j] = connectedPeers[j], connectedPeers[i]
		})
	}

	newLeader := ""
	if len(connectedPeers) > 0 {
		newLeader = connectedPeers[0]
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
