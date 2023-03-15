package stores

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/koenbollen/logging"
	"go.uber.org/zap"
)

type notificationPayload struct {
	Topic string `json:"t"`
	Data  []byte `json:"d"`
}

type PostgresStore struct {
	DB *pgxpool.Pool

	mutex             sync.Mutex
	callbacks         map[string]map[uint64]SubscriptionCallback
	nextCallbackIndex uint64
}

func NewPostgresStore(ctx context.Context, db *pgxpool.Pool) (*PostgresStore, error) {
	s := &PostgresStore{
		DB:        db,
		callbacks: make(map[string]map[uint64]SubscriptionCallback),
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

func (s *PostgresStore) CreateLobby(ctx context.Context, game, lobbyCode, peerID string) error {
	res, err := s.DB.Exec(ctx, `
		INSERT INTO lobbies (code, game, public)
		VALUES ($1, $2, true)
		ON CONFLICT DO NOTHING
	`, lobbyCode, game)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrAlreadyInLobby
	}
	return nil
}

func (s *PostgresStore) JoinLobby(ctx context.Context, game, lobbyCode, peerID string) ([]string, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

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
			return nil, ErrNotFound
		}
		return nil, err
	}

	for _, peer := range peerlist {
		if peer == peerID {
			return nil, ErrAlreadyInLobby
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE lobbies
		SET peers = array_append(peers, $1)
		WHERE code = $2
		AND game = $3
	`, peerID, lobbyCode, game)
	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return peerlist, nil
}

func (s *PostgresStore) IsPeerInLobby(ctx context.Context, game, lobbyCode, peerID string) (bool, error) {
	var count int
	err := s.DB.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM lobbies
		WHERE code = $1
		AND game = $2
		AND $3 = ANY(peers)
	`, lobbyCode, game, peerID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *PostgresStore) LeaveLobby(ctx context.Context, game, lobbyCode, peerID string) ([]string, error) {
	var peerlist []string
	err := s.DB.QueryRow(ctx, `
		UPDATE lobbies
		SET peers = array_remove(peers, $1)
		WHERE code = $2
		AND game = $3
		RETURNING peers
	`, peerID, lobbyCode, game).Scan(&peerlist)
	if err != nil {
		return nil, err
	}
	return peerlist, nil
}

func (s *PostgresStore) GetLobby(ctx context.Context, game, lobbyCode string) ([]string, error) {
	var peerlist []string
	err := s.DB.QueryRow(ctx, `
		SELECT peers
		FROM lobbies
		WHERE code = $1
		AND game = $2
	`, lobbyCode, game).Scan(&peerlist)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return peerlist, nil
}

func (s *PostgresStore) ListLobbies(ctx context.Context, game, filter string) ([]Lobby, error) {

	// TODO: Filters

	var lobbies []Lobby
	rows, err := s.DB.Query(ctx, `
		SELECT code, peers, meta
		FROM lobbies
		WHERE game = $1
		AND public = true
		ORDER BY created_at DESC
		LIMIT 50
	`, game)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var lobby Lobby
		var peers []string
		err = rows.Scan(&lobby.Code, &peers, &lobby.CustomData)
		if err != nil {
			return nil, err
		}
		lobby.PlayerCount = len(peers)
		lobbies = append(lobbies, lobby)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return lobbies, nil
}
