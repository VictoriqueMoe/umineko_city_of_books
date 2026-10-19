package comment

import "errors"

var (
	ErrEmptyBody      = errors.New("comment body is required")
	ErrNotFound       = errors.New("comment not found")
	ErrTargetNotFound = errors.New("the content being commented on was not found")
	ErrForbidden      = errors.New("not allowed to change this comment")
)
