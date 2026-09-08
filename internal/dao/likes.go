package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	likeDAO struct {
		sqlcSource
		q likeQuerier
	}
)

func newLikeDAO(db *sql.DB, q likeQuerier) *likeDAO {
	return &likeDAO{sqlcSource: newSQLCSource(db), q: q}
}

func (l *likeDAO) Like(ctx context.Context, s spec.Like, tx ...*sql.Tx) error {
	if err := l.q.Like(ctx, l.queries(tx), s); err != nil {
		return fmt.Errorf("like: %w", err)
	}

	return nil
}

func (l *likeDAO) Unlike(ctx context.Context, s spec.Like, tx ...*sql.Tx) error {
	if err := l.q.Unlike(ctx, l.queries(tx), s); err != nil {
		return fmt.Errorf("unlike: %w", err)
	}

	return nil
}

func (l *likeDAO) GetLikedBy(ctx context.Context, q spec.LikedByQuery, tx ...*sql.Tx) ([]model.PostLikeUser, error) {
	users, err := l.q.LikedBy(ctx, l.queries(tx), q)
	if err != nil {
		return nil, fmt.Errorf("get liked by: %w", err)
	}

	return users, nil
}
