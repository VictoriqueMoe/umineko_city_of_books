package admin

import "errors"

var (
	ErrUserNotFound            = errors.New("user not found")
	ErrProtectedUser           = errors.New("this user cannot be modified")
	ErrUnknownRole             = errors.New("unknown role")
	ErrRoleOutranksActor       = errors.New("cannot grant a role equal to or above your own")
	ErrSystemRole              = errors.New("cannot modify system role assignments")
	ErrVanityRoleNotFound      = errors.New("vanity role not found")
	ErrNoEmailAddress          = errors.New("your account has no email address set")
	ErrBannedGiphyInvalidInput = errors.New("could not recognise a Giphy URL or ID in the input")
	ErrBannedGiphyKindMismatch = errors.New("supplied kind does not match what was extracted from the URL")
)
