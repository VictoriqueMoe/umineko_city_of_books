package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

func (m *mediaDAO) CollectMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	paths, err := m.q.Paths(ctx, m.queries(tx), entityID)
	if err != nil {
		return nil, fmt.Errorf("collect media paths: %w", err)
	}

	return paths, nil
}

func (c *commentDAO[K]) CollectCommentMediaPaths(ctx context.Context, entityID K, tx ...*sql.Tx) ([]string, error) {
	paths, err := c.q.Paths(ctx, c.queries(tx), entityID)
	if err != nil {
		return nil, fmt.Errorf("collect comment media paths: %w", err)
	}

	return paths, nil
}

func (c *commentDAO[K]) CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	paths, err := c.q.SinglePaths(ctx, c.queries(tx), commentID)
	if err != nil {
		return nil, fmt.Errorf("collect comment media paths: %w", err)
	}

	return paths, nil
}
