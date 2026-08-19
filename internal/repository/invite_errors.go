package repository

import "errors"

var (
	ErrInviteUnavailable = errors.New("invite code is missing or already used")
)
