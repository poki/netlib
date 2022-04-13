package stores

import (
	"context"
	"sync"
)

type Memory struct {
	Lobbies map[string]map[string]struct{}

	mutex  *sync.Mutex
	topics map[string]map[chan []byte]struct{}
}

func NewMemoryStore() *Memory {
	m := &Memory{}
	m.Lobbies = make(map[string]map[string]struct{})
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

	m.Lobbies[key] = make(map[string]struct{})

	return nil
}

func (m *Memory) JoinLobby(ctx context.Context, game, lobby, id string) ([]string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := game + lobby

	peers, found := m.Lobbies[key]
	if !found {
		return nil, ErrNotFound
	}

	_, found = peers[id]
	if found {
		return nil, ErrAlreadyInLobby
	}

	peerlist := []string{}
	for id := range peers {
		peerlist = append(peerlist, id)
	}

	m.Lobbies[key][id] = struct{}{}

	go func() {
		<-ctx.Done()
		m.mutex.Lock()
		defer m.mutex.Unlock()
		delete(m.Lobbies[key], id)
		if len(m.Lobbies[key]) == 0 {
			delete(m.Lobbies, key)
		}
	}()

	return peerlist, nil
}

func (m *Memory) GetLobby(ctx context.Context, game, lobby string) ([]string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := game + lobby

	peers, found := m.Lobbies[key]
	if !found {
		return nil, ErrNotFound
	}

	peerlist := []string{}
	for id := range peers {
		peerlist = append(peerlist, id)
	}

	return peerlist, nil
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
