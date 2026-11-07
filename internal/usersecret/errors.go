package usersecret

import "errors"

var (
	ErrInvalidRequest    = errors.New("invalid request")
	ErrHuntAlreadySolved = errors.New("hunt already solved")
)
