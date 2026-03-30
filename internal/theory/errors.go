package theory

import "errors"

var (
	ErrCannotRespondToOwnTheory = errors.New("you cannot respond to your own theory")
	ErrRateLimited              = errors.New("daily limit reached")
)
