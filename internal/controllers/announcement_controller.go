package controllers

import (
	"errors"
	"strings"

	announcementsvc "umineko_city_of_books/internal/announcement"
	"umineko_city_of_books/internal/authz"
	ctrlutils "umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func (s *Service) getAllAnnouncementRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupListAnnouncements,
		s.setupGetAnnouncement,
		s.setupGetLatestAnnouncement,
		s.setupCreateAnnouncement,
		s.setupUpdateAnnouncement,
		s.setupDeleteAnnouncement,
		s.setupPinAnnouncement,
		s.setupCreateAnnouncementComment,
		s.setupUpdateAnnouncementComment,
		s.setupDeleteAnnouncementComment,
		s.setupLikeAnnouncementComment,
		s.setupUnlikeAnnouncementComment,
		s.setupUploadAnnouncementCommentMedia,
	}
}

func (s *Service) setupCreateAnnouncementComment(r fiber.Router) {
	r.Post("/announcements/:id/comments", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createAnnouncementComment)
}

func (s *Service) setupUpdateAnnouncementComment(r fiber.Router) {
	r.Put("/announcement-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateAnnouncementComment)
}

func (s *Service) setupDeleteAnnouncementComment(r fiber.Router) {
	r.Delete("/announcement-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteAnnouncementComment)
}

func (s *Service) setupLikeAnnouncementComment(r fiber.Router) {
	r.Post("/announcement-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.likeAnnouncementComment)
}

func (s *Service) setupUnlikeAnnouncementComment(r fiber.Router) {
	r.Delete("/announcement-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unlikeAnnouncementComment)
}

func (s *Service) setupUploadAnnouncementCommentMedia(r fiber.Router) {
	r.Post("/announcement-comments/:id/media", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.uploadAnnouncementCommentMedia)
}

func (s *Service) setupListAnnouncements(r fiber.Router) {
	r.Get("/announcements", s.listAnnouncements)
}

func (s *Service) setupGetAnnouncement(r fiber.Router) {
	r.Get("/announcements/:id", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.getAnnouncement)
}

func (s *Service) setupGetLatestAnnouncement(r fiber.Router) {
	r.Get("/announcements-latest", s.getLatestAnnouncement)
}

func (s *Service) setupCreateAnnouncement(r fiber.Router) {
	r.Post("/admin/announcements", s.requirePerm(authz.PermManageSettings), s.createAnnouncement)
}

func (s *Service) setupUpdateAnnouncement(r fiber.Router) {
	r.Put("/admin/announcements/:id", s.requirePerm(authz.PermManageSettings), s.updateAnnouncement)
}

func (s *Service) setupDeleteAnnouncement(r fiber.Router) {
	r.Delete("/admin/announcements/:id", s.requirePerm(authz.PermManageSettings), s.deleteAnnouncement)
}

func (s *Service) setupPinAnnouncement(r fiber.Router) {
	r.Post("/admin/announcements/:id/pin", s.requirePerm(authz.PermManageSettings), s.pinAnnouncement)
}

func (s *Service) listAnnouncements(ctx fiber.Ctx) error {
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	resp, err := s.AnnouncementService.List(ctx.Context(), limit, offset)
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to list announcements", err)
	}
	return ctx.JSON(resp)
}

func (s *Service) getAnnouncement(ctx fiber.Ctx) error {
	id, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}
	viewerID := ctrlutils.UserID(ctx)

	resp, err := s.AnnouncementService.GetDetail(ctx.Context(), id, viewerID)
	if err != nil {
		if errors.Is(err, announcementsvc.ErrNotFound) {
			return ctrlutils.NotFound(ctx, "announcement not found")
		}
		return ctrlutils.InternalError(ctx, "failed to get announcement", err)
	}
	return ctx.JSON(resp)
}

func (s *Service) getLatestAnnouncement(ctx fiber.Ctx) error {
	resp, err := s.AnnouncementService.GetLatest(ctx.Context())
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to get latest announcement", err)
	}
	return ctx.JSON(dto.AnnouncementLatestResponse{Announcement: resp})
}

func (s *Service) createAnnouncement(ctx fiber.Ctx) error {
	userID := ctrlutils.UserID(ctx)

	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctrlutils.BadRequest(ctx, "invalid request body")
	}

	id, err := s.AnnouncementService.Create(ctx.Context(), userID, req.Title, req.Body)
	if err != nil {
		if errors.Is(err, announcementsvc.ErrEmptyTitleOrBody) {
			return ctrlutils.BadRequest(ctx, "title and body are required")
		}
		return ctrlutils.InternalError(ctx, "failed to create announcement", err)
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateAnnouncement(ctx fiber.Ctx) error {
	id, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}

	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctrlutils.BadRequest(ctx, "invalid request body")
	}

	if err := s.AnnouncementService.Update(ctx.Context(), id, req.Title, req.Body); err != nil {
		if errors.Is(err, announcementsvc.ErrEmptyTitleOrBody) {
			return ctrlutils.BadRequest(ctx, "title and body are required")
		}
		return ctrlutils.InternalError(ctx, "failed to update announcement", err)
	}

	return ctrlutils.OK(ctx)
}

