package controllers

import (
	"context"
	"errors"

	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	secretsvc "umineko_city_of_books/internal/secret"
	"umineko_city_of_books/internal/upload"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllSecretRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupListSecrets,
		s.setupGetSecret,
		s.setupCreateSecretComment,
	}
}

func (s *Service) setupListSecrets(r fiber.Router) {
	r.Get("/secrets", s.optionalAuth(), s.listSecrets)
}

func (s *Service) setupGetSecret(r fiber.Router) {
	r.Get("/secrets/:id", s.optionalAuth(), s.getSecret)
}

func (s *Service) setupCreateSecretComment(r fiber.Router) {
	r.Post("/secrets/:id/comments", s.requireAuth(), s.requireEstablished(), s.createSecretComment)
}

func (s *Service) listSecrets(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	resp, err := s.SecretService.List(ctx.Context(), userID)
	if err != nil {
		return utils.InternalError(ctx, "failed to list secrets", err)
	}
	return ctx.JSON(resp)
}

func (s *Service) getSecret(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	userID := utils.UserID(ctx)
	resp, err := s.SecretService.Get(ctx.Context(), id, userID)
	if err != nil {
		if errors.Is(err, secretsvc.ErrNotFound) {
			return utils.NotFound(ctx, "secret not found")
		}
		return utils.InternalError(ctx, "failed to load secret", err)
	}
	return ctx.JSON(resp)
}

func (s *Service) createSecretComment(ctx fiber.Ctx) error {
	secretID := ctx.Params("id")
	userID := utils.UserID(ctx)

	return handleCreateComment(ctx, func(c context.Context, req dto.CreateCommentRequest, files []upload.FileUpload) (uuid.UUID, error) {
		return s.SecretService.CreateComment(c, secretID, userID, req, files)
	}, []errorResponse{
		{err: secretsvc.ErrNotFound, respond: utils.NotFound, message: "secret not found"},
	})
}
