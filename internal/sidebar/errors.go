package sidebar

import "errors"

var (
	ErrEmptyKey   = errors.New("key is required")
	ErrKeyTooLong = errors.New("key too long")
)
