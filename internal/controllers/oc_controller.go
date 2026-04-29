package controllers

import (
	"errors"
	"strconv"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/oc"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllOCRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupListOCs,
		s.setupGetOC,
		s.setupCreateOC,
		s.setupUpdateOC,
		s.setupDeleteOC,
		s.setupUploadOCImage,
		s.setupAddOCGalleryImage,
		s.setupUpdateOCGalleryImage,
		s.setupDeleteOCGalleryImage,
		s.setupVoteOC,
		s.setupFavouriteOC,
		s.setupCreateOCComment,
		s.setupUpdateOCComment,
		s.setupDeleteOCComment,
		s.setupLikeOCComment,
		s.setupUnlikeOCComment,
		s.setupUploadOCCommentMedia,
		s.setupListUserOCs,
		s.setupListUserOCSummaries,
	}
}

func (s *Service) setupListOCs(r fiber.Router) {
	r.Get("/ocs", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listOCs)
}

func (s *Service) setupGetOC(r fiber.Router) {
	r.Get("/ocs/:id", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.getOC)
}

func (s *Service) setupCreateOC(r fiber.Router) {
	r.Post("/ocs", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createOC)
}

func (s *Service) setupUpdateOC(r fiber.Router) {
	r.Put("/ocs/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateOC)
}

func (s *Service) setupDeleteOC(r fiber.Router) {
	r.Delete("/ocs/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteOC)
}

func (s *Service) setupUploadOCImage(r fiber.Router) {
	r.Post("/ocs/:id/image", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.uploadOCImage)
}

func (s *Service) setupAddOCGalleryImage(r fiber.Router) {
	r.Post("/ocs/:id/gallery", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.addOCGalleryImage)
}

func (s *Service) setupUpdateOCGalleryImage(r fiber.Router) {
	r.Patch("/ocs/:id/gallery/:imageID", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateOCGalleryImage)
}

func (s *Service) setupDeleteOCGalleryImage(r fiber.Router) {
	r.Delete("/ocs/:id/gallery/:imageID", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteOCGalleryImage)
}

func (s *Service) setupVoteOC(r fiber.Router) {
	r.Post("/ocs/:id/vote", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.voteOC)
}

func (s *Service) setupFavouriteOC(r fiber.Router) {
	r.Post("/ocs/:id/favourite", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.favouriteOC)
}

func (s *Service) setupCreateOCComment(r fiber.Router) {
	r.Post("/ocs/:id/comments", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createOCComment)
}

func (s *Service) setupUpdateOCComment(r fiber.Router) {
	r.Put("/oc-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateOCComment)
}

func (s *Service) setupDeleteOCComment(r fiber.Router) {
	r.Delete("/oc-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteOCComment)
}

func (s *Service) setupLikeOCComment(r fiber.Router) {
	r.Post("/oc-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.likeOCComment)
}

func (s *Service) setupUnlikeOCComment(r fiber.Router) {
	r.Delete("/oc-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unlikeOCComment)
}

func (s *Service) setupUploadOCCommentMedia(r fiber.Router) {
	r.Post("/oc-comments/:id/media", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.uploadOCCommentMedia)
}

func (s *Service) setupListUserOCs(r fiber.Router) {
	r.Get("/users/:id/ocs", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listUserOCs)
}

func (s *Service) setupListUserOCSummaries(r fiber.Router) {
	r.Get("/users/:id/oc-summaries", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listUserOCSummaries)
}

func (s *Service) listOCs(ctx fiber.Ctx) error {
	viewerID := utils.UserID(ctx)
	sort := ctx.Query("sort", "new")
	series := ctx.Query("series")
	customSeriesName := ctx.Query("custom")
	crackOnly := ctx.Query("crack") == "true"
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	var ownerID uuid.UUID
	if rawOwner := ctx.Query("user_id"); rawOwner != "" {
		parsed, err := uuid.Parse(rawOwner)
		if err != nil {
			return utils.BadRequest(ctx, "invalid user_id")
		}
		ownerID = parsed
	}

	result, err := s.OCService.ListOCs(ctx.Context(), viewerID, sort, crackOnly, series, customSeriesName, ownerID, limit, offset)
	if err != nil {
		return utils.InternalError(ctx, "failed to list ocs")
	}
	return ctx.JSON(result)
}

func (s *Service) getOC(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	viewerID := utils.UserID(ctx)
	result, err := s.OCService.GetOC(ctx.Context(), id, viewerID)
	if err != nil {
		if errors.Is(err, oc.ErrNotFound) {
			return utils.NotFound(ctx, "oc not found")
		}
		return utils.InternalError(ctx, "failed to get oc")
	}
	return ctx.JSON(result)
}

func (s *Service) createOC(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	req, ok := utils.BindJSON[dto.CreateOCRequest](ctx)
	if !ok {
		return nil
	}

	id, err := s.OCService.CreateOC(ctx.Context(), userID, req)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, oc.ErrEmptyName) || errors.Is(err, oc.ErrInvalidSeries) || errors.Is(err, oc.ErrEmptyCustomSeries) || errors.Is(err, oc.ErrDuplicateName) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to create oc")
	}
	s.Hub.BumpSidebarActivity("ocs")
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateOC(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.UpdateOCRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.OCService.UpdateOC(ctx.Context(), id, userID, req); err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, oc.ErrEmptyName) || errors.Is(err, oc.ErrInvalidSeries) || errors.Is(err, oc.ErrEmptyCustomSeries) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to update oc")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteOC(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.OCService.DeleteOC(ctx.Context(), id, userID); err != nil {
		return utils.InternalError(ctx, "failed to delete oc")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) uploadOCImage(ctx fiber.Ctx) error {
	ocID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	file, err := ctx.FormFile("image")
	if err != nil {
		return utils.BadRequest(ctx, "no image file provided")
	}
	reader, err := file.Open()
	if err != nil {
		return utils.InternalError(ctx, "failed to read file")
	}
	defer reader.Close()

	url, err := s.OCService.UploadOCImage(ctx.Context(), ocID, userID, file.Header.Get("Content-Type"), file.Size, reader)
	if err != nil {
		return utils.BadRequest(ctx, err.Error())
	}
	return ctx.JSON(fiber.Map{"image_url": url})
}

func (s *Service) addOCGalleryImage(ctx fiber.Ctx) error {
	ocID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	file, err := ctx.FormFile("image")
	if err != nil {
		return utils.BadRequest(ctx, "no image file provided")
	}
	reader, err := file.Open()
	if err != nil {
		return utils.InternalError(ctx, "failed to read file")
	}
	defer reader.Close()

	caption := ctx.FormValue("caption")
	result, err := s.OCService.AddGalleryImage(ctx.Context(), ocID, userID, caption, file.Header.Get("Content-Type"), file.Size, reader)
	if err != nil {
		return utils.BadRequest(ctx, err.Error())
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}

func (s *Service) updateOCGalleryImage(ctx fiber.Ctx) error {
	ocID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	imageID, err := strconv.ParseInt(ctx.Params("imageID"), 10, 64)
	if err != nil {
		return utils.BadRequest(ctx, "invalid imageID")
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.UpdateOCImageRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.OCService.UpdateGalleryImage(ctx.Context(), ocID, imageID, userID, req); err != nil {
		return utils.BadRequest(ctx, err.Error())
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteOCGalleryImage(ctx fiber.Ctx) error {
	ocID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	imageID, err := strconv.ParseInt(ctx.Params("imageID"), 10, 64)
	if err != nil {
		return utils.BadRequest(ctx, "invalid imageID")
	}
	userID := utils.UserID(ctx)

	if err := s.OCService.DeleteGalleryImage(ctx.Context(), ocID, imageID, userID); err != nil {
		return utils.BadRequest(ctx, err.Error())
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) voteOC(ctx fiber.Ctx) error {
	ocID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.VoteRequest](ctx)
	if !ok {
		return nil
	}
	if req.Value != 1 && req.Value != -1 && req.Value != 0 {
		return utils.BadRequest(ctx, "value must be 1, -1, or 0")
	}

	if err := s.OCService.Vote(ctx.Context(), userID, ocID, req.Value); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		return utils.InternalError(ctx, "failed to vote")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) favouriteOC(ctx fiber.Ctx) error {
	ocID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	favourited, err := s.OCService.ToggleFavourite(ctx.Context(), userID, ocID)
	if err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		if errors.Is(err, oc.ErrNotFound) {
			return utils.NotFound(ctx, "oc not found")
		}
		return utils.InternalError(ctx, "failed to favourite oc")
	}
	return ctx.JSON(fiber.Map{"favourited": favourited})
}

func (s *Service) createOCComment(ctx fiber.Ctx) error {
	ocID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.CreateCommentRequest](ctx)
	if !ok {
		return nil
	}

	id, err := s.OCService.CreateComment(ctx.Context(), ocID, userID, req)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		if errors.Is(err, oc.ErrEmptyBody) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to create comment")
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateOCComment(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.UpdateCommentRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.OCService.UpdateComment(ctx.Context(), id, userID, req); err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, oc.ErrEmptyBody) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to update comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteOCComment(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.OCService.DeleteComment(ctx.Context(), id, userID); err != nil {
		return utils.InternalError(ctx, "failed to delete comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) likeOCComment(ctx fiber.Ctx) error {
	commentID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.OCService.LikeComment(ctx.Context(), userID, commentID); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		return utils.InternalError(ctx, "failed to like comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unlikeOCComment(ctx fiber.Ctx) error {
	commentID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.OCService.UnlikeComment(ctx.Context(), userID, commentID); err != nil {
		return utils.InternalError(ctx, "failed to unlike comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) uploadOCCommentMedia(ctx fiber.Ctx) error {
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

	result, err := s.OCService.UploadCommentMedia(ctx.Context(), commentID, userID, file.Header.Get("Content-Type"), file.Size, reader)
	if err != nil {
		return utils.BadRequest(ctx, err.Error())
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}

func (s *Service) listUserOCs(ctx fiber.Ctx) error {
	userID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	viewerID := utils.UserID(ctx)
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.OCService.ListOCsByUser(ctx.Context(), userID, viewerID, limit, offset)
	if err != nil {
		return utils.InternalError(ctx, "failed to list user ocs")
	}
	return ctx.JSON(result)
}

func (s *Service) listUserOCSummaries(ctx fiber.Ctx) error {
	userID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	result, err := s.OCService.ListOCSummariesByUser(ctx.Context(), userID)
	if err != nil {
		return utils.InternalError(ctx, "failed to list user oc summaries")
	}
	if result == nil {
		result = []dto.OCSummary{}
	}
	return ctx.JSON(result)
}
