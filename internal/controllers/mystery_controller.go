package controllers

import (
	"errors"
	"strconv"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	mysterysvc "umineko_city_of_books/internal/mystery"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllMysteryRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupListMysteries,
		s.setupMysteryLeaderboard,
		s.setupGMLeaderboard,
		s.setupListUserMysteries,
		s.setupGetMystery,
		s.setupCreateMystery,
		s.setupUpdateMystery,
		s.setupDeleteMystery,
		s.setupCreateAttempt,
		s.setupDeleteAttempt,
		s.setupVoteAttempt,
		s.setupMarkSolved,
		s.setupMarkPermanentlySolved,
		s.setupAddClue,
		s.setupCreateMysteryComment,
		s.setupUpdateMysteryComment,
		s.setupDeleteMysteryComment,
		s.setupLikeMysteryComment,
		s.setupUnlikeMysteryComment,
		s.setupUploadMysteryCommentMedia,
		s.setupUploadMysteryAttachment,
		s.setupDeleteMysteryAttachment,
		s.setupUploadMysteryMedia,
		s.setupDeleteMysteryMedia,
		s.setupToggleMysteryPause,
		s.setupToggleMysteryGmAway,
		s.setupDeleteMysteryClue,
		s.setupUpdateMysteryClue,
	}
}

func (s *Service) setupListMysteries(r fiber.Router) {
	r.Get("/mysteries", s.optionalAuth(), s.listMysteries)
}

func (s *Service) setupGetMystery(r fiber.Router) {
	r.Get("/mysteries/:id", s.optionalAuth(), s.getMystery)
}

func (s *Service) setupCreateMystery(r fiber.Router) {
	r.Post("/mysteries", s.requireAuth(), s.createMystery)
}

func (s *Service) setupUpdateMystery(r fiber.Router) {
	r.Put("/mysteries/:id", s.requireAuth(), s.updateMystery)
}

func (s *Service) setupDeleteMystery(r fiber.Router) {
	r.Delete("/mysteries/:id", s.requireAuth(), s.deleteMystery)
}

func (s *Service) setupCreateAttempt(r fiber.Router) {
	r.Post("/mysteries/:id/attempts", s.requireAuth(), s.createAttempt)
}

func (s *Service) setupDeleteAttempt(r fiber.Router) {
	r.Delete("/mystery-attempts/:id", s.requireAuth(), s.deleteAttempt)
}

func (s *Service) setupVoteAttempt(r fiber.Router) {
	r.Post("/mystery-attempts/:id/vote", s.requireAuth(), s.voteAttempt)
}

func (s *Service) setupMarkSolved(r fiber.Router) {
	r.Post("/mysteries/:id/solve", s.requireAuth(), s.markSolved)
}

func (s *Service) setupMarkPermanentlySolved(r fiber.Router) {
	r.Post("/mysteries/:id/close", s.requireAuth(), s.markPermanentlySolved)
}

func (s *Service) setupAddClue(r fiber.Router) {
	r.Post("/mysteries/:id/clues", s.requireAuth(), s.addClue)
}

func (s *Service) setupMysteryLeaderboard(r fiber.Router) {
	r.Get("/mysteries/leaderboard", s.mysteryLeaderboard)
}

func (s *Service) setupListUserMysteries(r fiber.Router) {
	r.Get("/users/:id/mysteries", s.listUserMysteries)
}

func (s *Service) listMysteries(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	sort := ctx.Query("sort", "new")
	page := utils.Page(ctx, 20)

	var solved *bool
	solvedStr := ctx.Query("solved")
	if solvedStr == "true" {
		solved = new(true)
	} else if solvedStr == "false" {
		solved = new(false)
	}

	resp, err := s.MysteryService.ListMysteries(ctx.Context(), sort, solved, userID, page)
	if err != nil {
		return utils.InternalError(ctx, "failed to list mysteries")
	}
	return ctx.JSON(resp)
}

func (s *Service) getMystery(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	resp, err := s.MysteryService.GetMystery(ctx.Context(), id, userID)
	if err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return utils.NotFound(ctx, "mystery not found")
		}
		return utils.InternalError(ctx, "failed to get mystery")
	}
	return ctx.JSON(resp)
}

func (s *Service) createMystery(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.CreateMysteryRequest](ctx)
	if !ok {
		return nil
	}

	id, err := s.MysteryService.CreateMystery(ctx.Context(), userID, req)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, mysterysvc.ErrEmptyTitle) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to create mystery")
	}

	s.Hub.BumpSidebarActivity("mysteries")
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateMystery(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.CreateMysteryRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.MysteryService.UpdateMystery(ctx.Context(), id, userID, req); err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, mysterysvc.ErrContractLocked) {
			return utils.Conflict(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to update mystery")
	}
	return utils.OK(ctx)
}

