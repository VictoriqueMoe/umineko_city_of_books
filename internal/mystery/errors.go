package mystery

import "errors"

var (
	ErrNotFound       = errors.New("mystery not found")
	ErrAttachmentType = errors.New("only PDF, TXT, and DOCX files are allowed")
	ErrEmptyBody      = errors.New("body is required")
	ErrEmptyTitle     = errors.New("title and body are required")
	ErrAlreadySolved  = errors.New("this mystery has already been solved")
	ErrNotAuthor      = errors.New("only the author can perform this action")
	ErrMysteryPaused  = errors.New("this mystery is currently paused")
	ErrCannotReply    = errors.New("only the game master or the attempt author can reply")
	ErrInvalidVote    = errors.New("value must be 1, -1, or 0")
	ErrNotSolved      = errors.New("discussion comments are only available after the mystery is solved")
	ErrContractLocked = errors.New("the Knox contract is sealed once a piece has submitted an attempt")
)
