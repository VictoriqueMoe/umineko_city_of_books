package controllers

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	artsvc "umineko_city_of_books/internal/art"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllArtRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupListArt,
		s.setupGetArtCornerCounts,
		s.setupGetPopularTags,
		s.setupGetArt,
		s.setupCreateArt,
		s.setupUpdateArt,
		s.setupDeleteArt,
		s.setupLikeArt,
		s.setupUnlikeArt,
		s.setupCreateArtComment,
		s.setupUpdateArtComment,
		s.setupDeleteArtComment,
		s.setupLikeArtComment,
		s.setupUnlikeArtComment,
		s.setupUploadArtCommentMedia,
		s.setupListUserArt,
		s.setupCreateGallery,
		s.setupListAllGalleries,
		s.setupUpdateGallery,
		s.setupSetGalleryCover,
		s.setupDeleteGallery,
		s.setupGetGallery,
		s.setupListUserGalleries,
		s.setupSetArtGallery,
	}
}

func (s *Service) setupListArt(r fiber.Router) {
	r.Get("/art", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listArt)
}

func (s *Service) setupGetArtCornerCounts(r fiber.Router) {
	r.Get("/art/corner-counts", s.getArtCornerCounts)
}

func (s *Service) setupGetPopularTags(r fiber.Router) {
	r.Get("/art/tags", s.getPopularTags)
}

func (s *Service) setupGetArt(r fiber.Router) {
	r.Get("/art/:id", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.getArt)
}

func (s *Service) setupCreateArt(r fiber.Router) {
	r.Post("/art", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createArt)
}

