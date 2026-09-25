package mystery

import (
	"errors"
	"fmt"

	"umineko_city_of_books/internal/dao"
)

var (
	ErrNotFound       = fmt.Errorf("mystery not found: %w", dao.ErrNotFound)
	ErrEmptyBody      = errors.New("body is required")
	ErrEmptyTitle     = errors.New("title and body are required")
	ErrAlreadySolved  = errors.New("this mystery has already been solved")
	ErrNotAuthor      = errors.New("only the author can perform this action")
	ErrMysteryPaused  = errors.New("this mystery is currently paused")
	ErrCannotReply    = errors.New("only the game master or the attempt author can reply")
	ErrInvalidVote    = errors.New("value must be 1, -1, or 0")
	ErrNotSolved      = errors.New("discussion comments are only available after the mystery is solved")
	ErrContractLocked = errors.New("the Knox contract is sealed once a piece has submitted an attempt")

	ErrAttemptNotOnMystery = errors.New("that attempt does not belong to this mystery")
	ErrOwnAttempt          = errors.New("you cannot select your own attempt as the winner")
	ErrAlreadyWon          = errors.New("that player has already solved this mystery")
	ErrDuplicateAttachment = errors.New("a file with that name is already attached")
)
