package controllers

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	mysterysvc "umineko_city_of_books/internal/mystery"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

var allowedSniffedTypes = map[string]bool{
	"application/pdf": true,
	"text/plain":      true,
	"application/zip": true,
}

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
		s.setupAddClue,
		s.setupCreateMysteryComment,
		s.setupUpdateMysteryComment,
		s.setupDeleteMysteryComment,
		s.setupLikeMysteryComment,
		s.setupUnlikeMysteryComment,
		s.setupUploadMysteryCommentMedia,
		s.setupUploadMysteryAttachment,
		s.setupDeleteMysteryAttachment,
		s.setupToggleMysteryPause,
		s.setupToggleMysteryGmAway,
		s.setupDeleteMysteryClue,
		s.setupUpdateMysteryClue,
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

func (s *Service) setupListUserMysteries(r fiber.Router) {
	r.Get("/users/:id/mysteries", s.listUserMysteries)
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

	resp, err := s.MysteryService.ListMysteries(ctx.Context(), sort, solved, userID, limit, offset)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list mysteries"})
	}
	return ctx.JSON(resp)
}

func (s *Service) getMystery(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID, _ := ctx.Locals("userID").(uuid.UUID)

	resp, err := s.MysteryService.GetMystery(ctx.Context(), id, userID)
	if err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "mystery not found"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get mystery"})
	}
	return ctx.JSON(resp)
}

func (s *Service) createMystery(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.CreateMysteryRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	id, err := s.MysteryService.CreateMystery(ctx.Context(), userID, req)
	if err != nil {
		if errors.Is(err, mysterysvc.ErrEmptyTitle) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create mystery"})
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

	if err := s.MysteryService.UpdateMystery(ctx.Context(), id, userID, req); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update mystery"})
	}
	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) deleteMystery(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.MysteryService.DeleteMystery(ctx.Context(), id, userID); err != nil {
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

	var req dto.CreateAttemptRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	id, err := s.MysteryService.CreateAttempt(ctx.Context(), mysteryID, userID, req)
	if err != nil {
		if errors.Is(err, mysterysvc.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "mystery not found"})
		}
		if errors.Is(err, mysterysvc.ErrAlreadySolved) || errors.Is(err, mysterysvc.ErrCannotReply) || errors.Is(err, mysterysvc.ErrMysteryPaused) || errors.Is(err, block.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create attempt"})
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) deleteAttempt(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.MysteryService.DeleteAttempt(ctx.Context(), id, userID); err != nil {
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

	var req dto.VoteRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.MysteryService.VoteAttempt(ctx.Context(), attemptID, userID, req.Value); err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "attempt not found"})
		}
		if errors.Is(err, mysterysvc.ErrInvalidVote) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, block.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is blocked"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to vote"})
	}
	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) markSolved(ctx fiber.Ctx) error {
	mysteryID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req struct {
		AttemptID uuid.UUID `json:"attempt_id"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.AttemptID == uuid.Nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "attempt_id is required"})
	}

	if err := s.MysteryService.MarkSolved(ctx.Context(), mysteryID, userID, req.AttemptID); err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, mysterysvc.ErrNotAuthor) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to mark as solved"})
	}
	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) addClue(ctx fiber.Ctx) error {
	mysteryID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.CreateClueRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.MysteryService.AddClue(ctx.Context(), mysteryID, userID, req); err != nil {
		if errors.Is(err, mysterysvc.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, mysterysvc.ErrNotFound) || errors.Is(err, mysterysvc.ErrNotAuthor) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add clue"})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "ok"})
}

func (s *Service) mysteryLeaderboard(ctx fiber.Ctx) error {
	limit := fiber.Query[int](ctx, "limit", 20)
	resp, err := s.MysteryService.GetLeaderboard(ctx.Context(), limit)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load leaderboard"})
	}
	return ctx.JSON(resp)
}

func (s *Service) setupGMLeaderboard(r fiber.Router) {
	r.Get("/mysteries/gm-leaderboard", s.gmLeaderboard)
}

func (s *Service) gmLeaderboard(ctx fiber.Ctx) error {
	limit := fiber.Query[int](ctx, "limit", 20)
	resp, err := s.MysteryService.GetGMLeaderboard(ctx.Context(), limit)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load gm leaderboard"})
	}
	return ctx.JSON(resp)
}

func (s *Service) listUserMysteries(ctx fiber.Ctx) error {
	userID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	resp, err := s.MysteryService.ListByUser(ctx.Context(), userID, limit, offset)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list user mysteries"})
	}
	return ctx.JSON(resp)
}

func (s *Service) setupCreateMysteryComment(r fiber.Router) {
	r.Post("/mysteries/:id/comments", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createMysteryComment)
}

func (s *Service) setupUpdateMysteryComment(r fiber.Router) {
	r.Put("/mystery-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateMysteryComment)
}

func (s *Service) setupDeleteMysteryComment(r fiber.Router) {
	r.Delete("/mystery-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteMysteryComment)
}

func (s *Service) setupLikeMysteryComment(r fiber.Router) {
	r.Post("/mystery-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.likeMysteryComment)
}

func (s *Service) setupUnlikeMysteryComment(r fiber.Router) {
	r.Delete("/mystery-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unlikeMysteryComment)
}

func (s *Service) setupUploadMysteryCommentMedia(r fiber.Router) {
	r.Post("/mystery-comments/:id/media", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.uploadMysteryCommentMedia)
}

func (s *Service) createMysteryComment(ctx fiber.Ctx) error {
	mysteryID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.CreateCommentRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	id, err := s.MysteryService.CreateComment(ctx.Context(), mysteryID, userID, req)
	if err != nil {
		if errors.Is(err, mysterysvc.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, mysterysvc.ErrNotSolved) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, block.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is blocked"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create comment"})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateMysteryComment(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid comment id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.UpdateCommentRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.MysteryService.UpdateComment(ctx.Context(), id, userID, req); err != nil {
		if errors.Is(err, mysterysvc.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update comment"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteMysteryComment(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid comment id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.MysteryService.DeleteComment(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete comment"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) likeMysteryComment(ctx fiber.Ctx) error {
	commentID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid comment id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.MysteryService.LikeComment(ctx.Context(), userID, commentID); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is blocked"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to like comment"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unlikeMysteryComment(ctx fiber.Ctx) error {
	commentID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid comment id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.MysteryService.UnlikeComment(ctx.Context(), userID, commentID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to unlike comment"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) uploadMysteryCommentMedia(ctx fiber.Ctx) error {
	commentID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid comment id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	file, err := ctx.FormFile("media")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no media file provided"})
	}
	reader, err := file.Open()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read file"})
	}
	defer reader.Close()

	result, err := s.MysteryService.UploadCommentMedia(ctx.Context(), commentID, userID, file.Header.Get("Content-Type"), file.Size, reader)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}

func (s *Service) setupUploadMysteryAttachment(r fiber.Router) {
	r.Post("/mysteries/:id/attachments", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.uploadMysteryAttachment)
}

func (s *Service) setupDeleteMysteryAttachment(r fiber.Router) {
	r.Delete("/mysteries/:id/attachments/:attachmentId", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteMysteryAttachment)
}

func (s *Service) uploadMysteryAttachment(ctx fiber.Ctx) error {
	mysteryID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	file, err := ctx.FormFile("file")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no file provided"})
	}

	reader, err := file.Open()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read file"})
	}
	defer reader.Close()

	buf := make([]byte, 512)
	n, _ := reader.Read(buf)
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read file"})
	}

	sniffed := http.DetectContentType(buf[:n])
	if !allowedSniffedTypes[strings.SplitN(sniffed, ";", 2)[0]] {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "only PDF, TXT, and DOCX files are allowed"})
	}

	result, err := s.MysteryService.UploadAttachment(ctx.Context(), mysteryID, userID, file.Filename, file.Size, reader)
	if err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "mystery not found"})
		}
		if errors.Is(err, mysterysvc.ErrNotAuthor) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}

func (s *Service) deleteMysteryAttachment(ctx fiber.Ctx) error {
	mysteryID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	attachmentID, err := strconv.ParseInt(ctx.Params("attachmentId"), 10, 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid attachment id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.MysteryService.DeleteAttachment(ctx.Context(), attachmentID, mysteryID, userID); err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "mystery not found"})
		}
		if errors.Is(err, mysterysvc.ErrNotAuthor) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete attachment"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) setupToggleMysteryPause(r fiber.Router) {
	r.Post("/mysteries/:id/pause", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.toggleMysteryPause)
}

func (s *Service) toggleMysteryPause(ctx fiber.Ctx) error {
	mysteryID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req struct {
		Paused bool `json:"paused"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.MysteryService.SetPaused(ctx.Context(), mysteryID, userID, req.Paused); err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, mysterysvc.ErrNotAuthor) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to toggle pause"})
	}
	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) setupToggleMysteryGmAway(r fiber.Router) {
	r.Post("/mysteries/:id/away", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.toggleMysteryGmAway)
}

