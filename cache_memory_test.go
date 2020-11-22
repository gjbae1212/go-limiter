package limiter

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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

func TestMemoryCache_Increment(t *testing.T) {
	assert := assert.New(t)

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
		mc, err := NewMemoryCache()
		assert.NoError(err)

		err = mc.Increment(t.key, t.n, t.ttl)
		assert.Equal(t.isErr, err != nil)
		if err == nil {
			result, ok, err := mc.Get(t.key)
			assert.NoError(err)
			assert.True(ok)
			assert.Equal(t.n, result.(int64))
		}
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
