package controllers

import (
	"github.com/gofiber/fiber/v3"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
)

func (s *Service) setupAdminListBannedWords(r fiber.Router) {
	r.Get("/admin/banned-words", s.requirePerm(authz.PermManageBannedWords), s.adminListBannedWords)
}

func (s *Service) setupAdminCreateBannedWord(r fiber.Router) {
	r.Post("/admin/banned-words", s.requirePerm(authz.PermManageBannedWords), s.adminCreateBannedWord)
}

func (s *Service) setupAdminUpdateBannedWord(r fiber.Router) {
	r.Put("/admin/banned-words/:ruleID", s.requirePerm(authz.PermManageBannedWords), s.adminUpdateBannedWord)
}

func (s *Service) setupAdminDeleteBannedWord(r fiber.Router) {
	r.Delete("/admin/banned-words/:ruleID", s.requirePerm(authz.PermManageBannedWords), s.adminDeleteBannedWord)
}

func (s *Service) adminListBannedWords(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	rules, err := s.ChatService.ListGlobalBannedWords(ctx.Context(), actorID)
	if err != nil {
		return mapBannedWordError(ctx, err)
	}
	return ctx.JSON(dto.BannedWordRuleListResponse{Rules: rules})
}

func (s *Service) adminCreateBannedWord(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	req, ok := utils.BindJSON[dto.CreateBannedWordRequest](ctx)
	if !ok {
		return nil
	}
	rule, err := s.ChatService.CreateGlobalBannedWord(ctx.Context(), actorID, req)
	if err != nil {
		return mapBannedWordError(ctx, err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(rule)
}

func (s *Service) adminUpdateBannedWord(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	ruleID, ok := utils.ParseIDParam(ctx, "ruleID")
	if !ok {
		return nil
	}
	req, ok := utils.BindJSON[dto.UpdateBannedWordRequest](ctx)
	if !ok {
		return nil
	}
	rule, err := s.ChatService.UpdateGlobalBannedWord(ctx.Context(), actorID, ruleID, req)
	if err != nil {
		return mapBannedWordError(ctx, err)
	}
	return ctx.JSON(rule)
}

func (s *Service) adminDeleteBannedWord(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	ruleID, ok := utils.ParseIDParam(ctx, "ruleID")
	if !ok {
		return nil
	}
	if err := s.ChatService.DeleteGlobalBannedWord(ctx.Context(), actorID, ruleID); err != nil {
		return mapBannedWordError(ctx, err)
	}
	return utils.OK(ctx)
}
