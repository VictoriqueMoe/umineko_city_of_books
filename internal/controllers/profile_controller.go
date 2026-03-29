package controllers

import (
	"errors"
	"strings"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/profile"
	"umineko_city_of_books/internal/upload"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllProfileRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupUpdateProfileRoute,
		s.setupUploadAvatarRoute,
		s.setupUploadBannerRoute,
		s.setupChangePasswordRoute,
		s.setupDeleteAccountRoute,
		s.setupGetOnlineStatusRoute,
		s.setupGetUserActivityRoute,
		s.setupGetProfileRoute,
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

func (s *Service) setupUploadBannerRoute(r fiber.Router) {
	r.Post("/auth/banner", middleware.RequireAuth(s.AuthSession), s.uploadBanner)
}

func (s *Service) setupChangePasswordRoute(r fiber.Router) {
	r.Put("/auth/password", middleware.RequireAuth(s.AuthSession), s.changePassword)
}

func (s *Service) setupDeleteAccountRoute(r fiber.Router) {
	r.Delete("/auth/account", middleware.RequireAuth(s.AuthSession), s.deleteAccount)
}

func (s *Service) setupGetOnlineStatusRoute(r fiber.Router) {
	r.Get("/users/online", s.getOnlineStatus)
}

func (s *Service) setupGetUserActivityRoute(r fiber.Router) {
	r.Get("/users/:username/activity", s.getUserActivity)
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

	result.Online = s.Hub.IsOnline(result.ID)

	return ctx.JSON(result)
}

func (s *Service) updateProfile(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)

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
	userID := ctx.Locals("userID").(uuid.UUID)

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

func (s *Service) uploadBanner(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)

	file, err := ctx.FormFile("banner")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "banner file is required",
		})
	}

	src, err := file.Open()
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to read file",
		})
	}
	defer src.Close()

	bannerURL, err := s.ProfileService.UploadBanner(
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
			"error": "failed to upload banner",
		})
	}

	return ctx.JSON(fiber.Map{"banner_url": bannerURL})
}

func (s *Service) changePassword(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.ChangePasswordRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if len(req.NewPassword) < 8 {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "new password must be at least 8 characters",
		})
	}

	if err := s.ProfileService.ChangePassword(ctx.Context(), userID, req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) deleteAccount(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.DeleteAccountRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := s.ProfileService.DeleteAccount(ctx.Context(), userID, req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	s.clearSessionCookie(ctx)

	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) getOnlineStatus(ctx fiber.Ctx) error {
	idsParam := ctx.Query("ids")
	if idsParam == "" {
		return ctx.JSON(fiber.Map{})
	}

	parts := strings.Split(idsParam, ",")
	result := make(map[string]bool)
	for _, p := range parts {
		idStr := strings.TrimSpace(p)
		if idStr == "" {
			continue
		}
		parsed, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		result[idStr] = s.Hub.IsOnline(parsed)
	}

	return ctx.JSON(result)
}

func (s *Service) getUserActivity(ctx fiber.Ctx) error {
	username := ctx.Params("username")
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.ProfileService.GetActivity(ctx.Context(), username, limit, offset)
	if err != nil {
		if errors.Is(err, profile.ErrUserNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "user not found",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get activity",
		})
	}

	return ctx.JSON(result)
}
