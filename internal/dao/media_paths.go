package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

func scanMediaPaths(rows *sql.Rows, table string) ([]string, error) {
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var mediaURL, thumbnailURL string
		if err := rows.Scan(&mediaURL, &thumbnailURL); err != nil {
			return nil, fmt.Errorf("scan media path in %s: %w", table, err)
		}

		if mediaURL != "" {
			paths = append(paths, mediaURL)
		}

		if thumbnailURL != "" {
			paths = append(paths, thumbnailURL)
		}
	}

	return paths, rows.Err()
}

func (m *mediaDAO) CollectMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(m.db, tx).QueryContext(ctx,
		`SELECT media_url, thumbnail_url FROM `+m.table+` WHERE `+m.fk+` = $1`, entityID,
	)
	if err != nil {
		return nil, fmt.Errorf("collect media paths in %s: %w", m.table, err)
	}

	return scanMediaPaths(rows, m.table)
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
