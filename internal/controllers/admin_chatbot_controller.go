package controllers

import (
	"errors"
	"strconv"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/chatbot"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/openai"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllAdminChatbotRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupAdminListChatbots,
		s.setupAdminCreateChatbot,
		s.setupAdminUpdateChatbot,
		s.setupAdminDeleteChatbot,
		s.setupAdminChatbotUsage,
		s.setupAdminChatbotModels,
		s.setupAdminChatbotTest,
	}
}

func (s *Service) setupAdminListChatbots(r fiber.Router) {
	r.Get("/admin/chatbots", s.requirePerm(authz.PermManageSettings), s.adminListChatbots)
}

func (s *Service) setupAdminCreateChatbot(r fiber.Router) {
	r.Post("/admin/chatbots", s.requirePerm(authz.PermManageSettings), s.adminCreateChatbot)
}

func (s *Service) setupAdminUpdateChatbot(r fiber.Router) {
	r.Put("/admin/chatbots/:id", s.requirePerm(authz.PermManageSettings), s.adminUpdateChatbot)
}

func (s *Service) setupAdminDeleteChatbot(r fiber.Router) {
	r.Delete("/admin/chatbots/:id", s.requirePerm(authz.PermManageSettings), s.adminDeleteChatbot)
}

func (s *Service) setupAdminChatbotUsage(r fiber.Router) {
	r.Get("/admin/chatbots/usage", s.requirePerm(authz.PermManageSettings), s.adminChatbotUsage)
}

func (s *Service) setupAdminChatbotModels(r fiber.Router) {
	r.Get("/admin/chatbots/models", s.requirePerm(authz.PermManageSettings), s.adminChatbotModels)
}

func (s *Service) setupAdminChatbotTest(r fiber.Router) {
	r.Post("/admin/chatbots/test", s.requirePerm(authz.PermManageSettings), s.adminChatbotTest)
}

func (s *Service) getAllChatbotRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupListChatbots,
	}
}

func (s *Service) setupListChatbots(r fiber.Router) {
	r.Get("/chatbots", s.listChatbots)
}

func (s *Service) listChatbots(ctx fiber.Ctx) error {
	return ctx.JSON(dto.ChatbotListResponse{Chatbots: s.ChatbotService.Listing()})
}

func (s *Service) adminListChatbots(ctx fiber.Ctx) error {
	bots, err := s.ChatbotAdminService.List(ctx.Context())
	if err != nil {
		return utils.InternalError(ctx, err.Error())
	}

	return ctx.JSON(bots)
}

func (s *Service) adminCreateChatbot(ctx fiber.Ctx) error {
	var req dto.ChatbotUpsertRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request body")
	}

	bot, err := s.ChatbotAdminService.Create(ctx.Context(), req)
	if err != nil {
		return handleChatbotError(ctx, err)
	}

	return ctx.Status(fiber.StatusCreated).JSON(bot)
}

func (s *Service) adminUpdateChatbot(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return utils.BadRequest(ctx, "invalid chatbot id")
	}

	var req dto.ChatbotUpsertRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request body")
	}

	bot, updateErr := s.ChatbotAdminService.Update(ctx.Context(), id, req)
	if updateErr != nil {
		return handleChatbotError(ctx, updateErr)
	}

	return ctx.JSON(bot)
}

func (s *Service) adminDeleteChatbot(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return utils.BadRequest(ctx, "invalid chatbot id")
	}

	if err := s.ChatbotAdminService.Delete(ctx.Context(), id); err != nil {
		return handleChatbotError(ctx, err)
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) adminChatbotUsage(ctx fiber.Ctx) error {
	days, err := strconv.Atoi(ctx.Query("days", "7"))
	if err != nil || days <= 0 || days > 180 {
		days = 7
	}

	since := time.Now().AddDate(0, 0, -days)

	usage, usageErr := s.ChatbotAdminService.Usage(ctx.Context(), since)
	if usageErr != nil {
		return utils.InternalError(ctx, usageErr.Error())
	}

	return ctx.JSON(usage)
}

func (s *Service) adminChatbotModels(ctx fiber.Ctx) error {
	models, err := s.ChatbotAdminService.Models(ctx.Context())
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to list openai models for the admin panel")

		return ctx.JSON(dto.ChatbotModelsResponse{Models: []string{}, Error: openai.Reason(err)})
	}

	if models == nil {
		models = []string{}
	}

	return ctx.JSON(dto.ChatbotModelsResponse{Models: models})
}

func (s *Service) adminChatbotTest(ctx fiber.Ctx) error {
	var req dto.ChatbotTestRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request body")
	}

	ok, message, err := s.ChatbotAdminService.Test(ctx.Context(), req.Model)
	if err != nil {
		return ctx.JSON(dto.ChatbotTestResponse{OK: false, Error: err.Error()})
	}

	return ctx.JSON(dto.ChatbotTestResponse{OK: ok, Error: message})
}

func handleChatbotError(ctx fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, chatbot.ErrBotNotFound):
		return utils.NotFound(ctx, "chatbot not found")
	case errors.Is(err, chatbot.ErrBotUsernameUsed):
		return utils.BadRequest(ctx, "that username is already taken")
	case errors.Is(err, chatbot.ErrBotInvalid), errors.Is(err, chatbot.ErrBotUnknownModel):
		return utils.BadRequest(ctx, err.Error())
	default:
		return utils.InternalError(ctx, err.Error())
	}
}
