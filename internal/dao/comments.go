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
	commentDAO struct {
		sqlcSource
		tables map[spec.CommentKind]commentQuerier
	}
)

func (c *commentDAO) table(kind spec.CommentKind) (commentQuerier, error) {
	q, ok := c.tables[kind]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownCommentKind, kind)
	}

	return q, nil
}

func (c *commentDAO) CreateComment(ctx context.Context, s spec.NewComment, tx ...*sql.Tx) (*model.CommentRow, error) {
	table, err := c.table(s.Target.Kind)
	if err != nil {
		return nil, err
	}

	row, err := table.Create(ctx, c.queries(tx), s)
	if err != nil {
		return nil, fmt.Errorf("create comment: %w", err)
	}

	return row, nil
}

func (c *commentDAO) UpdateComment(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error {
	table, err := c.table(s.Kind)
	if err != nil {
		return err
	}

	n, err := table.Update(ctx, c.queries(tx), s)
	if err != nil {
		return fmt.Errorf("update comment: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("comment not found or not owned: %w", ErrNotFound)
	}

	return nil
}

func (c *commentDAO) DeleteComment(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) error {
	table, err := c.table(s.Kind)
	if err != nil {
		return err
	}

	n, err := table.Delete(ctx, c.queries(tx), s)
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}

	if n == 0 && !s.AsAdmin {
		return fmt.Errorf("comment not found or not owned: %w", ErrNotFound)
	}

	return nil
}

func (c *commentDAO) GetComments(ctx context.Context, q spec.CommentQuery, tx ...*sql.Tx) ([]model.CommentRow, int, error) {
	table, err := c.table(q.Target.Kind)
	if err != nil {
		return nil, 0, err
	}

	queries := c.queries(tx)

	total, err := table.Count(ctx, queries, q.Target, q.ExcludeUserIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("count comments: %w", err)
	}

	rows, err := table.List(ctx, queries, q)
	if err != nil {
		return nil, 0, fmt.Errorf("get comments: %w", err)
	}

	return rows, int(total), nil
}

func (c *commentDAO) GetCommentByID(ctx context.Context, r spec.CommentRef, tx ...*sql.Tx) (*model.CommentRow, error) {
	table, err := c.table(r.Kind)
	if err != nil {
		return nil, err
	}

	row, err := table.ByID(ctx, c.queries(tx), r.CommentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get comment by id: %w", err)
	}

	return row, nil
}

func (c *commentDAO) GetCommentTarget(ctx context.Context, r spec.CommentRef, tx ...*sql.Tx) (spec.CommentTarget, error) {
	table, err := c.table(r.Kind)
	if err != nil {
		return spec.CommentTarget{}, err
	}

	target, err := table.Target(ctx, c.queries(tx), r.CommentID)
	if errors.Is(err, sql.ErrNoRows) {
		return spec.CommentTarget{}, fmt.Errorf("get comment target: %w", ErrNotFound)
	}
	if err != nil {
		return spec.CommentTarget{}, fmt.Errorf("get comment target: %w", err)
	}

	if target.Kind == "" {
		target.Kind = r.Kind
	}

	return target, nil
}

func (c *commentDAO) GetCommentAuthorID(ctx context.Context, r spec.CommentRef, tx ...*sql.Tx) (uuid.UUID, error) {
	table, err := c.table(r.Kind)
	if err != nil {
		return uuid.Nil, err
	}

	userID, err := table.AuthorID(ctx, c.queries(tx), r.CommentID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("get comment author: %w", ErrNotFound)
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("get comment author: %w", err)
	}

	return userID, nil
}

func (c *commentDAO) GetTargetOwnerID(ctx context.Context, t spec.CommentTarget, tx ...*sql.Tx) (uuid.UUID, error) {
	table, err := c.table(t.Kind)
	if err != nil {
		return uuid.Nil, err
	}

	ownerID, err := table.Owner(ctx, c.queries(tx), t)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("get comment target owner: %w", ErrNotFound)
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("get comment target owner: %w", err)
	}

	return ownerID, nil
}

