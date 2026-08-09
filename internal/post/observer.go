package post

import (
	"context"
	"time"

	"umineko_city_of_books/internal/social"

	"github.com/google/uuid"
)

const (
	botObserveTimeout = 15 * time.Second
	maxBotMentions    = 20
)

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

	usernames := mentionedUsernames(body)
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

	mentioned := s.resolveMentionedIDs(ctx, usernames, authorID)
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

func mentionedUsernames(body string) []string {
	matches := social.MentionRegex.FindAllStringSubmatch(body, maxBotMentions)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if _, dup := seen[m[1]]; dup {
			continue
		}

		seen[m[1]] = struct{}{}
		out = append(out, m[1])
	}

	return out
}

func (s *service) resolveMentionedIDs(ctx context.Context, usernames []string, authorID uuid.UUID) map[uuid.UUID]struct{} {
	if len(usernames) == 0 {
		return nil
	}

	users, err := s.userRepo.GetByUsernames(ctx, usernames)
	if err != nil || len(users) == 0 {
		return nil
	}

	out := make(map[uuid.UUID]struct{}, len(users))
	for i := range users {
		if users[i].ID == authorID {
			continue
		}

		out[users[i].ID] = struct{}{}
	}

	return out
}
