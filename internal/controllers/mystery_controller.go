package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/utils"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllMysteryRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupListMysteries,
		s.setupMysteryLeaderboard,
		s.setupListUserMysteries,
		s.setupGetMystery,
		s.setupCreateMystery,
		s.setupUpdateMystery,
		s.setupDeleteMystery,
		s.setupCreateAttempt,
		s.setupDeleteAttempt,
		s.setupVoteAttempt,
		s.setupMarkSolved,
		s.setupAddClue,
	}
}

func (s *Service) setupListMysteries(r fiber.Router) {
	r.Get("/mysteries", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listMysteries)
}

func (s *Service) setupGetMystery(r fiber.Router) {
	r.Get("/mysteries/:id", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.getMystery)
}

func (s *Service) setupCreateMystery(r fiber.Router) {
	r.Post("/mysteries", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createMystery)
}

func (s *Service) setupUpdateMystery(r fiber.Router) {
	r.Put("/mysteries/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateMystery)
}

func (s *Service) setupDeleteMystery(r fiber.Router) {
	r.Delete("/mysteries/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteMystery)
}

func (s *Service) setupCreateAttempt(r fiber.Router) {
	r.Post("/mysteries/:id/attempts", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createAttempt)
}

func (s *Service) setupDeleteAttempt(r fiber.Router) {
	r.Delete("/mystery-attempts/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteAttempt)
}

func (s *Service) setupVoteAttempt(r fiber.Router) {
	r.Post("/mystery-attempts/:id/vote", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.voteAttempt)
}

func (s *Service) setupMarkSolved(r fiber.Router) {
	r.Post("/mysteries/:id/solve", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.markSolved)
}

func (s *Service) setupAddClue(r fiber.Router) {
	r.Post("/mysteries/:id/clues", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.addClue)
}

func (s *Service) setupMysteryLeaderboard(r fiber.Router) {
	r.Get("/mysteries/leaderboard", s.mysteryLeaderboard)
}

func (s *Service) listMysteries(ctx fiber.Ctx) error {
	userID, _ := ctx.Locals("userID").(uuid.UUID)
	sort := ctx.Query("sort", "new")
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	var solved *bool
	solvedStr := ctx.Query("solved")
	if solvedStr == "true" {
		t := true
		solved = &t
	} else if solvedStr == "false" {
		f := false
		solved = &f
	}

	blockedIDs, _ := s.BlockService.GetBlockedIDs(ctx.Context(), userID)
	rows, total, err := s.MysteryRepo.List(ctx.Context(), sort, solved, limit, offset, blockedIDs)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list mysteries"})
	}

	mysteries := make([]dto.MysteryResponse, len(rows))
	for i, r := range rows {
		resp := r.ToResponse()
		if len(resp.Body) > 200 {
			resp.Body = resp.Body[:200] + "..."
		}
		mysteries[i] = resp
	}

	return ctx.JSON(dto.MysteryListResponse{
		Mysteries: mysteries,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	})
}

func (s *Service) getMystery(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID, _ := ctx.Locals("userID").(uuid.UUID)

	row, err := s.MysteryRepo.GetByID(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get mystery"})
	}
	if row == nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "mystery not found"})
	}

	clues, _ := s.MysteryRepo.GetClues(ctx.Context(), id)
	if clues == nil {
		clues = []dto.MysteryClue{}
	}

	attemptRows, _ := s.MysteryRepo.GetAttempts(ctx.Context(), id, userID)
	flatAttempts := make([]dto.MysteryAttempt, len(attemptRows))
	for i, a := range attemptRows {
		flatAttempts[i] = dto.MysteryAttempt{
			ID:       a.ID,
			ParentID: a.ParentID,
			Author: dto.UserResponse{
				ID:          a.UserID,
				Username:    a.AuthorUsername,
				DisplayName: a.AuthorDisplayName,
				AvatarURL:   a.AuthorAvatarURL,
				Role:        role.Role(a.AuthorRole),
			},
			Body:      a.Body,
			IsWinner:  a.IsWinner,
			VoteScore: a.VoteScore,
			UserVote:  a.UserVote,
			CreatedAt: a.CreatedAt,
		}
	}

	attempts := utils.BuildTree(flatAttempts,
		func(a dto.MysteryAttempt) uuid.UUID { return a.ID },
		func(a dto.MysteryAttempt) *uuid.UUID { return a.ParentID },
		func(a *dto.MysteryAttempt, replies []dto.MysteryAttempt) { a.Replies = replies },
	)

	resp := dto.MysteryDetailResponse{
		ID:         row.ID,
		Title:      row.Title,
		Body:       row.Body,
		Difficulty: row.Difficulty,
		Solved:     row.Solved,
		SolvedAt:   row.SolvedAt,
		Author: dto.UserResponse{
			ID:          row.UserID,
			Username:    row.AuthorUsername,
			DisplayName: row.AuthorDisplayName,
			AvatarURL:   row.AuthorAvatarURL,
			Role:        role.Role(row.AuthorRole),
		},
		Clues:     clues,
		Attempts:  attempts,
		CreatedAt: row.CreatedAt,
	}
	if row.WinnerID != nil && row.WinnerUsername != nil {
		resp.Winner = &dto.UserResponse{
			ID:          *row.WinnerID,
			Username:    *row.WinnerUsername,
			DisplayName: *row.WinnerDisplayName,
			AvatarURL:   *row.WinnerAvatarURL,
			Role:        role.Role(*row.WinnerRole),
		}
	}

	return ctx.JSON(resp)
}