func (c *commentDAO) GetCommentMediaDir(kind spec.CommentKind) (string, error) {
	table, err := c.table(kind)
	if err != nil {
		return "", err
	}

	return table.MediaDir(), nil
}

func (c *commentDAO) LikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error {
	table, err := c.table(s.Kind)
	if err != nil {
		return err
	}

	if err := table.Like(ctx, c.queries(tx), s); err != nil {
		return fmt.Errorf("like comment: %w", err)
	}

	return nil
}

func (c *commentDAO) UnlikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error {
	table, err := c.table(s.Kind)
	if err != nil {
		return err
	}

	if err := table.Unlike(ctx, c.queries(tx), s); err != nil {
		return fmt.Errorf("unlike comment: %w", err)
	}

	return nil
}

func (c *commentDAO) AddCommentMedia(ctx context.Context, s spec.NewCommentMedia, tx ...*sql.Tx) (int64, error) {
	table, err := c.table(s.Kind)
	if err != nil {
		return 0, err
	}

	id, err := table.AddMedia(ctx, c.queries(tx), s.NewMedia)
	if err != nil {
		return 0, fmt.Errorf("add comment media: %w", err)
	}

	return id, nil
}

func (c *commentDAO) GetCommentMedia(ctx context.Context, r spec.CommentRef, tx ...*sql.Tx) ([]model.PostMediaRow, error) {
	table, err := c.table(r.Kind)
	if err != nil {
		return nil, err
	}

	rows, err := table.Media(ctx, c.queries(tx), r.CommentID)
	if err != nil {
		return nil, fmt.Errorf("get comment media: %w", err)
	}

	return rows, nil
}

func (c *commentDAO) GetCommentMediaBatch(ctx context.Context, s spec.CommentMediaBatch, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error) {
	if len(s.CommentIDs) == 0 {
		return nil, nil
	}

	table, err := c.table(s.Kind)
	if err != nil {
		return nil, err
	}

	rows, err := table.MediaBatch(ctx, c.queries(tx), s.CommentIDs)
	if err != nil {
		return nil, fmt.Errorf("batch get comment media: %w", err)
	}

	return rows, nil
}

func (c *commentDAO) UpdateCommentMediaURL(ctx context.Context, s spec.CommentMediaURLUpdate, tx ...*sql.Tx) error {
	table, err := c.table(s.Kind)
	if err != nil {
		return err
	}

	n, err := table.SetMediaURL(ctx, c.queries(tx), s.MediaURLUpdate)
	if err != nil {
		return fmt.Errorf("update comment media url: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("update comment media url %d: %w", s.ID, ErrNotFound)
	}

	return nil
}

func (c *commentDAO) UpdateCommentMediaThumbnail(ctx context.Context, s spec.CommentMediaURLUpdate, tx ...*sql.Tx) error {
	table, err := c.table(s.Kind)
	if err != nil {
		return err
	}

	n, err := table.SetMediaThumbnail(ctx, c.queries(tx), s.MediaURLUpdate)
	if err != nil {
		return fmt.Errorf("update comment media thumbnail: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("update comment media thumbnail %d: %w", s.ID, ErrNotFound)
	}

	return nil
}

func (c *commentDAO) CollectCommentMediaPaths(ctx context.Context, t spec.CommentTarget, tx ...*sql.Tx) ([]string, error) {
	table, err := c.table(t.Kind)
	if err != nil {
		return nil, err
	}

	paths, err := table.Paths(ctx, c.queries(tx), t)
	if err != nil {
		return nil, fmt.Errorf("collect comment media paths: %w", err)
	}

	return paths, nil
}

func (c *commentDAO) CollectSingleCommentMediaPaths(ctx context.Context, r spec.CommentRef, tx ...*sql.Tx) ([]string, error) {
	table, err := c.table(r.Kind)
	if err != nil {
		return nil, err
	}

	paths, err := table.SinglePaths(ctx, c.queries(tx), r.CommentID)
	if err != nil {
		return nil, fmt.Errorf("collect comment media paths: %w", err)
	}

	return paths, nil
}
