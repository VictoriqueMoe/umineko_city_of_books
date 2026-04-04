package middleware

import (
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/session"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func RequirePermission(mgr *session.Manager, authzSvc authz.Service, perm authz.Permission) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		userID, _, err := authenticateAndCheckBan(ctx, mgr, authzSvc)
		if err != nil {
			return err
		}

		if !authzSvc.Can(ctx.Context(), userID, perm) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "insufficient permissions",
			})
		}

		ctx.Locals("userID", userID)
		return ctx.Next()
	}
}

func OptionalAuth(mgr *session.Manager, authzSvc authz.Service) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		cookie := ctx.Cookies(session.CookieName)
		if cookie == "" {
			ctx.Locals("userID", uuid.Nil)
			return ctx.Next()
		}

		userID, err := mgr.Validate(ctx.Context(), cookie)
		if err != nil {
			ctx.Locals("userID", uuid.Nil)
			return ctx.Next()
		}

		if authzSvc.IsBanned(ctx.Context(), userID) {
			mgr.Delete(ctx.Context(), cookie)
			ctx.Locals("userID", uuid.Nil)
			return ctx.Next()
		}

		ctx.Locals("userID", userID)
		return ctx.Next()
	}
}

func RequireAuth(mgr *session.Manager, authzSvc authz.Service) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		userID, _, err := authenticateAndCheckBan(ctx, mgr, authzSvc)
		if err != nil {
			return err
		}

		ctx.Locals("userID", userID)
		return ctx.Next()
	}
}

func authenticateAndCheckBan(ctx fiber.Ctx, mgr *session.Manager, authzSvc authz.Service) (uuid.UUID, string, error) {
	cookie := ctx.Cookies(session.CookieName)
	if cookie == "" {
		return uuid.Nil, "", ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	userID, err := mgr.Validate(ctx.Context(), cookie)
	if err != nil {
		return uuid.Nil, "", ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid or expired session",
		})
	}

	if authzSvc.IsBanned(ctx.Context(), userID) {
		mgr.Delete(ctx.Context(), cookie)
		return uuid.Nil, "", ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "your account has been banned",
		})
	}

	return userID, cookie, nil
}
