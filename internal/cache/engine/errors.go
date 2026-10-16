package engine

import "errors"

var (
	ErrMiss            = errors.New("cache: miss")
	ErrUnavailable     = errors.New("cache: no engine available")
	ErrNotCounter      = errors.New("cache: value is not a counter")
	ErrUnexpectedReply = errors.New("cache: unexpected reply")
	ErrDeadlinePassed  = errors.New("cache: deadline must be in the future")
)
