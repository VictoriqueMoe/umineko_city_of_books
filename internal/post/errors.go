package post

import (
	"errors"
	"fmt"

	"umineko_city_of_books/internal/dao"
)

var (
	ErrNotFound         = fmt.Errorf("post not found: %w", dao.ErrNotFound)
	ErrEmptyBody        = errors.New("post body cannot be empty")
	ErrRateLimited      = errors.New("you have reached your daily post limit")
	ErrInvalidPoll      = errors.New("poll must have between 2 and 10 options")
	ErrInvalidDuration  = errors.New("invalid poll duration")
	ErrPollExpired      = errors.New("this poll has expired")
	ErrAlreadyVoted     = errors.New("you have already voted on this poll")
	ErrInvalidOption    = errors.New("invalid poll option")
	ErrInvalidShareType = errors.New("invalid shared content type")
)
