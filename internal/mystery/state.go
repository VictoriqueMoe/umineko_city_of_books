package mystery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

func (s *service) SetPaused(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, paused bool) error {
	authorID, err := s.mysteryAuthor(ctx, mysteryID)
	if err != nil {
		return err
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyWork) {
		return ErrNotAuthor
	}
	if err := s.mysteryRepo.SetPaused(ctx, spec.MysteryPauseUpdate{MysteryID: mysteryID, Paused: paused}); err != nil {
		return err
	}
	s.hub.BroadcastToTopic(mysteryTopic(mysteryID), ws.Message{
		Type: "mystery_paused",
		Data: map[string]any{
			"mystery_id": mysteryID,
			"paused":     paused,
		},
	})

	go func() {
		bgCtx := context.Background()
		playerIDs, err := s.mysteryRepo.GetPlayerIDs(bgCtx, mysteryID)
		if err != nil {
			logger.Ctx(bgCtx).Warn().Err(err).Str("mystery_id", mysteryID.String()).Msg("pause notification skipped, player lookup failed")

			return
		}
		if len(playerIDs) == 0 {
			return
		}
		notifType := dto.NotifMysteryPaused
		message := "paused a mystery you are playing"
		if !paused {
			notifType = dto.NotifMysteryUnpaused
			message = "resumed a mystery you are playing"
		}

		linkPath := fmt.Sprintf("/mystery/%s", mysteryID)
		for _, pid := range playerIDs {
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   pid,
				Type:          notifType,
				ReferenceID:   mysteryID,
				ReferenceType: "mystery",
				ActorID:       userID,
				Message:       message,
				EmailActor:    "The Game Master",
				EmailAction:   message,
				EmailLink:     linkPath,
			})
		}
	}()

	return nil
}

func (s *service) SetGmAway(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, away bool) error {
	authorID, err := s.mysteryAuthor(ctx, mysteryID)
	if err != nil {
		return err
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyWork) {
		return ErrNotAuthor
	}
	if err := s.mysteryRepo.SetGmAway(ctx, spec.MysteryGmAwayUpdate{MysteryID: mysteryID, Away: away}); err != nil {
		return err
	}
	s.hub.BroadcastToTopic(mysteryTopic(mysteryID), ws.Message{
		Type: "mystery_gm_away",
		Data: map[string]any{
			"mystery_id": mysteryID,
			"gm_away":    away,
		},
	})

	go func() {
		bgCtx := context.Background()
		playerIDs, err := s.mysteryRepo.GetPlayerIDs(bgCtx, mysteryID)
		if err != nil {
			logger.Ctx(bgCtx).Warn().Err(err).Str("mystery_id", mysteryID.String()).Msg("gm away notification skipped, player lookup failed")

			return
		}
		if len(playerIDs) == 0 {
			return
		}
		notifType := dto.NotifMysteryGmAway
		message := "marked themselves as away on a mystery you are playing"
		if !away {
			notifType = dto.NotifMysteryGmBack
			message = "is back on a mystery you are playing"
		}

		linkPath := fmt.Sprintf("/mystery/%s", mysteryID)
		for _, pid := range playerIDs {
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   pid,
				Type:          notifType,
				ReferenceID:   mysteryID,
				ReferenceType: "mystery",
				ActorID:       userID,
				Message:       message,
				EmailActor:    "The Game Master",
				EmailAction:   message,
				EmailLink:     linkPath,
			})
		}
	}()

	return nil
}

func (s *service) DeleteClue(ctx context.Context, mysteryID uuid.UUID, clueID int, userID uuid.UUID) error {
	authorID, err := s.mysteryAuthor(ctx, mysteryID)
	if err != nil {
		return err
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyWork) {
		return ErrNotAuthor
	}

	err = s.mysteryRepo.DeleteClue(ctx, spec.MysteryClueDeletion{MysteryID: mysteryID, ClueID: clueID})
	if errors.Is(err, daoerr.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if authorID != userID {
		s.audit(ctx, audit.NewEntry{
			ActorID:    userID,
			Action:     audit.ActionMysteryClueDelete,
			TargetType: audit.TargetMystery,
			TargetID:   mysteryID.String(),
			Details:    fmt.Sprintf("clue=%d", clueID),
			SubjectID:  authorID,
		})
	}

	s.hub.BroadcastToTopic(mysteryTopic(mysteryID), ws.Message{
		Type: "mystery_clue_updated",
		Data: map[string]any{"mystery_id": mysteryID},
	})
	return nil
}

func (s *service) UpdateClue(ctx context.Context, mysteryID uuid.UUID, clueID int, userID uuid.UUID, body string) error {
	if strings.TrimSpace(body) == "" {
		return ErrEmptyBody
	}
	authorID, err := s.mysteryAuthor(ctx, mysteryID)
	if err != nil {
		return err
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyWork) {
		return ErrNotAuthor
	}
	if err := s.contentFilter.Check(ctx, body); err != nil {
		return err
	}

	err = s.mysteryRepo.UpdateClue(ctx, spec.MysteryClueUpdate{MysteryID: mysteryID, ClueID: clueID, Body: strings.TrimSpace(body)})
	if errors.Is(err, daoerr.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if authorID != userID {
		s.audit(ctx, audit.NewEntry{
			ActorID:    userID,
			Action:     audit.ActionMysteryClueUpdate,
			TargetType: audit.TargetMystery,
			TargetID:   mysteryID.String(),
			Details:    fmt.Sprintf("clue=%d", clueID),
			SubjectID:  authorID,
		})
	}

	s.hub.BroadcastToTopic(mysteryTopic(mysteryID), ws.Message{
		Type: "mystery_clue_updated",
		Data: map[string]any{"mystery_id": mysteryID},
	})
	return nil
}
