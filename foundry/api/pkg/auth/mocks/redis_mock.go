package mocks

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// MockRedisClient is a mock implementation of the Redis client for testing
type MockRedisClient struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

// NewMockRedisClient creates a new mock Redis client
func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data: make(map[string]interface{}),
	}
}

// Set mocks the Redis SET command
func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
	return redis.NewStatusCmd(ctx, "OK")
}

// Get mocks the Redis GET command
func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.data[key]
	if !exists {
		return redis.NewStringCmd(ctx, redis.Nil)
	}

	return redis.NewStringCmd(ctx, value.(string))
}

// Del mocks the Redis DEL command
func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	deleted := int64(0)
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			deleted++
		}
	}

	return redis.NewIntCmd(ctx, deleted)
}

// Exists mocks the Redis EXISTS command
func (m *MockRedisClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	exists := int64(0)
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			exists++
		}
	}

	return redis.NewIntCmd(ctx, exists)
}

// FlushAll mocks the Redis FLUSHALL command
func (m *MockRedisClient) FlushAll(ctx context.Context) *redis.StatusCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[string]interface{})
	return redis.NewStatusCmd(ctx, "OK")
}

// Close mocks the Redis CLOSE command
func (m *MockRedisClient) Close() error {
	return nil
}

// Ping mocks the Redis PING command
func (m *MockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	return redis.NewStatusCmd(ctx, "PONG")
}

// MockRedisCmd is a mock implementation of redis.Cmd
type MockRedisCmd struct {
	val interface{}
	err error
}

func (m *MockRedisCmd) Result() (interface{}, error) {
	return m.val, m.err
}

func (m *MockRedisCmd) Err() error {
	return m.err
}

func (m *MockRedisCmd) String() (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if str, ok := m.val.(string); ok {
		return str, nil
	}
	return "", errors.New("value is not a string")
}

func (m *MockRedisCmd) Int64() (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	if i, ok := m.val.(int64); ok {
		return i, nil
	}
	return 0, errors.New("value is not an int64")
}