func (s *Service) deleteMystery(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.MysteryService.DeleteMystery(ctx.Context(), id, userID); err != nil {
		return utils.Forbidden(ctx, "cannot delete this mystery")
	}
	return utils.OK(ctx)
}

func (s *Service) createAttempt(ctx fiber.Ctx) error {
	mysteryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.CreateAttemptRequest](ctx)
	if !ok {
		return nil
	}

	id, err := s.MysteryService.CreateAttempt(ctx.Context(), mysteryID, userID, req)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, mysterysvc.ErrEmptyBody) {
			return utils.BadRequest(ctx, err.Error())
		}
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return utils.NotFound(ctx, "mystery not found")
		}
		if errors.Is(err, mysterysvc.ErrAlreadySolved) || errors.Is(err, mysterysvc.ErrCannotReply) || errors.Is(err, mysterysvc.ErrMysteryPaused) || errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to create attempt")
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) deleteAttempt(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.MysteryService.DeleteAttempt(ctx.Context(), id, userID); err != nil {
		return utils.Forbidden(ctx, "cannot delete this attempt")
	}
	return utils.OK(ctx)
}

func (s *Service) voteAttempt(ctx fiber.Ctx) error {
	attemptID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.VoteRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.MysteryService.VoteAttempt(ctx.Context(), attemptID, userID, req.Value); err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return utils.NotFound(ctx, "attempt not found")
		}
		if errors.Is(err, mysterysvc.ErrInvalidVote) {
			return utils.BadRequest(ctx, err.Error())
		}
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		return utils.InternalError(ctx, "failed to vote")
	}
	return utils.OK(ctx)
}

