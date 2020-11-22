package limiter

import (
	"context"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/patrickmn/go-cache"
)

// Cache is an interface which supports fundamental functionality for storage.
// This is used by RateLimiter.
type Cache interface {
	Get(key string) (interface{}, bool, error)
	MGet(keys ...string) ([]interface{}, bool, error)
	Set(key string, value interface{}, ttl time.Duration) error
	Increment(key string, n int64, ttl time.Duration) error
	Ping(ctx context.Context) bool
	Close() error
}

// NewRedisCache returns cache for redis
func NewRedisCache(cli *redis.Client) (Cache, error) {
	if cli == nil {
		return nil, ErrInvalidParams
	}
	rc := &redisCache{cli}
	return rc, nil
}

// NewMemoryCache returns cache for memory.
func NewMemoryCache() (Cache, error) {
	c := cache.New(cache.NoExpiration, 5*time.Minute)
	mc := &memoryCache{client: c}
	return mc, nil
}
