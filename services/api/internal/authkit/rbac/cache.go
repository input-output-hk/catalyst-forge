package rbac

import (
	"sync"
	"time"
)

type roleKey struct {
	slug string
	ver  int64
}

type principalVal struct {
	entries []RoleEntry
	exp     time.Time
}

type roleVal struct {
	entries []RoleEntry
	exp     time.Time
}

// MemoryCache is a simple in-memory Cache implementation for RBAC.
type MemoryCache struct {
	mu         sync.RWMutex
	roles      map[roleKey]roleVal
	principals map[string]principalVal
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		roles:      make(map[roleKey]roleVal),
		principals: make(map[string]principalVal),
	}
}

func (m *MemoryCache) GetRoleEntries(slug string, version int64) ([]RoleEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.roles[roleKey{slug: slug, ver: version}]
	if !ok {
		return nil, false
	}
	if !v.exp.IsZero() && time.Now().After(v.exp) {
		return nil, false
	}
	return v.entries, true
}

func (m *MemoryCache) SetRoleEntries(slug string, version int64, entries []RoleEntry, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	exp := time.Time{}
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	m.roles[roleKey{slug: slug, ver: version}] = roleVal{entries: entries, exp: exp}
}

func (m *MemoryCache) GetPrincipal(key string) ([]RoleEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.principals[key]
	if !ok {
		return nil, false
	}
	if !v.exp.IsZero() && time.Now().After(v.exp) {
		return nil, false
	}
	return v.entries, true
}

func (m *MemoryCache) SetPrincipal(key string, entries []RoleEntry, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	exp := time.Time{}
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	m.principals[key] = principalVal{entries: entries, exp: exp}
}
