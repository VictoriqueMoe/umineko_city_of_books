package controllers

import (
	"errors"
	"strconv"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
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
	r.Get("/ocs", s.optionalAuth(), s.listOCs)
}

func (s *Service) setupGetOC(r fiber.Router) {
	r.Get("/ocs/:id", s.optionalAuth(), s.getOC)
}

func (s *Service) setupCreateOC(r fiber.Router) {
	r.Post("/ocs", s.requireAuth(), s.createOC)
}

func (s *Service) setupUpdateOC(r fiber.Router) {
	r.Put("/ocs/:id", s.requireAuth(), s.updateOC)
}

func (s *Service) setupDeleteOC(r fiber.Router) {
	r.Delete("/ocs/:id", s.requireAuth(), s.deleteOC)
}

func (s *Service) setupUploadOCImage(r fiber.Router) {
	r.Post("/ocs/:id/image", s.requireAuth(), s.requireEstablished(), s.uploadOCImage)
}

func (s *Service) setupAddOCGalleryImage(r fiber.Router) {
	r.Post("/ocs/:id/gallery", s.requireAuth(), s.addOCGalleryImage)
}

func (s *Service) setupUpdateOCGalleryImage(r fiber.Router) {
	r.Patch("/ocs/:id/gallery/:imageID", s.requireAuth(), s.updateOCGalleryImage)
}

func (s *Service) setupDeleteOCGalleryImage(r fiber.Router) {
	r.Delete("/ocs/:id/gallery/:imageID", s.requireAuth(), s.deleteOCGalleryImage)
}

func (s *Service) setupVoteOC(r fiber.Router) {
	r.Post("/ocs/:id/vote", s.requireAuth(), s.voteOC)
}

func (s *Service) setupFavouriteOC(r fiber.Router) {
	r.Post("/ocs/:id/favourite", s.requireAuth(), s.favouriteOC)
}

func (s *Service) setupCreateOCComment(r fiber.Router) {
	r.Post("/ocs/:id/comments", s.requireAuth(), s.createOCComment)
}

func (s *Service) setupUpdateOCComment(r fiber.Router) {
	r.Put("/oc-comments/:id", s.requireAuth(), s.updateOCComment)
}

func (s *Service) setupDeleteOCComment(r fiber.Router) {
	r.Delete("/oc-comments/:id", s.requireAuth(), s.deleteOCComment)
}

func (s *Service) setupLikeOCComment(r fiber.Router) {
	r.Post("/oc-comments/:id/like", s.requireAuth(), s.likeOCComment)
}

func (s *Service) setupUnlikeOCComment(r fiber.Router) {
	r.Delete("/oc-comments/:id/like", s.requireAuth(), s.unlikeOCComment)
}

func (s *Service) setupUploadOCCommentMedia(r fiber.Router) {
	r.Post("/oc-comments/:id/media", s.requireAuth(), s.requireEstablished(), s.uploadOCCommentMedia)
}

func (s *Service) setupListUserOCs(r fiber.Router) {
	r.Get("/users/:id/ocs", s.optionalAuth(), s.listUserOCs)
}

func (s *Service) setupListUserOCSummaries(r fiber.Router) {
	r.Get("/users/:id/oc-summaries", s.optionalAuth(), s.listUserOCSummaries)
}

func (s *Service) listOCs(ctx fiber.Ctx) error {
	viewerID := utils.UserID(ctx)
	sort := ctx.Query("sort", "new")
	series := ctx.Query("series")
	customSeriesName := ctx.Query("custom")
	crackOnly := ctx.Query("crack") == "true"
	page := utils.Page(ctx, 20)

	var ownerID uuid.UUID
	if rawOwner := ctx.Query("user_id"); rawOwner != "" {
		parsed, err := uuid.Parse(rawOwner)
		if err != nil {
			return utils.BadRequest(ctx, "invalid user_id")
		}
		ownerID = parsed
	}

	result, err := s.OCService.ListOCs(ctx.Context(), viewerID, sort, crackOnly, series, customSeriesName, ownerID, page)
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
	return s.handleDeleteComment(ctx, s.OCService.DeleteComment)
}

func (s *Service) likeOCComment(ctx fiber.Ctx) error {
	return s.handleLikeComment(ctx, s.OCService.LikeComment)
}

func (s *Service) unlikeOCComment(ctx fiber.Ctx) error {
	return s.handleUnlikeComment(ctx, s.OCService.UnlikeComment)
}

func (s *Service) uploadOCCommentMedia(ctx fiber.Ctx) error {
	return handleUploadCommentMedia(ctx, s.OCService.UploadCommentMedia)
}

func (s *Service) listUserOCs(ctx fiber.Ctx) error {
	userID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	viewerID := utils.UserID(ctx)
	page := utils.Page(ctx, 20)

	result, err := s.OCService.ListOCsByUser(ctx.Context(), userID, viewerID, page)
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
