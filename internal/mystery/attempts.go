package mystery

import (
	"context"
	"fmt"
	"strings"
	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

func (s *service) CreateAttempt(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, req dto.CreateAttemptRequest) (uuid.UUID, error) {
	if strings.TrimSpace(req.Body) == "" {
		return uuid.Nil, ErrEmptyBody
	}
	if err := s.contentFilter.Check(ctx, req.Body); err != nil {
		return uuid.Nil, err
	}

	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return uuid.Nil, ErrNotFound
	}
	if solved, err := s.mysteryRepo.IsSolved(ctx, mysteryID); err != nil {
		return uuid.Nil, err
	} else if solved {
		return uuid.Nil, ErrAlreadySolved
	}
	if userID != authorID {
		if won, err := s.mysteryRepo.UserHasWinningAttempt(ctx, spec.MysterySolverQuery{MysteryID: mysteryID, UserID: userID}); err != nil {
			return uuid.Nil, err
		} else if won {
			return uuid.Nil, ErrAlreadySolved
		}
	}
	if paused, _ := s.mysteryRepo.IsPaused(ctx, mysteryID); paused && authorID != userID {
		return uuid.Nil, ErrMysteryPaused
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return uuid.Nil, block.ErrUserBlocked
	}

	if req.ParentID != nil {
		parentAuthor, err := s.mysteryRepo.GetAttemptAuthorID(ctx, *req.ParentID)
		if err != nil {
			return uuid.Nil, ErrNotFound
		}
		if userID != authorID && userID != parentAuthor {
			return uuid.Nil, ErrCannotReply
		}
	}

	body := strings.TrimSpace(req.Body)

	created, err := s.mysteryRepo.CreateAttempt(ctx, spec.NewMysteryAttempt{
		MysteryID: mysteryID,
		UserID:    userID,
		ParentID:  req.ParentID,
		Body:      body,
	})
	if err != nil {
		return uuid.Nil, err
	}

	s.mentionSvc.NotifyAsync(ctx, mention.Reference{
		Kind:     mention.KindMysteryAttempt,
		EntityID: mysteryID,
		ChildID:  created.ID,
	}, userID, body)

	wsData := map[string]any{
		"mystery_id":          mysteryID,
		"attempt_id":          created.ID,
		"parent_id":           req.ParentID,
		"author_id":           userID,
		"author_username":     created.AuthorUsername,
		"author_display_name": created.AuthorDisplayName,
		"author_avatar_url":   created.AuthorAvatarURL,
	}
	s.hub.Broadcast(ws.Message{
		Type: "mystery_attempt_created",
		Data: wsData,
	})

	go func() {
		bgCtx := context.Background()
		linkPath := fmt.Sprintf("/mystery/%s#attempt-%s", mysteryID, created.ID)
		attemptRef := fmt.Sprintf("mystery_attempt:%s", created.ID)

		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifMysteryAttempt,
			ReferenceID:   mysteryID,
			ReferenceType: attemptRef,
			ActorID:       userID,
			EmailActor:    "Someone",
			EmailAction:   "submitted an attempt on your mystery",
			EmailLink:     linkPath,
		})

		if req.ParentID != nil {
			if parentAuthor, err := s.mysteryRepo.GetAttemptAuthorID(bgCtx, *req.ParentID); err == nil && parentAuthor != authorID {
				_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
					RecipientID:   parentAuthor,
					Type:          dto.NotifMysteryReply,
					ReferenceID:   mysteryID,
					ReferenceType: attemptRef,
					ActorID:       userID,
					EmailActor:    "Someone",
					EmailAction:   "replied to your attempt",
					EmailLink:     linkPath,
				})
			}
		}
	}()

	return created.ID, nil
}

func (s *service) DeleteAttempt(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if !s.authz.Can(ctx, userID, authz.PermDeleteAnyComment) {
		return s.mysteryRepo.DeleteAttempt(ctx, spec.MysteryAttemptDeletion{ID: id, UserID: userID})
	}

	attemptAuthorID, err := s.mysteryRepo.GetAttemptAuthorID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	if err := s.mysteryRepo.DeleteAttemptAsAdmin(ctx, id); err != nil {
		return err
	}

	if attemptAuthorID != userID {
		s.audit(ctx, audit.NewEntry{
			ActorID:    userID,
			Action:     audit.ActionMysteryAttemptDeleteAdmin,
			TargetType: audit.TargetMysteryAttempt,
			TargetID:   id.String(),
			SubjectID:  attemptAuthorID,
		})
	}

	return nil
}

func (s *service) VoteAttempt(ctx context.Context, attemptID uuid.UUID, userID uuid.UUID, value int) error {
	if value != 1 && value != -1 && value != 0 {
		return ErrInvalidVote
	}

	attemptAuthorID, err := s.mysteryRepo.GetAttemptAuthorID(ctx, attemptID)
	if err != nil {
		return ErrNotFound
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, attemptAuthorID); blocked {
		return block.ErrUserBlocked
	}

	if err := s.mysteryRepo.VoteAttempt(ctx, spec.Vote{UserID: userID, TargetID: attemptID, Value: value}); err != nil {
		return err
	}

	if value != 0 {
		go func() {
			bgCtx := context.Background()
			mysteryID, err := s.mysteryRepo.GetAttemptMysteryID(bgCtx, attemptID)
			if err != nil {
				return
			}
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   attemptAuthorID,
				Type:          dto.NotifMysteryVote,
				ReferenceID:   mysteryID,
				ReferenceType: fmt.Sprintf("mystery_attempt:%s", attemptID),
				ActorID:       userID,
				EmailActor:    "Someone",
				EmailAction:   "voted on your attempt",
				EmailLink:     fmt.Sprintf("/mystery/%s#attempt-%s", mysteryID, attemptID),
			})
		}()
	}

	return nil
}
