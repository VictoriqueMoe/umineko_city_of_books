package controllers

import (
	"errors"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	secretsvc "umineko_city_of_books/internal/secret"

	"github.com/gofiber/fiber/v3"
)

func (s *Service) getAllSecretRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupListSecrets,
		s.setupGetSecret,
		s.setupCreateSecretComment,
		s.setupUpdateSecretComment,
		s.setupDeleteSecretComment,
		s.setupLikeSecretComment,
		s.setupUnlikeSecretComment,
		s.setupUploadSecretCommentMedia,
	}
}

func (s *Service) setupListSecrets(r fiber.Router) {
	r.Get("/secrets", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listSecrets)
}

func (s *Service) setupGetSecret(r fiber.Router) {
	r.Get("/secrets/:id", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.getSecret)
}

func (s *Service) setupCreateSecretComment(r fiber.Router) {
	r.Post("/secrets/:id/comments", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createSecretComment)
}

func (s *Service) setupUpdateSecretComment(r fiber.Router) {
	r.Put("/secret-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateSecretComment)
}

func (s *Service) setupDeleteSecretComment(r fiber.Router) {
	r.Delete("/secret-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteSecretComment)
}

func (s *Service) setupLikeSecretComment(r fiber.Router) {
	r.Post("/secret-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.likeSecretComment)
}

func (s *Service) setupUnlikeSecretComment(r fiber.Router) {
	r.Delete("/secret-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unlikeSecretComment)
}

func (s *Service) setupUploadSecretCommentMedia(r fiber.Router) {
	r.Post("/secret-comments/:id/media", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.uploadSecretCommentMedia)
}

func (s *Service) listSecrets(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	resp, err := s.SecretService.List(ctx.Context(), userID)
	if err != nil {
		return utils.InternalError(ctx, "failed to list secrets")
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
		return utils.InternalError(ctx, "failed to load secret")
	}
	return ctx.JSON(resp)
}

func (s *Service) createSecretComment(ctx fiber.Ctx) error {
	secretID := ctx.Params("id")
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.CreateSecretCommentRequest](ctx)
	if !ok {
		return nil
	}

	id, err := s.SecretService.CreateComment(ctx.Context(), secretID, userID, req)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, secretsvc.ErrNotFound) {
			return utils.NotFound(ctx, "secret not found")
		}
		if errors.Is(err, secretsvc.ErrEmptyBody) {
			return utils.BadRequest(ctx, err.Error())
		}
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		return utils.InternalError(ctx, "failed to create comment")
	}
	s.Hub.BumpSidebarActivity("secrets")
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateSecretComment(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.UpdateSecretCommentRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.SecretService.UpdateComment(ctx.Context(), id, userID, req); err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, secretsvc.ErrEmptyBody) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to update comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteSecretComment(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.SecretService.DeleteComment(ctx.Context(), id, userID); err != nil {
		return utils.InternalError(ctx, "failed to delete comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) likeSecretComment(ctx fiber.Ctx) error {
	commentID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.SecretService.LikeComment(ctx.Context(), userID, commentID); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		return utils.InternalError(ctx, "failed to like comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unlikeSecretComment(ctx fiber.Ctx) error {
	commentID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.SecretService.UnlikeComment(ctx.Context(), userID, commentID); err != nil {
		return utils.InternalError(ctx, "failed to unlike comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) uploadSecretCommentMedia(ctx fiber.Ctx) error {
	commentID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	file, err := ctx.FormFile("media")
	if err != nil {
		return utils.BadRequest(ctx, "no media file provided")
	}
	reader, err := file.Open()
	if err != nil {
		return utils.InternalError(ctx, "failed to read file")
	}
	defer reader.Close()

	result, err := s.SecretService.UploadCommentMedia(ctx.Context(), commentID, userID, file.Header.Get("Content-Type"), file.Size, reader)
	if err != nil {
		return utils.BadRequest(ctx, err.Error())
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}
