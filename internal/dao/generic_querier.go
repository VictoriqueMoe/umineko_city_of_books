package dao

import (
	"context"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	likeQuerier interface {
		Like(ctx context.Context, q *sqlcgen.Queries, s spec.Like) error
		Unlike(ctx context.Context, q *sqlcgen.Queries, s spec.Like) error
		LikedBy(ctx context.Context, q *sqlcgen.Queries, s spec.LikedByQuery) ([]model.PostLikeUser, error)
	}

	viewQuerier interface {
		Record(ctx context.Context, q *sqlcgen.Queries, s spec.ViewRecord) (int64, error)
	}

	voteQuerier interface {
		Clear(ctx context.Context, q *sqlcgen.Queries, s spec.Vote) error
		Upsert(ctx context.Context, q *sqlcgen.Queries, s spec.Vote) error
	}

	ownedQuerier interface {
		Delete(ctx context.Context, q *sqlcgen.Queries, s spec.OwnedDeletion) (int64, error)
		DeleteAsAdmin(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) error
		AuthorID(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (uuid.UUID, error)
		IncrementViewCount(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) error
	}

	mediaQuerier interface {
		Add(ctx context.Context, q *sqlcgen.Queries, s spec.NewMedia) (int64, error)
		URL(ctx context.Context, q *sqlcgen.Queries, s spec.MediaDeletion) (string, error)
		DeleteRow(ctx context.Context, q *sqlcgen.Queries, s spec.MediaDeletion) error
		SetURL(ctx context.Context, q *sqlcgen.Queries, s spec.MediaURLUpdate) error
		SetThumbnail(ctx context.Context, q *sqlcgen.Queries, s spec.MediaURLUpdate) error
		Media(ctx context.Context, q *sqlcgen.Queries, targetID uuid.UUID) ([]model.PostMediaRow, error)
		MediaBatch(ctx context.Context, q *sqlcgen.Queries, targetIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error)
		Paths(ctx context.Context, q *sqlcgen.Queries, targetID uuid.UUID) ([]string, error)
	}

	likeUserRow    = sqlcgen.GetPostLikedByRow
	entityMediaRow = sqlcgen.GetPostMediaRow
)

func likeUserFromRow(row likeUserRow) model.PostLikeUser {
	return model.PostLikeUser{
		ID:          row.ID,
		Username:    row.Username,
		DisplayName: row.DisplayName,
		AvatarURL:   row.AvatarUrl,
		Role:        row.Role,
	}
}

func mediaRowFromEntity(row entityMediaRow) model.PostMediaRow {
	return model.PostMediaRow{
		ID:           int(row.ID),
		PostID:       row.TargetID,
		MediaURL:     row.MediaUrl,
		MediaType:    row.MediaType,
		ThumbnailURL: row.ThumbnailUrl,
		Filename:     row.Filename,
		SortOrder:    int(row.SortOrder),
		IsSpoiler:    row.IsSpoiler,
	}
}
