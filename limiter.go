package limiter

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	formatCurrentLimitKey   = "limiter:%s:current_limit"
	formatCurrentLimitValue = "%d-%d"
	formatTimeWindow        = "limiter:%s:window:%d"
)

var (
	defaultOpts = []Option{
		WithMinLimit(150),
		WithMaxLimit(30000),
		WithEstimatePeriod(10 * time.Minute),
		WithErrorHandler(func(err error) {
			fmt.Printf("[err][limiter] %s\n", err.Error())
		}),
		WithRevaluateLimit(func(currentlyLimit, estimatedTimeWindowSize, estimatedTimeWindowUsedCount int64) int64 {
			var newLimit int64
			switch {
			case float64(currentlyLimit*estimatedTimeWindowSize)*0.9 <= float64(estimatedTimeWindowUsedCount):
				newLimit = currentlyLimit * 2
			case float64(currentlyLimit*estimatedTimeWindowSize)*0.5 >= float64(estimatedTimeWindowUsedCount):
				newLimit = currentlyLimit / 2
			default:
				newLimit = currentlyLimit
			}
			return newLimit
		}),
	}
)

// RevaluateLimit is to revaluate sending limit per 1 minute.
// currentlyLimit is currently applied limit.
// estimatedTimeWindowSize is the size of time-window for the duration of an estimated period.
// estimatedTimeWindowUsedCount is the sum of sending push count of time-window for the duration of an estimated period.
// Its return value is new limit per 1 minute.
type RevaluateLimit func(currentlyLimit, estimatedTimeWindowSize, estimatedTimeWindowUsedCount int64) int64

type RateLimiter interface {
	Acquire() (bool, error)
	CurrentLimit() (int64, error)
	HealthCheck(ctx context.Context) bool
	Close() error
}

type limiter struct {
	serviceName    string
	minLimit       int64
	maxLimit       int64
	estimatePeriod time.Duration
	errorHandler   ErrorHandler
	ticker         *time.Ticker
	revaluateLimit RevaluateLimit
	cache          Cache
}

// IsAcquire returns a value which exceeds limit or not currently.
func (l *limiter) Acquire() (bool, error) {
	used, err := l.getCurrentTimeWindow()
	if err != nil {
		return false, err
	}

	quota, _, err := l.getCurrentLimit()
	if err != nil {
		return false, err
	}

	// if quota is already full.
	if used >= quota {
		return false, nil
	}

	// increase count
	if err := l.increaseCurrentTimeWindow(); err != nil {
		return false, err
	}

	return true, nil
}

// CurrentLimit returns currently limit.
func (l *limiter) CurrentLimit() (int64, error) {
	limit, _, err := l.getCurrentLimit()
	if err != nil {
		return 0, err
	}
	return limit, nil
}

// HealthCheck checks to redis condition.
func (l *limiter) HealthCheck(ctx context.Context) bool {
	return l.cache.Ping(ctx)
}

// Close closes limiter.
func (l *limiter) Close() error {
	l.ticker.Stop()
	return l.cache.Close()
}

// getCurrentLimit returns current limit.
func (l *limiter) getCurrentLimit() (int64, time.Time, error) {
	result, ok, err := l.cache.Get(l.keyToCurrentLimit())
	if err != nil {
		return 0, time.Time{}, err
	}

	// if key to limit doesn't exist, default apply to a minimum limit.
	if !ok {
		return l.minLimit, time.Now().Add(-1 * time.Minute), nil
	}

	seps := strings.Split(result.(string), "-")
	if len(seps) != 2 {
		return 0, time.Time{}, ErrInvalidParams
	}

	limit, err := strconv.ParseInt(seps[0], 10, 64)
	if err != nil {
		return 0, time.Time{}, err
	}

	deadline, err := strconv.ParseInt(seps[1], 10, 64)
	if err != nil {
		return 0, time.Time{}, err
	}

	return limit, time.Unix(deadline, 0), nil
}

// setCurrentLimit sets current limit.
func (l *limiter) setCurrentLimit(limit int64) (int64, time.Time, error) {
	if limit <= 0 {
		return 0, time.Time{}, ErrInvalidParams
	}

	deadline := time.Now().UTC().Add(l.estimatePeriod)
	value := l.valueToCurrentLimit(limit, deadline)
	// cache expire time is twice as much as a estimated period.
	// it's not critical, but must grater than a estimated period.
	expire := l.estimatePeriod * 2

	if err := l.cache.Set(l.keyToCurrentLimit(), value, expire); err != nil {
		return 0, time.Time{}, err
	}

	return limit, deadline, nil
}

