package mystery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

func (s *service) MarkSolved(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, attemptID uuid.UUID) error {
	authorID, err := s.mysteryAuthor(ctx, mysteryID)
	if err != nil {
		return err
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return ErrNotAuthor
	}

	attemptAuthorID, err := s.mysteryRepo.GetAttemptAuthorID(ctx, attemptID)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	attemptMysteryID, err := s.mysteryRepo.GetAttemptMysteryID(ctx, attemptID)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if attemptMysteryID != mysteryID {
		return ErrAttemptNotOnMystery
	}
	if attemptAuthorID == authorID {
		return ErrOwnAttempt
	}

	row, err := s.mysteryRepo.GetByID(ctx, mysteryID)
	if err != nil {
		return err
	}
	if row == nil {
		return ErrNotFound
	}
	if row.Solved {
		return ErrAlreadySolved
	}
	ongoing := row.KeepOpenAfterSolve

	if alreadyWon, err := s.mysteryRepo.UserHasWinningAttempt(ctx, spec.MysterySolverQuery{MysteryID: mysteryID, UserID: attemptAuthorID}); err != nil {
		return err
	} else if alreadyWon {
		return ErrAlreadyWon
	}

	if err := s.mysteryRepo.MarkSolved(ctx, spec.MysterySolve{MysteryID: mysteryID, AttemptID: attemptID, LockMystery: !ongoing}); err != nil {
		return err
	}

	s.audit(ctx, audit.NewEntry{
		ActorID:    userID,
		Action:     audit.ActionMysterySolved,
		TargetType: audit.TargetMystery,
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
			playerIDs, err := s.mysteryRepo.GetPlayerIDs(bgCtx, mysteryID)
			if err != nil {
				logger.Ctx(bgCtx).Warn().Err(err).Str("mystery_id", mysteryID.String()).Msg("solved notification to players skipped, player lookup failed")
			}
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

		s.broadcastTopDetectives(bgCtx)
		if !ongoing {
			s.broadcastTopGMs(bgCtx)
		}
	}()

	return nil
}

func (s *service) broadcastTopDetectives(ctx context.Context) {
	topIDs, err := s.mysteryRepo.GetTopDetectiveIDs(ctx)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Msg("top detective broadcast skipped, lookup failed")

		return
	}

	s.hub.Broadcast(ws.Message{
		Type: "top_detective_changed",
		Data: map[string]any{
			"user_ids": topIDs,
		},
	})
}

func (s *service) broadcastTopGMs(ctx context.Context) {
	topGMIDs, err := s.mysteryRepo.GetTopGMIDs(ctx)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Msg("top gm broadcast skipped, lookup failed")

		return
	}

	s.hub.Broadcast(ws.Message{
		Type: "top_gm_changed",
		Data: map[string]any{
			"user_ids": topGMIDs,
		},
	})
}

func (s *service) MarkPermanentlySolved(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.mysteryAuthor(ctx, mysteryID)
	if err != nil {
		return err
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

	s.audit(ctx, audit.NewEntry{
		ActorID:    userID,
		Action:     audit.ActionMysteryClosed,
		TargetType: audit.TargetMystery,
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
		s.broadcastTopGMs(bgCtx)

		solvedLink := fmt.Sprintf("/mystery/%s", mysteryID)

		solverIDs, err := s.mysteryRepo.GetSolverIDs(bgCtx, mysteryID)
		if err != nil {
			logger.Ctx(bgCtx).Warn().Err(err).Str("mystery_id", mysteryID.String()).Msg("closed notification skipped, solver lookup failed")

			return
		}
		solverSet := make(map[uuid.UUID]struct{}, len(solverIDs))
		for _, sid := range solverIDs {
			solverSet[sid] = struct{}{}
		}

		playerIDs, err := s.mysteryRepo.GetPlayerIDs(bgCtx, mysteryID)
		if err != nil {
			logger.Ctx(bgCtx).Warn().Err(err).Str("mystery_id", mysteryID.String()).Msg("closed notification skipped, player lookup failed")

			return
		}
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

	authorID, err := s.mysteryAuthor(ctx, mysteryID)
	if err != nil {
		return err
	}
	if authorID != userID {
		return ErrNotAuthor
	}

	if req.TruthType == "" {
		req.TruthType = "red"
	}

	count, err := s.mysteryRepo.CountClues(ctx, mysteryID)
	if err != nil {
		return fmt.Errorf("count clues: %w", err)
	}

	if _, err := s.mysteryRepo.AddClue(ctx, spec.NewMysteryClue{
		MysteryID: mysteryID,
		NewClue: spec.NewClue{
			Body:      req.Body,
			TruthType: req.TruthType,
			SortOrder: count,
			PlayerID:  req.PlayerID,
		},
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
