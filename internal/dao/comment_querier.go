package dao

import (
	"context"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	commentQuerier[K comparable] interface {
		Create(ctx context.Context, q *sqlcgen.Queries, s spec.NewComment[K]) (*model.CommentRow, error)
		Update(ctx context.Context, q *sqlcgen.Queries, s spec.CommentUpdate) (int64, error)
		Delete(ctx context.Context, q *sqlcgen.Queries, s spec.CommentDeletion) (int64, error)
		Count(ctx context.Context, q *sqlcgen.Queries, target K, exclude []uuid.UUID) (int64, error)
		List(ctx context.Context, q *sqlcgen.Queries, s spec.CommentQuery[K]) ([]model.CommentRow, error)
		ByID(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (*model.CommentRow, error)
		EntityID(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (K, error)
		AuthorID(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (uuid.UUID, error)
		Like(ctx context.Context, q *sqlcgen.Queries, s spec.CommentLike) error
		Unlike(ctx context.Context, q *sqlcgen.Queries, s spec.CommentLike) error
		AddMedia(ctx context.Context, q *sqlcgen.Queries, s spec.NewMedia) (int64, error)
		Media(ctx context.Context, q *sqlcgen.Queries, commentID uuid.UUID) ([]model.PostMediaRow, error)
		MediaBatch(ctx context.Context, q *sqlcgen.Queries, ids []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error)
		SetMediaURL(ctx context.Context, q *sqlcgen.Queries, s spec.MediaURLUpdate) error
		SetMediaThumbnail(ctx context.Context, q *sqlcgen.Queries, s spec.MediaURLUpdate) error
		Paths(ctx context.Context, q *sqlcgen.Queries, target K) ([]string, error)
		SinglePaths(ctx context.Context, q *sqlcgen.Queries, commentID uuid.UUID) ([]string, error)
	}

	commentCreateRow = sqlcgen.CreatePostCommentRow
	commentListRow   = sqlcgen.ListPostCommentsRow
	commentMediaRow  = sqlcgen.GetPostCommentMediaRow
	mediaPathRow     = sqlcgen.CollectPostCommentMediaPathsRow
)

func commentRowFromCreate(row commentCreateRow) model.CommentRow {
	return model.CommentRow{
		ID:                row.ID,
		EntityID:          row.EntityID,
		ParentID:          row.ParentID,
		UserID:            row.UserID,
		Body:              row.Body,
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         nullTimeToStringPtr(row.UpdatedAt),
		AuthorUsername:    row.Username,
		AuthorDisplayName: row.DisplayName,
		AuthorAvatarURL:   row.AvatarUrl,
		AuthorRole:        row.AuthorRole,
		AuthorBanned:      row.AuthorBanned,
	}
}

func commentRowFromList(row commentListRow) model.CommentRow {
	out := commentRowFromCreate(commentCreateRow{
		ID:           row.ID,
		EntityID:     row.EntityID,
		ParentID:     row.ParentID,
		UserID:       row.UserID,
		Body:         row.Body,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
		Username:     row.Username,
		DisplayName:  row.DisplayName,
		AvatarUrl:    row.AvatarUrl,
		AuthorRole:   row.AuthorRole,
		AuthorBanned: row.AuthorBanned,
	})

	out.LikeCount = int(row.LikeCount)
	out.UserLiked = row.UserLiked

	return out
}

func mediaRowFromComment(row commentMediaRow) model.PostMediaRow {
	return model.PostMediaRow{
		ID:           int(row.ID),
		PostID:       row.CommentID,
		MediaURL:     row.MediaUrl,
		MediaType:    row.MediaType,
		ThumbnailURL: row.ThumbnailUrl,
		Filename:     row.Filename,
		SortOrder:    int(row.SortOrder),
		IsSpoiler:    row.IsSpoiler,
	}
}

func collectMediaPaths(rows []mediaPathRow) []string {
	var paths []string
	for _, row := range rows {
		if row.MediaUrl != "" {
			paths = append(paths, row.MediaUrl)
		}

		if row.ThumbnailUrl != "" {
			paths = append(paths, row.ThumbnailUrl)
		}
	}

	return paths
}
