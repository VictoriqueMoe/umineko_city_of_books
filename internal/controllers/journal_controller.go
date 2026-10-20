package controllers

import (
	"context"
	"errors"
	"strconv"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/journal"
	"umineko_city_of_books/internal/journal/params"
	"umineko_city_of_books/internal/upload"

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
		s.setupSetJournalPausedRoute,
		s.setupFollowJournalRoute,
		s.setupUnfollowJournalRoute,
		s.setupCreateJournalCommentRoute,
		s.setupGetJournalEntryRoute,
		s.setupCreateJournalEntryRoute,
		s.setupUpdateJournalEntryRoute,
		s.setupDeleteJournalEntryRoute,
		s.setupUploadJournalEntryMediaRoute,
		s.setupDeleteJournalEntryMediaRoute,
	}
}

func (s *Service) setupListJournalsRoute(r fiber.Router) {
	r.Get("/journals", s.optionalAuth(), s.listJournals)
}

func (s *Service) setupListUserJournalsRoute(r fiber.Router) {
	r.Get("/users/:id/journals", s.optionalAuth(), s.listUserJournals)
}

func (s *Service) setupListUserFollowedJournalsRoute(r fiber.Router) {
	r.Get("/users/:id/journal-follows", s.optionalAuth(), s.listUserFollowedJournals)
}

func (s *Service) setupCreateJournalRoute(r fiber.Router) {
	r.Post("/journals", s.requireAuth(), s.createJournal)
}

func (s *Service) setupGetJournalRoute(r fiber.Router) {
	r.Get("/journals/:id", s.optionalAuth(), s.getJournal)
}

func (s *Service) setupUpdateJournalRoute(r fiber.Router) {
	r.Put("/journals/:id", s.requireAuth(), s.updateJournal)
}

func (s *Service) setupDeleteJournalRoute(r fiber.Router) {
	r.Delete("/journals/:id", s.requireAuth(), s.deleteJournal)
}

func (s *Service) setupSetJournalPausedRoute(r fiber.Router) {
	r.Put("/journals/:id/pause", s.requireAuth(), s.setJournalPaused)
}

func (s *Service) setupFollowJournalRoute(r fiber.Router) {
	r.Post("/journals/:id/follow", s.requireAuth(), s.followJournal)
}

func (s *Service) setupUnfollowJournalRoute(r fiber.Router) {
	r.Delete("/journals/:id/follow", s.requireAuth(), s.unfollowJournal)
}

func (s *Service) setupCreateJournalCommentRoute(r fiber.Router) {
	r.Post("/journals/:id/comments", s.requireAuth(), s.requireEstablished(), s.createJournalComment)
}

func (s *Service) setupGetJournalEntryRoute(r fiber.Router) {
	r.Get("/journals/:id/entries/:number", s.optionalAuth(), s.getJournalEntry)
}

func (s *Service) setupCreateJournalEntryRoute(r fiber.Router) {
	r.Post("/journals/:id/entries", s.requireAuth(), s.requireEstablished(), s.createJournalEntry)
}

func (s *Service) setupUpdateJournalEntryRoute(r fiber.Router) {
	r.Put("/journal-entries/:id", s.requireAuth(), s.updateJournalEntry)
}

func (s *Service) setupDeleteJournalEntryRoute(r fiber.Router) {
	r.Delete("/journal-entries/:id", s.requireAuth(), s.deleteJournalEntry)
}

func (s *Service) setupUploadJournalEntryMediaRoute(r fiber.Router) {
	r.Post("/journal-entries/:id/media", s.requireAuth(), s.requireEstablished(), s.uploadJournalEntryMedia)
}

func (s *Service) setupDeleteJournalEntryMediaRoute(r fiber.Router) {
	r.Delete("/journal-entries/:id/media/:mediaId", s.requireAuth(), s.deleteJournalEntryMedia)
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
		return utils.InternalError(ctx, "failed to list user journals", err)
	}
	return ctx.JSON(result)
}

