package stores

import "errors"

var ErrAlreadyInLobby = errors.New("peer already in lobby")
var ErrLobbyExists = errors.New("lobby already exists")
var ErrNotFound = errors.New("lobby not found")
var ErrNoSuchTopic = errors.New("no such topic")

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
