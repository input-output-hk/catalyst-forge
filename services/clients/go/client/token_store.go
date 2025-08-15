package client

import "sync/atomic"

// TokenStore holds an access token in a thread-safe way.
type TokenStore struct{ v atomic.Value }

// NewTokenStore creates an empty token store.
func NewTokenStore() *TokenStore { return &TokenStore{} }

// Get returns the current token or empty string if none is set.
func (s *TokenStore) Get() string {
	if x := s.v.Load(); x != nil {
		if t, ok := x.(string); ok {
			return t
		}
	}
	return ""
}

// Set updates the current token.
func (s *TokenStore) Set(token string) { s.v.Store(token) }
