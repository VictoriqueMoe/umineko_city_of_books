package controllers

import (
	"umineko_city_of_books/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func (s *Service) getAllNotificationRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupListNotificationsRoute,
		s.setupMarkNotificationReadRoute,
		s.setupMarkAllNotificationsReadRoute,
		s.setupUnreadCountRoute,
	}
}

func (s *Service) setupListNotificationsRoute(r fiber.Router) {
	r.Get("/notifications", middleware.RequireAuth(s.AuthSession), s.listNotifications)
}

func (s *Service) setupMarkNotificationReadRoute(r fiber.Router) {
	r.Post("/notifications/:id<int>/read", middleware.RequireAuth(s.AuthSession), s.markNotificationRead)
}

func (s *Service) setupMarkAllNotificationsReadRoute(r fiber.Router) {
	r.Post("/notifications/read", middleware.RequireAuth(s.AuthSession), s.markAllNotificationsRead)
}

func (s *Service) setupUnreadCountRoute(r fiber.Router) {
	r.Get("/notifications/unread-count", middleware.RequireAuth(s.AuthSession), s.unreadCount)
}

func (s *Service) listNotifications(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(int)
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.NotificationService.List(ctx.Context(), userID, limit, offset)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list notifications",
		})
	}

	return ctx.JSON(result)
}

func (s *Service) markNotificationRead(ctx fiber.Ctx) error {
	id := fiber.Params[int](ctx, "id")
	userID := ctx.Locals("userID").(int)

	if err := s.NotificationService.MarkRead(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to mark notification as read",
		})
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) markAllNotificationsRead(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(int)

	if err := s.NotificationService.MarkAllRead(ctx.Context(), userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to mark all notifications as read",
		})
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) unreadCount(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(int)

	count, err := s.NotificationService.UnreadCount(ctx.Context(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get unread count",
		})
	}

	return ctx.JSON(fiber.Map{"count": count})
}
