package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	commentDAO[K comparable] struct {
		sqlcSource
		q commentQuerier[K]
	}
)

func newCommentDAO[K comparable](db *sql.DB, q commentQuerier[K]) *commentDAO[K] {
	return &commentDAO[K]{sqlcSource: newSQLCSource(db), q: q}
}

func (c *commentDAO[K]) CreateComment(ctx context.Context, s spec.NewComment[K], tx ...*sql.Tx) (*model.CommentRow, error) {
	row, err := c.q.Create(ctx, c.queries(tx), s)
	if err != nil {
		return nil, fmt.Errorf("create comment: %w", err)
	}

	return row, nil
}

func (c *commentDAO[K]) UpdateComment(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error {
	n, err := c.q.Update(ctx, c.queries(tx), s)
	if err != nil {
		return fmt.Errorf("update comment: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}

	return nil
}

func (c *commentDAO[K]) DeleteComment(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) error {
	n, err := c.q.Delete(ctx, c.queries(tx), s)
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}

	if n == 0 && !s.AsAdmin {
		return fmt.Errorf("comment not found or not owned")
	}

	return nil
}

func (c *commentDAO[K]) GetComments(ctx context.Context, q spec.CommentQuery[K], tx ...*sql.Tx) ([]model.CommentRow, int, error) {
	queries := c.queries(tx)

	total, err := c.q.Count(ctx, queries, q.TargetID, q.ExcludeUserIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("count comments: %w", err)
	}

	rows, err := c.q.List(ctx, queries, q)
	if err != nil {
		return nil, 0, fmt.Errorf("get comments: %w", err)
	}

	return rows, int(total), nil
}

func (c *commentDAO[K]) GetCommentByID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (*model.CommentRow, error) {
	row, err := c.q.ByID(ctx, c.queries(tx), commentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get comment by id: %w", err)
	}

	return row, nil
}

func (c *commentDAO[K]) GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (K, error) {
	entityID, err := c.q.EntityID(ctx, c.queries(tx), commentID)
	if err != nil {
		var zero K
		return zero, fmt.Errorf("get comment entity id: %w", err)
	}

	return entityID, nil
}

func (c *commentDAO[K]) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	userID, err := c.q.AuthorID(ctx, c.queries(tx), commentID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get comment author: %w", err)
	}

	return userID, nil
}

func (c *commentDAO[K]) LikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error {
	if err := c.q.Like(ctx, c.queries(tx), s); err != nil {
		return fmt.Errorf("like comment: %w", err)
	}

	return nil
}

func (c *commentDAO[K]) UnlikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error {
	if err := c.q.Unlike(ctx, c.queries(tx), s); err != nil {
		return fmt.Errorf("unlike comment: %w", err)
	}

	return nil
}

func (c *commentDAO[K]) AddCommentMedia(ctx context.Context, s spec.NewMedia, tx ...*sql.Tx) (int64, error) {
	id, err := c.q.AddMedia(ctx, c.queries(tx), s)
	if err != nil {
		return 0, fmt.Errorf("add comment media: %w", err)
	}

	return id, nil
}

func (c *commentDAO[K]) GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error) {
	rows, err := c.q.Media(ctx, c.queries(tx), commentID)
	if err != nil {
		return nil, fmt.Errorf("get comment media: %w", err)
	}

	return rows, nil
}

func (c *commentDAO[K]) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error) {
	if len(commentIDs) == 0 {
		return nil, nil
	}

	rows, err := c.q.MediaBatch(ctx, c.queries(tx), commentIDs)
	if err != nil {
		return nil, fmt.Errorf("batch get comment media: %w", err)
	}

	return rows, nil
}

func (c *commentDAO[K]) UpdateCommentMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error {
	if err := c.q.SetMediaURL(ctx, c.queries(tx), s); err != nil {
		return fmt.Errorf("update comment media url: %w", err)
	}

	return nil
}

func (c *commentDAO[K]) UpdateCommentMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error {
	if err := c.q.SetMediaThumbnail(ctx, c.queries(tx), s); err != nil {
		return fmt.Errorf("update comment media thumbnail: %w", err)
	}

	return nil
}
