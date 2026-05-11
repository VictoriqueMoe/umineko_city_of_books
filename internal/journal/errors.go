package journal

import "errors"

var (
	ErrNotFound        = errors.New("journal not found")
	ErrNotAuthor       = errors.New("not the journal author")
	ErrRateLimited     = errors.New("daily limit reached")
	ErrArchived        = errors.New("journal is archived")
	ErrCannotFollowOwn = errors.New("cannot follow your own journal")
	ErrEmptyBody       = errors.New("body is required")
	ErrEmptyTitle      = errors.New("title is required")
	ErrEntryNotFound   = errors.New("journal entry not found")
	ErrEntryMismatch   = errors.New("entry does not belong to this journal")
)
