package limiter

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v7"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

var (
	mockRedis         *miniredis.Miniredis
	mockMemory        *cache.Cache
	mockRedisCache    Cache
	mockMemoryCache   Cache
	mockRedisLimiter  RateLimiter
	mockMemoryLimiter RateLimiter
)

func TestNewRateLimiter(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		serviceName string
		cache       Cache
		opts        []Option
		isErr       bool
	}{
		"fail": {
			isErr: true,
		},
		"redis-limiter": {
			serviceName: "default",
			cache:       mockRedisCache,
			opts:        []Option{WithMinLimit(10), WithMaxLimit(20), WithEstimatePeriod(10 * time.Minute)},
		},
		"memory-limiter": {
			serviceName: "default",
			cache:       mockMemoryCache,
			opts:        []Option{WithMinLimit(10), WithMaxLimit(20), WithEstimatePeriod(10 * time.Minute)},
		},
	}

	for _, t := range tests {
		l, err := NewRateLimiter(t.serviceName, t.cache, t.opts...)
		assert.Equal(t.isErr, err != nil)
		if err == nil {
			ll := l.(*limiter)
			assert.Equal(ll.minLimit, int64(10))
			assert.Equal(ll.maxLimit, int64(20))
			assert.Equal(ll.estimatePeriod, 10*time.Minute)
		}
	}
}

func TestLimiter_HealthCheck(t *testing.T) {
	assert := assert.New(t)
	mockRedis.FlushAll()
	mockMemory.Flush()

	assert.Equal(true, mockRedisLimiter.HealthCheck(context.Background()))
	assert.Equal(true, mockMemoryLimiter.HealthCheck(context.Background()))
}

func TestLimiter_Acquire(t *testing.T) {
	assert := assert.New(t)
	mockRedis.FlushAll()
	mockMemory.Flush()

	tests := map[string]struct {
		limiter RateLimiter
	}{
		"redis":  {limiter: mockRedisLimiter},
		"memory": {limiter: mockMemoryLimiter},
	}

	for name, t := range tests {
		b, err := t.limiter.Acquire()
		assert.NoError(err)
		assert.True(b)
		result := map[int64]int64{}
		for i := 0; i < 50000; i++ {
			b, err := t.limiter.Acquire()
			assert.NoError(err)
			if b {
				t := time.Now().UTC().Truncate(time.Minute).Unix()
				result[t] += 1
			}
		}
		fmt.Println(name, " ## ", result)
		for _, r := range result {
			assert.LessOrEqual(r, int64(100))
		}
	}
}

func TestLimiter_CurrentLimit(t *testing.T) {
	assert := assert.New(t)
	mockRedis.FlushAll()
	mockMemory.Flush()

	tests := map[string]struct {
		limiter RateLimiter
	}{
		"redis":  {limiter: mockRedisLimiter},
		"memory": {limiter: mockMemoryLimiter},
	}

	for _, t := range tests {
		limit, err := t.limiter.CurrentLimit()
		assert.NoError(err)
		assert.True(limit > 0)
	}
}

func TestLimiter_GetAndSetCurrentLimit(t *testing.T) {
	assert := assert.New(t)
	mockRedis.FlushAll()
	mockMemory.Flush()

	tests := map[string]struct {
		limiter RateLimiter
	}{
		"redis":  {limiter: mockRedisLimiter},
		"memory": {limiter: mockMemoryLimiter},
	}

	for _, t := range tests {
		ll := t.limiter.(*limiter)
		// test getCurrentLimit
		quota, deadline, err := ll.getCurrentLimit()
		assert.NoError(err)
		assert.Equal(int64(100), quota)
		assert.True(time.Now().After(deadline))

		// test setCurrentLimit
		quota, deadline, err = ll.setCurrentLimit(5)
		assert.NoError(err)
		assert.Equal(int64(5), quota)
		assert.True(time.Now().Before(deadline))

		quota, deadline, err = ll.getCurrentLimit()
		assert.NoError(err)
		assert.Equal(int64(5), quota)
		assert.True(time.Now().Before(deadline))
	}
}

