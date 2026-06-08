package controllers

import (
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func (s *Service) getAllPushRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupRegisterDeviceRoute,
		s.setupUnregisterDeviceRoute,
	}
}

func (s *Service) setupRegisterDeviceRoute(r fiber.Router) {
	r.Post("/push/device", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.registerDeviceToken)
}

func (s *Service) setupUnregisterDeviceRoute(r fiber.Router) {
	r.Delete("/push/device", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unregisterDeviceToken)
}

func (s *Service) registerDeviceToken(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.DeviceTokenRequest](ctx)
	if !ok {
		return nil
	}

	if req.Token == "" {
		return utils.BadRequest(ctx, "token is required")
	}

	if err := s.PushService.RegisterToken(ctx.Context(), userID, req.Token, req.Platform); err != nil {
		return utils.InternalError(ctx, "failed to register device token")
	}

	return utils.OK(ctx)
}

func (s *Service) unregisterDeviceToken(ctx fiber.Ctx) error {
	req, ok := utils.BindJSON[dto.DeviceTokenRequest](ctx)
	if !ok {
		return nil
	}

	if req.Token == "" {
		return utils.BadRequest(ctx, "token is required")
	}

	if err := s.PushService.UnregisterToken(ctx.Context(), req.Token); err != nil {
		return utils.InternalError(ctx, "failed to unregister device token")
	}

	return utils.OK(ctx)
}
