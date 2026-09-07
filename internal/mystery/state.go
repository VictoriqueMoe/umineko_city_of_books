package mystery

import (
	"context"
	"fmt"
	"strings"
	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

func (s *service) SetPaused(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, paused bool) error {
	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return ErrNotAuthor
	}
	if err := s.mysteryRepo.SetPaused(ctx, spec.MysteryPauseUpdate{MysteryID: mysteryID, Paused: paused}); err != nil {
		return err
	}
	s.hub.Broadcast(ws.Message{
		Type: "mystery_paused",
		Data: map[string]any{
			"mystery_id": mysteryID,
			"paused":     paused,
		},
	})

	go func() {
		bgCtx := context.Background()
		playerIDs, err := s.mysteryRepo.GetPlayerIDs(bgCtx, mysteryID)
		if err != nil || len(playerIDs) == 0 {
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
	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return ErrNotAuthor
	}
	if err := s.mysteryRepo.SetGmAway(ctx, spec.MysteryGmAwayUpdate{MysteryID: mysteryID, Away: away}); err != nil {
		return err
	}
	s.hub.Broadcast(ws.Message{
		Type: "mystery_gm_away",
		Data: map[string]any{
			"mystery_id": mysteryID,
			"gm_away":    away,
		},
	})

	go func() {
		bgCtx := context.Background()
		playerIDs, err := s.mysteryRepo.GetPlayerIDs(bgCtx, mysteryID)
		if err != nil || len(playerIDs) == 0 {
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
	if err := s.mysteryRepo.DeleteClue(ctx, clueID); err != nil {
		return err
	}

	s.audit(ctx, audit.NewEntry{
		ActorID:    userID,
		Action:     audit.ActionMysteryClueDelete,
		TargetType: audit.TargetMystery,
		TargetID:   mysteryID.String(),
		Details:    fmt.Sprintf("clue=%d", clueID),
	})

	s.hub.Broadcast(ws.Message{
		Type: "mystery_clue_updated",
		Data: map[string]any{"mystery_id": mysteryID},
	})
	return nil
}

func (s *service) UpdateClue(ctx context.Context, mysteryID uuid.UUID, clueID int, userID uuid.UUID, body string) error {
	if strings.TrimSpace(body) == "" {
		return ErrEmptyBody
	}
	if err := s.contentFilter.Check(ctx, body); err != nil {
		return err
	}
	if err := s.mysteryRepo.UpdateClue(ctx, spec.MysteryClueUpdate{ClueID: clueID, Body: strings.TrimSpace(body)}); err != nil {
		return err
	}

	s.audit(ctx, audit.NewEntry{
		ActorID:    userID,
		Action:     audit.ActionMysteryClueUpdate,
		TargetType: audit.TargetMystery,
		TargetID:   mysteryID.String(),
		Details:    fmt.Sprintf("clue=%d", clueID),
	})

	s.hub.Broadcast(ws.Message{
		Type: "mystery_clue_updated",
		Data: map[string]any{"mystery_id": mysteryID},
	})
	return nil
}
