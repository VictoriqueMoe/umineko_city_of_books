package controllers

import (
	"errors"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/dto"
	fanficsvc "umineko_city_of_books/internal/fanfic"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/repository"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllFanficRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupListFanfics,
		s.setupGetFanfic,
		s.setupCreateFanfic,
		s.setupUpdateFanfic,
		s.setupDeleteFanfic,
		s.setupUploadFanficCover,
		s.setupDeleteFanficCover,
		s.setupGetFanficChapter,
		s.setupCreateFanficChapter,
		s.setupUpdateFanficChapter,
		s.setupDeleteFanficChapter,
		s.setupFavouriteFanfic,
		s.setupUnfavouriteFanfic,
		s.setupCreateFanficComment,
		s.setupUpdateFanficComment,
		s.setupDeleteFanficComment,
		s.setupLikeFanficComment,
		s.setupUnlikeFanficComment,
		s.setupUploadFanficCommentMedia,
		s.setupGetFanficLanguages,
		s.setupGetFanficSeries,
		s.setupSearchOCCharacters,
		s.setupListUserFanfics,
		s.setupListUserFanficFavourites,
	}
}

func (s *Service) setupListFanfics(r fiber.Router) {
	r.Get("/fanfics", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listFanfics)
}

func (s *Service) setupGetFanfic(r fiber.Router) {
	r.Get("/fanfics/:id", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.getFanfic)
}

func (s *Service) setupCreateFanfic(r fiber.Router) {
	r.Post("/fanfics", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createFanfic)
}

