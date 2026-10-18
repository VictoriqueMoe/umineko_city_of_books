package dao

import (
	"errors"
	"fmt"

	"umineko_city_of_books/internal/model"
)

type (
	ChatMessageReplayError struct {
		Existing model.ChatMessageRow
	}
)

var (
	ErrNotFound           = errors.New("not found")
	ErrAlreadyVoted       = errors.New("already voted")
	ErrUnknownCommentKind = errors.New("unknown comment kind")
)

func (e *ChatMessageReplayError) Error() string {
	return fmt.Sprintf("message %s was already sent with this client nonce", e.Existing.ID)
}