func TestLimiter_TimeWindow(t *testing.T) {
	assert := assert.New(t)
	mockRedis.FlushAll()
	mockMemory.Flush()

	tests := map[string]struct {
		limiter RateLimiter
	}{
		"redis":  {limiter: mockRedisLimiter},
		"memory": {limiter: mockMemoryLimiter},
	}

	for _, t := range tests {
		ll := t.limiter.(*limiter)

		// test getTimeRate
		used, err := ll.getCurrentTimeWindow()
		assert.NoError(err)
		assert.Equal(int64(0), used)

		// test increaseTimeWindow
		for i := 0; i < 5; i++ {
			err := ll.increaseCurrentTimeWindow()
			assert.NoError(err)
			used, err = ll.getCurrentTimeWindow()
			assert.NoError(err)
			assert.Equal(int64(i)+1, used)
		}
	}
}

func BenchmarkRedisLimiter_Acquire(b *testing.B) {
	var fail int
	var success int
	var error int
	id := time.Now().UnixNano()
	go func() {
		ticker := time.NewTicker(time.Minute)
		for range ticker.C {
			used, _ := mockRedisLimiter.(*limiter).getCurrentTimeWindow()
			quota, deadline, _ := mockRedisLimiter.(*limiter).getCurrentLimit()
			fmt.Println("[id]", id, "[success]", success, "[fail]", fail, "[error]", error,
				"[used]", used, "[quota]", quota, "[deadline]", deadline)
		}
	}()
	for i := 0; i < b.N; i++ {
		b, err := mockRedisLimiter.Acquire()
		if err != nil {
			error += 1
		} else {
			if b {
				success += 1
			} else {
				fail += 1
			}
		}
	}
}

func BenchmarkMemoryLimiter_Acquire(b *testing.B) {
	var fail int
	var success int
	var error int
	id := time.Now().UnixNano()
	go func() {
		ticker := time.NewTicker(time.Minute)
		for range ticker.C {
			used, _ := mockMemoryLimiter.(*limiter).getCurrentTimeWindow()
			quota, deadline, _ := mockMemoryLimiter.(*limiter).getCurrentLimit()
			fmt.Println("[id]", id, "[success]", success, "[fail]", fail, "[error]", error,
				"[used]", used, "[quota]", quota, "[deadline]", deadline)
		}
	}()
	for i := 0; i < b.N; i++ {
		b, err := mockRedisLimiter.Acquire()
		if err != nil {
			error += 1
		} else {
			if b {
				success += 1
			} else {
				fail += 1
			}
		}
	}
}

func TestMain(m *testing.M) {
	var err error
	mockRedis, err = miniredis.Run()
	if err != nil {
		panic(err)
	}

	redisCli := redis.NewClient(&redis.Options{Addr: mockRedis.Addr()})
	mockRedisCache, err = NewRedisCache(redisCli)
	if err != nil {
		panic(err)
	}

	mockMemoryCache, err = NewMemoryCache()
	if err != nil {
		panic(err)
	}

	mockMemory = mockMemoryCache.(*MemoryCache).client

	mockMemoryLimiter, err = NewRateLimiter("test-memory", mockMemoryCache,
		WithMinLimit(100),
		WithMaxLimit(10000),
		WithEstimatePeriod(2*time.Minute),
		WithRevaluateLimit(func(currentlyLimit, estimatedTimeWindowSize, estimatedTimeWindowUsedCount int64) int64 {
			var newLimit int64
			switch {
			case float64(currentlyLimit*estimatedTimeWindowSize)*0.85 <= float64(estimatedTimeWindowUsedCount):
				extra := float64(currentlyLimit) * 0.5
				newLimit = currentlyLimit + int64(extra)
			case float64(currentlyLimit*estimatedTimeWindowSize)*0.5 >= float64(estimatedTimeWindowUsedCount):
				extra := float64(currentlyLimit) * 0.3
				newLimit = currentlyLimit - int64(extra)
			default:
				newLimit = currentlyLimit
			}
			return newLimit
		}))
	if err != nil {
		panic(err)
	}

	mockRedisLimiter, err = NewRateLimiter("test-redis", mockRedisCache,
		WithMinLimit(100),
		WithMaxLimit(10000),
		WithEstimatePeriod(2*time.Minute),
		WithRevaluateLimit(func(currentlyLimit, estimatedTimeWindowSize, estimatedTimeWindowUsedCount int64) int64 {
			var newLimit int64
			switch {
			case float64(currentlyLimit*estimatedTimeWindowSize)*0.85 <= float64(estimatedTimeWindowUsedCount):
				extra := float64(currentlyLimit) * 0.5
				newLimit = currentlyLimit + int64(extra)
			case float64(currentlyLimit*estimatedTimeWindowSize)*0.5 >= float64(estimatedTimeWindowUsedCount):
				extra := float64(currentlyLimit) * 0.3
				newLimit = currentlyLimit - int64(extra)
			default:
				newLimit = currentlyLimit
			}
			return newLimit
		}))
	if err != nil {
		panic(err)
	}

	time.Sleep(3 * time.Second)
	os.Exit(m.Run())
}

