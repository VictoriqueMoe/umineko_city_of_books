package fanfic

import "errors"

var (
	ErrNotFound          = errors.New("fanfic not found")
	ErrEmptyTitle        = errors.New("title cannot be empty")
	ErrEmptyBody         = errors.New("body cannot be empty")
	ErrTooManyGenres     = errors.New("maximum 2 genres allowed")
	ErrTooManyCharacters = errors.New("maximum 4 characters allowed")
	ErrInvalidRating     = errors.New("invalid rating")
	ErrNotAuthor         = errors.New("only the author can perform this action")
)
