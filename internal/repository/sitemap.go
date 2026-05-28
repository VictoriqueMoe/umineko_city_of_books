package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type (
	SitemapEntry struct {
		ID      string
		LastMod time.Time
	}

	SitemapJournalRow struct {
		JournalID        string
		JournalUpdatedAt time.Time
		EntryNumber      sql.NullInt64
		EntryUpdatedAt   sql.NullTime
	}

	SitemapRepository interface {
		ListTheories(ctx context.Context) ([]SitemapEntry, error)
		ListPosts(ctx context.Context) ([]SitemapEntry, error)
		ListArt(ctx context.Context) ([]SitemapEntry, error)
		ListUsernames(ctx context.Context) ([]string, error)
		ListMysteries(ctx context.Context) ([]SitemapEntry, error)
		ListShips(ctx context.Context) ([]SitemapEntry, error)
		ListFanfics(ctx context.Context) ([]SitemapEntry, error)
		ListJournalRows(ctx context.Context) ([]SitemapJournalRow, error)
	}

	sitemapRepository struct {
		db *sql.DB
	}
)

func (r *sitemapRepository) listEntries(ctx context.Context, query, label string) ([]SitemapEntry, error) {
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", label, err)
	}
	defer rows.Close()

	var entries []SitemapEntry
	for rows.Next() {
		var e SitemapEntry
		if err := rows.Scan(&e.ID, &e.LastMod); err != nil {
			return nil, fmt.Errorf("scan %s: %w", label, err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *sitemapRepository) ListTheories(ctx context.Context) ([]SitemapEntry, error) {
	return r.listEntries(ctx, `SELECT id, created_at FROM theories ORDER BY created_at DESC`, "theories")
}

func (r *sitemapRepository) ListPosts(ctx context.Context) ([]SitemapEntry, error) {
	return r.listEntries(ctx, `SELECT id, created_at FROM posts ORDER BY created_at DESC`, "posts")
}

func (r *sitemapRepository) ListArt(ctx context.Context) ([]SitemapEntry, error) {
	return r.listEntries(ctx, `SELECT id, created_at FROM art ORDER BY created_at DESC`, "art")
}

func (r *sitemapRepository) ListMysteries(ctx context.Context) ([]SitemapEntry, error) {
	return r.listEntries(ctx, `SELECT id, created_at FROM mysteries ORDER BY created_at DESC`, "mysteries")
}

func (r *sitemapRepository) ListShips(ctx context.Context) ([]SitemapEntry, error) {
	return r.listEntries(ctx, `SELECT id, created_at FROM ships ORDER BY created_at DESC`, "ships")
}

func (r *sitemapRepository) ListFanfics(ctx context.Context) ([]SitemapEntry, error) {
	return r.listEntries(ctx, `SELECT id, created_at FROM fanfics ORDER BY created_at DESC`, "fanfics")
}

func (r *sitemapRepository) ListUsernames(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT username FROM users ORDER BY created_at DESC`)
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

func (r *sitemapRepository) ListJournalRows(ctx context.Context) ([]SitemapJournalRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT j.id, j.updated_at, e.entry_number, e.updated_at
		FROM journals j
		LEFT JOIN journal_entries e ON e.journal_id = j.id AND NOT e.is_draft
		WHERE j.archived_at IS NULL
		ORDER BY j.id, e.entry_number`)
	if err != nil {
		return nil, fmt.Errorf("list journal rows: %w", err)
	}
	defer rows.Close()

	var result []SitemapJournalRow
	for rows.Next() {
		var row SitemapJournalRow
		if err := rows.Scan(&row.JournalID, &row.JournalUpdatedAt, &row.EntryNumber, &row.EntryUpdatedAt); err != nil {
			return nil, fmt.Errorf("scan journal row: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
