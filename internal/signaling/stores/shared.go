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

type SubscriptionCallback func(context.Context, []byte)

type Store interface {
	CreateLobby(ctx context.Context, game, lobby, id string) error
	JoinLobby(ctx context.Context, game, lobby, id string) ([]string, error)
	IsPeerInLobby(ctx context.Context, game, lobby, id string) (bool, error)
	LeaveLobby(ctx context.Context, game, lobby, id string) ([]string, error)
	GetLobby(ctx context.Context, game, lobby string) ([]string, error)
	ListLobbies(ctx context.Context, game, filter string) ([]Lobby, error)

	Subscribe(ctx context.Context, topic string, callback SubscriptionCallback)
	Publish(ctx context.Context, topic string, data []byte) error

	TimeoutPeer(ctx context.Context, peerID, secret, gameID string, lobbies []string) error
	ReconnectPeer(ctx context.Context, peerID, secret, gameID string) (bool, error)
	ClaimNextTimedOutPeer(ctx context.Context, threshold time.Duration, callback func(peerID string, lobbies []string) error) (bool, error)
}

type Lobby struct {
	Code        string `json:"code"`
	PlayerCount int    `json:"playerCount"`

	Public     bool           `json:"public"`
	MaxPlayers int            `json:"maxPlayers"`
	Password   string         `json:"password"`
	CustomData map[string]any `json:"customData"`

	peers map[string]struct{}
}

func (l *Lobby) Build() Lobby {
	clone := Lobby{
		Code:        l.Code,
		PlayerCount: len(l.peers),
		Public:      l.Public,
		MaxPlayers:  l.MaxPlayers,
		Password:    l.Password,
		CustomData:  l.CustomData,
		peers:       make(map[string]struct{}),
	}
	for k, v := range l.CustomData {
		clone.CustomData[k] = v
	}
	for id := range l.peers {
		clone.peers[id] = struct{}{}
	}
	return clone
}
