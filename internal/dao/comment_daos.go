package dao

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	CommentDAO[K comparable] interface {
		CreateComment(ctx context.Context, s spec.NewComment[K], tx ...*sql.Tx) (*model.CommentRow, error)
		UpdateComment(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) error
		GetComments(ctx context.Context, q spec.CommentQuery[K], tx ...*sql.Tx) ([]model.CommentRow, int, error)
		GetCommentByID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (*model.CommentRow, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (K, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		LikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error
		AddCommentMedia(ctx context.Context, s spec.NewMedia, tx ...*sql.Tx) (int64, error)
		GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		UpdateCommentMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		CollectCommentMediaPaths(ctx context.Context, entityID K, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)
	}

	JournalCommentWriter interface {
		CreateComment(ctx context.Context, s spec.NewJournalComment, tx ...*sql.Tx) (*model.CommentRow, error)
	}

	CommentDAOs struct {
		ByID    map[string]CommentDAO[uuid.UUID]
		BySlug  map[string]CommentDAO[string]
		Journal JournalCommentWriter
	}
)
