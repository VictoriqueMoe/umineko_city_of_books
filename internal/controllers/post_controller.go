package controllers

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"

	"umineko_city_of_books/internal/block"
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
		s.setupGetCornerCounts,
		s.setupCreatePost,
		s.setupGetPost,
		s.setupUpdatePost,
		s.setupDeletePost,
		s.setupUploadPostMedia,
		s.setupDeletePostMedia,
		s.setupLikePost,
		s.setupUnlikePost,
		s.setupCreateComment,
		s.setupUpdateComment,
		s.setupDeleteComment,
		s.setupUploadCommentMedia,
		s.setupLikeComment,
		s.setupUnlikeComment,
		s.setupListUserPosts,
		s.setupFollowUser,
		s.setupUnfollowUser,
		s.setupGetFollowStats,
		s.setupGetFollowers,
		s.setupGetFollowing,
		s.setupVotePoll,
	}
}

func (s *Service) setupListPostFeed(r fiber.Router) {
	r.Get("/posts", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listPostFeed)
}

func (s *Service) setupCreatePost(r fiber.Router) {
	r.Post("/posts", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createPost)
}

func (s *Service) setupGetPost(r fiber.Router) {
	r.Get("/posts/:id", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.getPost)
}

func (s *Service) setupUpdatePost(r fiber.Router) {
	r.Put("/posts/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updatePost)
}

func (s *Service) setupDeletePost(r fiber.Router) {
	r.Delete("/posts/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deletePost)
}

func (s *Service) setupUploadPostMedia(r fiber.Router) {
	r.Post("/posts/:id/media", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.uploadPostMedia)
}

func (s *Service) setupDeletePostMedia(r fiber.Router) {
	r.Delete("/posts/:id/media/:mediaId", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deletePostMedia)
}

func (s *Service) setupLikePost(r fiber.Router) {
	r.Post("/posts/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.likePost)
}

func (s *Service) setupUnlikePost(r fiber.Router) {
	r.Delete("/posts/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unlikePost)
}

func (s *Service) setupCreateComment(r fiber.Router) {
	r.Post("/posts/:id/comments", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createComment)
}

func (s *Service) setupUpdateComment(r fiber.Router) {
	r.Put("/comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateComment)
}

func (s *Service) setupDeleteComment(r fiber.Router) {
	r.Delete("/comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteComment)
}

func (s *Service) setupUploadCommentMedia(r fiber.Router) {
	r.Post("/comments/:id/media", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.uploadCommentMedia)
}

func (s *Service) setupLikeComment(r fiber.Router) {
	r.Post("/comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.likeComment)
}

func (s *Service) setupUnlikeComment(r fiber.Router) {
	r.Delete("/comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unlikeComment)
}

func (s *Service) setupGetCornerCounts(r fiber.Router) {
	r.Get("/posts/corner-counts", s.getCornerCounts)
}

func (s *Service) setupListUserPosts(r fiber.Router) {
	r.Get("/users/:id/posts", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.listUserPosts)
}

func (s *Service) setupFollowUser(r fiber.Router) {
	r.Post("/users/:id/follow", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.followUser)
}

func (s *Service) setupUnfollowUser(r fiber.Router) {
	r.Delete("/users/:id/follow", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unfollowUser)
}

func (s *Service) setupGetFollowStats(r fiber.Router) {
	r.Get("/users/:id/follow-stats", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.getFollowStats)
}

func (s *Service) setupGetFollowers(r fiber.Router) {
	r.Get("/users/:id/followers", s.getFollowers)
}

func (s *Service) setupGetFollowing(r fiber.Router) {
	r.Get("/users/:id/following", s.getFollowing)
}

func (s *Service) getCornerCounts(ctx fiber.Ctx) error {
	counts, err := s.PostService.GetCornerCounts(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get counts"})
	}
	return ctx.JSON(counts)
}

func (s *Service) listPostFeed(ctx fiber.Ctx) error {
	viewerID, _ := ctx.Locals("userID").(uuid.UUID)
	tab := ctx.Query("tab", "everyone")
	corner := ctx.Query("corner", "general")
	search := ctx.Query("search")
	sort := ctx.Query("sort")
	seed := fiber.Query[int](ctx, "seed", 0)
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.PostService.ListFeed(ctx.Context(), tab, viewerID, corner, search, sort, seed, limit, offset)
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

func (s *Service) updatePost(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
	}

	userID := ctx.Locals("userID").(uuid.UUID)
	var req dto.UpdatePostRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.PostService.UpdatePost(ctx.Context(), id, userID, req); err != nil {
		if errors.Is(err, postsvc.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update post"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func viewerHash(ctx fiber.Ctx) string {
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

func (s *Service) getPost(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
	}

	viewerID, _ := ctx.Locals("userID").(uuid.UUID)
	result, err := s.PostService.GetPost(ctx.Context(), id, viewerID, viewerHash(ctx))
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

func (s *Service) deletePostMedia(ctx fiber.Ctx) error {
	postID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
	}

	mediaID := fiber.Query[int64](ctx, "mediaId", 0)
	if mediaID == 0 {
		mediaID, _ = strconv.ParseInt(ctx.Params("mediaId"), 10, 64)
	}
	if mediaID == 0 {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid media id"})
	}

	userID := ctx.Locals("userID").(uuid.UUID)
	if err := s.PostService.DeletePostMedia(ctx.Context(), postID, mediaID, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete media"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) likePost(ctx fiber.Ctx) error {
	postID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
	}

	userID := ctx.Locals("userID").(uuid.UUID)
	if err := s.PostService.LikePost(ctx.Context(), userID, postID); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is blocked"})
		}
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
		if errors.Is(err, block.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is blocked"})
		}
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

func (s *Service) updateComment(ctx fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid comment id"})
	}

	userID := ctx.Locals("userID").(uuid.UUID)
	var req dto.UpdateCommentRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.PostService.UpdateComment(ctx.Context(), id, userID, req); err != nil {
		if errors.Is(err, postsvc.ErrEmptyBody) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update comment"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) likeComment(ctx fiber.Ctx) error {
	commentID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid comment id"})
	}

	userID := ctx.Locals("userID").(uuid.UUID)
	if err := s.PostService.LikeComment(ctx.Context(), userID, commentID); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is blocked"})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to like comment"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unlikeComment(ctx fiber.Ctx) error {
	commentID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid comment id"})
	}

	userID := ctx.Locals("userID").(uuid.UUID)
	if err := s.PostService.UnlikeComment(ctx.Context(), userID, commentID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to unlike comment"})
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
		if errors.Is(err, block.ErrUserBlocked) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is blocked"})
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

func (s *Service) getFollowers(ctx fiber.Ctx) error {
	userID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	limit := fiber.Query[int](ctx, "limit", 50)
	offset := fiber.Query[int](ctx, "offset", 0)

	users, total, err := s.FollowService.GetFollowers(ctx.Context(), userID, limit, offset)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get followers"})
	}
	return ctx.JSON(fiber.Map{"users": users, "total": total})
}

func (s *Service) getFollowing(ctx fiber.Ctx) error {
	userID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	limit := fiber.Query[int](ctx, "limit", 50)
	offset := fiber.Query[int](ctx, "offset", 0)

	users, total, err := s.FollowService.GetFollowing(ctx.Context(), userID, limit, offset)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get following"})
	}
	return ctx.JSON(fiber.Map{"users": users, "total": total})
}

func (s *Service) setupVotePoll(r fiber.Router) {
	r.Post("/posts/:id/poll/vote", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.votePoll)
}

func (s *Service) votePoll(ctx fiber.Ctx) error {
	postID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
	}
	userID := ctx.Locals("userID").(uuid.UUID)

	var req dto.VotePollRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	poll, err := s.PostService.VotePoll(ctx.Context(), postID, userID, req.OptionID)
	if err != nil {
		if errors.Is(err, postsvc.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "poll not found"})
		}
		if errors.Is(err, postsvc.ErrPollExpired) {
			return ctx.Status(fiber.StatusGone).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, postsvc.ErrAlreadyVoted) {
			return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, postsvc.ErrInvalidOption) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to vote"})
	}
	return ctx.JSON(poll)
}
