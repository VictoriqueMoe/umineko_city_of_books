package controllers

import (
	"errors"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/report"

	"github.com/gofiber/fiber/v3"
)

func (s *Service) getAllReportRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupCreateReport,
		s.setupListReports,
		s.setupResolveReport,
	}
}

func (s *Service) setupCreateReport(r fiber.Router) {
	r.Post("/report", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createReport)
}

func (s *Service) setupListReports(r fiber.Router) {
	r.Get("/admin/reports", s.requirePerm(authz.PermViewUsers), s.listReports)
}

func (s *Service) setupResolveReport(r fiber.Router) {
	r.Post("/admin/reports/:id/resolve", s.requirePerm(authz.PermViewUsers), s.resolveReport)
}

func (s *Service) createReport(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[report.CreateReportRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.ReportService.Create(ctx.Context(), userID, req); err != nil {
		if errors.Is(err, report.ErrMissingFields) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to create report")
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "ok"})
}

func (s *Service) listReports(ctx fiber.Ctx) error {
	status := ctx.Query("status", "open")
	limit := fiber.Query[int](ctx, "limit", 50)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.ReportService.List(ctx.Context(), status, limit, offset)
	if err != nil {
		return utils.InternalError(ctx, "failed to list reports")
	}

	return ctx.JSON(result)
}

func (s *Service) resolveReport(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	id := fiber.Params[int](ctx, "id", 0)
	if id == 0 {
		return utils.BadRequest(ctx, "invalid report ID")
	}

	var req struct {
		Comment string `json:"comment"`
	}
	_ = ctx.Bind().JSON(&req)

	if err := s.ReportService.Resolve(ctx.Context(), id, userID, req.Comment); err != nil {
		return utils.InternalError(ctx, "failed to resolve report")
	}

	return utils.OK(ctx)
}
