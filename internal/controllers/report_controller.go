package controllers

import (
	"errors"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/report"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllReportRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupCreateReport,
		s.setupListReports,
		s.setupResolveReport,
	}
}

func (s *Service) setupCreateReport(r fiber.Router) {
	r.Post("/report", middleware.RequireAuth(s.AuthSession), s.createReport)
}

func (s *Service) setupListReports(r fiber.Router) {
	r.Get("/admin/reports", s.requirePerm(authz.PermViewUsers), s.listReports)
}

func (s *Service) setupResolveReport(r fiber.Router) {
	r.Post("/admin/reports/:id/resolve", s.requirePerm(authz.PermViewUsers), s.resolveReport)
}

func (s *Service) createReport(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)

	var req report.CreateReportRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := s.ReportService.Create(ctx.Context(), userID, req); err != nil {
		if errors.Is(err, report.ErrMissingFields) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create report",
		})
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "ok"})
}

func (s *Service) listReports(ctx fiber.Ctx) error {
	status := ctx.Query("status", "open")
	limit := fiber.Query[int](ctx, "limit", 50)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.ReportService.List(ctx.Context(), status, limit, offset)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list reports",
		})
	}

	return ctx.JSON(result)
}

func (s *Service) resolveReport(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)
	id := fiber.Params[int](ctx, "id", 0)
	if id == 0 {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid report ID",
		})
	}

	var req struct {
		Comment string `json:"comment"`
	}
	_ = ctx.Bind().JSON(&req)

	if err := s.ReportService.Resolve(ctx.Context(), id, userID, req.Comment); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to resolve report",
		})
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}
