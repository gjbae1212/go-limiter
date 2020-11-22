package limiter

import (
	"context"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

// memoryCache is Cache struct based on Memory.
type memoryCache struct {
	client *cache.Cache
	lock   sync.RWMutex
}

// memoryCache_Get gets item using key.
func (mc *memoryCache) Get(key string) (interface{}, bool, error) {
	if key == "" {
		return nil, false, nil
	}

	v, b := mc.client.Get(key)
	return v, b, nil
}

// memoryCache_MGet gets multiple items using key array.
func (mc *memoryCache) MGet(keys ...string) ([]interface{}, bool, error) {
	if len(keys) == 0 {
		return nil, false, nil
	}

	var ret []interface{}
	for _, key := range keys {
		v, _ := mc.client.Get(key)
		ret = append(ret, v)
	}
	return ret, true, nil
}

// memoryCache_Set sets key and value with ttl.
func (mc *memoryCache) Set(key string, value interface{}, ttl time.Duration) error {
	if key == "" || value == nil {
		return ErrInvalidParams
	}
	mc.client.Set(key, value, ttl)
	return nil
}

// RedisCache_Increment increases a value from key.
func (mc *memoryCache) Increment(key string, n int64, ttl time.Duration) error {
	if key == "" || n <= 0 {
		return ErrInvalidParams
	}
	mc.lock.Lock()
	defer mc.lock.Unlock()

	// if key limit doesn't exist, make key in cache.
	_, ok := mc.client.Get(key)
	if !ok {
		mc.client.Set(key, n, ttl)
		return nil
	}

	return mc.client.Increment(key, n)
}

// memoryCache_Ping does a server health check.
func (mc *memoryCache) Ping(ctx context.Context) bool {
	if ctx == nil {
		return false
	}

	if mc.client != nil {
		return true
	}
	return false
}

// memoryCache_Close closes memory cache.
func (mc *memoryCache) Close() error {
	mc.client.Flush()
	return nil
}
