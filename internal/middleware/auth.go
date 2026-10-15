package middleware

import (
	"context"
	"errors"
	"strings"

	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authctx"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/session"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const (
	authResultLocal = "auth_result"
)

type (
	Authenticator interface {
		Authenticate(ctx context.Context, token string) (model.SessionAuthState, error)
	}

	authResult struct {
		token string
		state model.SessionAuthState
		err   error
	}
)

func SessionToken(ctx fiber.Ctx) string {
	if bearer := ctx.Bearer(); bearer != "" {
		return bearer
	}

	return ctx.Cookies(session.CookieName)
}

func authenticate(ctx fiber.Ctx, authenticator Authenticator, token string) (model.SessionAuthState, error) {
	if cached, ok := ctx.Locals(authResultLocal).(authResult); ok && cached.token == token {
		return cached.state, cached.err
	}

	state, err := authenticator.Authenticate(ctx.Context(), token)
	ctx.SetLocal(authResultLocal, authResult{token: token, state: state, err: err})

	return state, err
}

func hasSession(ctx fiber.Ctx, authenticator Authenticator) bool {
	if authenticator == nil {
		return false
	}

	token := SessionToken(ctx)
	if token == "" {
		return false
	}

	_, err := authenticate(ctx, authenticator, token)

	return err == nil
}

func hasQuerySession(ctx fiber.Ctx, authenticator Authenticator) bool {
	if authenticator == nil {
		return false
	}

	token := ctx.Query("token")
	if token == "" {
		return false
	}

	_, err := authenticate(ctx, authenticator, token)

	return err == nil
}

func isWriteMethod(method string) bool {
	switch method {
	case fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete:
		return true
	default:
		return false
	}
}

func isVerifyExemptPath(method, path string) bool {
	if method != fiber.MethodPost {
		return false
	}
	switch path {
	case "/api/v1/auth/set-email", "/api/v1/auth/verify-email", "/api/v1/auth/resend-verification":
		return true
	case "/api/v1/notifications/read":
		return true
	}
	if strings.HasPrefix(path, "/api/v1/notifications/") && strings.HasSuffix(path, "/read") {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/chat/rooms/") && strings.HasSuffix(path, "/read") {
		return true
	}
	return false
}

func isLockExemptPath(method, path string) bool {
	if method != fiber.MethodPost {
		return false
	}
	if strings.HasPrefix(path, "/api/v1/notifications/") && strings.HasSuffix(path, "/read") {
		return true
	}
	if path == "/api/v1/notifications/read" {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/chat/rooms/") && strings.HasSuffix(path, "/read") {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/chat/rooms/") && strings.HasSuffix(path, "/messages") {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/chat/dm/") && strings.HasSuffix(path, "/messages") {
		return true
	}
	return false
}

func RequirePermission(authenticator Authenticator, authzSvc authz.Service, perm authz.Permission) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		userID, ok := requireAccount(ctx, authenticator)
		if !ok {
			return nil
		}

		if !authzSvc.Can(ctx.Context(), userID, perm) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "insufficient permissions",
			})
		}

		ctx.SetLocal("userID", userID)
		ctx.SetContext(authctx.WithUserID(ctx.Context(), userID))
		return ctx.Next()
	}
}

func OptionalAuth(authenticator Authenticator) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		token := SessionToken(ctx)
		if token == "" {
			ctx.SetLocal("userID", uuid.Nil)
			return ctx.Next()
		}

		state, err := authenticate(ctx, authenticator, token)
		if err != nil {
			ctx.SetLocal("userID", uuid.Nil)
			return ctx.Next()
		}

		ctx.SetLocal("userID", state.UserID)
		ctx.SetContext(authctx.WithUserID(ctx.Context(), state.UserID))
		return ctx.Next()
	}
}

func RequireAuth(authenticator Authenticator) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		userID, ok := requireAccount(ctx, authenticator)
		if !ok {
			return nil
		}

		ctx.SetLocal("userID", userID)
		ctx.SetContext(authctx.WithUserID(ctx.Context(), userID))
		return ctx.Next()
	}
}

func requireAccount(ctx fiber.Ctx, authenticator Authenticator) (uuid.UUID, bool) {
	token := SessionToken(ctx)
	if token == "" {
		_ = ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
		return uuid.Nil, false
	}

	state, err := authenticate(ctx, authenticator, token)
	if errors.Is(err, auth.ErrUserBanned) {
		_ = ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "your account has been banned",
		})
		return uuid.Nil, false
	}
	if err != nil {
		_ = ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid or expired session",
		})
		return uuid.Nil, false
	}

	if state.Locked && isWriteMethod(ctx.Method()) && !isLockExemptPath(ctx.Method(), ctx.Path()) {
		_ = ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "your account is locked",
		})
		return uuid.Nil, false
	}

	if state.RequiresEmailVerification && isWriteMethod(ctx.Method()) && !isVerifyExemptPath(ctx.Method(), ctx.Path()) {
		_ = ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "please verify your email address to continue",
			"code":  "email_unverified",
		})
		return uuid.Nil, false
	}

	return state.UserID, true
}
