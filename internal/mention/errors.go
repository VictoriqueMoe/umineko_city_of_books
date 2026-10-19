package mention

import "errors"

var (
	ErrUnknownKind      = errors.New("mention: kind is not registered")
	ErrInvalidReference = errors.New("mention: reference is incomplete for its kind")
)
