package announcement

import "errors"

var (
	ErrNotFound         = errors.New("announcement not found")
	ErrEmptyTitleOrBody = errors.New("title and body are required")
)
