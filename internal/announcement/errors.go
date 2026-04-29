package announcement

import "errors"

var (
	ErrNotFound         = errors.New("announcement not found")
	ErrForbidden        = errors.New("forbidden")
	ErrBlocked          = errors.New("user is blocked")
	ErrEmptyBody        = errors.New("body is required")
	ErrEmptyTitleOrBody = errors.New("title and body are required")
	ErrCommentNotFound  = errors.New("comment not found")
)
