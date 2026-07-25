package art

import "errors"

var (
	ErrNotFound    = errors.New("art not found")
	ErrEmptyTitle  = errors.New("art title cannot be empty")
	ErrRateLimited = errors.New("you have reached your daily art upload limit")
)
