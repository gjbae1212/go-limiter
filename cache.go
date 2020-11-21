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
	Ping(ctx context.Context) bool
	Close() error
}

// RedisCache is Cache struct based on Redis.
type RedisCache struct {
	*redis.Client
}

// RedisCache_Get gets item using key.
func (rc *RedisCache) Get(key string) (interface{}, bool, error) {
	if key == "" {
		return nil, false, nil
	}

	result, err := rc.Client.Get(key).Result()
	if err == redis.Nil {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	return result, true, nil
}

// RedisCache_MGet gets multiple items using key array.
func (rc *RedisCache) MGet(keys ...string) ([]interface{}, bool, error) {
	if len(keys) == 0 {
		return []interface{}{}, false, nil
	}

	result, err := rc.Client.MGet(keys...).Result()
	if err == redis.Nil {
		return []interface{}{}, false, nil
	} else if err != nil {
		return []interface{}{}, false, err
	}
	return result, true, nil
}

// RedisCache_Set sets key and value with ttl.
func (rc *RedisCache) Set(key string, value interface{}, ttl time.Duration) error {
	if key == "" || value == nil {
		return ErrInvalidParams
	}

	_, err := rc.Client.Set(key, value, ttl).Result()
	return err
}

// RedisCache_Ping does a server health check.
func (rc *RedisCache) Ping(ctx context.Context) bool {
	if ctx == nil {
		return false
	}

	ret := make(chan bool)
	go func() {
		if _, err := rc.Client.Ping().Result(); err != nil {
			ret <- false
		} else {
			ret <- true
		}
	}()

	select {
	case <-ctx.Done():
		return false
	case b := <-ret:
		return b
	}
}

// RedisCache_Close closes redis connection.
func (rc *RedisCache) Close() error {
	return rc.Client.Close()
}

// RedisCache is Cache struct based on Memory.
type MemoryCache struct {
	*cache.Cache
}

// MemoryCache_Get gets item using key.
func (mc *MemoryCache) Get(key string) (interface{}, bool, error) {
	if key == "" {
		return nil, false, nil
	}

	v, b := mc.Cache.Get(key)
	return v, b, nil
}

// MemoryCache_MGet gets multiple items using key array.
func (mc *MemoryCache) MGet(keys ...string) ([]interface{}, bool, error) {
	if len(keys) == 0 {
		return nil, false, nil
	}

	var ret []interface{}
	for _, key := range keys {
		v, _ := mc.Cache.Get(key)
		ret = append(ret, v)
	}
	return ret, true, nil
}

// MemoryCache_Set sets key and value with ttl.
func (mc *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
	if key == "" || value == nil {
		return ErrInvalidParams
	}
	mc.Cache.Set(key, value, ttl)
	return nil
}

// MemoryCache_Ping does a server health check.
func (mc *MemoryCache) Ping(ctx context.Context) bool {
	if ctx == nil {
		return false
	}

	if mc.Cache != nil {
		return true
	}
	return false
}

// MemoryCache_Close closes memory cache.
func (mc *MemoryCache) Close() error {
	mc.Cache.Flush()
	return nil
}

// NewRedisCache returns cache for redis
func NewRedisCache(cli *redis.Client) (Cache, error) {
	if cli == nil {
		return nil, ErrInvalidParams
	}

	rc := &RedisCache{cli}
	return rc, nil
}

// NewMemoryCache returns cache for memory.
func NewMemoryCache() (Cache, error) {
	c := cache.New(cache.NoExpiration, 5*time.Minute)
	mc := &MemoryCache{c}
	return mc, nil
}
