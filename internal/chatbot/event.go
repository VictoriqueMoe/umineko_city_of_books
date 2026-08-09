package chatbot

import (
	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/post"

	"github.com/google/uuid"
)

const (
	SurfaceChat        Surface = "chat"
	SurfacePost        Surface = "post"
	SurfacePostComment Surface = "post_comment"
)

type (
	Surface string

	scopeKey struct {
		surface Surface
		id      uuid.UUID
	}

	botEvent struct {
		Surface      Surface
		IsDM         bool
		ScopeID      uuid.UUID
		ItemID       uuid.UUID
		SenderID     uuid.UUID
		SenderName   string
		SenderHandle string
		Body         string
		MentionedIDs map[uuid.UUID]struct{}
		ParentID     *uuid.UUID
		ParentAuthor uuid.UUID
		Audience     []uuid.UUID
	}
)

func (s Surface) gameBoard() bool {
	return s == SurfacePost || s == SurfacePostComment
}

func (s Surface) scope() Surface {
	if s == SurfacePostComment {
		return SurfacePost
	}

	return s
}

func (ev botEvent) scopeKey() scopeKey {
	return scopeKey{surface: ev.Surface.scope(), id: ev.ScopeID}
}

func (s *service) ObserveMessage(ev chat.BotMessageEvent) {
	s.observe(botEvent{
		Surface:      SurfaceChat,
		IsDM:         ev.RoomType == string(dto.RoomTypeDM),
		ScopeID:      ev.RoomID,
		ItemID:       ev.MessageID,
		SenderID:     ev.SenderID,
		SenderName:   ev.SenderName,
		SenderHandle: ev.SenderHandle,
		Body:         ev.Body,
		MentionedIDs: ev.MentionedIDs,
		ParentID:     ev.ReplyToID,
		ParentAuthor: ev.ReplyToAuthor,
		Audience:     ev.Members,
	})
}

func (s *service) ObserveComment(ev post.BotContentEvent) {
	surface := SurfacePost
	itemID := ev.PostID

	if ev.CommentID != nil {
		surface = SurfacePostComment
		itemID = *ev.CommentID
	}

	s.observe(botEvent{
		Surface:      surface,
		ScopeID:      ev.PostID,
		ItemID:       itemID,
		SenderID:     ev.AuthorID,
		SenderName:   ev.AuthorName,
		SenderHandle: ev.AuthorHandle,
		Body:         ev.Body,
		MentionedIDs: ev.MentionedIDs,
		ParentID:     ev.ParentID,
		ParentAuthor: ev.ParentAuthor,
	})
}
