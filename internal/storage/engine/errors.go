package engine

import "errors"

var (
	ErrNotFound   = errors.New("storage: object not found")
	ErrInvalidKey = errors.New("storage: invalid key")
	ErrDisabled   = errors.New("storage: engine is not configured")
)
