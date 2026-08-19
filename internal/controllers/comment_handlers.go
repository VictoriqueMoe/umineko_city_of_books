package controllers

import (
	"context"
	"errors"
	"io"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) handleDeleteComment(ctx fiber.Ctx, del func(context.Context, uuid.UUID, uuid.UUID) error) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	userID := utils.UserID(ctx)
	if err := del(ctx.Context(), id, userID); err != nil {
		return utils.InternalError(ctx, "failed to delete comment")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) handleLikeComment(ctx fiber.Ctx, like func(context.Context, uuid.UUID, uuid.UUID) error) error {
	commentID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	userID := utils.UserID(ctx)
	if err := like(ctx.Context(), userID, commentID); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}

		return utils.InternalError(ctx, "failed to like comment")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) handleUnlikeComment(ctx fiber.Ctx, unlike func(context.Context, uuid.UUID, uuid.UUID) error) error {
	commentID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	userID := utils.UserID(ctx)
	if err := unlike(ctx.Context(), userID, commentID); err != nil {
		return utils.InternalError(ctx, "failed to unlike comment")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func handleUploadCommentMedia[T any](ctx fiber.Ctx, upload func(context.Context, uuid.UUID, uuid.UUID, string, string, int64, io.Reader) (T, error)) error {
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

	result, err := upload(ctx.Context(), commentID, userID, file.Header.Get("Content-Type"), file.Filename, file.Size, reader)
	if err != nil {
		return utils.BadRequest(ctx, err.Error())
	}

	return ctx.Status(fiber.StatusCreated).JSON(result)
}
