package limiter

import "time"

type ErrorHandler func(error)

// Option is an interface for dependency injection.
type Option interface {
	apply(l *limiter)
}

// OptionFunc is a function for Option interface.
type OptionFunc func(l *limiter)

func (o OptionFunc) apply(l *limiter) { o(l) }

// WithMinLimit returns a function which sets minimum limit per minute in limiter.
func WithMinLimit(min int64) OptionFunc {
	return func(l *limiter) {
		l.minLimit = min
	}
}

// WithMaxLimit returns a function which sets maximum limit per minute in limiter.
func WithMaxLimit(max int64) OptionFunc {
	return func(l *limiter) {
		l.maxLimit = max
	}
}

// WithEstimatePeriod returns a function which sets a period to estimate threshold.
func WithEstimatePeriod(d time.Duration) OptionFunc {
	return func(l *limiter) {
		l.estimatePeriod = d
	}
}

// WithErrorHandler returns a function which sets handler to process error.
func WithErrorHandler(h ErrorHandler) OptionFunc {
	return func(l *limiter) {
		l.errorHandler = h
	}
}

// WithRevaluateLimit returns a function which sets function used to revaluate limit.
func WithRevaluateLimit(f RevaluateLimit) OptionFunc {
	return func(l *limiter) {
		l.revaluateLimit = f
	}
}
