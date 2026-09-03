package mystery

import (
	"context"
	"fmt"
	"strings"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

func (s *service) MarkSolved(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, attemptID uuid.UUID) error {
	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return ErrNotAuthor
	}

	attemptAuthorID, err := s.mysteryRepo.GetAttemptAuthorID(ctx, attemptID)
	if err != nil {
		return ErrNotFound
	}
	attemptMysteryID, err := s.mysteryRepo.GetAttemptMysteryID(ctx, attemptID)
	if err != nil {
		return ErrNotFound
	}
	if attemptMysteryID != mysteryID {
		return fmt.Errorf("attempt does not belong to this mystery")
	}
	if attemptAuthorID == authorID {
		return fmt.Errorf("cannot select your own attempt as the winner")
	}

	row, err := s.mysteryRepo.GetByID(ctx, mysteryID)
	if err != nil || row == nil {
		return ErrNotFound
	}
	if row.Solved {
		return ErrAlreadySolved
	}
	ongoing := row.KeepOpenAfterSolve

	if alreadyWon, err := s.mysteryRepo.UserHasWinningAttempt(ctx, mysteryID, attemptAuthorID); err != nil {
		return err
	} else if alreadyWon {
		return fmt.Errorf("user has already solved this mystery")
	}

	if err := s.mysteryRepo.MarkSolved(ctx, mysteryID, attemptID, !ongoing); err != nil {
		return err
	}

	s.audit(ctx, repository.NewAuditEntry{
		ActorID:    userID,
		Action:     repository.AuditActionMysterySolved,
		TargetType: repository.AuditTargetMystery,
		TargetID:   mysteryID.String(),
		Details:    fmt.Sprintf("attempt=%s", attemptID),
		SubjectID:  attemptAuthorID,
	})

	if ongoing {
		s.hub.Broadcast(ws.Message{
			Type: "mystery_winner_added",
			Data: map[string]any{
				"mystery_id": mysteryID,
				"winner_id":  attemptAuthorID,
			},
		})
	} else {
		s.hub.Broadcast(ws.Message{
			Type: "mystery_solved",
			Data: map[string]any{
				"mystery_id": mysteryID,
				"attempt_id": attemptID,
				"winner_id":  attemptAuthorID,
			},
		})
	}

	go func() {
		bgCtx := context.Background()
		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   attemptAuthorID,
			Type:          dto.NotifMysterySolved,
			ReferenceID:   mysteryID,
			ReferenceType: fmt.Sprintf("mystery_attempt:%s", attemptID),
			ActorID:       userID,
			EmailActor:    "Someone",
			EmailAction:   "chose your attempt as the winner!",
			EmailLink:     fmt.Sprintf("/mystery/%s#attempt-%s", mysteryID, attemptID),
		})

		if !ongoing {
			playerIDs, _ := s.mysteryRepo.GetPlayerIDs(bgCtx, mysteryID)
			solvedLink := fmt.Sprintf("/mystery/%s", mysteryID)
			params := make([]dto.NotifyParams, 0, len(playerIDs))
			for _, pid := range playerIDs {
				if pid == attemptAuthorID {
					continue
				}
				params = append(params, dto.NotifyParams{
					RecipientID:   pid,
					Type:          dto.NotifMysterySolvedAll,
					ReferenceID:   mysteryID,
					ReferenceType: "mystery",
					ActorID:       userID,
					Message:       "a mystery you were playing has been solved",
					EmailActor:    "The Game Master",
					EmailAction:   "solved a mystery you were playing",
					EmailLink:     solvedLink,
				})
			}
			s.notifService.NotifyMany(bgCtx, params)
		}

		topIDs, err := s.mysteryRepo.GetTopDetectiveIDs(bgCtx)
		if err == nil {
			s.hub.Broadcast(ws.Message{
				Type: "top_detective_changed",
				Data: map[string]any{
					"user_ids": topIDs,
				},
			})
		}
		if !ongoing {
			topGMIDs, err := s.mysteryRepo.GetTopGMIDs(bgCtx)
			if err == nil {
				s.hub.Broadcast(ws.Message{
					Type: "top_gm_changed",
					Data: map[string]any{
						"user_ids": topGMIDs,
					},
				})
			}
		}
	}()

	return nil
}