func (s *Service) toggleMysteryGmAway(ctx fiber.Ctx) error {
	mysteryID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req struct {
		Away bool `json:"away"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.MysteryService.SetGmAway(ctx.Context(), mysteryID, userID, req.Away); err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, mysterysvc.ErrNotAuthor) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to toggle away"})
	}
	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) setupDeleteMysteryClue(r fiber.Router) {
	r.Delete("/mysteries/:id/clues/:clueId", middleware.RequirePermission(s.AuthSession, s.AuthzService, authz.PermEditAnyTheory), s.deleteMysteryClue)
}

func (s *Service) deleteMysteryClue(ctx fiber.Ctx) error {
	mysteryID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid mystery id"})
	}
	clueID, err := strconv.Atoi(ctx.Params("clueId"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid clue id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.MysteryService.DeleteClue(ctx.Context(), mysteryID, clueID, userID); err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) || errors.Is(err, mysterysvc.ErrNotAuthor) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete clue"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) setupUpdateMysteryClue(r fiber.Router) {
	r.Put("/mysteries/:id/clues/:clueId", middleware.RequirePermission(s.AuthSession, s.AuthzService, authz.PermEditAnyTheory), s.updateMysteryClue)
}

func (s *Service) updateMysteryClue(ctx fiber.Ctx) error {
	mysteryID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid mystery id"})
	}
	clueID, err := strconv.Atoi(ctx.Params("clueId"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid clue id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req struct {
		Body string `json:"body"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.MysteryService.UpdateClue(ctx.Context(), mysteryID, clueID, userID, req.Body); err != nil {
		if errors.Is(err, mysterysvc.ErrNotFound) || errors.Is(err, mysterysvc.ErrNotAuthor) || errors.Is(err, mysterysvc.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update clue"})
	}
	return ctx.JSON(fiber.Map{"status": "ok"})
}
