package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	AnnouncementDAO interface {
		Create(ctx context.Context, s spec.NewAnnouncement, tx ...*sql.Tx) (*model.AnnouncementRow, error)
		Update(ctx context.Context, s spec.AnnouncementUpdate, tx ...*sql.Tx) error
		Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.AnnouncementRow, error)
		List(ctx context.Context, q spec.AnnouncementListQuery, tx ...*sql.Tx) ([]model.AnnouncementRow, int, error)
		GetLatest(ctx context.Context, tx ...*sql.Tx) (*model.AnnouncementRow, error)
		SetPinned(ctx context.Context, s spec.AnnouncementPinUpdate, tx ...*sql.Tx) error

		UpdateComment(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) error
		GetComments(ctx context.Context, q spec.CommentQuery[uuid.UUID], tx ...*sql.Tx) ([]model.CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		LikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error

		AddCommentMedia(ctx context.Context, s spec.NewMedia, tx ...*sql.Tx) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)
	}

	announcementDAO struct {
		db  *sql.DB
		gen *sqlcgen.Queries
		*commentDAO[uuid.UUID]
	}

	announcementJoinRow = sqlcgen.GetAnnouncementByIDRow
)

func (r *announcementDAO) q(tx []*sql.Tx) *sqlcgen.Queries {
	if len(tx) > 0 && tx[0] != nil {
		return r.gen.WithTx(tx[0])
	}

	return r.gen
}

func toAnnouncementRow(row announcementJoinRow) model.AnnouncementRow {
	out := model.AnnouncementRow{
		ID:                row.ID,
		Title:             row.Title,
		Body:              row.Body,
		Pinned:            row.Pinned,
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         row.UpdatedAt.UTC().Format(time.RFC3339),
		AuthorUsername:    row.AuthorUsername,
		AuthorDisplayName: row.AuthorDisplayName,
		AuthorAvatarURL:   row.AuthorAvatarUrl,
		AuthorRole:        row.AuthorRole,
	}

	if row.AuthorID != nil {
		out.AuthorID = *row.AuthorID
	}

	return out
}

func (r *announcementDAO) Create(ctx context.Context, s spec.NewAnnouncement, tx ...*sql.Tx) (*model.AnnouncementRow, error) {
	created, err := r.q(tx).CreateAnnouncement(ctx, sqlcgen.CreateAnnouncementParams{
		AuthorID: &s.AuthorID,
		Title:    s.Title,
		Body:     s.Body,
	})
	if err != nil {
		return nil, fmt.Errorf("create announcement: %w", err)
	}

	return new(toAnnouncementRow(announcementJoinRow(created))), nil
}

func (r *announcementDAO) Update(ctx context.Context, s spec.AnnouncementUpdate, tx ...*sql.Tx) error {
	err := r.q(tx).UpdateAnnouncement(ctx, sqlcgen.UpdateAnnouncementParams{
		Title: s.Title,
		Body:  s.Body,
		ID:    s.ID,
	})
	if err != nil {
		return fmt.Errorf("update announcement: %w", err)
	}

	return nil
}

func (r *announcementDAO) Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	if err := r.q(tx).DeleteAnnouncement(ctx, id); err != nil {
		return fmt.Errorf("delete announcement: %w", err)
	}

	return nil
}

func (r *announcementDAO) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.AnnouncementRow, error) {
	row, err := r.q(tx).GetAnnouncementByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get announcement: %w", err)
	}

	return new(toAnnouncementRow(row)), nil
}

func (r *announcementDAO) List(ctx context.Context, q spec.AnnouncementListQuery, tx ...*sql.Tx) ([]model.AnnouncementRow, int, error) {
	queries := r.q(tx)

	total, err := queries.CountAnnouncements(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count announcements: %w", err)
	}

	rows, err := queries.ListAnnouncements(ctx, sqlcgen.ListAnnouncementsParams{
		Limit:  int32(q.Limit),
		Offset: int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list announcements: %w", err)
	}

	var result []model.AnnouncementRow
	for _, row := range rows {
		result = append(result, toAnnouncementRow(announcementJoinRow(row)))
	}

	return result, int(total), nil
}

func (r *announcementDAO) GetLatest(ctx context.Context, tx ...*sql.Tx) (*model.AnnouncementRow, error) {
	row, err := r.q(tx).GetLatestAnnouncement(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest announcement: %w", err)
	}

	return new(toAnnouncementRow(announcementJoinRow(row))), nil
}

func (r *announcementDAO) SetPinned(ctx context.Context, s spec.AnnouncementPinUpdate, tx ...*sql.Tx) error {
	err := r.q(tx).SetAnnouncementPinned(ctx, sqlcgen.SetAnnouncementPinnedParams{
		Pinned: s.Pinned,
		ID:     s.ID,
	})
	if err != nil {
		return fmt.Errorf("set pinned: %w", err)
	}

	return nil
}
