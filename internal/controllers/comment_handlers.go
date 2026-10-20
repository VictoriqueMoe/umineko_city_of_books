package controllers

import (
	"context"
	"errors"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/comment"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/upload"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type (
	errorResponse struct {
		err     error
		respond func(fiber.Ctx, string) error
		message string
	}

	commentRoute struct {
		kind       spec.CommentKind
		path       string
		createPath string
	}
)

var (
	commentRoutes = []commentRoute{
		{kind: spec.CommentKindPost, path: "/comments"},
		{kind: spec.CommentKindArt, path: "/art-comments", createPath: "/art/:id/comments"},
		{kind: spec.CommentKindAnnouncement, path: "/announcement-comments", createPath: "/announcements/:id/comments"},
		{kind: spec.CommentKindMystery, path: "/mystery-comments"},
		{kind: spec.CommentKindShip, path: "/ship-comments", createPath: "/ships/:id/comments"},
		{kind: spec.CommentKindOC, path: "/oc-comments", createPath: "/ocs/:id/comments"},
		{kind: spec.CommentKindFanfic, path: "/fanfic-comments"},
		{kind: spec.CommentKindJournal, path: "/journal-comments"},
		{kind: spec.CommentKindSecret, path: "/secret-comments"},
	}
)

func (s *Service) getAllCommentRoutes() []FSetupRoute {
	return []FSetupRoute{s.setupCommentRoutes}
}

func (s *Service) setupCommentRoutes(r fiber.Router) {
	for _, route := range commentRoutes {
		if route.createPath != "" {
			r.Post(route.createPath, s.requireAuth(), s.requireEstablished(), s.createComment(route.kind))
		}

		r.Put(route.path+"/:id", s.requireAuth(), s.updateComment(route.kind))
		r.Delete(route.path+"/:id", s.requireAuth(), s.deleteComment(route.kind))
		r.Post(route.path+"/:id/like", s.requireAuth(), s.likeComment(route.kind))
		r.Delete(route.path+"/:id/like", s.requireAuth(), s.unlikeComment(route.kind))
		r.Post(route.path+"/:id/media", s.requireAuth(), s.requireEstablished(), s.uploadCommentMedia(route.kind))
	}
}

func (s *Service) createComment(kind spec.CommentKind) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		entityID, ok := utils.ParseID(ctx)
		if !ok {
			return nil
		}

		userID := utils.UserID(ctx)
		target := spec.CommentTarget{Kind: kind, ID: entityID}

		return handleCreateComment(ctx, func(c context.Context, req dto.CreateCommentRequest, files []upload.FileUpload) (uuid.UUID, error) {
			return s.CommentService.Create(c, target, userID, req, files)
		}, nil)
	}
}

func (s *Service) updateComment(kind spec.CommentKind) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		commentID, ok := utils.ParseID(ctx)
		if !ok {
			return nil
		}

		req, ok := utils.BindJSON[dto.UpdateCommentRequest](ctx)
		if !ok {
			return nil
		}

		ref := spec.CommentRef{Kind: kind, CommentID: commentID}
		if err := s.CommentService.Update(ctx.Context(), ref, utils.UserID(ctx), req.Body); err != nil {
			return commentError(ctx, err, "failed to update comment")
		}

		return ctx.SendStatus(fiber.StatusNoContent)
	}
}

func (s *Service) deleteComment(kind spec.CommentKind) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		commentID, ok := utils.ParseID(ctx)
		if !ok {
			return nil
		}

		ref := spec.CommentRef{Kind: kind, CommentID: commentID}
		if err := s.CommentService.Delete(ctx.Context(), ref, utils.UserID(ctx)); err != nil {
			return commentError(ctx, err, "failed to delete comment")
		}

		return ctx.SendStatus(fiber.StatusNoContent)
	}
}

func (s *Service) likeComment(kind spec.CommentKind) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		commentID, ok := utils.ParseID(ctx)
		if !ok {
			return nil
		}

		ref := spec.CommentRef{Kind: kind, CommentID: commentID}
		if err := s.CommentService.Like(ctx.Context(), ref, utils.UserID(ctx)); err != nil {
			return commentError(ctx, err, "failed to like comment")
		}

		return ctx.SendStatus(fiber.StatusNoContent)
	}
}

func (s *Service) unlikeComment(kind spec.CommentKind) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		commentID, ok := utils.ParseID(ctx)
		if !ok {
			return nil
		}

		ref := spec.CommentRef{Kind: kind, CommentID: commentID}
		if err := s.CommentService.Unlike(ctx.Context(), ref, utils.UserID(ctx)); err != nil {
			return commentError(ctx, err, "failed to unlike comment")
		}

		return ctx.SendStatus(fiber.StatusNoContent)
	}
}

func (s *Service) uploadCommentMedia(kind spec.CommentKind) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		commentID, ok := utils.ParseID(ctx)
		if !ok {
			return nil
		}

		file, err := ctx.FormFile("media")
		if err != nil {
			return utils.BadRequest(ctx, "no media file provided")
		}

		reader, err := file.Open()
		if err != nil {
			return utils.InternalError(ctx, "failed to read file", err)
		}
		defer reader.Close()

		ref := spec.CommentRef{Kind: kind, CommentID: commentID}
		result, err := s.CommentService.UploadMedia(ctx.Context(), ref, utils.UserID(ctx), file.Header.Get("Content-Type"), file.Filename, file.Size, reader, isSpoilerUpload(ctx))
		if err != nil {
			return commentError(ctx, err, "failed to upload media")
		}

		return ctx.Status(fiber.StatusCreated).JSON(result)
	}
}

func handleCreateComment[T any](
	ctx fiber.Ctx,
	create func(context.Context, T, []upload.FileUpload) (uuid.UUID, error),
	familyErrors []errorResponse,
) error {
	req, ok := utils.BindWithMedia[T](ctx)
	if !ok {
		return nil
	}

	id, err := create(ctx.Context(), req.Payload, req.Media)
	if err != nil {
		for _, fe := range familyErrors {
			if !errors.Is(err, fe.err) {
				continue
			}

			if fe.message == "" {
				return fe.respond(ctx, err.Error())
			}

			return fe.respond(ctx, fe.message)
		}

		return commentError(ctx, err, "failed to create comment")
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func commentError(ctx fiber.Ctx, err error, failure string) error {
	if utils.MapFilterError(ctx, err) {
		return nil
	}
	if errors.Is(err, comment.ErrEmptyBody) {
		return utils.BadRequest(ctx, err.Error())
	}
	if errors.Is(err, comment.ErrTargetNotFound) {
		return utils.NotFound(ctx, err.Error())
	}
	if errors.Is(err, comment.ErrNotFound) || errors.Is(err, daoerr.ErrNotFound) {
		return utils.NotFound(ctx, "comment not found")
	}
	if errors.Is(err, comment.ErrForbidden) || errors.Is(err, authz.ErrNotCommentAuthor) {
		return utils.Forbidden(ctx, err.Error())
	}
	if errors.Is(err, block.ErrUserBlocked) {
		return utils.Forbidden(ctx, "user is blocked")
	}
	if utils.IsUploadRejection(err) {
		return utils.UploadRejected(ctx, err)
	}

	return utils.InternalError(ctx, failure, err)
}

func isSpoilerUpload(ctx fiber.Ctx) bool {
	return ctx.FormValue("is_spoiler") == "true"
}