func (s *Service) markSolved(ctx fiber.Ctx) error {
	mysteryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	var req struct {
		AttemptID uuid.UUID `json:"attempt_id"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request body")
	}
	if req.AttemptID == uuid.Nil {
		return utils.BadRequest(ctx, "attempt_id is required")
	}

	if err := s.MysteryService.MarkSolved(ctx.Context(), mysteryID, userID, req.AttemptID); err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return utils.NotFound(ctx, err.Error())
		}
		if errors.Is(err, mysterysvc.ErrNotAuthor) {
			return utils.Forbidden(ctx, err.Error())
		}
		if errors.Is(err, mysterysvc.ErrAlreadySolved) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to mark as solved")
	}
	return utils.OK(ctx)
}

func (s *Service) markPermanentlySolved(ctx fiber.Ctx) error {
	mysteryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.MysteryService.MarkPermanentlySolved(ctx.Context(), mysteryID, userID); err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return utils.NotFound(ctx, err.Error())
		}
		if errors.Is(err, mysterysvc.ErrNotAuthor) {
			return utils.Forbidden(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to close mystery")
	}
	return utils.OK(ctx)
}

func (s *Service) addClue(ctx fiber.Ctx) error {
	mysteryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.CreateClueRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.MysteryService.AddClue(ctx.Context(), mysteryID, userID, req); err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, mysterysvc.ErrEmptyBody) {
			return utils.BadRequest(ctx, err.Error())
		}
		if errors.Is(err, mysterysvc.ErrNotFound) || errors.Is(err, mysterysvc.ErrNotAuthor) {
			return utils.Forbidden(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to add clue")
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "ok"})
}

func (s *Service) mysteryLeaderboard(ctx fiber.Ctx) error {
	page := bounds.NewPage(fiber.Query[int](ctx, "limit", 20), 0)

	resp, err := s.MysteryService.GetLeaderboard(ctx.Context(), page)
	if err != nil {
		return utils.InternalError(ctx, "failed to load leaderboard")
	}
	return ctx.JSON(resp)
}

func (s *Service) setupGMLeaderboard(r fiber.Router) {
	r.Get("/mysteries/gm-leaderboard", s.gmLeaderboard)
}

func (s *Service) gmLeaderboard(ctx fiber.Ctx) error {
	page := bounds.NewPage(fiber.Query[int](ctx, "limit", 20), 0)

	resp, err := s.MysteryService.GetGMLeaderboard(ctx.Context(), page)
	if err != nil {
		return utils.InternalError(ctx, "failed to load gm leaderboard")
	}
	return ctx.JSON(resp)
}

func (s *Service) listUserMysteries(ctx fiber.Ctx) error {
	userID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	page := utils.Page(ctx, 20)

	resp, err := s.MysteryService.ListByUser(ctx.Context(), userID, page)
	if err != nil {
		return utils.InternalError(ctx, "failed to list user mysteries")
	}
	return ctx.JSON(resp)
}

func (s *Service) setupCreateMysteryComment(r fiber.Router) {
	r.Post("/mysteries/:id/comments", s.requireAuth(), s.createMysteryComment)
}

func (s *Service) setupUpdateMysteryComment(r fiber.Router) {
	r.Put("/mystery-comments/:id", s.requireAuth(), s.updateMysteryComment)
}

func (s *Service) setupDeleteMysteryComment(r fiber.Router) {
	r.Delete("/mystery-comments/:id", s.requireAuth(), s.deleteMysteryComment)
}

func (s *Service) setupLikeMysteryComment(r fiber.Router) {
	r.Post("/mystery-comments/:id/like", s.requireAuth(), s.likeMysteryComment)
}

func (s *Service) setupUnlikeMysteryComment(r fiber.Router) {
	r.Delete("/mystery-comments/:id/like", s.requireAuth(), s.unlikeMysteryComment)
}

func (s *Service) setupUploadMysteryCommentMedia(r fiber.Router) {
	r.Post("/mystery-comments/:id/media", s.requireAuth(), s.uploadMysteryCommentMedia)
}

func (s *Service) createMysteryComment(ctx fiber.Ctx) error {
	mysteryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.CreateCommentRequest](ctx)
	if !ok {
		return nil
	}

	id, err := s.MysteryService.CreateComment(ctx.Context(), mysteryID, userID, req)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, mysterysvc.ErrEmptyBody) {
			return utils.BadRequest(ctx, err.Error())
		}
		if errors.Is(err, mysterysvc.ErrNotSolved) {
			return utils.Forbidden(ctx, err.Error())
		}
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		return utils.InternalError(ctx, "failed to create comment")
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateMysteryComment(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.UpdateCommentRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.MysteryService.UpdateComment(ctx.Context(), id, userID, req); err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, mysterysvc.ErrEmptyBody) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to update comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteMysteryComment(ctx fiber.Ctx) error {
	return s.handleDeleteComment(ctx, s.MysteryService.DeleteComment)
}

func (s *Service) likeMysteryComment(ctx fiber.Ctx) error {
	return s.handleLikeComment(ctx, s.MysteryService.LikeComment)
}

func (s *Service) unlikeMysteryComment(ctx fiber.Ctx) error {
	return s.handleUnlikeComment(ctx, s.MysteryService.UnlikeComment)
}

func (s *Service) uploadMysteryCommentMedia(ctx fiber.Ctx) error {
	return handleUploadCommentMedia(ctx, s.MysteryService.UploadCommentMedia)
}

func (s *Service) setupUploadMysteryAttachment(r fiber.Router) {
	r.Post("/mysteries/:id/attachments", s.requireAuth(), s.uploadMysteryAttachment)
}

func (s *Service) setupDeleteMysteryAttachment(r fiber.Router) {
	r.Delete("/mysteries/:id/attachments/:attachmentId", s.requireAuth(), s.deleteMysteryAttachment)
}

func (s *Service) uploadMysteryAttachment(ctx fiber.Ctx) error {
	mysteryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	file, err := ctx.FormFile("file")
	if err != nil {
		return utils.BadRequest(ctx, "no file provided")
	}

	reader, err := file.Open()
	if err != nil {
		return utils.InternalError(ctx, "failed to read file")
	}
	defer reader.Close()

	result, err := s.MysteryService.UploadAttachment(ctx.Context(), mysteryID, userID, file.Filename, file.Size, reader)
	if err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return utils.NotFound(ctx, "mystery not found")
		}
		if errors.Is(err, mysterysvc.ErrNotAuthor) {
			return utils.Forbidden(ctx, err.Error())
		}
		return utils.BadRequest(ctx, err.Error())
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}

func (s *Service) deleteMysteryAttachment(ctx fiber.Ctx) error {
	mysteryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	attachmentID, err := strconv.ParseInt(ctx.Params("attachmentId"), 10, 64)
	if err != nil {
		return utils.BadRequest(ctx, "invalid attachment id")
	}
	userID := utils.UserID(ctx)

	if err := s.MysteryService.DeleteAttachment(ctx.Context(), attachmentID, mysteryID, userID); err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return utils.NotFound(ctx, "mystery not found")
		}
		if errors.Is(err, mysterysvc.ErrNotAuthor) {
			return utils.Forbidden(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to delete attachment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) setupUploadMysteryMedia(r fiber.Router) {
	r.Post("/mysteries/:id/media", s.requireAuth(), s.uploadMysteryMedia)
}

func (s *Service) setupDeleteMysteryMedia(r fiber.Router) {
	r.Delete("/mysteries/:id/media/:mediaId", s.requireAuth(), s.deleteMysteryMedia)
}

func (s *Service) uploadMysteryMedia(ctx fiber.Ctx) error {
	mysteryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	file, err := ctx.FormFile("media")
	if err != nil {
		return utils.BadRequest(ctx, "no media file provided")
	}
	reader, err := file.Open()
	if err != nil {
		return utils.InternalError(ctx, "failed to read file")
	}
	defer reader.Close()

	result, err := s.MysteryService.UploadMedia(ctx.Context(), mysteryID, userID, file.Header.Get("Content-Type"), file.Size, reader)
	if err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return utils.NotFound(ctx, "mystery not found")
		}
		if errors.Is(err, mysterysvc.ErrNotAuthor) {
			return utils.Forbidden(ctx, err.Error())
		}
		return utils.BadRequest(ctx, err.Error())
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}

func (s *Service) deleteMysteryMedia(ctx fiber.Ctx) error {
	mysteryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	mediaID, err := strconv.ParseInt(ctx.Params("mediaId"), 10, 64)
	if err != nil {
		return utils.BadRequest(ctx, "invalid media id")
	}
	userID := utils.UserID(ctx)

	if err := s.MysteryService.DeleteMedia(ctx.Context(), mediaID, mysteryID, userID); err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return utils.NotFound(ctx, "mystery not found")
		}
		if errors.Is(err, mysterysvc.ErrNotAuthor) {
			return utils.Forbidden(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to delete media")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) setupToggleMysteryPause(r fiber.Router) {
	r.Post("/mysteries/:id/pause", s.requireAuth(), s.toggleMysteryPause)
}

func (s *Service) toggleMysteryPause(ctx fiber.Ctx) error {
	mysteryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	var req struct {
		Paused bool `json:"paused"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request body")
	}

	if err := s.MysteryService.SetPaused(ctx.Context(), mysteryID, userID, req.Paused); err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return utils.NotFound(ctx, err.Error())
		}
		if errors.Is(err, mysterysvc.ErrNotAuthor) {
			return utils.Forbidden(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to toggle pause")
	}
	return utils.OK(ctx)
}

