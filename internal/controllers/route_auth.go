package controllers

import (
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func (s *Service) requireAuth() fiber.Handler {
	return middleware.RequireAuth(s.AuthSession, s.AuthzService)
}

func (s *Service) optionalAuth() fiber.Handler {
	return middleware.OptionalAuth(s.AuthSession, s.AuthzService)
}

func (s *Service) requireEstablished() fiber.Handler {
	return middleware.RequireEstablishedAccount(s.AuthzService, s.SettingsService)
}

func (s *Service) requirePerm(perm authz.Permission) fiber.Handler {
	return middleware.RequirePermission(s.AuthSession, s.AuthzService, perm)
}
