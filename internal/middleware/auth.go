package middleware

import (
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/session"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func RequirePermission(mgr *session.Manager, authzSvc authz.Service, perm authz.Permission) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		userID, _, ok := authenticateAndCheckBan(ctx, mgr, authzSvc)
		if !ok {
			return nil
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
		userID, _, ok := authenticateAndCheckBan(ctx, mgr, authzSvc)
		if !ok {
			return nil
		}

		ctx.Locals("userID", userID)
		return ctx.Next()
	}
}

// authenticateAndCheckBan validates the session cookie and ban status. On any
// failure it writes the appropriate response and returns ok=false; callers
// must then `return nil` so fiber does not run subsequent handlers.
func authenticateAndCheckBan(ctx fiber.Ctx, mgr *session.Manager, authzSvc authz.Service) (uuid.UUID, string, bool) {
	cookie := ctx.Cookies(session.CookieName)
	if cookie == "" {
		_ = ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
		return uuid.Nil, "", false
	}

	userID, err := mgr.Validate(ctx.Context(), cookie)
	if err != nil {
		_ = ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid or expired session",
		})
		return uuid.Nil, "", false
	}

	if authzSvc.IsBanned(ctx.Context(), userID) {
		mgr.Delete(ctx.Context(), cookie)
		_ = ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "your account has been banned",
		})
		return uuid.Nil, "", false
	}

	return userID, cookie, true
}
