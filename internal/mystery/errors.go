package mystery

import "errors"

var (
	ErrNotFound      = errors.New("mystery not found")
	ErrEmptyBody     = errors.New("body is required")
	ErrEmptyTitle    = errors.New("title and body are required")
	ErrAlreadySolved = errors.New("this mystery has already been solved")
	ErrNotAuthor     = errors.New("only the author can perform this action")
	ErrCannotReply   = errors.New("only the game master or the attempt author can reply")
	ErrInvalidVote   = errors.New("value must be 1, -1, or 0")
)
