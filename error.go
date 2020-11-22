package limiter

import (
	"errors"
)

var (
	ErrInvalidParams = errors.New("[err] invalid params")
	ErrUnknown       = errors.New("[err] unknown")
)
