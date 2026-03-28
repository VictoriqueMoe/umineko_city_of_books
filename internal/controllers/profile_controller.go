package controllers

import (
	"errors"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/profile"
	"umineko_city_of_books/internal/upload"

	"github.com/gofiber/fiber/v3"
)

func (s *Service) getAllProfileRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupGetProfileRoute,
		s.setupUpdateProfileRoute,
		s.setupUploadAvatarRoute,
	}
}

func (s *Service) setupGetProfileRoute(r fiber.Router) {
	r.Get("/users/:username", s.getProfile)
}

func (s *Service) setupUpdateProfileRoute(r fiber.Router) {
	r.Put("/auth/profile", middleware.RequireAuth(s.AuthSession), s.updateProfile)
}

func (s *Service) setupUploadAvatarRoute(r fiber.Router) {
	r.Post("/auth/avatar", middleware.RequireAuth(s.AuthSession), s.uploadAvatar)
}

func (s *Service) getProfile(ctx fiber.Ctx) error {
	username := ctx.Params("username")

	result, err := s.ProfileService.GetProfile(ctx.Context(), username)
	if err != nil {
		if errors.Is(err, profile.ErrUserNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "user not found",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get profile",
		})
	}

	return ctx.JSON(result)
}

func (s *Service) updateProfile(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(int)

	var req dto.UpdateProfileRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.DisplayName == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "display name is required",
		})
	}

	if err := s.ProfileService.UpdateProfile(ctx.Context(), userID, req); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update profile",
		})
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) uploadAvatar(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(int)

	file, err := ctx.FormFile("avatar")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "avatar file is required",
		})
	}

	src, err := file.Open()
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to read file",
		})
	}
	defer src.Close()

	avatarURL, err := s.ProfileService.UploadAvatar(
		ctx.Context(),
		userID,
		file.Header.Get("Content-Type"),
		file.Size,
		src,
	)
	if err != nil {
		if errors.Is(err, upload.ErrFileTooLarge) || errors.Is(err, upload.ErrInvalidFileType) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to upload avatar",
		})
	}

	return ctx.JSON(fiber.Map{"avatar_url": avatarURL})
}
