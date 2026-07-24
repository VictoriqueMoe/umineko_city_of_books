package dao

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"umineko_city_of_books/internal/repository"
)

type (
	searchDAO struct {
		db *sql.DB
	}
)

func (r *searchDAO) Search(ctx context.Context, query string, types []repository.SearchEntityType, limit, offset int) ([]repository.SearchResult, int, error) {
	srcs := repository.ResolveSearchTypes(types)
	if len(srcs) == 0 {
		return nil, 0, nil
	}

	subqueries := make([]string, len(srcs))
	for i, src := range srcs {
		subqueries[i] = src.BuildSubquery()
	}
	union := strings.Join(subqueries, "\nUNION ALL\n")

	countSQL := fmt.Sprintf(`WITH q AS (SELECT websearch_to_tsquery('english', $1) AS tsq, $1 AS qstr)
        SELECT COUNT(*) FROM (%s) results`, union)

	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, query).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("search count: %w", err)
	}

	dataSQL := fmt.Sprintf(`WITH q AS (SELECT websearch_to_tsquery('english', $1) AS tsq, $1 AS qstr)
        SELECT entity_type, id, parent_id, parent_title, title, snippet,
               author_id, author_username, author_display_name, author_avatar_url, created_at, rank
        FROM (%s) results
        ORDER BY rank DESC, created_at DESC
        LIMIT $2 OFFSET $3`, union)

	rows, err := r.db.QueryContext(ctx, dataSQL, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	results, err := scanSearchRows(rows, limit)
	if err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (r *searchDAO) QuickSearch(ctx context.Context, query string, perTypeLimit int) ([]repository.SearchResult, error) {
	sources := repository.SearchSources()

	subqueries := make([]string, len(sources))
	for i, src := range sources {
		subqueries[i] = fmt.Sprintf(`(SELECT * FROM (%s) sub ORDER BY rank DESC, created_at DESC LIMIT %d)`, src.BuildSubquery(), perTypeLimit)
	}
	union := strings.Join(subqueries, "\nUNION ALL\n")

	sqlStr := fmt.Sprintf(`WITH q AS (SELECT websearch_to_tsquery('english', $1) AS tsq, $1 AS qstr)
        SELECT entity_type, id, parent_id, parent_title, title, snippet,
               author_id, author_username, author_display_name, author_avatar_url, created_at, rank
        FROM (%s) results
        ORDER BY rank DESC, created_at DESC`, union)

	rows, err := r.db.QueryContext(ctx, sqlStr, query)
	if err != nil {
		return nil, fmt.Errorf("quick search: %w", err)
	}
	defer rows.Close()

	return scanSearchRows(rows, perTypeLimit*len(sources))
}

func scanSearchRowsWithTotal(rows *sql.Rows, capacity int) ([]repository.SearchResult, int, error) {
	results := make([]repository.SearchResult, 0, capacity)
	total := 0

	for rows.Next() {
		var (
			r         repository.SearchResult
			createdAt time.Time
			entityT   string
			rowTotal  int
		)
		if err := rows.Scan(
			&entityT, &r.ID, &r.ParentID, &r.ParentTitle, &r.Title, &r.Snippet,
			&r.AuthorID, &r.AuthorUsername, &r.AuthorDisplayName, &r.AuthorAvatarURL,
			&createdAt, &r.Rank, &rowTotal,
		); err != nil {
			return nil, 0, fmt.Errorf("search scan: %w", err)
		}

		r.EntityType = repository.SearchEntityType(entityT)
		r.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		results = append(results, r)
		total = rowTotal
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("search rows: %w", err)
	}

	return results, total, nil
}

func scanSearchRows(rows *sql.Rows, capacity int) ([]repository.SearchResult, error) {
	results := make([]repository.SearchResult, 0, capacity)
	for rows.Next() {
		var (
			r         repository.SearchResult
			createdAt time.Time
			entityT   string
		)
		if err := rows.Scan(
			&entityT, &r.ID, &r.ParentID, &r.ParentTitle, &r.Title, &r.Snippet,
			&r.AuthorID, &r.AuthorUsername, &r.AuthorDisplayName, &r.AuthorAvatarURL,
			&createdAt, &r.Rank,
		); err != nil {
			return nil, fmt.Errorf("search scan: %w", err)
		}
		r.EntityType = repository.SearchEntityType(entityT)
		r.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search rows: %w", err)
	}
	return results, nil
}
