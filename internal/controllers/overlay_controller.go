package controllers

import (
	"errors"

	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/overlay"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllOverlayRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupOverlayConnect,
	}
}

func (s *Service) setupOverlayConnect(r fiber.Router) {
	r.Get("/overlay", s.OverlayService.Handler())
}

func (s *Service) getAllOverlayTokenRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupOverlayToken,
		s.setupOverlayTokenReset,
		s.setupOverlayConnectorSEF,
		s.setupOverlayTest,
	}
}

func (s *Service) setupOverlayToken(r fiber.Router) {
	r.Get("/overlay/token", s.requireAuth(), s.overlayToken)
}

func (s *Service) setupOverlayTokenReset(r fiber.Router) {
	r.Post("/overlay/token/reset", s.requireAuth(), s.overlayTokenReset)
}

func (s *Service) setupOverlayConnectorSEF(r fiber.Router) {
	r.Get("/overlay/connector.sef", s.requireAuth(), s.overlayConnectorSEF)
}

func (s *Service) setupOverlayTest(r fiber.Router) {
	r.Post("/overlay/test", s.requireAuth(), s.overlayTest)
}

func (s *Service) overlayToken(ctx fiber.Ctx) error {
	return s.respondOverlayConnection(ctx, utils.UserID(ctx))
}

func (s *Service) overlayTokenReset(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	if _, err := s.OverlayService.ResetToken(ctx.Context(), userID); err != nil {
		logger.Log.Warn().Err(err).Msg("overlay token reset failed")
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	return s.respondOverlayConnection(ctx, userID)
}

func (s *Service) overlayConnectorSEF(ctx fiber.Ctx) error {
	sef, err := s.OverlayService.BuildSEF(ctx.Context(), utils.UserID(ctx))
	if err != nil {
		logger.Log.Warn().Err(err).Msg("overlay sef build failed")
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	ctx.Set("Content-Type", "text/plain; charset=utf-8")
	ctx.Set("Content-Disposition", `attachment; filename="overlay-connector.sef"`)

	return ctx.SendString(sef)
}

func (s *Service) overlayTest(ctx fiber.Ctx) error {
	if err := s.OverlayService.TestFire(utils.UserID(ctx)); err != nil {
		if errors.Is(err, overlay.ErrNotConnected) {
			return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "no overlay connection"})
		}

		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	return ctx.JSON(fiber.Map{"ok": true})
}

func (s *Service) respondOverlayConnection(ctx fiber.Ctx, userID uuid.UUID) error {
	token, err := s.OverlayService.Token(ctx.Context(), userID)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("overlay token failed")
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	connectURL, err := s.OverlayService.ConnectURL(ctx.Context(), userID)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("overlay connect url failed")
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	return ctx.JSON(fiber.Map{
		"token":       token,
		"connect_url": connectURL,
		"connected":   s.OverlayService.IsConnected(userID),
	})
}