func (s *service) MarkPermanentlySolved(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return ErrNotAuthor
	}

	if err := s.mysteryRepo.MarkPermanentlySolved(ctx, mysteryID); err != nil {
		return err
	}

	by := "author"
	if authorID != userID {
		by = "staff"
	}

	s.audit(ctx, repository.NewAuditEntry{
		ActorID:    userID,
		Action:     repository.AuditActionMysteryClosed,
		TargetType: repository.AuditTargetMystery,
		TargetID:   mysteryID.String(),
		Details:    fmt.Sprintf("by=%s", by),
		SubjectID:  authorID,
	})

	s.hub.Broadcast(ws.Message{
		Type: "mystery_solved",
		Data: map[string]any{
			"mystery_id": mysteryID,
		},
	})

	go func() {
		bgCtx := context.Background()
		solvedLink := fmt.Sprintf("/mystery/%s", mysteryID)

		solverIDs, _ := s.mysteryRepo.GetSolverIDs(bgCtx, mysteryID)
		solverSet := make(map[uuid.UUID]struct{}, len(solverIDs))
		for _, sid := range solverIDs {
			solverSet[sid] = struct{}{}
		}

		playerIDs, _ := s.mysteryRepo.GetPlayerIDs(bgCtx, mysteryID)
		params := make([]dto.NotifyParams, 0, len(playerIDs))
		for _, pid := range playerIDs {
			if _, isSolver := solverSet[pid]; isSolver {
				continue
			}
			params = append(params, dto.NotifyParams{
				RecipientID:   pid,
				Type:          dto.NotifMysterySolvedAll,
				ReferenceID:   mysteryID,
				ReferenceType: "mystery",
				ActorID:       userID,
				Message:       "a mystery you were playing has been closed",
				EmailActor:    "The Game Master",
				EmailAction:   "closed a mystery you were playing",
				EmailLink:     solvedLink,
			})
		}
		s.notifService.NotifyMany(bgCtx, params)

		topGMIDs, err := s.mysteryRepo.GetTopGMIDs(bgCtx)
		if err == nil {
			s.hub.Broadcast(ws.Message{
				Type: "top_gm_changed",
				Data: map[string]any{
					"user_ids": topGMIDs,
				},
			})
		}
	}()

	return nil
}

func (s *service) AddClue(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, req dto.CreateClueRequest) error {
	if strings.TrimSpace(req.Body) == "" {
		return ErrEmptyBody
	}
	if err := s.contentFilter.Check(ctx, req.Body); err != nil {
		return err
	}

	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID {
		return ErrNotAuthor
	}

	if req.TruthType == "" {
		req.TruthType = "red"
	}

	count, _ := s.mysteryRepo.CountClues(ctx, mysteryID)
	if _, err := s.mysteryRepo.AddClue(ctx, mysteryID, repository.NewClue{
		Body:      req.Body,
		TruthType: req.TruthType,
		SortOrder: count,
		PlayerID:  req.PlayerID,
	}); err != nil {
		return err
	}

	wsData := map[string]any{
		"mystery_id": mysteryID,
		"truth_type": req.TruthType,
	}
	if req.PlayerID != nil {
		wsData["player_id"] = req.PlayerID
		s.hub.SendToUser(*req.PlayerID, ws.Message{
			Type: "mystery_clue_added",
			Data: wsData,
		})

		recipient := *req.PlayerID
		go func() {
			bgCtx := context.Background()
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   recipient,
				Type:          dto.NotifMysteryPrivateClue,
				ReferenceID:   mysteryID,
				ReferenceType: "mystery",
				ActorID:       userID,
				EmailActor:    "The Game Master",
				EmailAction:   "revealed a private red truth to you",
				EmailLink:     fmt.Sprintf("/mystery/%s", mysteryID),
			})
		}()
	}
	s.hub.Broadcast(ws.Message{
		Type: "mystery_clue_added",
		Data: wsData,
	})

	return nil
}
