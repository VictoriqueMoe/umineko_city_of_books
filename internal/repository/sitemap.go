package repository

import (
	"context"
	"database/sql"
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
)
