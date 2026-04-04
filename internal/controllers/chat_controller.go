package controllers

import (
	"errors"

	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllChatRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupCreateDMRoute,
		s.setupCreateGroupRoomRoute,
		s.setupListRoomsRoute,
		s.setupGetMessagesRoute,
		s.setupSendMessageRoute,
		s.setupDeleteChatRoute,
	}
}

func (s *Service) setupCreateDMRoute(r fiber.Router) {
	r.Post("/chat/dm", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createDM)
}

func (s *Service) setupCreateGroupRoomRoute(r fiber.Router) {
	r.Post("/chat/rooms", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createGroupRoom)
}

func (s *Service) setupListRoomsRoute(r fiber.Router) {
	r.Get("/chat/rooms", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.listRooms)
}

func (s *Service) setupGetMessagesRoute(r fiber.Router) {
	r.Get("/chat/rooms/:roomID/messages", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.getMessages)
}

func (s *Service) createDM(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.CreateDMRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	room, err := s.ChatService.GetOrCreateDMRoom(ctx.Context(), userID, req)
	if err != nil {
		if errors.Is(err, chat.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you cannot message this user",
			})
		}
		if errors.Is(err, chat.ErrDmsDisabled) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "recipient has DMs disabled",
			})
		}
		if errors.Is(err, chat.ErrUserNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "user not found",
			})
		}
		if errors.Is(err, chat.ErrCannotDMSelf) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "cannot create DM with yourself",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create DM room",
		})
	}

	return ctx.JSON(room)
}

func (s *Service) createGroupRoom(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.CreateGroupRoomRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	room, err := s.ChatService.CreateGroupRoom(ctx.Context(), userID, req)
	if err != nil {
		if errors.Is(err, chat.ErrMissingFields) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "room name is required",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create group room",
		})
	}

	return ctx.Status(fiber.StatusCreated).JSON(room)
}

func (s *Service) listRooms(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)

	resp, err := s.ChatService.ListRooms(ctx.Context(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list rooms",
		})
	}

	return ctx.JSON(resp)
}

func (s *Service) setupSendMessageRoute(r fiber.Router) {
	r.Post("/chat/rooms/:roomID/messages", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.sendMessage)
}

func (s *Service) sendMessage(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)

	roomID, err := uuid.Parse(ctx.Params("roomID"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid room ID",
		})
	}

	var req dto.SendMessageRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	resp, err := s.ChatService.SendMessage(ctx.Context(), userID, roomID, req)
	if err != nil {
		if errors.Is(err, chat.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you cannot message this user",
			})
		}
		if errors.Is(err, chat.ErrMissingFields) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "message body is required",
			})
		}
		if errors.Is(err, chat.ErrNotMember) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you are not a member of this room",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to send message",
		})
	}

	return ctx.Status(fiber.StatusCreated).JSON(resp)
}

func (s *Service) getMessages(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)

	roomID, err := uuid.Parse(ctx.Params("roomID"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid room ID",
		})
	}

	limit := fiber.Query[int](ctx, "limit", 50)
	offset := fiber.Query[int](ctx, "offset", 0)

	resp, err := s.ChatService.GetMessages(ctx.Context(), userID, roomID, limit, offset)
	if err != nil {
		if errors.Is(err, chat.ErrNotMember) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you are not a member of this room",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get messages",
		})
	}

	return ctx.JSON(resp)
}

func (s *Service) setupDeleteChatRoute(r fiber.Router) {
	r.Delete("/chat/rooms/:roomID", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteChat)
}

func (s *Service) deleteChat(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)

	roomID, err := uuid.Parse(ctx.Params("roomID"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid room ID",
		})
	}

	if err := s.ChatService.DeleteChat(ctx.Context(), roomID, userID); err != nil {
		if errors.Is(err, chat.ErrNotMember) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you are not a member of this chat",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete chat",
		})
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}
