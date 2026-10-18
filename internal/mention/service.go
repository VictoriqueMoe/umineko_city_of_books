package mention

import (
	"context"
	"fmt"
	"time"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

const (
	recipientBudget = 40 * time.Second
	fanOutTimeout   = MaxMatches * recipientBudget
	emailAction     = "mentioned you"
)

type (
	Service interface {
		Notify(ctx context.Context, ref Reference, actorID uuid.UUID, body string) error
		NotifyAsync(ctx context.Context, ref Reference, actorID uuid.UUID, body string)
		Recipients(ctx context.Context, usernames []string, actorID uuid.UUID) ([]model.User, error)
	}

	service struct {
		userRepo repository.UserRepository
		blockSvc block.Service
		notifSvc notification.Service
	}
)

func NewService(userRepo repository.UserRepository, blockSvc block.Service, notifSvc notification.Service) Service {
	return &service{
		userRepo: userRepo,
		blockSvc: blockSvc,
		notifSvc: notifSvc,
	}
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
