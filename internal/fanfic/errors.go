package fanfic

import "errors"

var (
	ErrNotFound          = errors.New("fanfic not found")
	ErrEmptyTitle        = errors.New("title cannot be empty")
	ErrEmptyBody         = errors.New("body cannot be empty")
	ErrTooManyGenres     = errors.New("maximum 2 genres allowed")
	ErrTooManyCharacters = errors.New("maximum 4 characters allowed")
	ErrTooManyTags       = errors.New("maximum 10 tags allowed")
	ErrTagTooLong        = errors.New("tags must be 30 characters or fewer")
	ErrInvalidRating     = errors.New("invalid rating")
	ErrNotAuthor         = errors.New("only the author can perform this action")
)
