package authz

import "errors"

var (
	ErrNotCommentAuthor = errors.New("not the comment author")
)
