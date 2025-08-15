package rate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLimiter implements a Redis-based rate limiter.
type RedisLimiter struct {
	client redis.UniversalClient
	prefix string
}

// NewRedisLimiter creates a new Redis-based rate limiter.
func NewRedisLimiter(client redis.UniversalClient, prefix string) *RedisLimiter {
	if prefix == "" {
		prefix = "ratelimit"
	}
	
	return &RedisLimiter{
		client: client,
		prefix: prefix,
	}
}

// Allow checks if n requests are allowed for the given key within the specified duration.
func (rl *RedisLimiter) Allow(ctx context.Context, key Key, n int, per time.Duration) (bool, int, time.Time, error) {
	// Create a Redis key with window timestamp
	now := time.Now().UTC()
	window := now.Truncate(per)
	redisKey := fmt.Sprintf("%s:%s:%d", rl.prefix, string(key), window.Unix())
	
	// Get the limit for this key type
	limit := rl.getLimit(key, per)
	
	// Use a Lua script for atomic operation
	script := redis.NewScript(`
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local ttl = tonumber(ARGV[2])
		local n = tonumber(ARGV[3])
		
		local current = redis.call('GET', key)
		if current == false then
			current = 0
		else
			current = tonumber(current)
		end
		
		if current + n <= limit then
			local new = redis.call('INCRBY', key, n)
			redis.call('EXPIRE', key, ttl)
			return {1, limit - new}
		else
			return {0, limit - current}
		end
	`)
	
	result, err := script.Run(ctx, rl.client, []string{redisKey},
		limit, int(per.Seconds()), n).Result()
	if err != nil {
		return false, 0, time.Time{}, fmt.Errorf("redis script failed: %w", err)
	}
	
	vals, ok := result.([]interface{})
	if !ok || len(vals) != 2 {
		return false, 0, time.Time{}, fmt.Errorf("unexpected redis response")
	}
	
	allowed := vals[0].(int64) == 1
	remaining := int(vals[1].(int64))
	if remaining < 0 {
		remaining = 0
	}
	resetAt := window.Add(per)
	
	return allowed, remaining, resetAt, nil
}

// getLimit returns the rate limit for a given key and duration.
func (rl *RedisLimiter) getLimit(key Key, per time.Duration) int {
	keyStr := string(key)
	
	// Determine limits based on key prefix and duration
	switch {
	case strings.HasPrefix(keyStr, "login:"):
		// Login: 5 attempts per 15 minutes
		if per == 15*time.Minute {
			return 5
		}
		// Scale proportionally for other durations
		return int(5 * (per.Minutes() / 15))
		
	case strings.HasPrefix(keyStr, "invite:"):
		// Invite: 5 attempts per lifetime (use 10 per hour as backstop)
		return 10
		
	case strings.HasPrefix(keyStr, "recovery:"):
		// Recovery: 3 attempts per hour for init, 5 for verify
		if strings.Contains(keyStr, ":init") {
			if per == time.Hour {
				return 3
			}
			return int(3 * (per.Hours()))
		}
		// Recovery verify
		if per == time.Hour {
			return 5
		}
		return int(5 * (per.Hours()))
		
	case strings.HasPrefix(keyStr, "refresh:"):
		// Refresh: 100 attempts per hour
		if per == time.Hour {
			return 100
		}
		return int(100 * (per.Hours()))
		
	case strings.HasPrefix(keyStr, "credential_add:"):
		// Credentials add: 10 attempts per hour
		if per == time.Hour {
			return 10
		}
		return int(10 * (per.Hours()))
		
	default:
		// Default: 60 requests per minute as a reasonable default
		return int(60 * (per.Minutes()))
	}
}