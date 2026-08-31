package post

import (
	"context"
	"time"

	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/mention"

	"github.com/google/uuid"
)

const botObserveTimeout = 15 * time.Second

type (
	CommentObserver interface {
		Enabled() bool
		ObserveComment(ev BotContentEvent)
	}

	BotContentEvent struct {
		PostID       uuid.UUID
		CommentID    *uuid.UUID
		AuthorID     uuid.UUID
		AuthorName   string
		AuthorHandle string
		Body         string
		MentionedIDs map[uuid.UUID]struct{}
		ParentID     *uuid.UUID
		ParentAuthor uuid.UUID
	}
)

func (s *service) SetCommentObserver(obs CommentObserver) {
	s.botObserver = obs
}

func (s *service) observeForBot(postID uuid.UUID, commentID *uuid.UUID, authorID uuid.UUID, body string, parentID *uuid.UUID) {
	if s.botObserver == nil || !s.botObserver.Enabled() {
		return
	}

	usernames := mention.Usernames(body)
	if len(usernames) == 0 && parentID == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), botObserveTimeout)
	defer cancel()

	author, err := s.userRepo.GetByID(ctx, authorID)
	if err != nil || author == nil || author.IsBot {
		return
	}

	var parentAuthor uuid.UUID
	if parentID != nil {
		parent, err := s.postRepo.GetCommentByID(ctx, *parentID)
		if err == nil && parent != nil && parent.EntityID == postID.String() {
			parentAuthor = parent.UserID
		}
	}

	recipients, err := s.mentionSvc.Recipients(ctx, usernames, authorID)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Msg("resolve bot mention recipients failed")
	}

	mentioned := make(map[uuid.UUID]struct{}, len(recipients))
	for _, u := range recipients {
		mentioned[u.ID] = struct{}{}
	}

	if len(mentioned) == 0 && parentAuthor == uuid.Nil {
		return
	}

	s.botObserver.ObserveComment(BotContentEvent{
		PostID:       postID,
		CommentID:    commentID,
		AuthorID:     authorID,
		AuthorName:   author.DisplayLabel(),
		AuthorHandle: author.Username,
		Body:         body,
		MentionedIDs: mentioned,
		ParentID:     parentID,
		ParentAuthor: parentAuthor,
	})
}
