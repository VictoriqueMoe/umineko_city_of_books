package controllers

import (
	"errors"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/quotefinder"
	"umineko_city_of_books/internal/ship"

	"github.com/gofiber/fiber/v3"
)

func (s *Service) getAllShipRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupListShips,
		s.setupGetShip,
		s.setupCreateShip,
		s.setupUpdateShip,
		s.setupDeleteShip,
		s.setupUploadShipImage,
		s.setupVoteShip,
		s.setupCreateShipComment,
		s.setupUpdateShipComment,
		s.setupDeleteShipComment,
		s.setupLikeShipComment,
		s.setupUnlikeShipComment,
		s.setupUploadShipCommentMedia,
		s.setupListCharacters,
		s.setupListUserShips,
	}
}

func (s *Service) setupListShips(r fiber.Router) {
	r.Get("/ships", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listShips)
}

func (s *Service) setupGetShip(r fiber.Router) {
	r.Get("/ships/:id", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.getShip)
}

func (s *Service) setupCreateShip(r fiber.Router) {
	r.Post("/ships", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createShip)
}

func (s *Service) setupUpdateShip(r fiber.Router) {
	r.Put("/ships/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateShip)
}

func (s *Service) setupDeleteShip(r fiber.Router) {
	r.Delete("/ships/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteShip)
}

func (s *Service) setupUploadShipImage(r fiber.Router) {
	r.Post("/ships/:id/image", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.uploadShipImage)
}

func (s *Service) setupVoteShip(r fiber.Router) {
	r.Post("/ships/:id/vote", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.voteShip)
}

func (s *Service) setupCreateShipComment(r fiber.Router) {
	r.Post("/ships/:id/comments", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createShipComment)
}

func (s *Service) setupUpdateShipComment(r fiber.Router) {
	r.Put("/ship-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateShipComment)
}

func (s *Service) setupDeleteShipComment(r fiber.Router) {
	r.Delete("/ship-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteShipComment)
}

func (s *Service) setupLikeShipComment(r fiber.Router) {
	r.Post("/ship-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.likeShipComment)
}

func (s *Service) setupUnlikeShipComment(r fiber.Router) {
	r.Delete("/ship-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unlikeShipComment)
}

func (s *Service) setupUploadShipCommentMedia(r fiber.Router) {
	r.Post("/ship-comments/:id/media", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.uploadShipCommentMedia)
}

func (s *Service) setupListCharacters(r fiber.Router) {
	r.Get("/characters/:series", s.listCharacters)
}

func (s *Service) listShips(ctx fiber.Ctx) error {
	viewerID := utils.UserID(ctx)
	sort := ctx.Query("sort", "new")
	series := ctx.Query("series")
	characterID := ctx.Query("character")
	crackshipsOnly := ctx.Query("crackships") == "true"
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.ShipService.ListShips(ctx.Context(), viewerID, sort, crackshipsOnly, series, characterID, limit, offset)
	if err != nil {
		return utils.InternalError(ctx, "failed to list ships")
	}
	return ctx.JSON(result)
}

func (s *Service) getShip(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	viewerID := utils.UserID(ctx)
	result, err := s.ShipService.GetShip(ctx.Context(), id, viewerID)
	if err != nil {
		if errors.Is(err, ship.ErrNotFound) {
			return utils.NotFound(ctx, "ship not found")
		}
		return utils.InternalError(ctx, "failed to get ship")
	}
	return ctx.JSON(result)
}

func (s *Service) createShip(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	req, ok := utils.BindJSON[dto.CreateShipRequest](ctx)
	if !ok {
		return nil
	}

	id, err := s.ShipService.CreateShip(ctx.Context(), userID, req)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, ship.ErrEmptyTitle) || errors.Is(err, ship.ErrTooFewCharacters) || errors.Is(err, ship.ErrDuplicateCharacters) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to create ship")
	}
	s.Hub.BumpSidebarActivity("ships")
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateShip(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.UpdateShipRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.ShipService.UpdateShip(ctx.Context(), id, userID, req); err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, ship.ErrEmptyTitle) || errors.Is(err, ship.ErrTooFewCharacters) || errors.Is(err, ship.ErrDuplicateCharacters) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to update ship")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteShip(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.ShipService.DeleteShip(ctx.Context(), id, userID); err != nil {
		return utils.InternalError(ctx, "failed to delete ship")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) uploadShipImage(ctx fiber.Ctx) error {
	shipID, ok := utils.ParseID(ctx)
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

	url, err := s.ShipService.UploadShipImage(ctx.Context(), shipID, userID, file.Header.Get("Content-Type"), file.Size, reader)
	if err != nil {
		return utils.BadRequest(ctx, err.Error())
	}
	return ctx.JSON(fiber.Map{"image_url": url})
}

func (s *Service) voteShip(ctx fiber.Ctx) error {
	shipID, ok := utils.ParseID(ctx)
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

	if err := s.ShipService.Vote(ctx.Context(), userID, shipID, req.Value); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		return utils.InternalError(ctx, "failed to vote")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) createShipComment(ctx fiber.Ctx) error {
	shipID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.CreateCommentRequest](ctx)
	if !ok {
		return nil
	}

	id, err := s.ShipService.CreateComment(ctx.Context(), shipID, userID, req)
	if err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		if errors.Is(err, ship.ErrEmptyBody) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to create comment")
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateShipComment(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.UpdateCommentRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.ShipService.UpdateComment(ctx.Context(), id, userID, req); err != nil {
		if utils.MapFilterError(ctx, err) {
			return nil
		}
		if errors.Is(err, ship.ErrEmptyBody) {
			return utils.BadRequest(ctx, err.Error())
		}
		return utils.InternalError(ctx, "failed to update comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteShipComment(ctx fiber.Ctx) error {
	id, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.ShipService.DeleteComment(ctx.Context(), id, userID); err != nil {
		return utils.InternalError(ctx, "failed to delete comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) likeShipComment(ctx fiber.Ctx) error {
	commentID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.ShipService.LikeComment(ctx.Context(), userID, commentID); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return utils.Forbidden(ctx, "user is blocked")
		}
		return utils.InternalError(ctx, "failed to like comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unlikeShipComment(ctx fiber.Ctx) error {
	commentID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := utils.UserID(ctx)

	if err := s.ShipService.UnlikeComment(ctx.Context(), userID, commentID); err != nil {
		return utils.InternalError(ctx, "failed to unlike comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) uploadShipCommentMedia(ctx fiber.Ctx) error {
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

	result, err := s.ShipService.UploadCommentMedia(ctx.Context(), commentID, userID, file.Header.Get("Content-Type"), file.Size, reader)
	if err != nil {
		return utils.BadRequest(ctx, err.Error())
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}

func (s *Service) listCharacters(ctx fiber.Ctx) error {
	series, err := quotefinder.ParseSeries(ctx.Params("series"))
	if err != nil {
		return utils.BadRequest(ctx, err.Error())
	}
	chars, err := s.ShipService.ListCharacters(series)
	if err != nil {
		return utils.InternalError(ctx, "failed to list characters")
	}
	return ctx.JSON(dto.CharacterListResponse{
		Series:     string(series),
		Characters: chars,
	})
}

func (s *Service) setupListUserShips(r fiber.Router) {
	r.Get("/users/:id/ships", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listUserShips)
}

func (s *Service) listUserShips(ctx fiber.Ctx) error {
	userID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	viewerID := utils.UserID(ctx)
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.ShipService.ListShipsByUser(ctx.Context(), userID, viewerID, limit, offset)
	if err != nil {
		return utils.InternalError(ctx, "failed to list user ships")
	}
	return ctx.JSON(result)
}
