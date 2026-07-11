package cache_utils

import (
	"sync"
	"time"
)

// entry is one stored value with its absolute expiry. A zero expiresAt means
// no expiry.
type entry struct {
	value     string
	expiresAt time.Time
}

// memoryStore is a process-local key/value store with per-key TTL. DockVol runs
// as a single process on a single node, so an in-memory map with a mutex is a
// complete cache: there are no other instances to share state with. Expiry is
// lazy (checked on read); a background janitor would only reclaim memory sooner
// and is not needed at DockVol's key volume.
type memoryStore struct {
	mu      sync.RWMutex
	entries map[string]entry
}

var store = &memoryStore{entries: make(map[string]entry)}

func (m *memoryStore) get(key string) (string, bool) {
	m.mu.RLock()
	stored, exists := m.entries[key]
	m.mu.RUnlock()

	if !exists {
		return "", false
	}

	if !stored.expiresAt.IsZero() && time.Now().UTC().After(stored.expiresAt) {
		m.mu.Lock()
		if current, stillExists := m.entries[key]; stillExists && current.expiresAt.Equal(stored.expiresAt) {
			delete(m.entries, key)
		}
		m.mu.Unlock()

		return "", false
	}

	return stored.value, true
}

func (m *memoryStore) set(key, value string, ttl time.Duration) {
	expiresAt := time.Time{}
	if ttl > 0 {
		expiresAt = time.Now().UTC().Add(ttl)
	}

	m.mu.Lock()
	m.entries[key] = entry{value: value, expiresAt: expiresAt}
	m.mu.Unlock()
}

// getAndDelete atomically returns the live value and removes the key, so a token
// can be consumed exactly once even under concurrent callers.
func (m *memoryStore) getAndDelete(key string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stored, exists := m.entries[key]
	if !exists {
		return "", false
	}

	delete(m.entries, key)

	if !stored.expiresAt.IsZero() && time.Now().UTC().After(stored.expiresAt) {
		return "", false
	}

	return stored.value, true
}

func (m *memoryStore) delete(key string) {
	m.mu.Lock()
	delete(m.entries, key)
	m.mu.Unlock()
}

func (m *memoryStore) clear() {
	m.mu.Lock()
	m.entries = make(map[string]entry)
	m.mu.Unlock()
}

// FlushAll wipes every cached key. There is a single in-process keyspace, so it
// clears the same map as ClearAllCache.
func FlushAll() error {
	store.clear()

	return nil
}

func ClearAllCache() error {
	store.clear()

	return nil
}
