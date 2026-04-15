package profile

import "errors"

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrPasswordTooShort = errors.New("password is too short")
	ErrInvalidDOB       = errors.New("dob must use YYYY-MM-DD format")
	ErrFutureDOB        = errors.New("dob cannot be in the future")
)
