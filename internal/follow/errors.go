package follow

import "errors"

var (
	ErrCannotFollowSelf = errors.New("cannot follow yourself")
)