func (s *Service) setupUpdateArt(r fiber.Router) {
	r.Put("/art/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateArt)
}

func (s *Service) setupDeleteArt(r fiber.Router) {
	r.Delete("/art/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteArt)
}

func (s *Service) setupLikeArt(r fiber.Router) {
	r.Post("/art/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.likeArt)
}

func (s *Service) setupUnlikeArt(r fiber.Router) {
	r.Delete("/art/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unlikeArt)
}

func (s *Service) setupCreateArtComment(r fiber.Router) {
	r.Post("/art/:id/comments", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createArtComment)
}

func (s *Service) setupUpdateArtComment(r fiber.Router) {
	r.Put("/art-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateArtComment)
}

func (s *Service) setupDeleteArtComment(r fiber.Router) {
	r.Delete("/art-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteArtComment)
}

func (s *Service) setupLikeArtComment(r fiber.Router) {
	r.Post("/art-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.likeArtComment)
}

func (s *Service) setupUnlikeArtComment(r fiber.Router) {
	r.Delete("/art-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unlikeArtComment)
}

func (s *Service) setupUploadArtCommentMedia(r fiber.Router) {
	r.Post("/art-comments/:id/media", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.uploadArtCommentMedia)
}

func (s *Service) setupListUserArt(r fiber.Router) {
	r.Get("/users/:id/art", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listUserArt)
}

func artViewerHash(ctx fiber.Ctx) string {
	userID, ok := ctx.Locals("userID").(uuid.UUID)
	var raw string
	if ok && userID != uuid.Nil {
		raw = userID.String()
	} else {
		raw = ctx.IP()
	}
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h[:16])
}

func (s *Service) listArt(ctx fiber.Ctx) error {
	viewerID := utils.UserID(ctx)
	corner := ctx.Query("corner", "general")
	artType := ctx.Query("type")
	search := ctx.Query("search")
	tag := ctx.Query("tag")
	sort := ctx.Query("sort", "new")
	limit := fiber.Query[int](ctx, "limit", 24)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.ArtService.ListArt(ctx.Context(), viewerID, corner, artType, search, tag, sort, limit, offset)
	if err != nil {
		return utils.InternalError(ctx, "failed to list art")
	}
	return ctx.JSON(result)
}

func (s *Service) getArtCornerCounts(ctx fiber.Ctx) error {
	counts, err := s.ArtService.GetCornerCounts(ctx.Context())
	if err != nil {
		return utils.InternalError(ctx, "failed to get art counts")
	}
	return ctx.JSON(counts)
}

func (s *Service) getPopularTags(ctx fiber.Ctx) error {
	corner := ctx.Query("corner")
	tags, err := s.ArtService.GetPopularTags(ctx.Context(), corner)
	if err != nil {
		return utils.InternalError(ctx, "failed to get tags")
	}
	return ctx.JSON(tags)
}

func (s *Service) getArt(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	viewerID := utils.UserID(ctx)
	result, err := s.ArtService.GetArt(ctx.Context(), id, viewerID, artViewerHash(ctx))
	if err != nil {
		if errors.Is(err, artsvc.ErrNotFound) {
			return utils.NotFound(ctx, "art not found")
		}
		return utils.InternalError(ctx, "failed to get art")
	}
	return ctx.JSON(result)
}

func (s *Service) createArt(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	metadataStr := ctx.FormValue("metadata")
	if metadataStr == "" {
		return utils.BadRequest(ctx, "metadata is required")
	}

	var req dto.CreateArtRequest
	if err := json.Unmarshal([]byte(metadataStr), &req); err != nil {
		return utils.BadRequest(ctx, "invalid metadata")
	}

	file, err := ctx.FormFile("image")
	if err != nil {
		return utils.BadRequest(ctx, "image file is required")
	}

	reader, err := file.Open()
	if err != nil {
		return utils.InternalError(ctx, "failed to read file")
	}
	defer reader.Close()

	id, err := s.ArtService.CreateArt(ctx.Context(), userID, req, file.Header.Get("Content-Type"), file.Size, reader)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, artsvc.ErrEmptyTitle) {
			return utils.BadRequest(ctx, err.Error())
		}
		if errors.Is(err, artsvc.ErrRateLimited) {
			return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": err.Error()})
		}
		return utils.BadRequest(ctx, err.Error())
	}
	corner := req.Corner
	if corner == "" {
		corner = "general"
	}
	s.Hub.BumpSidebarActivity("gallery_" + corner)
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateArt(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	userID := utils.UserID(ctx)
	req, ok := utils.BindJSON[dto.UpdateArtRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.ArtService.UpdateArt(ctx.Context(), id, userID, req); err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, artsvc.ErrEmptyTitle) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to update art")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteArt(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	userID := utils.UserID(ctx)
	if err := s.ArtService.DeleteArt(ctx.Context(), id, userID); err != nil {
		return utils.InternalError(ctx, "failed to delete art")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) likeArt(ctx fiber.Ctx) error {
	artID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	userID := utils.UserID(ctx)
	if err := s.ArtService.LikeArt(ctx.Context(), userID, artID); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		return utils.InternalError(ctx, "failed to like art")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unlikeArt(ctx fiber.Ctx) error {
	artID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	userID := utils.UserID(ctx)
	if err := s.ArtService.UnlikeArt(ctx.Context(), userID, artID); err != nil {
		return utils.InternalError(ctx, "failed to unlike art")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) createArtComment(ctx fiber.Ctx) error {
	artID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	userID := utils.UserID(ctx)
	req, ok := utils.BindJSON[dto.CreateCommentRequest](ctx)
	if !ok {
		return nil
	}

	id, err := s.ArtService.CreateComment(ctx.Context(), artID, userID, req)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		return utils.InternalError(ctx, "failed to create comment")
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateArtComment(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	userID := utils.UserID(ctx)
	req, ok := utils.BindJSON[dto.UpdateCommentRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.ArtService.UpdateComment(ctx.Context(), id, userID, req); err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		return utils.InternalError(ctx, "failed to update comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteArtComment(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	userID := utils.UserID(ctx)
	if err := s.ArtService.DeleteComment(ctx.Context(), id, userID); err != nil {
		return utils.InternalError(ctx, "failed to delete comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) likeArtComment(ctx fiber.Ctx) error {
	commentID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	userID := utils.UserID(ctx)
	if err := s.ArtService.LikeComment(ctx.Context(), userID, commentID); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		return utils.InternalError(ctx, "failed to like comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unlikeArtComment(ctx fiber.Ctx) error {
	commentID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	userID := utils.UserID(ctx)
	if err := s.ArtService.UnlikeComment(ctx.Context(), userID, commentID); err != nil {
		return utils.InternalError(ctx, "failed to unlike comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) uploadArtCommentMedia(ctx fiber.Ctx) error {
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

	result, err := s.ArtService.UploadCommentMedia(ctx.Context(), commentID, userID, file.Header.Get("Content-Type"), file.Size, reader)
	if err != nil {
		return utils.BadRequest(ctx, err.Error())
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}

func (s *Service) listUserArt(ctx fiber.Ctx) error {
	userID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	viewerID := utils.UserID(ctx)
	limit := fiber.Query[int](ctx, "limit", 24)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.ArtService.ListByUser(ctx.Context(), userID, viewerID, limit, offset)
	if err != nil {
		return utils.InternalError(ctx, "failed to list user art")
	}
	return ctx.JSON(result)
}

func (s *Service) setupCreateGallery(r fiber.Router) {
	r.Post("/galleries", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createGallery)
}

func (s *Service) setupListAllGalleries(r fiber.Router) {
	r.Get("/galleries", s.listAllGalleries)
}

func (s *Service) setupUpdateGallery(r fiber.Router) {
	r.Put("/galleries/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateGallery)
}

func (s *Service) setupSetGalleryCover(r fiber.Router) {
	r.Put("/galleries/:id/cover", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.setGalleryCover)
}

func (s *Service) setupDeleteGallery(r fiber.Router) {
	r.Delete("/galleries/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteGallery)
}

func (s *Service) setupGetGallery(r fiber.Router) {
	r.Get("/galleries/:id", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.getGallery)
}

func (s *Service) setupListUserGalleries(r fiber.Router) {
	r.Get("/users/:id/galleries", s.listUserGalleries)
}

func (s *Service) setupSetArtGallery(r fiber.Router) {
	r.Put("/art/:id/gallery", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.setArtGallery)
}

func (s *Service) createGallery(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	req, ok := utils.BindJSON[dto.CreateGalleryRequest](ctx)
	if !ok {
		return nil
	}

	id, err := s.ArtService.CreateGallery(ctx.Context(), userID, req)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, artsvc.ErrEmptyTitle) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to create gallery")
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) listAllGalleries(ctx fiber.Ctx) error {
	corner := ctx.Query("corner")
	galleries, err := s.ArtService.ListAllGalleries(ctx.Context(), corner)
	if err != nil {
		return utils.InternalError(ctx, "failed to list galleries")
	}
	return ctx.JSON(galleries)
}

func (s *Service) updateGallery(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	userID := utils.UserID(ctx)
	req, ok := utils.BindJSON[dto.UpdateGalleryRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.ArtService.UpdateGallery(ctx.Context(), id, userID, req); err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		return utils.InternalError(ctx, "failed to update gallery")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) setGalleryCover(ctx fiber.Ctx) error {
	galleryID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	userID := utils.UserID(ctx)
	var body struct {
		CoverArtID *uuid.UUID `json:"cover_art_id"`
	}
	if err := ctx.Bind().JSON(&body); err != nil {
		return utils.BadRequest(ctx, "invalid request body")
	}

	if err := s.ArtService.SetGalleryCover(ctx.Context(), galleryID, userID, body.CoverArtID); err != nil {
		return utils.InternalError(ctx, "failed to set cover")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteGallery(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	userID := utils.UserID(ctx)
	if err := s.ArtService.DeleteGallery(ctx.Context(), id, userID); err != nil {
		return utils.InternalError(ctx, "failed to delete gallery")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) getGallery(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	viewerID := utils.UserID(ctx)
	limit := fiber.Query[int](ctx, "limit", 24)
	offset := fiber.Query[int](ctx, "offset", 0)

	gallery, art, total, err := s.ArtService.GetGallery(ctx.Context(), id, viewerID, limit, offset)
	if err != nil {
		if errors.Is(err, artsvc.ErrNotFound) {
			return utils.NotFound(ctx, "gallery not found")
		}
		return utils.InternalError(ctx, "failed to get gallery")
	}
	return ctx.JSON(fiber.Map{
		"gallery": gallery,
		"art":     art,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

func (s *Service) listUserGalleries(ctx fiber.Ctx) error {
	userID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	galleries, err := s.ArtService.ListUserGalleries(ctx.Context(), userID)
	if err != nil {
		return utils.InternalError(ctx, "failed to list galleries")
	}
	return ctx.JSON(galleries)
}

func (s *Service) setArtGallery(ctx fiber.Ctx) error {
	artID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	userID := utils.UserID(ctx)
	var body struct {
		GalleryID *uuid.UUID `json:"gallery_id"`
	}
	if err := ctx.Bind().JSON(&body); err != nil {
		return utils.BadRequest(ctx, "invalid request body")
	}

	if err := s.ArtService.SetArtGallery(ctx.Context(), artID, userID, body.GalleryID); err != nil {
		return utils.InternalError(ctx, "failed to set gallery")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}
