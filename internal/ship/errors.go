package ship

import (
	"errors"
	"fmt"

	"umineko_city_of_books/internal/dao"
)

var (
	ErrNotFound            = fmt.Errorf("ship not found: %w", dao.ErrNotFound)
	ErrEmptyTitle          = errors.New("ship title cannot be empty")
	ErrEmptyBody           = errors.New("comment body cannot be empty")
	ErrTooFewCharacters    = errors.New("a ship must contain at least 2 characters")
	ErrDuplicateCharacters = errors.New("ship cannot contain duplicate characters")
)
