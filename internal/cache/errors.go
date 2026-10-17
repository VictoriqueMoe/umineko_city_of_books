package cache

import (
	"errors"

	"umineko_city_of_books/internal/cache/engine"
)

var (
	ErrMiss             = engine.ErrMiss
	ErrUnavailable      = engine.ErrUnavailable
	ErrInvalidAllowance = errors.New("cache: an allowance needs a limit of at least one and a window of at least one millisecond")
)

func errorIsMiss(err error) bool {
	return errors.Is(err, ErrMiss)
}
