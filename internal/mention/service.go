package mention

import (
	"context"
	"errors"
	"fmt"
	"time"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

const (
	recipientBudget = 40 * time.Second
	fanOutTimeout   = MaxMatches * recipientBudget
	emailAction     = "mentioned you"
)

type (
	CommentSpec struct {
		Kind                 Kind
		EntityID             uuid.UUID
		EntityKey            string
		EntryID              *uuid.UUID
		EntryNumber          int
		ParentID             *uuid.UUID
		AuthorID             uuid.UUID
		Body                 string
		RecordAuthorActivity bool
	}

	Service interface {
		Notify(ctx context.Context, ref Reference, actorID uuid.UUID, body string) error
		NotifyAsync(ctx context.Context, ref Reference, actorID uuid.UUID, body string)
		CreateComment(ctx context.Context, spec CommentSpec) (uuid.UUID, error)
		Recipients(ctx context.Context, usernames []string, actorID uuid.UUID) ([]model.User, error)
	}

	service struct {
		userRepo repository.UserRepository
		blockSvc block.Service
		notifSvc notification.Service
		comments repository.CommentDAOs
	}
)

var ErrNoCommentDAO = errors.New("mention: kind has no comment dao")

func NewService(userRepo repository.UserRepository, blockSvc block.Service, notifSvc notification.Service, comments repository.CommentDAOs) Service {
	return &service{
		userRepo: userRepo,
		blockSvc: blockSvc,
		notifSvc: notifSvc,
		comments: comments,
	}
}

func ValidateCommentDAOs(comments repository.CommentDAOs) error {
	for _, kind := range CommentKinds() {
		if !hasCommentDAO(kind, comments) {
			return fmt.Errorf("%w: %q", ErrNoCommentDAO, kind)
		}
	}

	return nil
}

func hasCommentDAO(kind Kind, comments repository.CommentDAOs) bool {
	if isJournalCommentKind(kind) {
		return comments.Journal != nil
	}

	if comments.BySlug[string(kind)] != nil {
		return true
	}

	return comments.ByID[string(kind)] != nil
}

func isJournalCommentKind(kind Kind) bool {
	return kind == KindJournalComment || kind == KindJournalEntryComment
}

func (s *service) CreateComment(ctx context.Context, spec CommentSpec) (uuid.UUID, error) {
	commentID, err := s.writeComment(ctx, spec)
	if err != nil {
		return uuid.Nil, err
	}

	s.notifyChild(ctx, spec, commentID)

	return commentID, nil
}

func (s *service) writeComment(ctx context.Context, spec CommentSpec) (uuid.UUID, error) {
	if isJournalCommentKind(spec.Kind) {
		if s.comments.Journal == nil {
			return uuid.Nil, fmt.Errorf("%w: %q", ErrNoCommentDAO, spec.Kind)
		}

		created, err := s.comments.Journal.CreateComment(ctx, repository.NewJournalComment{
			JournalID:            spec.EntityID,
			EntryID:              spec.EntryID,
			ParentID:             spec.ParentID,
			UserID:               spec.AuthorID,
			Body:                 spec.Body,
			RecordAuthorActivity: spec.RecordAuthorActivity,
		})
		if err != nil {
			return uuid.Nil, err
		}

		return created.ID, nil
	}

	if bySlug := s.comments.BySlug[string(spec.Kind)]; bySlug != nil {
		created, err := bySlug.CreateComment(ctx, spec.EntityKey, spec.ParentID, spec.AuthorID, spec.Body)
		if err != nil {
			return uuid.Nil, err
		}

		return created.ID, nil
	}

	byID := s.comments.ByID[string(spec.Kind)]
	if byID == nil {
		return uuid.Nil, fmt.Errorf("%w: %q", ErrNoCommentDAO, spec.Kind)
	}

	created, err := byID.CreateComment(ctx, spec.EntityID, spec.ParentID, spec.AuthorID, spec.Body)
	if err != nil {
		return uuid.Nil, err
	}

	return created.ID, nil
}

func (s *service) notifyChild(ctx context.Context, spec CommentSpec, childID uuid.UUID) {
	s.NotifyAsync(ctx, Reference{
		Kind:        spec.Kind,
		EntityID:    spec.EntityID,
		EntityKey:   spec.EntityKey,
		EntryNumber: spec.EntryNumber,
		ChildID:     childID,
	}, spec.AuthorID, spec.Body)
}

func (s *service) NotifyAsync(ctx context.Context, ref Reference, actorID uuid.UUID, body string) {
	detached, cancel := context.WithTimeout(context.WithoutCancel(ctx), fanOutTimeout)

	go func() {
		defer cancel()

		if err := s.Notify(detached, ref, actorID, body); err != nil {
			logger.Ctx(detached).Error().Err(err).
				Str("kind", string(ref.Kind)).
				Str("actor_id", actorID.String()).
				Msg("mention fan-out failed")
		}
	}()
}

func (s *service) Notify(ctx context.Context, ref Reference, actorID uuid.UUID, body string) error {
	target, err := ref.Resolve()
	if err != nil {
		return err
	}

	usernames := Usernames(body)
	if len(usernames) == 0 {
		return nil
	}

	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return fmt.Errorf("load mention actor %s: %w", actorID, err)
	}
	if actor == nil || actor.IsBot {
		return nil
	}

	mentioned, err := s.Recipients(ctx, usernames, actorID)
	if err != nil {
		return err
	}

	referenceID := target.ReferenceID(ref)
	referenceType := target.ReferenceType(ref)
	linkURL := target.LinkURL(ref)

	for _, recipient := range mentioned {
		blocked, err := s.blockSvc.IsBlockedEither(ctx, actorID, recipient.ID)
		if err != nil {
			logger.Ctx(ctx).Warn().Err(err).
				Str("actor_id", actorID.String()).
				Str("recipient_id", recipient.ID.String()).
				Msg("mention block check failed")
		}
		if blocked {
			continue
		}

		if err := s.notifSvc.Notify(ctx, dto.NotifyParams{
			RecipientID:   recipient.ID,
			Type:          dto.NotifMention,
			ReferenceID:   referenceID,
			ReferenceType: referenceType,
			ActorID:       actorID,
			EmailActor:    actor.DisplayName,
			EmailAction:   emailAction,
			EmailLink:     linkURL,
		}); err != nil {
			logger.Ctx(ctx).Warn().Err(err).
				Str("recipient_id", recipient.ID.String()).
				Str("reference_type", referenceType).
				Msg("mention notification failed")
		}
	}

	return nil
}

func (s *service) Recipients(ctx context.Context, usernames []string, actorID uuid.UUID) ([]model.User, error) {
	if len(usernames) == 0 {
		return nil, nil
	}

	users, err := s.userRepo.GetByUsernames(ctx, usernames)
	if err != nil {
		return nil, fmt.Errorf("resolve mentioned usernames: %w", err)
	}

	out := make([]model.User, 0, len(users))
	for _, u := range users {
		if u.ID == actorID {
			continue
		}

		out = append(out, u)
	}

	return out, nil
}
