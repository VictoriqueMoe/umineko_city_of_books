package theory

import "errors"

var (
	ErrCannotRespondToOwnTheory = errors.New("you cannot respond to your own theory")
)
