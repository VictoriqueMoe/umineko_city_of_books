package controllers

import (
	"errors"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/overlay"
	"umineko_city_of_books/internal/session"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type (
	OverlayHandler struct {
		svc     overlay.Service
		session *session.Manager
		authz   authz.Service
	}
)

func NewOverlayHandler(svc overlay.Service, sessionMgr *session.Manager, authzSvc authz.Service) *OverlayHandler {
	return &OverlayHandler{svc: svc, session: sessionMgr, authz: authzSvc}
}

func (h *OverlayHandler) Register(app fiber.Router) {
	auth := middleware.RequireAuth(h.session, h.authz)
	r := app.Group("/api/v1/overlay")
	r.Get("/token", auth, h.getToken)
	r.Post("/token/reset", auth, h.resetToken)
	r.Get("/connector.sef", auth, h.downloadSEF)
	r.Post("/test", auth, h.test)
}

func (h *OverlayHandler) getToken(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("userID").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return ctx.SendStatus(fiber.StatusUnauthorized)
	}

	return h.respondConnection(ctx, userID)
}

func (h *OverlayHandler) resetToken(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("userID").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return ctx.SendStatus(fiber.StatusUnauthorized)
	}

	if _, err := h.svc.ResetToken(ctx.Context(), userID); err != nil {
		logger.Log.Warn().Err(err).Msg("overlay token reset failed")
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	return h.respondConnection(ctx, userID)
}

func (h *OverlayHandler) downloadSEF(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("userID").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return ctx.SendStatus(fiber.StatusUnauthorized)
	}

	sef, err := h.svc.BuildSEF(ctx.Context(), userID)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("overlay sef build failed")
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	ctx.Set("Content-Type", "text/plain; charset=utf-8")
	ctx.Set("Content-Disposition", `attachment; filename="overlay-connector.sef"`)
	return ctx.SendString(sef)
}

func (h *OverlayHandler) test(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("userID").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return ctx.SendStatus(fiber.StatusUnauthorized)
	}

	if err := h.svc.TestFire(userID); err != nil {
		if errors.Is(err, overlay.ErrNotConnected) {
			return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "no overlay connection"})
		}
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	return ctx.JSON(fiber.Map{"ok": true})
}

func (h *OverlayHandler) respondConnection(ctx fiber.Ctx, userID uuid.UUID) error {
	token, err := h.svc.Token(ctx.Context(), userID)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("overlay token failed")
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	connectURL, err := h.svc.ConnectURL(ctx.Context(), userID)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("overlay connect url failed")
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	return ctx.JSON(fiber.Map{
		"token":       token,
		"connect_url": connectURL,
		"connected":   h.svc.IsConnected(userID),
	})
}
