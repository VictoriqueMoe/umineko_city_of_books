package post

import "errors"

var (
	ErrNotFound    = errors.New("post not found")
	ErrEmptyBody   = errors.New("post body cannot be empty")
	ErrRateLimited = errors.New("you have reached your daily post limit")
)
