package post

import "errors"

var (
	ErrNotFound        = errors.New("post not found")
	ErrEmptyBody       = errors.New("post body cannot be empty")
	ErrRateLimited     = errors.New("you have reached your daily post limit")
	ErrInvalidPoll     = errors.New("poll must have between 2 and 10 options")
	ErrInvalidDuration = errors.New("invalid poll duration")
	ErrPollExpired     = errors.New("this poll has expired")
	ErrAlreadyVoted    = errors.New("you have already voted on this poll")
	ErrInvalidOption   = errors.New("invalid poll option")
)
