package stores

import "errors"

var ErrAlreadyInLobby = errors.New("peer already in lobby")
var ErrLobbyExists = errors.New("lobby already exists")
var ErrNotFound = errors.New("lobby not found")
var ErrNoSuchTopic = errors.New("no such topic")
