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

type sitemapRepository struct {
	dao SitemapRepository
}

func NewSitemapRepo(dao SitemapRepository) SitemapRepository {
	return &sitemapRepository{dao: dao}
}

func (r *sitemapRepository) ListTheories(ctx context.Context) ([]SitemapEntry, error) {
	return r.dao.ListTheories(ctx)
}

func (r *sitemapRepository) ListPosts(ctx context.Context) ([]SitemapEntry, error) {
	return r.dao.ListPosts(ctx)
}

func (r *sitemapRepository) ListArt(ctx context.Context) ([]SitemapEntry, error) {
	return r.dao.ListArt(ctx)
}

func (r *sitemapRepository) ListUsernames(ctx context.Context) ([]string, error) {
	return r.dao.ListUsernames(ctx)
}

func (r *sitemapRepository) ListMysteries(ctx context.Context) ([]SitemapEntry, error) {
	return r.dao.ListMysteries(ctx)
}

func (r *sitemapRepository) ListShips(ctx context.Context) ([]SitemapEntry, error) {
	return r.dao.ListShips(ctx)
}

func (r *sitemapRepository) ListFanfics(ctx context.Context) ([]SitemapEntry, error) {
	return r.dao.ListFanfics(ctx)
}

func (r *sitemapRepository) ListJournalRows(ctx context.Context) ([]SitemapJournalRow, error) {
	return r.dao.ListJournalRows(ctx)
}
