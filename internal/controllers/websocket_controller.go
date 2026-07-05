package controllers

import (
	"context"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/ws"

	"github.com/gofiber/fiber/v3"
)

func (s *Service) getAllWebSocketRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupWebSocket,
	}
}

func (s *Service) setupWebSocket(r fiber.Router) {
	r.Get("/ws", ws.Handler(s.Hub, s.AuthSession, s.ChatService, s.GameRoomService, s.ChatService, func(ctx context.Context) string {
		return s.SettingsService.Get(ctx, config.SettingBaseURL)
	}))
}
