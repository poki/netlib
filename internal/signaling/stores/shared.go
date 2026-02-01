package stores

import (
	"context"
	"errors"
	"time"
)

var ErrAlreadyInLobby = errors.New("peer already in lobby")
var ErrLobbyExists = errors.New("lobby already exists")
var ErrNotFound = errors.New("lobby not found")
var ErrInvalidLobbyCode = errors.New("invalid lobby code")
var ErrInvalidPeerID = errors.New("invalid peer id")
var ErrInvalidPassword = errors.New("invalid password")
var ErrLobbyIsFull = errors.New("lobby is full")

type SubscriptionCallback func(context.Context, []byte)

type LobbyOptions struct {
	Public      *bool
	CustomData  *map[string]any
	CanUpdateBy *string
	Password    *string
	MaxPlayers  *int
}

type Store interface {
	CreateLobby(ctx context.Context, Game, LobbyCode, PeerID string, options LobbyOptions) error
	JoinLobby(ctx context.Context, game, lobby, id, password string) error
	LeaveLobby(ctx context.Context, game, lobby, id string) error
	GetLobby(ctx context.Context, game, lobby string) (Lobby, error)
	ListLobbies(ctx context.Context, game string, filter, sort string, limit int) ([]Lobby, error)

	Subscribe(ctx context.Context, callback SubscriptionCallback, game, lobby, peerID string)
	Publish(ctx context.Context, topic string, data []byte) error

	CreatePeer(ctx context.Context, peerID, secret, gameID string) error
	MarkPeerAsActive(ctx context.Context, peerID string) error
	MarkPeerAsDisconnected(ctx context.Context, peerID string) error
	MarkPeerAsReconnected(ctx context.Context, peerID, secret, gameID string) (bool, []string, error)
	ClaimNextTimedOutPeer(ctx context.Context, threshold time.Duration) (string, bool, map[string][]string, error)
	ResetAllPeerLastSeen(ctx context.Context) error

	CleanEmptyLobbies(ctx context.Context, olderThan time.Time) error

	// DoLeaderElection attempts to elect a leader for the given lobby. If a correct leader already exists it will return nil.
	// If no leader can be elected, it will return an ElectionResult with a nil leader.
	DoLeaderElection(ctx context.Context, gameID, lobbyCode string) (*ElectionResult, error)

	UpdateLobby(ctx context.Context, Game, LobbyCode, PeerID string, options LobbyOptions) error
}

const (
	CanUpdateByCreator = "creator"
	CanUpdateByLeader  = "leader"
	CanUpdateByAnyone  = "anyone"
	CanUpdateByNone    = "none"
)

type Lobby struct {
	Code        string   `json:"code"`
	Peers       []string `json:"peers,omitempty"`
	PlayerCount int      `json:"playerCount"`
	Creator     string   `json:"creator"`

	Public      bool           `json:"public"`
	MaxPlayers  int            `json:"maxPlayers"`
	HasPassword bool           `json:"hasPassword"`
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
