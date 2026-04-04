package block

import "errors"

var (
	ErrCannotBlockSelf  = errors.New("cannot block yourself")
	ErrCannotBlockStaff = errors.New("cannot block staff members")
	ErrUserBlocked      = errors.New("user is blocked")
)
