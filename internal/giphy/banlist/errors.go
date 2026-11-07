package banlist

import "errors"

var (
	ErrInvalidKind   = errors.New("invalid kind")
	ErrValueRequired = errors.New("value required")
)
