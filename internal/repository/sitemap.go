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
		ListTheories(ctx context.Context, tx ...*sql.Tx) ([]SitemapEntry, error)
		ListPosts(ctx context.Context, tx ...*sql.Tx) ([]SitemapEntry, error)
		ListArt(ctx context.Context, tx ...*sql.Tx) ([]SitemapEntry, error)
		ListUsernames(ctx context.Context, tx ...*sql.Tx) ([]string, error)
		ListMysteries(ctx context.Context, tx ...*sql.Tx) ([]SitemapEntry, error)
		ListShips(ctx context.Context, tx ...*sql.Tx) ([]SitemapEntry, error)
		ListFanfics(ctx context.Context, tx ...*sql.Tx) ([]SitemapEntry, error)
		ListJournalRows(ctx context.Context, tx ...*sql.Tx) ([]SitemapJournalRow, error)
	}
)

type sitemapRepository struct {
	dao SitemapRepository
}

func NewSitemapRepo(dao SitemapRepository) SitemapRepository {
	return &sitemapRepository{dao: dao}
}

func (r *sitemapRepository) ListTheories(ctx context.Context, tx ...*sql.Tx) ([]SitemapEntry, error) {
	return r.dao.ListTheories(ctx, tx...)
}

func (r *sitemapRepository) ListPosts(ctx context.Context, tx ...*sql.Tx) ([]SitemapEntry, error) {
	return r.dao.ListPosts(ctx, tx...)
}

func (r *sitemapRepository) ListArt(ctx context.Context, tx ...*sql.Tx) ([]SitemapEntry, error) {
	return r.dao.ListArt(ctx, tx...)
}

func (r *sitemapRepository) ListUsernames(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	return r.dao.ListUsernames(ctx, tx...)
}

func (r *sitemapRepository) ListMysteries(ctx context.Context, tx ...*sql.Tx) ([]SitemapEntry, error) {
	return r.dao.ListMysteries(ctx, tx...)
}

func (r *sitemapRepository) ListShips(ctx context.Context, tx ...*sql.Tx) ([]SitemapEntry, error) {
	return r.dao.ListShips(ctx, tx...)
}

func (r *sitemapRepository) ListFanfics(ctx context.Context, tx ...*sql.Tx) ([]SitemapEntry, error) {
	return r.dao.ListFanfics(ctx, tx...)
}

func (r *sitemapRepository) ListJournalRows(ctx context.Context, tx ...*sql.Tx) ([]SitemapJournalRow, error) {
	return r.dao.ListJournalRows(ctx, tx...)
}
