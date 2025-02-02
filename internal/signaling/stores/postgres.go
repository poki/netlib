package stores

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
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

type PostgresStore struct {
	DB *pgxpool.Pool

	mutex             sync.Mutex
	callbacks         map[string]map[uint64]SubscriptionCallback
	nextCallbackIndex uint64
	filterConverter   *filter.Converter
}

func NewPostgresStore(ctx context.Context, db *pgxpool.Pool) (*PostgresStore, error) {
	filterConverter, err := filter.NewConverter(
		filter.WithNestedJSONB("custom_data", "code", "playerCount", "createdAt", "updatedAt"),
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
	if len(topic) > 76 {
		return fmt.Errorf("topic too long")
	}
	if strings.ContainsRune(topic, ':') {
		return fmt.Errorf("topic contains : character")
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

	for _, peer := range peerlist {
		if peer == peerID {
			return ErrAlreadyInLobby
		}
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

func (s *PostgresStore) ListLobbies(ctx context.Context, game, filter string) ([]Lobby, error) {
	// TODO: Remove this.
	if filter == "" {
		filter = "{}"
	}

	where, values, err := s.filterConverter.Convert([]byte(filter), 2)
	if err != nil {
		logger := logging.GetLogger(ctx)
		logger.Warn("failed to convert filter", zap.String("filter", filter), zap.Error(err))
		return nil, fmt.Errorf("invalid filter: %w", err)
	}

	var lobbies []Lobby
	rows, err := s.DB.Query(ctx, `
		WITH lobbies AS (
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
				max_players
			FROM lobbies
			WHERE game = $1
			AND public = true
		)
		SELECT *
		FROM lobbies
		WHERE `+where+`
		ORDER BY "createdAt" DESC
		LIMIT 50
	`, append([]any{game}, values...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var lobby Lobby
		err = rows.Scan(&lobby.Code, &lobby.PlayerCount, &lobby.Public, &lobby.CustomData, &lobby.CreatedAt, &lobby.UpdatedAt, &lobby.Leader, &lobby.Term, &lobby.CanUpdateBy, &lobby.Creator, &lobby.HasPassword, &lobby.MaxPlayers)
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

func (s *PostgresStore) TimeoutPeer(ctx context.Context, peerID, secret, gameID string, lobbies []string) error {
	if len(peerID) > 20 {
		logger := logging.GetLogger(ctx)
		logger.Warn("peer id too long", zap.String("peerID", peerID))
		return ErrInvalidPeerID
	}
	for _, lobby := range lobbies {
		if len(lobby) > 20 {
			logger := logging.GetLogger(ctx)
			logger.Warn("lobby code too long", zap.String("lobbyCode", lobby))
			return ErrInvalidLobbyCode
		}
	}

	now := util.NowUTC(ctx)
	_, err := s.DB.Exec(ctx, `
		INSERT INTO timeouts (peer, secret, game, lobbies, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		ON CONFLICT (peer) DO UPDATE
		SET
			secret = $2,
			game = $3,
			lobbies = $4,
			last_seen = $5,
			updated_at = $5
	`, peerID, secret, gameID, lobbies, now)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) ReconnectPeer(ctx context.Context, peerID, secret, gameID string) (bool, []string, error) {
	var lobbies []string
	err := s.DB.QueryRow(ctx, `
		DELETE FROM timeouts
		WHERE peer = $1
		AND secret = $2
		AND game = $3
		RETURNING lobbies
	`, peerID, secret, gameID).Scan(&lobbies)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil, nil
		}
		return false, nil, err
	}
	if len(lobbies) == 0 {
		lobbies = nil
	}
	return true, lobbies, nil
}

func (s *PostgresStore) ClaimNextTimedOutPeer(ctx context.Context, threshold time.Duration, callback func(peerID, gameID string, lobbies []string) error) (more bool, err error) {
	now := util.NowUTC(ctx)

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	// DELETE FROM timeouts will lock the row for this peer in this transaction.
	// This means we can safely remove the peer from lobbies without getting a
	// race condition with DoLeaderElection.
	// It is important that both ClaimNextTimedOutPeer and DoLeaderElection always
	// lock timeouts first to avoid deadlocks.

	var peerID string
	var gameID string
	var lobbies []string
	err = tx.QueryRow(ctx, `
		WITH d AS (
			SELECT peer, game, lobbies
			FROM timeouts
			WHERE last_seen < $1
			LIMIT 1
		)
		DELETE FROM timeouts
		USING d
		WHERE timeouts.peer = d.peer
		AND timeouts.game = d.game
		RETURNING d.peer, d.game, d.lobbies
	`, now.Add(-threshold)).Scan(&peerID, &gameID, &lobbies)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if err := tx.Commit(ctx); err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return false, err
			}
			return false, nil
		}
		return false, err
	}

	for _, lobby := range lobbies {
		_, err := tx.Exec(ctx, `
			UPDATE lobbies
			SET
				peers = array_remove(peers, $1),
				updated_at = $2
			WHERE code = $3
			AND game = $4
		`, peerID, now, lobby, gameID)
		if err != nil {
			return false, err
		}
	}

	err = callback(peerID, gameID, lobbies)
	if err != nil {
		return false, err
	}

	return true, tx.Commit(ctx)
}

func (s *PostgresStore) MarkAllPeersAsActive(ctx context.Context) error {
	now := util.NowUTC(ctx)
	_, err := s.DB.Exec(ctx, `
		INSERT INTO peer_activity
		SELECT UNNEST(peers) AS peer, $1 AS updated_at
		FROM lobbies
		ON CONFLICT (peer) DO NOTHING
	`, now)
	return err
}

func (s *PostgresStore) UpdatePeerActivity(ctx context.Context, peerID string) error {
	_, err := s.DB.Exec(ctx, `
		INSERT INTO peer_activity (peer, updated_at)
		VALUES ($1, $2)
		ON CONFLICT (peer) DO UPDATE
		SET updated_at = $2
	`, peerID, util.NowUTC(ctx))
	return err
}

func (s *PostgresStore) RemovePeerActivity(ctx context.Context, peerID string) error {
	_, err := s.DB.Exec(ctx, `
		DELETE FROM peer_activity
		WHERE peer = $1
	`, peerID)
	return err
}

func (s *PostgresStore) ClaimNextInactivePeer(ctx context.Context, threshold time.Duration) (string, []string, []string, error) {
	now := util.NowUTC(ctx)

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return "", nil, nil, err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	var peerID string
	err = tx.QueryRow(ctx, `
		WITH d AS (
			SELECT peer
			FROM peer_activity
			WHERE updated_at < $1
			LIMIT 1
		)
		DELETE FROM peer_activity
		USING d
		WHERE peer_activity.peer = d.peer
		RETURNING d.peer
	`, now.Add(-threshold)).Scan(&peerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if err := tx.Commit(ctx); err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return "", nil, nil, err
			}

			rows, err := s.DB.Query(ctx, `
				SELECT peer, updated_at
				FROM peer_activity
			`)
			if err == nil {
				for rows.Next() {
					var peer string
					var updated time.Time

					err = rows.Scan(&peer, &updated)
					if err == nil {
						fmt.Printf("peer active: %s, since: %s\n", peer, now.Sub(updated))
					}
				}
			}

			return "", nil, nil, nil
		}
		return "", nil, nil, err
	}

	var games []string
	var lobbies []string

	rows, err := tx.Query(ctx, `
		UPDATE lobbies
		SET
			peers = array_remove(peers, $1),
			updated_at = $2
		WHERE $1 = ANY(peers)
		RETURNING game, code
	`, peerID, now)
	if err != nil {
		return "", nil, nil, err
	}

	for rows.Next() {
		var game string
		var lobby string

		err = rows.Scan(&game, &lobby)
		if err != nil {
			return "", nil, nil, err
		}

		games = append(games, game)
		lobbies = append(lobbies, lobby)
	}

	if err = rows.Err(); err != nil {
		return "", nil, nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return "", nil, nil, err
	}

	return peerID, games, lobbies, nil
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
		LOCK TABLE timeouts IN EXCLUSIVE MODE
	`)
	if err != nil {
		return nil, err
	}

	var timedOutPeers []string
	rows, err := tx.Query(ctx, `
		SELECT peer
		FROM timeouts
		WHERE game = $1
		AND $2 = ANY(lobbies)
	`, gameID, lobbyCode)
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

	needNewLeader := false
	if currentLeader == "" {
		needNewLeader = true
	}

	if !needNewLeader {
		found := false
		for _, peer := range peers {
			if currentLeader == peer {
				found = true
				break
			}
		}
		if !found {
			needNewLeader = true
		}
	}

	if !needNewLeader {
		found := false
		for _, peer := range timedOutPeers {
			if currentLeader == peer {
				found = true
				break
			}
		}
		if found {
			needNewLeader = true
		}
	}

	if !needNewLeader {
		return nil, nil
	}

	// Randomize the order of the peers to avoid always picking the same leader.
	rand.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})

	newLeader := ""
	for _, peer := range peers {
		found := false
		for _, timedOutPeer := range timedOutPeers {
			if peer == timedOutPeer {
				found = true
				break
			}
		}
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