func (s *Service) createMystery(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.CreateMysteryRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Body) == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title and body are required"})
	}
	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}

	id := uuid.New()
	if err := s.MysteryRepo.Create(ctx.Context(), id, userID, req.Title, req.Body, req.Difficulty); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create mystery"})
	}

	for i, clue := range req.Clues {
		if strings.TrimSpace(clue.Body) == "" {
			continue
		}
		truthType := clue.TruthType
		if truthType == "" {
			truthType = "red"
		}
		if err := s.MysteryRepo.AddClue(ctx.Context(), id, clue.Body, truthType, i); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add clue"})
		}
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateMystery(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.CreateMysteryRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.MysteryRepo.Update(ctx.Context(), id, userID, req.Title, req.Body, req.Difficulty); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update mystery"})
	}

	_ = s.MysteryRepo.DeleteClues(ctx.Context(), id)
	for i, clue := range req.Clues {
		if strings.TrimSpace(clue.Body) == "" {
			continue
		}
		truthType := clue.TruthType
		if truthType == "" {
			truthType = "red"
		}
		_ = s.MysteryRepo.AddClue(ctx.Context(), id, clue.Body, truthType, i)
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) deleteMystery(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if s.AuthzService.Can(ctx.Context(), userID, authz.PermDeleteAnyTheory) {
		if err := s.MysteryRepo.DeleteAsAdmin(ctx.Context(), id); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete mystery"})
		}
	} else if err := s.MysteryRepo.Delete(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "cannot delete this mystery"})
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) createAttempt(ctx fiber.Ctx) error {
	mysteryID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	authorID, err := s.MysteryRepo.GetAuthorID(ctx.Context(), mysteryID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "mystery not found"})
	}
	if blocked, _ := s.BlockService.IsBlockedEither(ctx.Context(), userID, authorID); blocked {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is blocked"})
	}

	var req dto.CreateAttemptRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if strings.TrimSpace(req.Body) == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body is required"})
	}

	if req.ParentID != nil {
		parentAuthor, err := s.MysteryRepo.GetAttemptAuthorID(ctx.Context(), *req.ParentID)
		if err != nil {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "parent attempt not found"})
		}
		if userID != authorID && userID != parentAuthor {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only the game master or the attempt author can reply"})
		}
	}

	id := uuid.New()
	if err := s.MysteryRepo.CreateAttempt(ctx.Context(), id, mysteryID, userID, req.ParentID, strings.TrimSpace(req.Body)); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create attempt"})
	}

	go func() {
		bgCtx := context.Background()
		baseURL := s.SettingsService.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/mysteries/%s#attempt-%s", baseURL, mysteryID, id)
		attemptRef := fmt.Sprintf("mystery_attempt:%s", id)

		subject, body := notification.NotifEmail("Someone", "submitted an attempt on your mystery", "", linkURL)
		_ = s.NotificationService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifMysteryAttempt,
			ReferenceID:   mysteryID,
			ReferenceType: attemptRef,
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     body,
		})

		if req.ParentID != nil {
			if parentAuthor, err := s.MysteryRepo.GetAttemptAuthorID(bgCtx, *req.ParentID); err == nil && parentAuthor != authorID {
				replySubject, replyBody := notification.NotifEmail("Someone", "replied to your attempt", "", linkURL)
				_ = s.NotificationService.Notify(bgCtx, dto.NotifyParams{
					RecipientID:   parentAuthor,
					Type:          dto.NotifMysteryAttempt,
					ReferenceID:   mysteryID,
					ReferenceType: attemptRef,
					ActorID:       userID,
					EmailSubject:  replySubject,
					EmailBody:     replyBody,
				})
			}
		}
	}()

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) deleteAttempt(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if s.AuthzService.Can(ctx.Context(), userID, authz.PermDeleteAnyComment) {
		if err := s.MysteryRepo.DeleteAttemptAsAdmin(ctx.Context(), id); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete attempt"})
		}
	} else if err := s.MysteryRepo.DeleteAttempt(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "cannot delete this attempt"})
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) voteAttempt(ctx fiber.Ctx) error {
	attemptID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	attemptAuthorID, err := s.MysteryRepo.GetAttemptAuthorID(ctx.Context(), attemptID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "attempt not found"})
	}
	if blocked, _ := s.BlockService.IsBlockedEither(ctx.Context(), userID, attemptAuthorID); blocked {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is blocked"})
	}

	var req dto.VoteRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.Value != 1 && req.Value != -1 && req.Value != 0 {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "value must be 1, -1, or 0"})
	}

	if err := s.MysteryRepo.VoteAttempt(ctx.Context(), userID, attemptID, req.Value); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is blocked"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to vote"})
	}

	if req.Value != 0 {
		go func() {
			bgCtx := context.Background()
			mysteryID, err := s.MysteryRepo.GetAttemptMysteryID(bgCtx, attemptID)
			if err != nil {
				return
			}
			baseURL := s.SettingsService.Get(bgCtx, config.SettingBaseURL)
			linkURL := fmt.Sprintf("%s/mysteries/%s#attempt-%s", baseURL, mysteryID, attemptID)
			subject, body := notification.NotifEmail("Someone", "voted on your attempt", "", linkURL)
			_ = s.NotificationService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   attemptAuthorID,
				Type:          dto.NotifMysteryVote,
				ReferenceID:   mysteryID,
				ReferenceType: fmt.Sprintf("mystery_attempt:%s", attemptID),
				ActorID:       userID,
				EmailSubject:  subject,
				EmailBody:     body,
			})
		}()
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) markSolved(ctx fiber.Ctx) error {
	mysteryID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	authorID, err := s.MysteryRepo.GetAuthorID(ctx.Context(), mysteryID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "mystery not found"})
	}
	if authorID != userID && !s.AuthzService.Can(ctx.Context(), userID, authz.PermEditAnyTheory) {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only the author can mark this as solved"})
	}

	var req struct {
		WinnerID uuid.UUID `json:"winner_id"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.WinnerID == uuid.Nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "winner_id is required"})
	}

	if err := s.MysteryRepo.MarkSolved(ctx.Context(), mysteryID, req.WinnerID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to mark as solved"})
	}

	go func() {
		bgCtx := context.Background()
		baseURL := s.SettingsService.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/mysteries/%s", baseURL, mysteryID)
		subject, body := notification.NotifEmail("Someone", "chose your attempt as the winner!", "", linkURL)
		_ = s.NotificationService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   req.WinnerID,
			Type:          dto.NotifMysterySolved,
			ReferenceID:   mysteryID,
			ReferenceType: "mystery",
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     body,
		})
	}()

	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) addClue(ctx fiber.Ctx) error {
	mysteryID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	authorID, err := s.MysteryRepo.GetAuthorID(ctx.Context(), mysteryID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "mystery not found"})
	}
	if authorID != userID {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only the author can add clues"})
	}

	var req dto.CreateClueRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if strings.TrimSpace(req.Body) == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body is required"})
	}
	if req.TruthType == "" {
		req.TruthType = "red"
	}

	count, _ := s.MysteryRepo.CountAttempts(ctx.Context(), mysteryID)
	if err := s.MysteryRepo.AddClue(ctx.Context(), mysteryID, req.Body, req.TruthType, count); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add clue"})
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "ok"})
}

func (s *Service) mysteryLeaderboard(ctx fiber.Ctx) error {
	limit := fiber.Query[int](ctx, "limit", 20)
	rows, err := s.MysteryRepo.GetLeaderboard(ctx.Context(), limit)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load leaderboard"})
	}

	entries := make([]dto.MysteryLeaderboardEntry, len(rows))
	for i, r := range rows {
		entries[i] = dto.MysteryLeaderboardEntry{
			User: dto.UserResponse{
				ID:          r.UserID,
				Username:    r.Username,
				DisplayName: r.DisplayName,
				AvatarURL:   r.AvatarURL,
				Role:        role.Role(r.Role),
			},
			SolvedCount: r.SolvedCount,
		}
	}
	return ctx.JSON(dto.MysteryLeaderboardResponse{Entries: entries})
}

func (s *Service) setupListUserMysteries(r fiber.Router) {
	r.Get("/users/:id/mysteries", s.listUserMysteries)
}

func (s *Service) listUserMysteries(ctx fiber.Ctx) error {
	userID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	rows, total, err := s.MysteryRepo.ListByUser(ctx.Context(), userID, limit, offset)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list user mysteries"})
	}

	mysteries := make([]dto.MysteryResponse, len(rows))
	for i, r := range rows {
		resp := r.ToResponse()
		if len(resp.Body) > 200 {
			resp.Body = resp.Body[:200] + "..."
		}
		mysteries[i] = resp
	}

	return ctx.JSON(dto.MysteryListResponse{
		Mysteries: mysteries,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	})
}
