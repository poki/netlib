package stores

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
)

type Lobby struct {
	MaxPlayers int
	Players    map[string]struct{}
}

type Memory struct {
	Lobbies map[string]*Lobby

	mutex  *sync.Mutex
	topics map[string]map[chan []byte]struct{}
}

func NewMemoryStore() *Memory {
	m := &Memory{}
	m.Lobbies = make(map[string]*Lobby)
	m.mutex = &sync.Mutex{}
	m.topics = make(map[string]map[chan []byte]struct{})
	return m
}

func (m *Memory) CreateLobby(ctx context.Context, game, lobby, id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := game + lobby
	_, found := m.Lobbies[key]
	if found {
		return ErrLobbyExists
	}

	m.Lobbies[key] = &Lobby{
		MaxPlayers: 8, // TODO: combine some args into a struct and add this.
		Players: map[string]struct{}{
			id: {},
		},
	}

	return nil
}

func (m *Memory) JoinLobby(ctx context.Context, game, lobby, id string) ([]string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := game + lobby

	lb, found := m.Lobbies[key]
	if !found {
		return nil, ErrNotFound
	}

	_, found = lb.Players[id]
	if found {
		return nil, ErrAlreadyInLobby
	}

	if len(lb.Players) >= lb.MaxPlayers {
		return nil, ErrLobbyFull
	}

	peerlist := []string{}
	for id := range lb.Players {
		peerlist = append(peerlist, id)
	}

	lb.Players[id] = struct{}{}

	return peerlist, nil
}

// JoinOrCreateLobby joins the first none-full lobby with lobbyPrefix or
// otherwise creates a new lobby with a random identifier starting with lobbyPrefix.
func (m *Memory) JoinOrCreateLobby(ctx context.Context, game, lobbyPrefix, id string) (string, []string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	prefix := game + lobbyPrefix

	lobby := ""
	peerlist := []string{}

	for key, lb := range m.Lobbies {
		if strings.HasPrefix(key, prefix) && len(lb.Players) < lb.MaxPlayers {
			for id := range lb.Players {
				peerlist = append(peerlist, id)
			}

			lobby = key
			lb.Players[id] = struct{}{}

			break
		}
	}

	if lobby == "" {
		for {
			lobby = game + lobbyPrefix + fmt.Sprintf("%d", rand.Int63())
			_, found := m.Lobbies[lobby]
			if !found {
				break
			}
		}

		m.Lobbies[lobby] = &Lobby{
			MaxPlayers: 2, // TODO: combine some args into a struct and add this.
			Players: map[string]struct{}{
				id: {},
			},
		}
	}

	return lobby, peerlist, nil
}

func (m *Memory) Subscribe(ctx context.Context, topic string, callback func(context.Context, []byte)) {
	m.mutex.Lock()
	if _, found := m.topics[topic]; !found {
		m.topics[topic] = make(map[chan []byte]struct{})
	}
	channel := make(chan []byte)
	m.topics[topic][channel] = struct{}{}
	m.mutex.Unlock()

	go func() {
		defer func() {
			m.mutex.Lock()
			close(channel)
			if channels, found := m.topics[topic]; found {
				delete(channels, channel)
				if len(channels) == 0 {
					delete(m.topics, topic)
				}
			}
			m.mutex.Unlock()
		}()

		for ctx.Err() == nil {
			select {
			case payload := <-channel:
				callback(ctx, payload)
			case <-ctx.Done():
				break
			}
		}
	}()
}

func (m *Memory) Publish(ctx context.Context, topic string, data []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.topics == nil {
		return ErrNoSuchTopic
	}
	channels, found := m.topics[topic]
	if !found {
		return ErrNoSuchTopic
	}

	for channel := range channels {
		select {
		case channel <- data:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