//func TestRedisLimiter_Integration(t *testing.T) {
//	assert := assert.New(t)
//	mockRedis.FlushAll()
//	mockMemory.Flush()
//
//	var fail int
//	var success int
//	var error int
//	id := time.Now().UnixNano()
//	go func() {
//		ticker := time.NewTicker(30 * time.Second)
//		for range ticker.C {
//			used, _ := mockRedisLimiter.(*limiter).getCurrentTimeWindow()
//			quota, deadline, _ := mockRedisLimiter.(*limiter).getCurrentLimit()
//			fmt.Println(time.Now(), "[id]", id, "[success]", success, "[fail]", fail, "[error]", error,
//				"[used]", used, "[quota]", quota, "[deadline]", deadline)
//		}
//	}()
//
//	sleep := 10 * time.Millisecond
//
//	go func() {
//		ticker := time.NewTicker(10 * time.Minute)
//		for range ticker.C {
//			sleep += time.Second
//		}
//	}()
//
//	for {
//		b, err := mockRedisLimiter.Acquire()
//		if err != nil {
//			error += 1
//		} else {
//			if b {
//				success += 1
//			} else {
//				fail += 1
//			}
//		}
//		if sleep == 0 {
//			continue
//		} else {
//			time.Sleep(sleep)
//		}
//	}
//
//	_ = assert
//}

//func TestMemoryLimiter_Integration(t *testing.T) {
//	assert := assert.New(t)
//	mockRedis.FlushAll()
//	mockMemory.Flush()
//
//	var fail int
//	var success int
//	var error int
//	id := time.Now().UnixNano()
//	go func() {
//		ticker := time.NewTicker(30 * time.Second)
//		for range ticker.C {
//			used, _ := mockMemoryLimiter.(*limiter).getCurrentTimeWindow()
//			quota, deadline, _ := mockMemoryLimiter.(*limiter).getCurrentLimit()
//			fmt.Println(time.Now(), "[id]", id, "[success]", success, "[fail]", fail, "[error]", error,
//				"[used]", used, "[quota]", quota, "[deadline]", deadline)
//		}
//	}()
//
//	sleep := 10 * time.Millisecond
//
//	go func() {
//		ticker := time.NewTicker(10 * time.Minute)
//		for range ticker.C {
//			sleep += time.Second
//		}
//	}()
//
//	for {
//		b, err := mockMemoryLimiter.Acquire()
//		if err != nil {
//			error += 1
//		} else {
//			if b {
//				success += 1
//			} else {
//				fail += 1
//			}
//		}
//		if sleep == 0 {
//			continue
//		} else {
//			time.Sleep(sleep)
//		}
//	}
//
//	_ = assert
//}
