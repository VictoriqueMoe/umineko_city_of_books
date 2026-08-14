package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/repository"
)

type (
	sitemapDAO struct {
		db *sql.DB
	}
)

func (r *sitemapDAO) listEntries(ctx context.Context, query, label string, tx ...*sql.Tx) ([]repository.SitemapEntry, error) {
	rows, err := getDb(r.db, tx).QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", label, err)
	}
	defer rows.Close()

	var entries []repository.SitemapEntry
	for rows.Next() {
		var e repository.SitemapEntry
		if err := rows.Scan(&e.ID, &e.LastMod); err != nil {
			return nil, fmt.Errorf("scan %s: %w", label, err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *sitemapDAO) ListTheories(ctx context.Context, tx ...*sql.Tx) ([]repository.SitemapEntry, error) {
	return r.listEntries(ctx, `SELECT id, created_at FROM theories ORDER BY created_at DESC`, "theories", tx...)
}

func (r *sitemapDAO) ListPosts(ctx context.Context, tx ...*sql.Tx) ([]repository.SitemapEntry, error) {
	return r.listEntries(ctx, `SELECT id, created_at FROM posts ORDER BY created_at DESC`, "posts", tx...)
}

func (r *sitemapDAO) ListArt(ctx context.Context, tx ...*sql.Tx) ([]repository.SitemapEntry, error) {
	return r.listEntries(ctx, `SELECT id, created_at FROM art ORDER BY created_at DESC`, "art", tx...)
}

func (r *sitemapDAO) ListMysteries(ctx context.Context, tx ...*sql.Tx) ([]repository.SitemapEntry, error) {
	return r.listEntries(ctx, `SELECT id, created_at FROM mysteries ORDER BY created_at DESC`, "mysteries", tx...)
}

func (r *sitemapDAO) ListShips(ctx context.Context, tx ...*sql.Tx) ([]repository.SitemapEntry, error) {
	return r.listEntries(ctx, `SELECT id, created_at FROM ships ORDER BY created_at DESC`, "ships", tx...)
}

func (r *sitemapDAO) ListFanfics(ctx context.Context, tx ...*sql.Tx) ([]repository.SitemapEntry, error) {
	return r.listEntries(ctx, `SELECT id, created_at FROM fanfics ORDER BY created_at DESC`, "fanfics", tx...)
}

func (r *sitemapDAO) ListUsernames(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	rows, err := getDb(r.db, tx).QueryContext(ctx, `SELECT username FROM users WHERE NOT is_bot ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list usernames: %w", err)
	}
	defer rows.Close()

	var usernames []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, fmt.Errorf("scan username: %w", err)
		}
		usernames = append(usernames, u)
	}
	return usernames, rows.Err()
}

func (r *sitemapDAO) ListJournalRows(ctx context.Context, tx ...*sql.Tx) ([]repository.SitemapJournalRow, error) {
	rows, err := getDb(r.db, tx).QueryContext(ctx,
		`SELECT j.id, COALESCE(j.updated_at, j.created_at), e.entry_number, e.updated_at
		FROM journals j
		LEFT JOIN journal_entries e ON e.journal_id = j.id AND NOT e.is_draft
		WHERE j.archived_at IS NULL
		ORDER BY j.id, e.entry_number`)
	if err != nil {
		return nil, fmt.Errorf("list journal rows: %w", err)
	}
	defer rows.Close()

	var result []repository.SitemapJournalRow
	for rows.Next() {
		var row repository.SitemapJournalRow
		if err := rows.Scan(&row.JournalID, &row.JournalUpdatedAt, &row.EntryNumber, &row.EntryUpdatedAt); err != nil {
			return nil, fmt.Errorf("scan journal row: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
