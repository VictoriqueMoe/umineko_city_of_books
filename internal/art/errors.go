package art

import "errors"

var (
	ErrNotFound    = errors.New("art not found")
	ErrEmptyTitle  = errors.New("art title cannot be empty")
	ErrNoImage     = errors.New("art must include an image")
	ErrRateLimited = errors.New("you have reached your daily art upload limit")
)
