package controllers

import (
	"errors"

	announcementsvc "umineko_city_of_books/internal/announcement"
	"umineko_city_of_books/internal/authz"
	ctrlutils "umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"

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
	}
}

func (s *Service) setupListAnnouncements(r fiber.Router) {
	r.Get("/announcements", s.listAnnouncements)
}

func (s *Service) setupGetAnnouncement(r fiber.Router) {
	r.Get("/announcements/:id", s.optionalAuth(), s.getAnnouncement)
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
	page := ctrlutils.Page(ctx, 20)

	resp, err := s.AnnouncementService.List(ctx.Context(), page)
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

	if err := s.AnnouncementService.Update(ctx.Context(), ctrlutils.UserID(ctx), id, req.Title, req.Body); err != nil {
		if errors.Is(err, announcementsvc.ErrEmptyTitleOrBody) {
			return ctrlutils.BadRequest(ctx, "title and body are required")
		}
		if errors.Is(err, announcementsvc.ErrNotFound) {
			return ctrlutils.NotFound(ctx, "announcement not found")
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

	if err := s.AnnouncementService.Delete(ctx.Context(), ctrlutils.UserID(ctx), id); err != nil {
		if errors.Is(err, announcementsvc.ErrNotFound) {
			return ctrlutils.NotFound(ctx, "announcement not found")
		}
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

	if err := s.AnnouncementService.SetPinned(ctx.Context(), ctrlutils.UserID(ctx), id, req.Pinned); err != nil {
		if errors.Is(err, announcementsvc.ErrNotFound) {
			return ctrlutils.NotFound(ctx, "announcement not found")
		}
		return ctrlutils.InternalError(ctx, "failed to pin announcement", err)
	}

	return ctrlutils.OK(ctx)
}
