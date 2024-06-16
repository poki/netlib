package stores

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
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
)

type PostgresStore struct {
	DB *pgxpool.Pool

	mutex             sync.Mutex
	callbacks         map[string]map[uint64]SubscriptionCallback
	nextCallbackIndex uint64
	filterConverter   *filter.Converter
}

func NewPostgresStore(ctx context.Context, db *pgxpool.Pool) (*PostgresStore, error) {
	s := &PostgresStore{
		DB:        db,
		callbacks: make(map[string]map[uint64]SubscriptionCallback),
		filterConverter: filter.NewConverter(
			filter.WithNestedJSONB("meta", "code", "playerCount", "createdAt", "updatedAt"),
			filter.WithEmptyCondition("TRUE"), // No filter returns all lobbies.
		),
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

func (s *PostgresStore) Subscribe(ctx context.Context, topic string, callback SubscriptionCallback) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, found := s.callbacks[topic]; !found {
		s.callbacks[topic] = make(map[uint64]SubscriptionCallback)
	}

	id := s.nextCallbackIndex
	s.nextCallbackIndex += 1
	s.callbacks[topic][id] = callback

	go func() {
		defer func() {
			s.mutex.Lock()
			defer s.mutex.Unlock()

			delete(s.callbacks[topic], id)
			if len(s.callbacks[topic]) == 0 {
				delete(s.callbacks, topic)
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

func (s *PostgresStore) CreateLobby(ctx context.Context, game, lobbyCode, peerID string, public bool, customData map[string]any) error {
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
	now := util.Now(ctx)
	res, err := s.DB.Exec(ctx, `
		INSERT INTO lobbies (code, game, peers, public, meta, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		ON CONFLICT DO NOTHING
	`, lobbyCode, game, []string{peerID}, public, customData, now)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrLobbyExists
	}
	return nil
}

func (s *PostgresStore) JoinLobby(ctx context.Context, game, lobbyCode, peerID string) error {
	if len(peerID) > 20 {
		logger := logging.GetLogger(ctx)
		logger.Warn("peer id too long", zap.String("peerID", peerID))
		return ErrInvalidPeerID
	}

	now := util.Now(ctx)

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	var peerlist []string
	err = tx.QueryRow(ctx, `
		SELECT peers
		FROM lobbies
		WHERE code = $1
		AND game = $2
		FOR UPDATE
	`, lobbyCode, game).Scan(&peerlist)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
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
	now := util.Now(ctx)

	_, err := s.DB.Exec(ctx, `
		UPDATE lobbies
		SET
			peers = array_remove(peers, $1),
			updated_at = $2
		WHERE code = $3
		AND game = $4
		RETURNING peers
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
			meta,
			created_at AS "createdAt",
			updated_at AS "updatedAt"
		FROM lobbies
		WHERE code = $1
		AND game = $2
	`, lobbyCode, game).Scan(&lobby.Code, &lobby.Peers, &lobby.PlayerCount, &lobby.Public, &lobby.CustomData, &lobby.CreatedAt, &lobby.UpdatedAt)
	if err != nil {
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
				meta,
				created_at AS "createdAt",
				updated_at AS "updatedAt"
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
		err = rows.Scan(&lobby.Code, &lobby.PlayerCount, &lobby.Public, &lobby.CustomData, &lobby.CreatedAt, &lobby.UpdatedAt)
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

	now := util.Now(ctx)
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
	now := util.Now(ctx)

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

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
	err = callback(peerID, gameID, lobbies)
	if err != nil {
		return false, err
	}

	return true, tx.Commit(ctx)
}

func (s *PostgresStore) CleanEmptyLobbies(ctx context.Context, olderThan time.Time) error {
	_, err := s.DB.Exec(ctx, `
		DELETE FROM lobbies
		WHERE updated_at < $1
		AND peers = '{}'
	`, olderThan)
	return err
}
