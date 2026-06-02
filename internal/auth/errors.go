package auth

import "errors"

var (
	ErrRegistrationDisabled     = errors.New("registration is currently disabled")
	ErrInviteRequired           = errors.New("an invite code is required to register")
	ErrInvalidInvite            = errors.New("invalid or already used invite code")
	ErrPasswordTooShort         = errors.New("password is too short")
	ErrInvalidUsername          = errors.New("username must be 3-30 characters and contain only letters, numbers, underscores, or hyphens")
	ErrTurnstileFailed          = errors.New("verification failed")
	ErrUserBanned               = errors.New("your account has been banned")
	ErrUserNotFound             = errors.New("user not found")
	ErrNoEmailAddress           = errors.New("user has no email set")
	ErrEmailDisabled            = errors.New("password reset is not available")
	ErrInvalidResetToken        = errors.New("reset link is invalid or has expired")
	ErrInvalidEmail             = errors.New("a valid email address is required")
	ErrEmailTaken               = errors.New("that email address is already in use")
	ErrInvalidVerificationToken = errors.New("verification link is invalid or has expired")
	ErrEmailAlreadyVerified     = errors.New("your email is already verified")
)