func (s *Service) setupUpdateFanfic(r fiber.Router) {
	r.Put("/fanfics/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateFanfic)
}

func (s *Service) setupDeleteFanfic(r fiber.Router) {
	r.Delete("/fanfics/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteFanfic)
}

func (s *Service) setupUploadFanficCover(r fiber.Router) {
	r.Post("/fanfics/:id/cover", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.uploadFanficCover)
}

func (s *Service) setupDeleteFanficCover(r fiber.Router) {
	r.Delete("/fanfics/:id/cover", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteFanficCover)
}

func (s *Service) setupGetFanficChapter(r fiber.Router) {
	r.Get("/fanfics/:id/chapters/:number", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.getFanficChapter)
}

func (s *Service) setupCreateFanficChapter(r fiber.Router) {
	r.Post("/fanfics/:id/chapters", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createFanficChapter)
}

func (s *Service) setupUpdateFanficChapter(r fiber.Router) {
	r.Put("/fanfic-chapters/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateFanficChapter)
}

func (s *Service) setupDeleteFanficChapter(r fiber.Router) {
	r.Delete("/fanfic-chapters/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteFanficChapter)
}

func (s *Service) setupFavouriteFanfic(r fiber.Router) {
	r.Post("/fanfics/:id/favourite", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.favouriteFanfic)
}

func (s *Service) setupUnfavouriteFanfic(r fiber.Router) {
	r.Delete("/fanfics/:id/favourite", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unfavouriteFanfic)
}

func (s *Service) setupCreateFanficComment(r fiber.Router) {
	r.Post("/fanfics/:id/comments", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createFanficComment)
}

func (s *Service) setupUpdateFanficComment(r fiber.Router) {
	r.Put("/fanfic-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateFanficComment)
}

func (s *Service) setupDeleteFanficComment(r fiber.Router) {
	r.Delete("/fanfic-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteFanficComment)
}

func (s *Service) setupLikeFanficComment(r fiber.Router) {
	r.Post("/fanfic-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.likeFanficComment)
}

func (s *Service) setupUnlikeFanficComment(r fiber.Router) {
	r.Delete("/fanfic-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unlikeFanficComment)
}

func (s *Service) setupUploadFanficCommentMedia(r fiber.Router) {
	r.Post("/fanfic-comments/:id/media", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.uploadFanficCommentMedia)
}

func (s *Service) setupGetFanficLanguages(r fiber.Router) {
	r.Get("/fanfic-languages", s.getFanficLanguages)
}

func (s *Service) setupGetFanficSeries(r fiber.Router) {
	r.Get("/fanfic-series", s.getFanficSeries)
}

func (s *Service) setupSearchOCCharacters(r fiber.Router) {
	r.Get("/fanfic-oc-characters", s.searchOCCharacters)
}

func (s *Service) setupListUserFanfics(r fiber.Router) {
	r.Get("/users/:id/fanfics", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listUserFanfics)
}

func (s *Service) listFanfics(ctx fiber.Ctx) error {
	viewerID, _ := ctx.Locals("userID").(uuid.UUID)
	params := repository.FanficListParams{
		Sort:       ctx.Query("sort", "updated"),
		Series:     ctx.Query("series"),
		Rating:     ctx.Query("rating"),
		GenreA:     ctx.Query("genre_a"),
		GenreB:     ctx.Query("genre_b"),
		Language:   ctx.Query("language"),
		Status:     ctx.Query("status"),
		Tag:        ctx.Query("tag"),
		CharacterA: ctx.Query("char_a"),
		CharacterB: ctx.Query("char_b"),
		CharacterC: ctx.Query("char_c"),
		CharacterD: ctx.Query("char_d"),
		IsPairing:  ctx.Query("pairing") == "true",
		ShowLemons: ctx.Query("lemons") == "true",
		Search:     ctx.Query("search"),
		Limit:      fiber.Query[int](ctx, "limit", 25),
		Offset:     fiber.Query[int](ctx, "offset", 0),
	}
	result, err := s.FanficService.ListFanfics(ctx.Context(), viewerID, params)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list fanfics"})
	}
	return ctx.JSON(result)
}

func (s *Service) getFanfic(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid fanfic id"})
	}

	viewerID, _ := ctx.Locals("userID").(uuid.UUID)
	result, err := s.FanficService.GetFanfic(ctx.Context(), id, viewerID, viewerHash(ctx))
	if err != nil {
		if errors.Is(err, fanficsvc.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "fanfic not found"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get fanfic"})
	}
	return ctx.JSON(result)
}

func (s *Service) createFanfic(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)
	var req dto.CreateFanficRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	id, err := s.FanficService.CreateFanfic(ctx.Context(), userID, req)
	if err != nil {
		if errors.Is(err, fanficsvc.ErrEmptyTitle) || errors.Is(err, fanficsvc.ErrTooManyGenres) || errors.Is(err, fanficsvc.ErrTooManyCharacters) || errors.Is(err, fanficsvc.ErrTooManyTags) || errors.Is(err, fanficsvc.ErrTagTooLong) || errors.Is(err, fanficsvc.ErrInvalidRating) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create fanfic"})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateFanfic(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid fanfic id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.UpdateFanficRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.FanficService.UpdateFanfic(ctx.Context(), id, userID, req); err != nil {
		if errors.Is(err, fanficsvc.ErrEmptyTitle) || errors.Is(err, fanficsvc.ErrTooManyGenres) || errors.Is(err, fanficsvc.ErrTooManyCharacters) || errors.Is(err, fanficsvc.ErrTooManyTags) || errors.Is(err, fanficsvc.ErrTagTooLong) || errors.Is(err, fanficsvc.ErrInvalidRating) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, fanficsvc.ErrNotAuthor) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, fanficsvc.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "fanfic not found"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update fanfic"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteFanfic(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid fanfic id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.FanficService.DeleteFanfic(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete fanfic"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) uploadFanficCover(ctx fiber.Ctx) error {
	fanficID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid fanfic id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	file, err := ctx.FormFile("image")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no image file provided"})
	}
	reader, err := file.Open()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read file"})
	}
	defer reader.Close()

	url, err := s.FanficService.UploadCoverImage(ctx.Context(), fanficID, userID, file.Header.Get("Content-Type"), file.Size, reader)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(fiber.Map{"image_url": url})
}

func (s *Service) deleteFanficCover(ctx fiber.Ctx) error {
	fanficID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid fanfic id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.FanficService.RemoveCoverImage(ctx.Context(), fanficID, userID); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) getFanficChapter(ctx fiber.Ctx) error {
	fanficID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid fanfic id"})
	}
	number := fiber.Params[int](ctx, "number", 0)
	if number < 1 {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid chapter number"})
	}
	viewerID, _ := ctx.Locals("userID").(uuid.UUID)
	result, err := s.FanficService.GetChapter(ctx.Context(), fanficID, number, viewerID)
	if err != nil {
		if errors.Is(err, fanficsvc.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "chapter not found"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get chapter"})
	}
	return ctx.JSON(result)
}

func (s *Service) createFanficChapter(ctx fiber.Ctx) error {
	fanficID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid fanfic id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.CreateChapterRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	id, err := s.FanficService.CreateChapter(ctx.Context(), fanficID, userID, req)
	if err != nil {
		if errors.Is(err, fanficsvc.ErrNotAuthor) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, fanficsvc.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, fanficsvc.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "fanfic not found"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create chapter"})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateFanficChapter(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid chapter id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.UpdateChapterRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.FanficService.UpdateChapter(ctx.Context(), id, userID, req); err != nil {
		if errors.Is(err, fanficsvc.ErrNotAuthor) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, fanficsvc.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, fanficsvc.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "chapter not found"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update chapter"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteFanficChapter(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid chapter id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.FanficService.DeleteChapter(ctx.Context(), id, userID); err != nil {
		if errors.Is(err, fanficsvc.ErrNotAuthor) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, fanficsvc.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "chapter not found"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete chapter"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) favouriteFanfic(ctx fiber.Ctx) error {
	fanficID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid fanfic id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.FanficService.Favourite(ctx.Context(), userID, fanficID); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is blocked"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to favourite fanfic"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unfavouriteFanfic(ctx fiber.Ctx) error {
	fanficID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid fanfic id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.FanficService.Unfavourite(ctx.Context(), userID, fanficID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to unfavourite fanfic"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) createFanficComment(ctx fiber.Ctx) error {
	fanficID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid fanfic id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.CreateCommentRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	id, err := s.FanficService.CreateComment(ctx.Context(), fanficID, userID, req)
	if err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is blocked"})
		}
		if errors.Is(err, fanficsvc.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create comment"})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateFanficComment(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid comment id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.UpdateCommentRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.FanficService.UpdateComment(ctx.Context(), id, userID, req); err != nil {
		if errors.Is(err, fanficsvc.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update comment"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteFanficComment(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid comment id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.FanficService.DeleteComment(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete comment"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) likeFanficComment(ctx fiber.Ctx) error {
	commentID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid comment id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.FanficService.LikeComment(ctx.Context(), userID, commentID); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is blocked"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to like comment"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unlikeFanficComment(ctx fiber.Ctx) error {
	commentID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid comment id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	if err := s.FanficService.UnlikeComment(ctx.Context(), userID, commentID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to unlike comment"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) uploadFanficCommentMedia(ctx fiber.Ctx) error {
	commentID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid comment id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	file, err := ctx.FormFile("media")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no media file provided"})
	}
	reader, err := file.Open()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read file"})
	}
	defer reader.Close()

	result, err := s.FanficService.UploadCommentMedia(ctx.Context(), commentID, userID, file.Header.Get("Content-Type"), file.Size, reader)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}

func (s *Service) getFanficLanguages(ctx fiber.Ctx) error {
	langs, err := s.FanficService.GetLanguages(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get languages"})
	}
	return ctx.JSON(fiber.Map{"languages": langs})
}

func (s *Service) getFanficSeries(ctx fiber.Ctx) error {
	series, err := s.FanficService.GetSeries(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get series"})
	}
	return ctx.JSON(fiber.Map{"series": series})
}

func (s *Service) searchOCCharacters(ctx fiber.Ctx) error {
	q := ctx.Query("q")
	results, err := s.FanficService.SearchOCCharacters(ctx.Context(), q)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to search characters"})
	}
	return ctx.JSON(fiber.Map{"characters": results})
}

func (s *Service) listUserFanfics(ctx fiber.Ctx) error {
	userID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}
	viewerID, _ := ctx.Locals("userID").(uuid.UUID)
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.FanficService.ListFanficsByUser(ctx.Context(), userID, viewerID, limit, offset)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list user fanfics"})
	}
	return ctx.JSON(result)
}

func (s *Service) setupListUserFanficFavourites(r fiber.Router) {
	r.Get("/users/:id/fanfic-favourites", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listUserFanficFavourites)
}

func (s *Service) listUserFanficFavourites(ctx fiber.Ctx) error {
	userID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}
	viewerID, _ := ctx.Locals("userID").(uuid.UUID)
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.FanficService.ListFavourites(ctx.Context(), userID, viewerID, limit, offset)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list favourites"})
	}
	return ctx.JSON(result)
}
