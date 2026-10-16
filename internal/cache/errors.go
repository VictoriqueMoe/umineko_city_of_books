package cache

import (
	"errors"

	"umineko_city_of_books/internal/cache/engine"
)

var (
	ErrMiss           = engine.ErrMiss
	ErrUnavailable    = engine.ErrUnavailable
	ErrTTLTooShort    = errors.New("cache: ttl must be at least one millisecond")
	ErrDeadlinePassed = engine.ErrDeadlinePassed
	ErrNotDeadline    = errors.New("cache: value is not a deadline")
)

func errorIsMiss(err error) bool {
	return errors.Is(err, ErrMiss)
}
