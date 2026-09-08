package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
)

type (
	SitemapDAO interface {
		ListTheories(ctx context.Context, tx ...*sql.Tx) ([]model.SitemapEntry, error)
		ListPosts(ctx context.Context, tx ...*sql.Tx) ([]model.SitemapEntry, error)
		ListArt(ctx context.Context, tx ...*sql.Tx) ([]model.SitemapEntry, error)
		ListUsernames(ctx context.Context, tx ...*sql.Tx) ([]string, error)
		ListMysteries(ctx context.Context, tx ...*sql.Tx) ([]model.SitemapEntry, error)
		ListShips(ctx context.Context, tx ...*sql.Tx) ([]model.SitemapEntry, error)
		ListFanfics(ctx context.Context, tx ...*sql.Tx) ([]model.SitemapEntry, error)
		ListJournalRows(ctx context.Context, tx ...*sql.Tx) ([]model.SitemapJournalRow, error)
	}

	sitemapDAO struct {
		db *sql.DB
	}

	sitemapEntryRow = sqlcgen.SitemapListTheoriesRow
)

func toSitemapEntry(row sitemapEntryRow) model.SitemapEntry {
	return model.SitemapEntry{
		ID:      row.ID.String(),
		LastMod: row.CreatedAt,
	}
}

func (r *sitemapDAO) ListTheories(ctx context.Context, tx ...*sql.Tx) ([]model.SitemapEntry, error) {
	rows, err := genQueries(r.db, tx).SitemapListTheories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list theories: %w", err)
	}

	var entries []model.SitemapEntry
	for _, row := range rows {
		entries = append(entries, toSitemapEntry(row))
	}

	return entries, nil
}

func (r *sitemapDAO) ListPosts(ctx context.Context, tx ...*sql.Tx) ([]model.SitemapEntry, error) {
	rows, err := genQueries(r.db, tx).SitemapListPosts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list posts: %w", err)
	}

	var entries []model.SitemapEntry
	for _, row := range rows {
		entries = append(entries, toSitemapEntry(sitemapEntryRow(row)))
	}

	return entries, nil
}

func (r *sitemapDAO) ListArt(ctx context.Context, tx ...*sql.Tx) ([]model.SitemapEntry, error) {
	rows, err := genQueries(r.db, tx).SitemapListArt(ctx)
	if err != nil {
		return nil, fmt.Errorf("list art: %w", err)
	}

	var entries []model.SitemapEntry
	for _, row := range rows {
		entries = append(entries, toSitemapEntry(sitemapEntryRow(row)))
	}

	return entries, nil
}

func (r *sitemapDAO) ListMysteries(ctx context.Context, tx ...*sql.Tx) ([]model.SitemapEntry, error) {
	rows, err := genQueries(r.db, tx).SitemapListMysteries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list mysteries: %w", err)
	}

	var entries []model.SitemapEntry
	for _, row := range rows {
		entries = append(entries, toSitemapEntry(sitemapEntryRow(row)))
	}

	return entries, nil
}

func (r *sitemapDAO) ListShips(ctx context.Context, tx ...*sql.Tx) ([]model.SitemapEntry, error) {
	rows, err := genQueries(r.db, tx).SitemapListShips(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ships: %w", err)
	}

	var entries []model.SitemapEntry
	for _, row := range rows {
		entries = append(entries, toSitemapEntry(sitemapEntryRow(row)))
	}

	return entries, nil
}

func (r *sitemapDAO) ListFanfics(ctx context.Context, tx ...*sql.Tx) ([]model.SitemapEntry, error) {
	rows, err := genQueries(r.db, tx).SitemapListFanfics(ctx)
	if err != nil {
		return nil, fmt.Errorf("list fanfics: %w", err)
	}

	var entries []model.SitemapEntry
	for _, row := range rows {
		entries = append(entries, toSitemapEntry(sitemapEntryRow(row)))
	}

	return entries, nil
}

func (r *sitemapDAO) ListUsernames(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	usernames, err := genQueries(r.db, tx).SitemapListUsernames(ctx)
	if err != nil {
		return nil, fmt.Errorf("list usernames: %w", err)
	}

	return usernames, nil
}

func (r *sitemapDAO) ListJournalRows(ctx context.Context, tx ...*sql.Tx) ([]model.SitemapJournalRow, error) {
	rows, err := genQueries(r.db, tx).SitemapListJournalRows(ctx)
	if err != nil {
		return nil, fmt.Errorf("list journal rows: %w", err)
	}

	var result []model.SitemapJournalRow
	for _, row := range rows {
		entry := model.SitemapJournalRow{
			JournalID:        row.ID.String(),
			JournalUpdatedAt: row.JournalUpdatedAt,
			EntryUpdatedAt:   row.UpdatedAt,
		}

		if row.EntryNumber.Valid {
			entry.EntryNumber = sql.NullInt64{Int64: int64(row.EntryNumber.Int32), Valid: true}
		}

		result = append(result, entry)
	}

	return result, nil
}
