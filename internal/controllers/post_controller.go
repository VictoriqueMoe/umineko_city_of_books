package controllers

import (
	"errors"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/follow"
	"umineko_city_of_books/internal/middleware"
	postsvc "umineko_city_of_books/internal/post"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllPostRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupListPostFeed,
		s.setupCreatePost,
		s.setupGetPost,
		s.setupDeletePost,
		s.setupUploadPostMedia,
		s.setupLikePost,
		s.setupUnlikePost,
		s.setupCreateComment,
		s.setupDeleteComment,
		s.setupUploadCommentMedia,
		s.setupListUserPosts,
		s.setupFollowUser,
		s.setupUnfollowUser,
		s.setupGetFollowStats,
	}
}

func (s *Service) setupListPostFeed(r fiber.Router) {
	r.Get("/posts", middleware.OptionalAuth(s.AuthSession), s.listPostFeed)
}

func (s *Service) setupCreatePost(r fiber.Router) {
	r.Post("/posts", middleware.RequireAuth(s.AuthSession), s.createPost)
}

func (s *Service) setupGetPost(r fiber.Router) {
	r.Get("/posts/:id", middleware.OptionalAuth(s.AuthSession), s.getPost)
}

func (s *Service) setupDeletePost(r fiber.Router) {
	r.Delete("/posts/:id", middleware.RequireAuth(s.AuthSession), s.deletePost)
}

func (s *Service) setupUploadPostMedia(r fiber.Router) {
	r.Post("/posts/:id/media", middleware.RequireAuth(s.AuthSession), s.uploadPostMedia)
}

func (s *Service) setupLikePost(r fiber.Router) {
	r.Post("/posts/:id/like", middleware.RequireAuth(s.AuthSession), s.likePost)
}

func (s *Service) setupUnlikePost(r fiber.Router) {
	r.Delete("/posts/:id/like", middleware.RequireAuth(s.AuthSession), s.unlikePost)
}

func (s *Service) setupCreateComment(r fiber.Router) {
	r.Post("/posts/:id/comments", middleware.RequireAuth(s.AuthSession), s.createComment)
}

func (s *Service) setupDeleteComment(r fiber.Router) {
	r.Delete("/comments/:id", middleware.RequireAuth(s.AuthSession), s.deleteComment)
}

func (s *Service) setupUploadCommentMedia(r fiber.Router) {
	r.Post("/comments/:id/media", middleware.RequireAuth(s.AuthSession), s.uploadCommentMedia)
}

func (s *Service) setupListUserPosts(r fiber.Router) {
	r.Get("/users/:id/posts", middleware.OptionalAuth(s.AuthSession), s.listUserPosts)
}

func (s *Service) setupFollowUser(r fiber.Router) {
	r.Post("/users/:id/follow", middleware.RequireAuth(s.AuthSession), s.followUser)
}

func (s *Service) setupUnfollowUser(r fiber.Router) {
	r.Delete("/users/:id/follow", middleware.RequireAuth(s.AuthSession), s.unfollowUser)
}

func (s *Service) setupGetFollowStats(r fiber.Router) {
	r.Get("/users/:id/follow-stats", middleware.OptionalAuth(s.AuthSession), s.getFollowStats)
}

func (s *Service) listPostFeed(ctx fiber.Ctx) error {
	viewerID, _ := ctx.Locals("userID").(uuid.UUID)
	tab := ctx.Query("tab", "everyone")
	search := ctx.Query("search")
	sort := ctx.Query("sort")
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.PostService.ListFeed(ctx.Context(), tab, viewerID, search, sort, limit, offset)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list posts"})
	}
	return ctx.JSON(result)
}

func (s *Service) createPost(ctx fiber.Ctx) error {
	userID := ctx.Locals("userID").(uuid.UUID)
	var req dto.CreatePostRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	id, err := s.PostService.CreatePost(ctx.Context(), userID, req)
	if err != nil {
		if errors.Is(err, postsvc.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, postsvc.ErrRateLimited) {
			return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create post"})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) getPost(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
	}

	viewerID, _ := ctx.Locals("userID").(uuid.UUID)
	result, err := s.PostService.GetPost(ctx.Context(), id, viewerID)
	if err != nil {
		if errors.Is(err, postsvc.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get post"})
	}
	return ctx.JSON(result)
}

func (s *Service) deletePost(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
	}

	userID := ctx.Locals("userID").(uuid.UUID)
	if err := s.PostService.DeletePost(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete post"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) uploadPostMedia(ctx fiber.Ctx) error {
	postID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
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

	result, err := s.PostService.UploadPostMedia(ctx.Context(), postID, userID, file.Header.Get("Content-Type"), file.Size, reader)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}

func (s *Service) likePost(ctx fiber.Ctx) error {
	postID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
	}

	userID := ctx.Locals("userID").(uuid.UUID)
	if err := s.PostService.LikePost(ctx.Context(), userID, postID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to like post"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unlikePost(ctx fiber.Ctx) error {
	postID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
	}

	userID := ctx.Locals("userID").(uuid.UUID)
	if err := s.PostService.UnlikePost(ctx.Context(), userID, postID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to unlike post"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) createComment(ctx fiber.Ctx) error {
	postID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
	}

	userID := ctx.Locals("userID").(uuid.UUID)
	var req dto.CreateCommentRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	id, err := s.PostService.CreateComment(ctx.Context(), postID, userID, req)
	if err != nil {
		if errors.Is(err, postsvc.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create comment"})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) deleteComment(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid comment id"})
	}

	userID := ctx.Locals("userID").(uuid.UUID)
	if err := s.PostService.DeleteComment(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete comment"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) uploadCommentMedia(ctx fiber.Ctx) error {
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

	result, err := s.PostService.UploadCommentMedia(ctx.Context(), commentID, userID, file.Header.Get("Content-Type"), file.Size, reader)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}

func (s *Service) listUserPosts(ctx fiber.Ctx) error {
	userID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	viewerID, _ := ctx.Locals("userID").(uuid.UUID)
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.PostService.ListUserPosts(ctx.Context(), userID, viewerID, limit, offset)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list user posts"})
	}
	return ctx.JSON(result)
}

func (s *Service) followUser(ctx fiber.Ctx) error {
	targetID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	userID := ctx.Locals("userID").(uuid.UUID)
	if err := s.FollowService.Follow(ctx.Context(), userID, targetID); err != nil {
		if errors.Is(err, follow.ErrCannotFollowSelf) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to follow user"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unfollowUser(ctx fiber.Ctx) error {
	targetID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	userID := ctx.Locals("userID").(uuid.UUID)
	if err := s.FollowService.Unfollow(ctx.Context(), userID, targetID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to unfollow user"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) getFollowStats(ctx fiber.Ctx) error {
	userID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	viewerID, _ := ctx.Locals("userID").(uuid.UUID)
	stats, err := s.FollowService.GetFollowStats(ctx.Context(), userID, viewerID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get follow stats"})
	}
	return ctx.JSON(stats)
}
