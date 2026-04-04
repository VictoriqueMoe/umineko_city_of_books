package ship

import "errors"

var (
	ErrNotFound            = errors.New("ship not found")
	ErrEmptyTitle          = errors.New("ship title cannot be empty")
	ErrEmptyBody           = errors.New("comment body cannot be empty")
	ErrTooFewCharacters    = errors.New("a ship must contain at least 2 characters")
	ErrDuplicateCharacters = errors.New("ship cannot contain duplicate characters")
)