// getCurrentTimeWindow returns count of current time window.
func (l *limiter) getCurrentTimeWindow() (int64, error) {
	result, ok, err := l.cache.Get(l.keyToTimeWindow(time.Now().UTC().Truncate(time.Minute)))
	if err != nil {
		return 0, err
	}

	// if key to limit doesn't exist, directly return 0, nil
	if !ok {
		return 0, nil
	}

	return interfaceToInt64(result)
}

// increaseTimeWindow is to increase current time window.
func (l *limiter) increaseCurrentTimeWindow() error {
	currentWindow := time.Now().UTC().Truncate(time.Minute)

	// cache expire time is twice as much as a estimated period.
	// it's not critical, but must grater than a estimated period.
	key := l.keyToTimeWindow(currentWindow)
	expire := l.estimatePeriod * 2

	return l.cache.Increment(key, 1, expire)
}

// getEstimateTimeWindow returns total time window count and list size.
func (l *limiter) getEstimateTimeWindow() (int64, int64, error) {
	// truncate seconds
	endWindow := time.Now().UTC().Add(-time.Minute).Truncate(time.Minute)
	startWindow := endWindow.Add(-l.estimatePeriod).Truncate(time.Minute)

	// extract time windows
	var keys []string
	var windowSize int64
	for ; startWindow.Unix() <= endWindow.Unix(); startWindow = startWindow.Add(time.Minute) {
		keys = append(keys, l.keyToTimeWindow(startWindow))
		windowSize += 1
	}

	// summary time window count
	var used int64
	result, ok, err := l.cache.MGet(keys...)
	if err != nil {
		return 0, 0, err
	}

	// if getting time windows don't exist.
	if !ok {
		used = 0
	} else {
		for _, e := range result {
			if e != nil {
				count, err := interfaceToInt64(e)
				if err != nil {
					return 0, 0, err
				}
				used += count
			}
		}
	}

	return used, windowSize, nil
}

// estimateCurrentLimit estimates current limit.
func (l *limiter) estimateCurrentLimit() error {
	now := time.Now()

	quota, deadline, err := l.getCurrentLimit()
	if err != nil {
		return err
	}

	// currently not estimate time.
	if now.Before(deadline) {
		return nil
	}

	// get estimate time window information.
	used, windowSize, err := l.getEstimateTimeWindow()
	if err != nil {
		return err
	}

	newQuota := l.revaluateLimit(quota, windowSize, used)
	// newQuota must be between min and max limit.
	if newQuota < l.minLimit {
		newQuota = l.minLimit
	} else if newQuota > l.maxLimit {
		newQuota = l.maxLimit
	}

	// set current limit
	if _, _, err := l.setCurrentLimit(newQuota); err != nil {
		return err
	}

	return nil
}

// estimateLoop runs loop for estimating threshold.
func (l *limiter) estimateLoop() {
	l.ticker = time.NewTicker(time.Minute)
	go func() {
		for _ = range l.ticker.C {
			if err := l.estimateCurrentLimit(); err != nil {
				l.errorHandler(err)
			}
		}
	}()
}

// keyToTimeWindow returns time window string
func (l *limiter) keyToTimeWindow(t time.Time) string {
	return fmt.Sprintf(formatTimeWindow, l.serviceName, t.Unix())
}

// keyToCurrentLimit returns current limit key.
func (l *limiter) keyToCurrentLimit() string {
	return fmt.Sprintf(formatCurrentLimitKey, l.serviceName)
}

// valueToCurrentLimit returns current limit value.
func (l *limiter) valueToCurrentLimit(v int64, t time.Time) string {
	return fmt.Sprintf(formatCurrentLimitValue, v, t.Unix())
}

// NewRateLimiter returns RateLimiter interface.
func NewRateLimiter(serviceName string, cache Cache, opts ...Option) (RateLimiter, error) {
	if serviceName == "" || cache == nil {
		return nil, ErrInvalidParams
	}

	l := &limiter{serviceName: serviceName, cache: cache}

	var mergeOpt []Option
	mergeOpt = append(mergeOpt, defaultOpts...)
	mergeOpt = append(mergeOpt, opts...)
	for _, opt := range mergeOpt {
		opt.apply(l)
	}

	// minimal estimate period 1 minute.
	if l.estimatePeriod < time.Minute {
		l.estimatePeriod = time.Minute
	}

	// run estimate loop
	l.estimateLoop()

	return l, nil
}

// interfaceToInt64 converts value having interface type to value having int64.
func interfaceToInt64(i interface{}) (int64, error) {
	if i == nil {
		return 0, ErrInvalidParams
	}

	switch i.(type) {
	case int:
		return int64(i.(int)), nil
	case int64:
		return i.(int64), nil
	case int32:
		return int64(i.(int32)), nil
	case float32:
		return int64(i.(float32)), nil
	case float64:
		return int64(i.(float64)), nil
	case string:
		return strconv.ParseInt(i.(string), 10, 64)
	default:
		return 0, ErrUnknown
	}
}
