package stores

import (
	"context"
	"strings"
	"sync"
)

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

func (m *Memory) DebugTotalLobbyCount() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return len(m.Lobbies)
}

func (m *Memory) CreateLobby(ctx context.Context, game, lobbyCode, id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := game + lobbyCode
	_, found := m.Lobbies[key]
	if found {
		return ErrLobbyExists
	}

	m.Lobbies[key] = &Lobby{
		Code:   lobbyCode,
		Public: true,
		peers:  make(map[string]struct{}),
	}

	return nil
}

func (m *Memory) JoinLobby(ctx context.Context, game, lobbyCode, id string) ([]string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := game + lobbyCode

	lobby, found := m.Lobbies[key]
	if !found {
		return nil, ErrNotFound
	}

	_, found = lobby.peers[id]
	if found {
		return nil, ErrAlreadyInLobby
	}

	peerlist := []string{}
	for id := range lobby.peers {
		peerlist = append(peerlist, id)
	}

	lobby.peers[id] = struct{}{}

	go func() {
		<-ctx.Done()
		m.mutex.Lock()
		defer m.mutex.Unlock()
		if lobby, ok := m.Lobbies[key]; ok {
			delete(lobby.peers, id)
			if len(lobby.peers) == 0 {
				delete(m.Lobbies, key)
			}
		}
	}()

	return peerlist, nil
}

func (m *Memory) LeaveLobby(ctx context.Context, game, lobbyCode, id string) ([]string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := game + lobbyCode

	lobby, found := m.Lobbies[key]
	if !found {
		return nil, nil
	}

	_, found = lobby.peers[id]
	if !found {
		return nil, nil
	}

	delete(lobby.peers, id)

	if len(lobby.peers) == 0 {
		delete(m.Lobbies, key)
		return []string{}, nil
	}

	peerlist := []string{}
	for id := range lobby.peers {
		peerlist = append(peerlist, id)
	}

	return peerlist, nil
}

func (m *Memory) GetLobby(ctx context.Context, game, lobbyCode string) ([]string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := game + lobbyCode

	lobby, found := m.Lobbies[key]
	if !found {
		return nil, ErrNotFound
	}

	peerlist := []string{}
	for id := range lobby.peers {
		peerlist = append(peerlist, id)
	}

	return peerlist, nil
}

func (m *Memory) ListLobbies(ctx context.Context, game, filter string) ([]Lobby, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	lobbies := []Lobby{}
	for key, lobby := range m.Lobbies {
		if !strings.HasPrefix(key, game) || !lobby.Public {
			continue
		}

		// TODO: Filter lobby on given filter

		lobbies = append(lobbies, lobby.Build())

		if len(lobbies) >= 50 {
			break
		}
	}

	return lobbies, nil
}

func (m *Memory) IsPeerInLobby(ctx context.Context, game, lobbyCode, id string) (bool, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := game + lobbyCode

	lobby, found := m.Lobbies[key]
	if !found {
		return false, ErrNotFound
	}

	_, found = lobby.peers[id]
	return found, nil
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
			if channels, found := m.topics[topic]; found {
				delete(channels, channel)
				if len(channels) == 0 {
					delete(m.topics, topic)
				}
			}
			m.mutex.Unlock()

			close(channel)
		}()

		for ctx.Err() == nil {
			select {
			case payload := <-channel:
				callback(ctx, payload)
			case <-ctx.Done():
			}
		}
	}()
}

func (m *Memory) Publish(ctx context.Context, topic string, data []byte) error {
	// Use a closure here to be able to defer m.mutex.Unlock() without
	// having the channel sending code below blocking the mutex.
	channels, err := func() (map[chan []byte]struct{}, error) {
		m.mutex.Lock()
		defer m.mutex.Unlock()

		if m.topics == nil {
			return nil, ErrNoSuchTopic
		}
		channels, found := m.topics[topic]
		if !found {
			return nil, ErrNoSuchTopic
		}
		return channels, nil
	}()
	if err != nil {
		return err
	}

	for channel := range channels {
		err := func() error {
			// The send on channel can panic as the channel can be closed after we
			// got it from the topics. We just recover from the panic and ignore it as
			// the receiving party has disconnected already.
			_ = recover()

			select {
			case channel <- data:
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
}