func (s *Service) deleteAnnouncement(ctx fiber.Ctx) error {
	id, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}

	if err := s.AnnouncementService.Delete(ctx.Context(), id); err != nil {
		return ctrlutils.InternalError(ctx, "failed to delete announcement", err)
	}

	return ctrlutils.OK(ctx)
}

func (s *Service) pinAnnouncement(ctx fiber.Ctx) error {
	id, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}

	var req struct {
		Pinned bool `json:"pinned"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctrlutils.BadRequest(ctx, "invalid request body")
	}

	if err := s.AnnouncementService.SetPinned(ctx.Context(), id, req.Pinned); err != nil {
		return ctrlutils.InternalError(ctx, "failed to pin announcement", err)
	}

	return ctrlutils.OK(ctx)
}

func (s *Service) createAnnouncementComment(ctx fiber.Ctx) error {
	announcementID, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := ctrlutils.UserID(ctx)

	req, ok := ctrlutils.BindJSON[dto.CreateCommentRequest](ctx)
	if !ok {
		return nil
	}

	id, err := s.AnnouncementService.CreateComment(ctx.Context(), announcementID, userID, req.ParentID, strings.TrimSpace(req.Body))
	if err != nil {
		switch {
		case errors.Is(err, announcementsvc.ErrEmptyBody):
			return ctrlutils.BadRequest(ctx, "body is required")
		case errors.Is(err, announcementsvc.ErrNotFound):
			return ctrlutils.NotFound(ctx, "announcement not found")
		case errors.Is(err, announcementsvc.ErrBlocked):
			return ctrlutils.Forbidden(ctx, "user is blocked")
		}
		return ctrlutils.InternalError(ctx, "failed to create comment", err)
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateAnnouncementComment(ctx fiber.Ctx) error {
	id, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := ctrlutils.UserID(ctx)

	req, ok := ctrlutils.BindJSON[dto.UpdateCommentRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.AnnouncementService.UpdateComment(ctx.Context(), id, userID, strings.TrimSpace(req.Body)); err != nil {
		switch {
		case errors.Is(err, announcementsvc.ErrEmptyBody):
			return ctrlutils.BadRequest(ctx, "body is required")
		case errors.Is(err, announcementsvc.ErrForbidden):
			return ctrlutils.Forbidden(ctx, "cannot update this comment")
		}
		return ctrlutils.InternalError(ctx, "failed to update comment", err)
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteAnnouncementComment(ctx fiber.Ctx) error {
	id, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := ctrlutils.UserID(ctx)

	if err := s.AnnouncementService.DeleteComment(ctx.Context(), id, userID); err != nil {
		if errors.Is(err, announcementsvc.ErrForbidden) {
			return ctrlutils.Forbidden(ctx, "cannot delete this comment")
		}
		return ctrlutils.InternalError(ctx, "failed to delete comment", err)
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) likeAnnouncementComment(ctx fiber.Ctx) error {
	commentID, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := ctrlutils.UserID(ctx)

	if err := s.AnnouncementService.LikeComment(ctx.Context(), userID, commentID); err != nil {
		switch {
		case errors.Is(err, announcementsvc.ErrCommentNotFound):
			return ctrlutils.NotFound(ctx, "comment not found")
		case errors.Is(err, announcementsvc.ErrBlocked):
			return ctrlutils.Forbidden(ctx, "user is blocked")
		}
		return ctrlutils.InternalError(ctx, "failed to like comment", err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unlikeAnnouncementComment(ctx fiber.Ctx) error {
	commentID, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := ctrlutils.UserID(ctx)

	if err := s.AnnouncementService.UnlikeComment(ctx.Context(), userID, commentID); err != nil {
		return ctrlutils.InternalError(ctx, "failed to unlike comment", err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) uploadAnnouncementCommentMedia(ctx fiber.Ctx) error {
	commentID, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := ctrlutils.UserID(ctx)

	file, err := ctx.FormFile("media")
	if err != nil {
		return ctrlutils.BadRequest(ctx, "no media file provided")
	}
	reader, err := file.Open()
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to read file", err)
	}
	defer reader.Close()

	resp, err := s.AnnouncementService.UploadCommentMedia(ctx.Context(), commentID, userID, file.Header.Get("Content-Type"), file.Size, reader)
	if err != nil {
		switch {
		case errors.Is(err, announcementsvc.ErrCommentNotFound):
			return ctrlutils.NotFound(ctx, "comment not found")
		case errors.Is(err, announcementsvc.ErrForbidden):
			return ctrlutils.Forbidden(ctx, "not the comment author")
		}
		return ctrlutils.BadRequest(ctx, err.Error())
	}

	return ctx.Status(fiber.StatusCreated).JSON(resp)
}
