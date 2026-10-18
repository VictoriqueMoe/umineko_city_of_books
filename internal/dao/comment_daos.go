package dao

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	CommentDAO interface {
		CreateComment(ctx context.Context, s spec.NewComment, tx ...*sql.Tx) (*model.CommentRow, error)
		UpdateComment(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) error
		GetComments(ctx context.Context, q spec.CommentQuery, tx ...*sql.Tx) ([]model.CommentRow, int, error)
		GetCommentByID(ctx context.Context, r spec.CommentRef, tx ...*sql.Tx) (*model.CommentRow, error)
		GetCommentTarget(ctx context.Context, r spec.CommentRef, tx ...*sql.Tx) (spec.CommentTarget, error)
		GetCommentAuthorID(ctx context.Context, r spec.CommentRef, tx ...*sql.Tx) (uuid.UUID, error)
		GetTargetOwnerID(ctx context.Context, t spec.CommentTarget, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentMediaDir(kind spec.CommentKind) (string, error)
		LikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error
		AddCommentMedia(ctx context.Context, s spec.NewCommentMedia, tx ...*sql.Tx) (int64, error)
		GetCommentMedia(ctx context.Context, r spec.CommentRef, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, s spec.CommentMediaBatch, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		UpdateCommentMediaURL(ctx context.Context, s spec.CommentMediaURLUpdate, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, s spec.CommentMediaURLUpdate, tx ...*sql.Tx) error
		CollectCommentMediaPaths(ctx context.Context, t spec.CommentTarget, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, r spec.CommentRef, tx ...*sql.Tx) ([]string, error)
	}
)
