package admin

import "errors"

var (
	ErrPermissionDenied        = errors.New("permission denied")
	ErrUserNotFound            = errors.New("user not found")
	ErrProtectedUser           = errors.New("this user cannot be modified")
	ErrSystemRole              = errors.New("cannot modify system role assignments")
	ErrVanityRoleNotFound      = errors.New("vanity role not found")
	ErrBannedGiphyInvalidInput = errors.New("could not recognise a Giphy URL or ID in the input")
	ErrBannedGiphyKindMismatch = errors.New("supplied kind does not match what was extracted from the URL")
)
