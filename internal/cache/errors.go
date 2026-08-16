package cache

import (
	"errors"

	"umineko_city_of_books/internal/cache/engine"
)

var (
	ErrMiss = engine.ErrMiss
)

func errorIsMiss(err error) bool {
	return errors.Is(err, ErrMiss)
}
