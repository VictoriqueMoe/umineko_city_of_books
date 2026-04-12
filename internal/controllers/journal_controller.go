package controllers

import (
	"errors"
	"strings"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/journal"
	"umineko_city_of_books/internal/journal/params"
	"umineko_city_of_books/internal/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllJournalRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupListJournalsRoute,
		s.setupListUserJournalsRoute,
		s.setupListUserFollowedJournalsRoute,
		s.setupCreateJournalRoute,
		s.setupGetJournalRoute,
		s.setupUpdateJournalRoute,
		s.setupDeleteJournalRoute,
		s.setupFollowJournalRoute,
		s.setupUnfollowJournalRoute,
		s.setupCreateJournalCommentRoute,
		s.setupUpdateJournalCommentRoute,
		s.setupDeleteJournalCommentRoute,
		s.setupLikeJournalCommentRoute,
		s.setupUnlikeJournalCommentRoute,
		s.setupUploadJournalCommentMediaRoute,
	}
}

func (s *Service) setupListJournalsRoute(r fiber.Router) {
	r.Get("/journals", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listJournals)
}

func (s *Service) setupListUserJournalsRoute(r fiber.Router) {
	r.Get("/users/:id/journals", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listUserJournals)
}

func (s *Service) setupListUserFollowedJournalsRoute(r fiber.Router) {
	r.Get("/users/:id/journal-follows", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listUserFollowedJournals)
}

func (s *Service) setupCreateJournalRoute(r fiber.Router) {
	r.Post("/journals", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createJournal)
}

func (s *Service) setupGetJournalRoute(r fiber.Router) {
	r.Get("/journals/:id", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.getJournal)
}

func (s *Service) setupUpdateJournalRoute(r fiber.Router) {
	r.Put("/journals/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateJournal)
}

func (s *Service) setupDeleteJournalRoute(r fiber.Router) {
	r.Delete("/journals/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteJournal)
}

func (s *Service) setupFollowJournalRoute(r fiber.Router) {
	r.Post("/journals/:id/follow", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.followJournal)
}

func (s *Service) setupUnfollowJournalRoute(r fiber.Router) {
	r.Delete("/journals/:id/follow", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unfollowJournal)
}

func (s *Service) setupCreateJournalCommentRoute(r fiber.Router) {
	r.Post("/journals/:id/comments", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createJournalComment)
}

func (s *Service) setupUpdateJournalCommentRoute(r fiber.Router) {
	r.Put("/journal-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateJournalComment)
}

func (s *Service) setupDeleteJournalCommentRoute(r fiber.Router) {
	r.Delete("/journal-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteJournalComment)
}

func (s *Service) setupLikeJournalCommentRoute(r fiber.Router) {
	r.Post("/journal-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.likeJournalComment)
}

func (s *Service) setupUnlikeJournalCommentRoute(r fiber.Router) {
	r.Delete("/journal-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unlikeJournalComment)
}

func (s *Service) setupUploadJournalCommentMediaRoute(r fiber.Router) {
	r.Post("/journal-comments/:id/media", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.uploadJournalCommentMedia)
}

func (s *Service) listUserJournals(ctx fiber.Ctx) error {
	userID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}
	viewerID, _ := ctx.Locals("userID").(uuid.UUID)
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.JournalService.ListJournalsByUser(ctx.Context(), userID, viewerID, limit, offset)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list user journals"})
	}
	return ctx.JSON(result)
}

func (s *Service) listUserFollowedJournals(ctx fiber.Ctx) error {
	userID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}
	viewerID, _ := ctx.Locals("userID").(uuid.UUID)
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.JournalService.ListFollowedByUser(ctx.Context(), userID, viewerID, limit, offset)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list followed journals"})
	}
	return ctx.JSON(result)
}

func (s *Service) listJournals(ctx fiber.Ctx) error {
	sort := ctx.Query("sort", "new")
	work := ctx.Query("work")
	authorIDStr := ctx.Query("author")
	search := ctx.Query("search")
	includeArchived := ctx.Query("include_archived") == "true"
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)
	userID, _ := ctx.Locals("userID").(uuid.UUID)

	var authorID uuid.UUID
	if authorIDStr != "" {
		parsed, err := uuid.Parse(authorIDStr)
		if err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid author ID"})
		}
		authorID = parsed
	}

	p := params.NewListParams(sort, work, authorID, search, includeArchived, limit, offset)
	result, err := s.JournalService.ListJournals(ctx.Context(), p, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list journals"})
	}
	return ctx.JSON(result)
}

func (s *Service) createJournal(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.CreateJournalRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	id, err := s.JournalService.CreateJournal(ctx.Context(), userID, req)
	if err != nil {
		if errors.Is(err, journal.ErrRateLimited) {
			return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "daily journal limit reached"})
		}
		if errors.Is(err, journal.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title and body are required"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create journal"})
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) getJournal(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	viewerID, _ := ctx.Locals("userID").(uuid.UUID)

	result, err := s.JournalService.GetJournalDetail(ctx.Context(), id, viewerID)
	if err != nil {
		if errors.Is(err, journal.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "journal not found"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get journal"})
	}
	return ctx.JSON(result)
}

func (s *Service) updateJournal(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.CreateJournalRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.JournalService.UpdateJournal(ctx.Context(), id, userID, req); err != nil {
		if errors.Is(err, journal.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title and body are required"})
		}
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "cannot update this journal"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteJournal(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.JournalService.DeleteJournal(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "cannot delete this journal"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) followJournal(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.JournalService.FollowJournal(ctx.Context(), id, userID); err != nil {
		if errors.Is(err, journal.ErrCannotFollowOwn) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot follow your own journal"})
		}
		if errors.Is(err, journal.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "journal not found"})
		}
		if errors.Is(err, block.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is blocked"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to follow"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unfollowJournal(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.JournalService.UnfollowJournal(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to unfollow"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) createJournalComment(ctx fiber.Ctx) error {
	journalID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.CreateCommentRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	id, err := s.JournalService.CreateComment(ctx.Context(), journalID, userID, req.ParentID, req.Body)
	if err != nil {
		if errors.Is(err, journal.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body is required"})
		}
		if errors.Is(err, journal.ErrArchived) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "journal is archived"})
		}
		if errors.Is(err, journal.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "journal not found"})
		}
		if errors.Is(err, block.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is blocked"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create comment"})
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateJournalComment(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.UpdateCommentRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.JournalService.UpdateComment(ctx.Context(), id, userID, req.Body); err != nil {
		if errors.Is(err, journal.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body is required"})
		}
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "cannot update this comment"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteJournalComment(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.JournalService.DeleteComment(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "cannot delete this comment"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) likeJournalComment(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.JournalService.LikeComment(ctx.Context(), id, userID); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is blocked"})
		}
		if errors.Is(err, journal.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "comment not found"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to like comment"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unlikeJournalComment(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.JournalService.UnlikeComment(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to unlike comment"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) uploadJournalCommentMedia(ctx fiber.Ctx) error {
	commentID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
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

	contentType := file.Header.Get("Content-Type")
	result, err := s.JournalService.UploadCommentMedia(ctx.Context(), commentID, userID, contentType, file.Size, reader)
	if err != nil {
		if errors.Is(err, journal.ErrNotAuthor) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not the comment author"})
		}
		if errors.Is(err, journal.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "comment not found"})
		}
		if strings.Contains(err.Error(), "too large") {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to upload media"})
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}