func (s *Service) setupToggleMysteryGmAway(r fiber.Router) {
	r.Post("/mysteries/:id/away", s.requireAuth(), s.toggleMysteryGmAway)
}

func (s *Service) toggleMysteryGmAway(ctx fiber.Ctx) error {
	mysteryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	var req struct {
		Away bool `json:"away"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request body")
	}

	if err := s.MysteryService.SetGmAway(ctx.Context(), mysteryID, userID, req.Away); err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return utils.NotFound(ctx, err.Error())
		}
		if errors.Is(err, mysterysvc.ErrNotAuthor) {
			return utils.Forbidden(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to toggle away")
	}
	return utils.OK(ctx)
}

func (s *Service) setupDeleteMysteryClue(r fiber.Router) {
	r.Delete("/mysteries/:id/clues/:clueId", middleware.RequirePermission(s.AuthSession, s.AuthzService, authz.PermEditAnyTheory), s.deleteMysteryClue)
}

func (s *Service) deleteMysteryClue(ctx fiber.Ctx) error {
	mysteryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	clueID, err := strconv.Atoi(ctx.Params("clueId"))
	if err != nil {
		return utils.BadRequest(ctx, "invalid clue id")
	}
	userID := utils.UserID(ctx)

	if err := s.MysteryService.DeleteClue(ctx.Context(), mysteryID, clueID, userID); err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) || errors.Is(err, mysterysvc.ErrNotAuthor) {
			return utils.Forbidden(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to delete clue")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) setupUpdateMysteryClue(r fiber.Router) {
	r.Put("/mysteries/:id/clues/:clueId", middleware.RequirePermission(s.AuthSession, s.AuthzService, authz.PermEditAnyTheory), s.updateMysteryClue)
}

func (s *Service) updateMysteryClue(ctx fiber.Ctx) error {
	mysteryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	clueID, err := strconv.Atoi(ctx.Params("clueId"))
	if err != nil {
		return utils.BadRequest(ctx, "invalid clue id")
	}
	userID := utils.UserID(ctx)

	var req struct {
		Body string `json:"body"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request body")
	}

	if err := s.MysteryService.UpdateClue(ctx.Context(), mysteryID, clueID, userID, req.Body); err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, mysterysvc.ErrNotFound) || errors.Is(err, mysterysvc.ErrNotAuthor) || errors.Is(err, mysterysvc.ErrEmptyBody) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to update clue")
	}
	return utils.OK(ctx)
}
