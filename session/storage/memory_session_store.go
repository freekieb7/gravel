package storage

import (
	"errors"

	"github.com/freekieb7/gravel/session"
)

var ErrSessionNotFound = errors.New("session store: session not found")

const MemorySessionStoreName = "memory"

type MemorySessionStore struct {
	data map[string]map[string]any
}

func NewMemorySessionStore() SessionStore {
	return &MemorySessionStore{
		data: make(map[string]map[string]any),
	}
}

func (m *MemorySessionStore) Close() error {
	return nil
}

func (m *MemorySessionStore) Has(id string) bool {
	_, found := m.data[id]
	return found
}

func (m *MemorySessionStore) Get(id string) (map[string]any, error) {
	data, found := m.data[id]
	if !found {
		return nil, ErrSessionNotFound
	}

	return data, nil
}

func (m *MemorySessionStore) Save(session session.Session) error {
	m.data[session.GetId()] = session.All()
	return nil
}

func (m *MemorySessionStore) Delete(id string) error {
	delete(m.data, id)
	return nil
}
