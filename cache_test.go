package limiter

import (
	"testing"

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
