package controllers

import (
	"errors"
	"strings"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils"
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
	userID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	viewerID := utils.UserID(ctx)
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.JournalService.ListJournalsByUser(ctx.Context(), userID, viewerID, limit, offset)
	if err != nil {
		return utils.InternalError(ctx, "failed to list user journals")
	}
	return ctx.JSON(result)
}

func (s *Service) listUserFollowedJournals(ctx fiber.Ctx) error {
	userID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	viewerID := utils.UserID(ctx)
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.JournalService.ListFollowedByUser(ctx.Context(), userID, viewerID, limit, offset)
	if err != nil {
		return utils.InternalError(ctx, "failed to list followed journals")
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
	userID := utils.UserID(ctx)

	var authorID uuid.UUID
	if authorIDStr != "" {
		parsed, err := uuid.Parse(authorIDStr)
		if err != nil {
			return utils.BadRequest(ctx, "invalid author ID")
		}
		authorID = parsed
	}

	p := params.NewListParams(sort, work, authorID, search, includeArchived, limit, offset)
	result, err := s.JournalService.ListJournals(ctx.Context(), p, userID)
	if err != nil {
		return utils.InternalError(ctx, "failed to list journals")
	}
	return ctx.JSON(result)
}

func (s *Service) createJournal(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.CreateJournalRequest](ctx)
	if !ok {
		return nil
	}

	id, err := s.JournalService.CreateJournal(ctx.Context(), userID, req)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, journal.ErrRateLimited) {
			return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "daily journal limit reached"})
		}
		if errors.Is(err, journal.ErrEmptyBody) {
			return utils.BadRequest(ctx, "title and body are required")
		}
		return utils.InternalError(ctx, "failed to create journal")
	}

	s.Hub.BumpSidebarActivity("journals")
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) getJournal(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	viewerID := utils.UserID(ctx)

	result, err := s.JournalService.GetJournalDetail(ctx.Context(), id, viewerID)
	if err != nil {
		if errors.Is(err, journal.ErrNotFound) {
			return utils.NotFound(ctx, "journal not found")
		}
		return utils.InternalError(ctx, "failed to get journal")
	}
	return ctx.JSON(result)
}

func (s *Service) updateJournal(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.CreateJournalRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.JournalService.UpdateJournal(ctx.Context(), id, userID, req); err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, journal.ErrEmptyBody) {
			return utils.BadRequest(ctx, "title and body are required")
		}
		return utils.Forbidden(ctx, "cannot update this journal")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteJournal(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.JournalService.DeleteJournal(ctx.Context(), id, userID); err != nil {
		return utils.Forbidden(ctx, "cannot delete this journal")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) followJournal(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.JournalService.FollowJournal(ctx.Context(), id, userID); err != nil {
		if errors.Is(err, journal.ErrCannotFollowOwn) {
			return utils.BadRequest(ctx, "cannot follow your own journal")
		}
		if errors.Is(err, journal.ErrNotFound) {
			return utils.NotFound(ctx, "journal not found")
		}
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		return utils.InternalError(ctx, "failed to follow")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unfollowJournal(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.JournalService.UnfollowJournal(ctx.Context(), id, userID); err != nil {
		return utils.InternalError(ctx, "failed to unfollow")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) createJournalComment(ctx fiber.Ctx) error {
	journalID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.CreateCommentRequest](ctx)
	if !ok {
		return nil
	}

	id, err := s.JournalService.CreateComment(ctx.Context(), journalID, userID, req.ParentID, req.Body)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, journal.ErrEmptyBody) {
			return utils.BadRequest(ctx, "body is required")
		}
		if errors.Is(err, journal.ErrArchived) {
			return utils.Forbidden(ctx, "journal is archived")
		}
		if errors.Is(err, journal.ErrNotFound) {
			return utils.NotFound(ctx, "journal not found")
		}
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		return utils.InternalError(ctx, "failed to create comment")
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateJournalComment(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.UpdateCommentRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.JournalService.UpdateComment(ctx.Context(), id, userID, req.Body); err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, journal.ErrEmptyBody) {
			return utils.BadRequest(ctx, "body is required")
		}
		return utils.Forbidden(ctx, "cannot update this comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteJournalComment(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.JournalService.DeleteComment(ctx.Context(), id, userID); err != nil {
		return utils.Forbidden(ctx, "cannot delete this comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) likeJournalComment(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.JournalService.LikeComment(ctx.Context(), id, userID); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		if errors.Is(err, journal.ErrNotFound) {
			return utils.NotFound(ctx, "comment not found")
		}
		return utils.InternalError(ctx, "failed to like comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unlikeJournalComment(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.JournalService.UnlikeComment(ctx.Context(), id, userID); err != nil {
		return utils.InternalError(ctx, "failed to unlike comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) uploadJournalCommentMedia(ctx fiber.Ctx) error {
	commentID, ok := utils.ParseID(ctx)
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

	contentType := file.Header.Get("Content-Type")
	result, err := s.JournalService.UploadCommentMedia(ctx.Context(), commentID, userID, contentType, file.Size, reader)
	if err != nil {
		if errors.Is(err, journal.ErrNotAuthor) {
			return utils.Forbidden(ctx, "not the comment author")
		}
		if errors.Is(err, journal.ErrNotFound) {
			return utils.NotFound(ctx, "comment not found")
		}
		if strings.Contains(err.Error(), "too large") {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to upload media")
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}
