package chat

import "errors"

var (
	ErrDmsDisabled   = errors.New("recipient has DMs disabled")
	ErrUserNotFound  = errors.New("user not found")
	ErrNotMember     = errors.New("not a member of this room")
	ErrRoomNotFound  = errors.New("room not found")
	ErrMissingFields = errors.New("missing required fields")
	ErrCannotDMSelf  = errors.New("cannot create DM with yourself")
	ErrUserBlocked   = errors.New("you cannot message this user")
)
