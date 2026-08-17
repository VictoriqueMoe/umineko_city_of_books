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
	rows, err := txOrDB(c.db, tx).QueryContext(ctx,
		`SELECT m.media_url, m.thumbnail_url
		 FROM `+c.mediaTable+` m
		 JOIN `+c.table+` cm ON cm.id = m.comment_id
		 WHERE cm.`+c.fk+` = $1`, entityID,
	)
	if err != nil {
		return nil, fmt.Errorf("collect comment media paths in %s: %w", c.mediaTable, err)
	}

	return scanMediaPaths(rows, c.mediaTable)
}

func (c *commentDAO[K]) CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(c.db, tx).QueryContext(ctx,
		`WITH RECURSIVE tree AS (
		     SELECT id FROM `+c.table+` WHERE id = $1
		     UNION ALL
		     SELECT child.id FROM `+c.table+` child JOIN tree ON child.parent_id = tree.id
		 )
		 SELECT m.media_url, m.thumbnail_url
		 FROM `+c.mediaTable+` m
		 JOIN tree ON tree.id = m.comment_id`, commentID,
	)
	if err != nil {
		return nil, fmt.Errorf("collect comment media paths in %s: %w", c.mediaTable, err)
	}

	return scanMediaPaths(rows, c.mediaTable)
}
