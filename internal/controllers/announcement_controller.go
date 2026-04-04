package controllers

import (
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/ws"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
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
	r.Get("/announcements/:id", s.getAnnouncement)
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

	rows, total, err := s.AnnouncementRepo.List(ctx.Context(), limit, offset)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list announcements"})
	}

	items := make([]fiber.Map, len(rows))
	for i, r := range rows {
		items[i] = fiber.Map{
			"id":         r.ID,
			"title":      r.Title,
			"body":       r.Body,
			"pinned":     r.Pinned,
			"created_at": r.CreatedAt,
			"updated_at": r.UpdatedAt,
			"author": dto.UserResponse{
				ID:          r.AuthorID,
				Username:    r.AuthorUsername,
				DisplayName: r.AuthorDisplayName,
				AvatarURL:   r.AuthorAvatarURL,
				Role:        role.Role(r.AuthorRole),
			},
		}
	}

	if items == nil {
		items = []fiber.Map{}
	}

	return ctx.JSON(fiber.Map{
		"announcements": items,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	})
}

func (s *Service) getAnnouncement(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	row, err := s.AnnouncementRepo.GetByID(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get announcement"})
	}
	if row == nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "announcement not found"})
	}

	return ctx.JSON(fiber.Map{
		"id":         row.ID,
		"title":      row.Title,
		"body":       row.Body,
		"pinned":     row.Pinned,
		"created_at": row.CreatedAt,
		"updated_at": row.UpdatedAt,
		"author": dto.UserResponse{
			ID:          row.AuthorID,
			Username:    row.AuthorUsername,
			DisplayName: row.AuthorDisplayName,
			AvatarURL:   row.AuthorAvatarURL,
			Role:        role.Role(row.AuthorRole),
		},
	})
}

func (s *Service) getLatestAnnouncement(ctx fiber.Ctx) error {
	row, err := s.AnnouncementRepo.GetLatest(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get latest announcement"})
	}
	if row == nil {
		return ctx.JSON(fiber.Map{"announcement": nil})
	}

	return ctx.JSON(fiber.Map{
		"announcement": fiber.Map{
			"id":         row.ID,
			"title":      row.Title,
			"body":       row.Body,
			"pinned":     row.Pinned,
			"created_at": row.CreatedAt,
			"updated_at": row.UpdatedAt,
			"author": dto.UserResponse{
				ID:          row.AuthorID,
				Username:    row.AuthorUsername,
				DisplayName: row.AuthorDisplayName,
				AvatarURL:   row.AuthorAvatarURL,
				Role:        role.Role(row.AuthorRole),
			},
		},
	})
}

func (s *Service) createAnnouncement(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)

	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.Title == "" || req.Body == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title and body are required"})
	}

	id := uuid.New()
	if err := s.AnnouncementRepo.Create(ctx.Context(), id, userID, req.Title, req.Body); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create announcement"})
	}

	s.Hub.Broadcast(ws.Message{
		Type: "new_announcement",
		Data: map[string]interface{}{
			"id":        id,
			"title":     req.Title,
			"author_id": userID,
		},
	})

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateAnnouncement(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.Title == "" || req.Body == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title and body are required"})
	}

	if err := s.AnnouncementRepo.Update(ctx.Context(), id, req.Title, req.Body); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update announcement"})
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) deleteAnnouncement(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := s.AnnouncementRepo.Delete(ctx.Context(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete announcement"})
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) pinAnnouncement(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req struct {
		Pinned bool `json:"pinned"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.AnnouncementRepo.SetPinned(ctx.Context(), id, req.Pinned); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to pin announcement"})
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}