func (s *Service) listUserFollowedJournals(ctx fiber.Ctx) error {
	userID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	viewerID := utils.UserID(ctx)
	page := utils.Page(ctx, 20)

	result, err := s.JournalService.ListFollowedByUser(ctx.Context(), userID, viewerID, page)
	if err != nil {
		return utils.InternalError(ctx, "failed to list followed journals", err)
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
		return utils.InternalError(ctx, "failed to list journals", err)
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
		if errors.Is(err, journal.ErrEmptyTitle) {
			return utils.BadRequest(ctx, "title is required")
		}
		return utils.InternalError(ctx, "failed to create journal", err)
	}

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
		return utils.InternalError(ctx, "failed to get journal", err)
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
		if errors.Is(err, journal.ErrEmptyTitle) {
			return utils.BadRequest(ctx, "title is required")
		}
		if errors.Is(err, journal.ErrNotFound) {
			return utils.NotFound(ctx, "journal not found")
		}
		if errors.Is(err, daoerr.ErrNotFound) {
			return utils.Forbidden(ctx, "cannot update this journal")
		}
		return utils.InternalError(ctx, "failed to update journal", err)
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
		if errors.Is(err, journal.ErrNotFound) {
			return utils.NotFound(ctx, "journal not found")
		}
		if errors.Is(err, daoerr.ErrNotFound) {
			return utils.Forbidden(ctx, "cannot delete this journal")
		}
		return utils.InternalError(ctx, "failed to delete journal", err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) setJournalPaused(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	var req struct {
		Paused bool `json:"paused"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request")
	}

	if err := s.JournalService.SetJournalPaused(ctx.Context(), id, userID, req.Paused); err != nil {
		if errors.Is(err, journal.ErrNotFound) {
			return utils.NotFound(ctx, "journal not found")
		}
		if errors.Is(err, journal.ErrNotAuthor) {
			return utils.Forbidden(ctx, "not the journal author")
		}
		return utils.InternalError(ctx, "failed to update the journal", err)
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
		return utils.InternalError(ctx, "failed to follow", err)
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
		return utils.InternalError(ctx, "failed to unfollow", err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) createJournalComment(ctx fiber.Ctx) error {
	journalID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	return handleCreateComment(ctx, func(c context.Context, req dto.CreateJournalCommentRequest, files []upload.FileUpload) (uuid.UUID, error) {
		return s.JournalService.CreateComment(c, journalID, userID, req, files)
	}, []errorResponse{
		{err: journal.ErrArchived, respond: utils.Forbidden, message: "journal is archived"},
		{err: journal.ErrNotFound, respond: utils.NotFound, message: "journal not found"},
		{err: journal.ErrEntryNotFound, respond: utils.NotFound, message: "entry not found"},
		{err: journal.ErrEntryMismatch, respond: utils.BadRequest, message: "entry does not belong to this journal"},
	})
}

func (s *Service) deleteJournalEntryMedia(ctx fiber.Ctx) error {
	entryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	mediaID, err := strconv.ParseInt(ctx.Params("mediaId"), 10, 64)
	if err != nil {
		return utils.BadRequest(ctx, "invalid media id")
	}
	userID := utils.UserID(ctx)

	if err := s.JournalService.DeleteEntryMedia(ctx.Context(), entryID, mediaID, userID); err != nil {
		if errors.Is(err, journal.ErrNotAuthor) {
			return utils.Forbidden(ctx, "not the entry author")
		}
		if errors.Is(err, journal.ErrNotFound) {
			return utils.NotFound(ctx, "entry not found")
		}
		return utils.InternalError(ctx, "failed to delete media", err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) uploadJournalEntryMedia(ctx fiber.Ctx) error {
	entryID, ok := utils.ParseID(ctx)
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
		return utils.InternalError(ctx, "failed to read file", err)
	}
	defer reader.Close()

	contentType := file.Header.Get("Content-Type")
	result, err := s.JournalService.UploadEntryMedia(ctx.Context(), entryID, userID, contentType, file.Filename, file.Size, reader, isSpoilerUpload(ctx))
	if err != nil {
		if errors.Is(err, journal.ErrNotAuthor) {
			return utils.Forbidden(ctx, "not the entry author")
		}
		if errors.Is(err, journal.ErrNotFound) {
			return utils.NotFound(ctx, "entry not found")
		}
		if utils.IsUploadRejection(err) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to upload media", err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}

func (s *Service) getJournalEntry(ctx fiber.Ctx) error {
	journalID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	numStr := ctx.Params("number")
	number, err := strconv.Atoi(numStr)
	if err != nil || number < 1 {
		return utils.BadRequest(ctx, "invalid entry number")
	}
	viewerID := utils.UserID(ctx)

	entry, comments, err := s.JournalService.GetEntry(ctx.Context(), journalID, number, viewerID)
	if err != nil {
		if errors.Is(err, journal.ErrEntryNotFound) || errors.Is(err, journal.ErrNotFound) {
			return utils.NotFound(ctx, "entry not found")
		}
		return utils.InternalError(ctx, "failed to get entry", err)
	}
	return ctx.JSON(fiber.Map{"entry": entry, "comments": comments})
}

func (s *Service) createJournalEntry(ctx fiber.Ctx) error {
	journalID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindWithMedia[dto.CreateJournalEntryRequest](ctx)
	if !ok {
		return nil
	}

	id, entryNumber, err := s.JournalService.CreateEntry(ctx.Context(), journalID, userID, req.Payload, req.Media)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, journal.ErrEmptyBody) {
			return utils.BadRequest(ctx, "body is required")
		}
		if errors.Is(err, journal.ErrNotFound) {
			return utils.NotFound(ctx, "journal not found")
		}
		if errors.Is(err, journal.ErrNotAuthor) {
			return utils.Forbidden(ctx, "not the journal author")
		}
		if utils.IsUploadRejection(err) {
			return utils.UploadRejected(ctx, err)
		}
		return utils.InternalError(ctx, "failed to create entry", err)
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id, "entry_number": entryNumber})
}

func (s *Service) updateJournalEntry(ctx fiber.Ctx) error {
	entryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.UpdateJournalEntryRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.JournalService.UpdateEntry(ctx.Context(), entryID, userID, req); err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, journal.ErrEmptyBody) {
			return utils.BadRequest(ctx, "body is required")
		}
		if errors.Is(err, journal.ErrEntryNotFound) {
			return utils.NotFound(ctx, "entry not found")
		}
		if errors.Is(err, journal.ErrNotAuthor) {
			return utils.Forbidden(ctx, "not the entry author")
		}
		return utils.InternalError(ctx, "failed to update entry", err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteJournalEntry(ctx fiber.Ctx) error {
	entryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.JournalService.DeleteEntry(ctx.Context(), entryID, userID); err != nil {
		if errors.Is(err, journal.ErrEntryNotFound) {
			return utils.NotFound(ctx, "entry not found")
		}
		if errors.Is(err, journal.ErrNotAuthor) {
			return utils.Forbidden(ctx, "not the entry author")
		}
		return utils.InternalError(ctx, "failed to delete entry", err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}
