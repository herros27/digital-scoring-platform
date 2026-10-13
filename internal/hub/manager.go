package hub

import (
	"fmt"
	"sync"
)

type Manager struct {
	mu   sync.RWMutex
	hubs map[string]*Hub
}

func NewManager() *Manager {
	return &Manager{
		hubs: make(map[string]*Hub),
	}
}

func (m *Manager) GetOrCreate(arenaID string) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()

	if h, ok := m.hubs[arenaID]; ok {
		return h
	}

	h := NewHub(arenaID)
	m.hubs[arenaID] = h
	go h.Run()

	return h
}

func (m *Manager) Get(arenaID string) (*Hub, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	h, ok := m.hubs[arenaID]
	if !ok {
		return nil, fmt.Errorf("arena hub not found: %s", arenaID)
	}
	return h, nil
}
