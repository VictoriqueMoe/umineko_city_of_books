package auth

import "errors"

var (
	ErrRegistrationDisabled = errors.New("registration is currently disabled")
	ErrInviteRequired       = errors.New("an invite code is required to register")
	ErrInvalidInvite        = errors.New("invalid or already used invite code")
	ErrPasswordTooShort     = errors.New("password is too short")
	ErrInvalidUsername      = errors.New("username must be 3-30 characters and contain only letters, numbers, underscores, or hyphens")
	ErrTurnstileFailed     = errors.New("verification failed")
)
