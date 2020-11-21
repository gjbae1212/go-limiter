package limiter

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/go-redis/redis/v7"

	"github.com/stretchr/testify/assert"
)

func TestNewMemoryCache(t *testing.T) {
	assert := assert.New(t)

	cli := redis.NewClient(&redis.Options{Addr: mockRedis.Addr()})

	tests := map[string]struct {
		input *redis.Client
		isErr bool
	}{
		"fail":    {isErr: true},
		"success": {input: cli},
	}

	for _, t := range tests {
		_, err := NewRedisCache(t.input)
		assert.Equal(t.isErr, err != nil)
	}

}

func TestNewRedisCache(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		isErr bool
	}{
		"success": {},
	}

	for _, t := range tests {
		_, err := NewMemoryCache()
		assert.Equal(t.isErr, err != nil)
	}
}

func TestMemoryCache_Get(t *testing.T) {
	assert := assert.New(t)

	mc, err := NewMemoryCache()
	assert.NoError(err)
	mc.Set("expired", 1, 100*time.Millisecond)
	mc.Set("get", 1, 1*time.Hour)
	time.Sleep(200 * time.Millisecond)

	tests := map[string]struct {
		input  string
		output interface{}
		ok     bool
		isErr  bool
	}{
		"no-exist": {
			input: "no-exist", ok: false, isErr: false,
		},
		"expired": {
			input: "expired", ok: false, isErr: false,
		},
		"get": {
			input: "get", output: 1, ok: true, isErr: false,
		},
	}

	for _, t := range tests {
		v, ok, err := mc.Get(t.input)
		assert.Equal(t.output, v)
		assert.Equal(t.ok, ok)
		assert.Equal(t.isErr, err != nil)
	}
}

func TestMemoryCache_MGet(t *testing.T) {
	assert := assert.New(t)

	mc, err := NewMemoryCache()
	assert.NoError(err)
	mc.Set("expired-1", 1, 100*time.Millisecond)
	mc.Set("expired-2", 1, 100*time.Millisecond)
	mc.Set("expired-3", 1, 100*time.Millisecond)
	mc.Set("get-1", 1, 1*time.Hour)
	mc.Set("get-2", 1, 1*time.Hour)
	mc.Set("get-3", 1, 1*time.Hour)
	time.Sleep(200 * time.Millisecond)

	tests := map[string]struct {
		input  []string
		output []interface{}
		ok     bool
		isErr  bool
	}{
		"no-exist": {
			input: []string{}, ok: false, isErr: false,
		},
		"expired": {
			input: []string{"expired-1", "expired-2", "expired-3"}, output: []interface{}{nil, nil, nil}, ok: true, isErr: false,
		},
		"get": {
			input: []string{"get-1", "get-2", "get-3"}, output: []interface{}{1, 1, 1}, ok: true, isErr: false,
		},
		"mix": {
			input:  []string{"expired-1", "get-1", "expired-2", "get-2", "expired-3", "get-3"},
			output: []interface{}{nil, 1, nil, 1, nil, 1}, ok: true, isErr: false,
		},
	}

	for _, t := range tests {
		result, ok, err := mc.MGet(t.input...)
		assert.Equal(t.isErr, err != nil)
		assert.Equal(t.ok, ok)
		assert.True(reflect.DeepEqual(t.output, result))
	}
}

func TestMemoryCache_Close(t *testing.T) {
	assert := assert.New(t)

	mc, err := NewMemoryCache()
	assert.NoError(err)

	tests := map[string]struct {
		isErr bool
	}{
		"success": {},
	}

	for _, t := range tests {
		assert.Equal(t.isErr, mc.Close() != nil)
	}
}

func TestMemoryCache_Ping(t *testing.T) {
	assert := assert.New(t)

	mc, err := NewMemoryCache()
	assert.NoError(err)

	tests := map[string]struct {
		live bool
	}{
		"success": {live: true},
	}

	for _, t := range tests {
		assert.Equal(t.live, mc.Ping(context.Background()))
	}
}

func TestMemoryCache_Set(t *testing.T) {
	assert := assert.New(t)

	mc, err := NewMemoryCache()
	assert.NoError(err)

	tests := map[string]struct {
		key   string
		value interface{}
		ttl   time.Duration
		isErr bool
	}{
		"fail-1":  {value: 1, isErr: true},
		"fail-2":  {key: "test", isErr: true},
		"success": {key: "allan", value: 1, isErr: false},
	}

	for _, t := range tests {
		err := mc.Set(t.key, t.value, t.ttl)
		assert.Equal(t.isErr, err != nil)
	}
}

func TestRedisCache_Get(t *testing.T) {
	assert := assert.New(t)

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

func TestRedisCache_Close(t *testing.T) {
	assert := assert.New(t)

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

func TestRedisCache_Ping(t *testing.T) {
	assert := assert.New(t)

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

func TestRedisCache_Set(t *testing.T) {
	assert := assert.New(t)

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
