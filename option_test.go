package limiter

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithEstimatePeriod(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		input time.Duration
	}{
		"success": {input: time.Hour},
	}

	for _, t := range tests {
		l := &limiter{}
		o := WithEstimatePeriod(t.input)
		o(l)
		assert.Equal(l.estimatePeriod, t.input)
	}
}

func TestWithMaxLimit(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		input int64
	}{
		"success": {input: 10},
	}

	for _, t := range tests {
		l := &limiter{}
		o := WithMinLimit(t.input)
		o(l)
		assert.Equal(l.minLimit, t.input)
	}
}

func TestWithMinLimit(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		input int64
	}{
		"success": {input: 10},
	}

	for _, t := range tests {
		l := &limiter{}
		o := WithMaxLimit(t.input)
		o(l)
		assert.Equal(l.maxLimit, t.input)
	}
}

func TestWithErrorHandler(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		input ErrorHandler
	}{
		"success": {input: func(err error) { fmt.Println(err) }},
	}

	for _, t := range tests {
		l := &limiter{}
		o := WithErrorHandler(t.input)
		o(l)
		l.errorHandler(fmt.Errorf("[err] TestWithErrorHandler test"))
		_ = assert
	}
}

func TestWithRevaluateLimit(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		input RevaluateLimit
	}{
		"success": {input: func(currentlyLimit, estimatedTimeWindowSize, estimatedTimeWindowUsedCount int64) int64 { return 10 }},
	}

	for _, t := range tests {
		l := &limiter{}
		o := WithRevaluateLimit(t.input)
		o(l)
		vv := l.revaluateLimit(1, 1, 1)
		assert.Equal(int64(10), vv)
	}
}
