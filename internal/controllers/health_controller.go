package controllers

import (
	"github.com/gofiber/fiber/v3"
	healthgo "github.com/hellofresh/health-go/v5"
)

func (s *Service) getAllHealthRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupHealth,
	}
}

func (s *Service) setupHealth(r fiber.Router) {
	r.Get("/health", s.health)
	r.Get("/livez", s.livez)
}

func (s *Service) livez(ctx fiber.Ctx) error {
	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) health(ctx fiber.Ctx) error {
	check := s.HealthService.Measure(ctx.Context())

	code := fiber.StatusOK
	if check.Status == healthgo.StatusUnavailable {
		code = fiber.StatusServiceUnavailable
	}

	return ctx.Status(code).JSON(check)
}
