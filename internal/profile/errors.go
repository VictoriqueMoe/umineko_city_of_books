package profile

import "errors"

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrPasswordTooShort = errors.New("password is too short")
)
