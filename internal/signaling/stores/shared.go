package stores

import (
	"context"
	"errors"
	"time"
)

var ErrAlreadyInLobby = errors.New("peer already in lobby")
var ErrLobbyExists = errors.New("lobby already exists")
var ErrNotFound = errors.New("lobby not found")
var ErrNoSuchTopic = errors.New("no such topic")
var ErrInvalidLobbyCode = errors.New("invalid lobby code")
var ErrInvalidPeerID = errors.New("invalid peer id")

type SubscriptionCallback func(context.Context, []byte)

type Store interface {
	CreateLobby(ctx context.Context, game, lobby, peerID string, public bool, customData map[string]any) error
	JoinLobby(ctx context.Context, game, lobby, id string) error
	IsPeerInLobby(ctx context.Context, game, lobby, id string) (bool, error)
	LeaveLobby(ctx context.Context, game, lobby, id string) error
	GetLobby(ctx context.Context, game, lobby string) (Lobby, error)
	ListLobbies(ctx context.Context, game, filter string) ([]Lobby, error)

	Subscribe(ctx context.Context, callback SubscriptionCallback, topics ...string)
	Publish(ctx context.Context, topic string, data []byte) error

	TimeoutPeer(ctx context.Context, peerID, secret, gameID string, lobbies []string) error
	ReconnectPeer(ctx context.Context, peerID, secret, gameID string) (bool, []string, error)
	ClaimNextTimedOutPeer(ctx context.Context, threshold time.Duration, callback func(peerID, gameID string, lobbies []string) error) (bool, error)

	CleanEmptyLobbies(ctx context.Context, olderThan time.Time) error
}

type Lobby struct {
	Code        string   `json:"code"`
	Peers       []string `json:"peers"`
	PlayerCount int      `json:"playerCount"`

	Public     bool           `json:"public"`
	MaxPlayers int            `json:"maxPlayers"`
	Password   string         `json:"password,omitempty"`
	CustomData map[string]any `json:"customData"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
