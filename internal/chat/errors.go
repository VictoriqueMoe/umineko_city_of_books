package chat

import "errors"

var (
	ErrDmsDisabled       = errors.New("recipient has DMs disabled")
	ErrUserNotFound      = errors.New("user not found")
	ErrNotMember         = errors.New("not a member of this room")
	ErrRoomNotFound      = errors.New("room not found")
	ErrMissingFields     = errors.New("missing required fields")
	ErrCannotDMSelf      = errors.New("cannot create DM with yourself")
	ErrUserBlocked       = errors.New("you cannot message this user")
	ErrCannotLeaveAsHost = errors.New("host cannot leave their own room")
	ErrNotHost           = errors.New("only the host can do this")
	ErrCannotKickHost    = errors.New("cannot kick the host")
	ErrRoomFull          = errors.New("room is full")
	ErrNotPublic         = errors.New("room is not public")
	ErrAlreadyMember     = errors.New("already a member")
	ErrNotGroupRoom      = errors.New("not a group room")
	ErrRateLimited       = errors.New("daily limit reached")
	ErrSystemRoom        = errors.New("system rooms are managed automatically")
)
