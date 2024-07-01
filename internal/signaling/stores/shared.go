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
var ErrNotAllowed = errors.New("not allowed")

type SubscriptionCallback func(context.Context, []byte)

type Store interface {
	CreateLobby(ctx context.Context, game, lobby, peerID string, public bool, customData map[string]any, canUpdateBy string) error
	JoinLobby(ctx context.Context, game, lobby, id string) error
	LeaveLobby(ctx context.Context, game, lobby, id string) error
	GetLobby(ctx context.Context, game, lobby string) (Lobby, error)
	ListLobbies(ctx context.Context, game, filter string) ([]Lobby, error)

	Subscribe(ctx context.Context, callback SubscriptionCallback, game, lobby, peerID string)
	Publish(ctx context.Context, topic string, data []byte) error

	TimeoutPeer(ctx context.Context, peerID, secret, gameID string, lobbies []string) error
	ReconnectPeer(ctx context.Context, peerID, secret, gameID string) (bool, []string, error)
	ClaimNextTimedOutPeer(ctx context.Context, threshold time.Duration, callback func(peerID, gameID string, lobbies []string) error) (bool, error)

	CleanEmptyLobbies(ctx context.Context, olderThan time.Time) error

	// DoLeaderElection attempts to elect a leader for the given lobby. If a correct leader already exists it will return nil.
	// If no leader can be elected, it will return an ElectionResult with a nil leader.
	DoLeaderElection(ctx context.Context, gameID, lobbyCode string) (*ElectionResult, error)

	UpdateCustomData(ctx context.Context, game, lobby, peer string, public *bool, customData *map[string]any, canUpdateBy *string) error
}

const (
	CanUpdateByCreator = "creator"
	CanUpdateByLeader  = "leader"
	CanUpdateByAnyone  = "anyone"
	CanUpdateByNone    = "none"
)

type Lobby struct {
	Code        string   `json:"code"`
	Peers       []string `json:"peers"`
	PlayerCount int      `json:"playerCount"`
	Creator     string   `json:"creator"`

	Public      bool           `json:"public"`
	MaxPlayers  int            `json:"maxPlayers"`
	Password    string         `json:"password,omitempty"`
	CustomData  map[string]any `json:"customData"`
	CanUpdateBy string         `json:"canUpdateBy"`

	Leader string `json:"leader,omitempty"`
	Term   int    `json:"term"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ElectionResult struct {
	Leader string
	Term   int
}
