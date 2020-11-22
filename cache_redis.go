package limiter

import (
	"context"
	"time"

	"github.com/go-redis/redis/v7"
)

// redisCache is Cache struct based on Redis.
type redisCache struct {
	client *redis.Client
}

// redisCache_Get gets item using key.
func (rc *redisCache) Get(key string) (interface{}, bool, error) {
	if key == "" {
		return nil, false, nil
	}

	result, err := rc.client.Get(key).Result()
	if err == redis.Nil {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	return result, true, nil
}

// redisCache_MGet gets multiple items using key array.
func (rc *redisCache) MGet(keys ...string) ([]interface{}, bool, error) {
	if len(keys) == 0 {
		return []interface{}{}, false, nil
	}

	result, err := rc.client.MGet(keys...).Result()
	if err == redis.Nil {
		return []interface{}{}, false, nil
	} else if err != nil {
		return []interface{}{}, false, err
	}
	return result, true, nil
}

// redisCache_Set sets key and value with ttl.
func (rc *redisCache) Set(key string, value interface{}, ttl time.Duration) error {
	if key == "" || value == nil {
		return ErrInvalidParams
	}

	_, err := rc.client.Set(key, value, ttl).Result()
	return err
}

// redisCache_Increment increases a value from key.
func (rc *redisCache) Increment(key string, n int64, ttl time.Duration) error {
	if key == "" || n <= 0 {
		return ErrInvalidParams
	}

	// start redis pipeline.
	pipe := rc.client.Pipeline()
	pipe.IncrBy(key, n)
	if ttl > 0 {
		pipe.Expire(key, ttl)
	}

	// end redis pipeline.
	if _, err := pipe.Exec(); err != nil {
		return err
	}

	return nil
}

// redisCache_Ping does a server health check.
func (rc *redisCache) Ping(ctx context.Context) bool {
	if ctx == nil {
		return false
	}

	ret := make(chan bool)
	go func() {
		if _, err := rc.client.Ping().Result(); err != nil {
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

// redisCache_Close closes redis connection.
func (rc *redisCache) Close() error {
	return rc.client.Close()
}
