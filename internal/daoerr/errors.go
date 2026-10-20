package daoerr

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

	ErrArtNotOwned = errors.New("art or gallery not found or not owned")

	ErrBotNotFound        = errors.New("chatbot not found")
	ErrBasePromptNotFound = errors.New("base prompt not found")
	ErrBasePromptNameUsed = errors.New("that base prompt name is already taken")
	ErrBasePromptInUse    = errors.New("that base prompt is still used by a chatbot")

	ErrRoomNotActive = errors.New("game room is not active")

	ErrInviteUnavailable = errors.New("invite code is missing or already used")

	ErrLiveStreamCapacity     = errors.New("live stream capacity reached")
	ErrLiveStreamActiveExists = errors.New("user already has an active live stream")

	ErrRefutationRejected = errors.New("the response cannot refute this theory")
)

func (e *ChatMessageReplayError) Error() string {
	return fmt.Sprintf("message %s was already sent with this client nonce", e.Existing.ID)
}
