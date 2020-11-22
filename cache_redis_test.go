package limiter

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/stretchr/testify/assert"
)

func TestRedisCache_Get(t *testing.T) {
	assert := assert.New(t)
	mockRedis.FlushAll()

	client := redis.NewClient(&redis.Options{Addr: mockRedis.Addr()})
	rc, err := NewRedisCache(client)
	assert.NoError(err)
	rc.Set("get", 1, 1*time.Hour)

	tests := map[string]struct {
		input  string
		output interface{}
		ok     bool
		isErr  bool
	}{
		"no-exist": {
			input: "no-exist", ok: false, isErr: false,
		},
		"get": {
			input: "get", output: "1", ok: true, isErr: false,
		},
	}

	for _, t := range tests {
		v, ok, err := rc.Get(t.input)
		assert.Equal(t.output, v)
		assert.Equal(t.ok, ok)
		assert.Equal(t.isErr, err != nil)
	}
}

func TestRedisCache_MGet(t *testing.T) {
	assert := assert.New(t)
	mockRedis.FlushAll()

	client := redis.NewClient(&redis.Options{Addr: mockRedis.Addr()})
	rc, err := NewRedisCache(client)
	assert.NoError(err)
	rc.Set("mget-1", 1, 1*time.Hour)
	rc.Set("mget-2", 2, 1*time.Hour)
	rc.Set("mget-3", 3, 1*time.Hour)

	tests := map[string]struct {
		input  []string
		output []interface{}
		ok     bool
		isErr  bool
	}{
		"empty": {output: []interface{}{}, ok: false},
		"one of part": {input: []string{"mget-1", "empty-1", "empty-2"},
			output: []interface{}{"1", nil, nil}, ok: true},
		"all": {input: []string{"mget-1", "mget-2", "mget-3"},
			output: []interface{}{"1", "2", "3"}, ok: true},
	}

	for _, t := range tests {
		output, ok, err := rc.MGet(t.input...)
		assert.Equal(t.isErr, err != nil)
		assert.Equal(t.ok, ok)
		assert.Equal(t.output, output)
	}
}

func TestRedisCache_Set(t *testing.T) {
	assert := assert.New(t)
	mockRedis.FlushAll()

	client := redis.NewClient(&redis.Options{Addr: mockRedis.Addr()})
	rc, err := NewRedisCache(client)
	assert.NoError(err)

	tests := map[string]struct {
		key   string
		value interface{}
		ttl   time.Duration
		isErr bool
	}{
		"fail":    {isErr: true},
		"success": {key: "allan", value: 1, ttl: time.Second},
	}

	for _, t := range tests {
		err := rc.Set(t.key, t.value, t.ttl)
		assert.Equal(t.isErr, err != nil)
	}
}

func TestRedisCache_Ping(t *testing.T) {
	assert := assert.New(t)
	mockRedis.FlushAll()

	client := redis.NewClient(&redis.Options{Addr: mockRedis.Addr()})
	rc, err := NewRedisCache(client)
	assert.NoError(err)

	tests := map[string]struct {
		live bool
	}{
		"success": {live: true},
	}

	for _, t := range tests {
		assert.Equal(t.live, rc.Ping(context.Background()))
	}
}

func TestRedisCache_Increment(t *testing.T) {
	assert := assert.New(t)
	mockRedis.FlushAll()

	client := redis.NewClient(&redis.Options{Addr: mockRedis.Addr()})
	rc, err := NewRedisCache(client)
	assert.NoError(err)

	tests := map[string]struct {
		key   string
		n     int64
		ttl   time.Duration
		isErr bool
	}{
		"fail":    {isErr: true},
		"success": {key: "allan", n: 10},
	}

	for _, t := range tests {
		err = rc.Increment(t.key, t.n, t.ttl)
		assert.Equal(t.isErr, err != nil)
		if err == nil {
			result, ok, err := rc.Get(t.key)
			assert.NoError(err)
			assert.True(ok)
			assert.Equal(strconv.Itoa(int(t.n)), result)
		}
	}

}

func TestRedisCache_Close(t *testing.T) {
	assert := assert.New(t)
	mockRedis.FlushAll()

	client := redis.NewClient(&redis.Options{Addr: mockRedis.Addr()})
	rc, err := NewRedisCache(client)
	assert.NoError(err)

	tests := map[string]struct {
		isErr bool
	}{
		"success": {},
	}

	for _, t := range tests {
		assert.Equal(t.isErr, rc.Close() != nil)
	}

}
