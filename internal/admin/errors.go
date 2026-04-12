package admin

import "errors"

var (
	ErrPermissionDenied   = errors.New("permission denied")
	ErrUserNotFound       = errors.New("user not found")
	ErrProtectedUser      = errors.New("this user cannot be modified")
	ErrSystemRole         = errors.New("cannot modify system role assignments")
	ErrVanityRoleNotFound = errors.New("vanity role not found")
)
