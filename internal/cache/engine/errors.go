package engine

import "errors"

var (
	ErrMiss        = errors.New("cache: miss")
	ErrUnavailable = errors.New("cache: no engine available")
)
