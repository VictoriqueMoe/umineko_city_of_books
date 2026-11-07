package controllers

import (
	"context"
	"strings"

	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/upload"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const (
	uploadFileField = "file"
)

func (s *Service) getAllPendingUploadRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupBeginMediaUpload,
		s.setupBeginFileUpload,
		s.setupUploadStatuses,
		s.setupRemoveUpload,
	}
}

func (s *Service) setupBeginMediaUpload(r fiber.Router) {
	r.Post("/uploads/media", s.requireAuth(), s.requireEstablished(), s.beginUpload(s.UploadService.BeginMediaUpload))
}

func (s *Service) setupBeginFileUpload(r fiber.Router) {
	r.Post("/uploads/files", s.requireAuth(), s.requireEstablished(), s.beginUpload(s.UploadService.BeginFileUpload))
}

func (s *Service) setupUploadStatuses(r fiber.Router) {
	r.Get("/uploads", s.requireAuth(), s.uploadStatuses)
}

func (s *Service) setupRemoveUpload(r fiber.Router) {
	r.Delete("/uploads/:id", s.requireAuth(), s.removeUpload)
}

func (s *Service) beginUpload(begin func(context.Context, uuid.UUID, upload.FileUpload) (*dto.PendingUploadResponse, error)) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		file, err := utils.ReceiveFile(ctx, s.UploadService, uploadFileField)
		if err != nil {
			return utils.UploadFailed(ctx, err)
		}
		defer file.Close()

		result, err := begin(ctx.Context(), utils.UserID(ctx), file.FileUpload())
		if err != nil {
			if utils.IsUploadRejection(err) {
				return utils.UploadRejected(ctx, err)
			}

			return utils.InternalError(ctx, "failed to accept the upload", err)
		}

		return ctx.Status(fiber.StatusAccepted).JSON(result)
	}
}

func (s *Service) uploadStatuses(ctx fiber.Ctx) error {
	var ids []uuid.UUID
	for raw := range strings.SplitSeq(ctx.Query("ids"), ",") {
		if raw == "" {
			continue
		}

		id, err := uuid.Parse(raw)
		if err != nil {
			return utils.BadRequest(ctx, "invalid upload id")
		}

		ids = append(ids, id)
	}

	statuses, err := s.UploadService.UploadStatuses(ctx.Context(), utils.UserID(ctx), ids)
	if err != nil {
		return utils.InternalError(ctx, "failed to load upload statuses", err)
	}

	return ctx.JSON(fiber.Map{"uploads": statuses})
}

func (s *Service) removeUpload(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	if err := s.UploadService.RemoveUpload(ctx.Context(), utils.UserID(ctx), id); err != nil {
		return utils.InternalError(ctx, "failed to remove the upload", err)
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}
