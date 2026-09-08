package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	HomeFeedDAO interface {
		ListRecentActivity(ctx context.Context, limit int, tx ...*sql.Tx) ([]model.HomeActivityRow, error)
		ListEchoes(ctx context.Context, q spec.HomeEchoQuery, tx ...*sql.Tx) ([]model.HomeEchoRow, error)
		ListRecentMembers(ctx context.Context, limit int, tx ...*sql.Tx) ([]model.HomeMemberRow, error)
		ListPublicRooms(ctx context.Context, limit int, tx ...*sql.Tx) ([]model.HomePublicRoomRow, error)
		ListCornerActivity24h(ctx context.Context, tx ...*sql.Tx) ([]model.HomeCornerActivityRow, error)
		ListSidebarActivity(ctx context.Context, tx ...*sql.Tx) ([]model.SidebarActivityEntry, error)
	}

	homeFeedDAO struct {
		db *sql.DB
	}
)

func (r *homeFeedDAO) ListRecentActivity(ctx context.Context, limit int, tx ...*sql.Tx) ([]model.HomeActivityRow, error) {
	rows, err := genQueries(r.db, tx).ListHomeRecentActivity(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("home feed activity: %w", err)
	}

	var out []model.HomeActivityRow
	for _, row := range rows {
		out = append(out, model.HomeActivityRow{
			Kind:        row.Kind,
			ID:          row.ID,
			Title:       row.Title,
			Body:        row.Body,
			Corner:      row.Corner,
			CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
			AuthorID:    row.AuthorID,
			Username:    row.Username,
			DisplayName: row.DisplayName,
			AvatarURL:   row.AvatarUrl,
		})
	}

	return out, nil
}

func (r *homeFeedDAO) ListEchoes(ctx context.Context, q spec.HomeEchoQuery, tx ...*sql.Tx) ([]model.HomeEchoRow, error) {
	rows, err := genQueries(r.db, tx).ListHomeEchoes(ctx, sqlcgen.ListHomeEchoesParams{
		Ago:      q.Ago,
		RowLimit: int32(q.Limit),
	})
	if err != nil {
		return nil, fmt.Errorf("home feed echoes: %w", err)
	}

	var out []model.HomeEchoRow
	for _, row := range rows {
		out = append(out, model.HomeEchoRow{
			Kind:        row.Kind,
			ID:          row.ID,
			Title:       row.Title,
			Body:        row.Body,
			Corner:      row.Corner,
			Episode:     int(row.Episode),
			IsSpoiler:   row.IsSpoiler,
			CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
			AuthorID:    row.AuthorID,
			Username:    row.Username,
			DisplayName: row.DisplayName,
			AvatarURL:   row.AvatarUrl,
		})
	}

	return out, nil
}

func (r *homeFeedDAO) ListRecentMembers(ctx context.Context, limit int, tx ...*sql.Tx) ([]model.HomeMemberRow, error) {
	rows, err := genQueries(r.db, tx).ListHomeRecentMembers(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("home feed members: %w", err)
	}

	var out []model.HomeMemberRow
	for _, row := range rows {
		out = append(out, model.HomeMemberRow{
			ID:          row.ID,
			Username:    row.Username,
			DisplayName: row.DisplayName,
			AvatarURL:   row.AvatarUrl,
			CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	return out, nil
}

func (r *homeFeedDAO) ListCornerActivity24h(ctx context.Context, tx ...*sql.Tx) ([]model.HomeCornerActivityRow, error) {
	rows, err := genQueries(r.db, tx).ListHomeCornerActivity24h(ctx)
	if err != nil {
		return nil, fmt.Errorf("home feed corner activity: %w", err)
	}

	var out []model.HomeCornerActivityRow
	for _, row := range rows {
		out = append(out, model.HomeCornerActivityRow{
			Corner:        row.Corner,
			PostCount:     int(row.PostCount),
			UniquePosters: int(row.UniquePosters),
			LastPostAt:    new(row.LastPostAt.UTC().Format(time.RFC3339)),
		})
	}

	return out, nil
}

func (r *homeFeedDAO) ListSidebarActivity(ctx context.Context, tx ...*sql.Tx) ([]model.SidebarActivityEntry, error) {
	rows, err := genQueries(r.db, tx).ListHomeSidebarActivity(ctx)
	if err != nil {
		return nil, fmt.Errorf("sidebar activity: %w", err)
	}

	var out []model.SidebarActivityEntry
	for _, row := range rows {
		out = append(out, model.SidebarActivityEntry{
			Key:      row.Key,
			LatestAt: row.LatestAt.UTC().Format(time.RFC3339),
		})
	}

	return out, nil
}

func (r *homeFeedDAO) ListPublicRooms(ctx context.Context, limit int, tx ...*sql.Tx) ([]model.HomePublicRoomRow, error) {
	rows, err := genQueries(r.db, tx).ListHomePublicRooms(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("home feed public rooms: %w", err)
	}

	var out []model.HomePublicRoomRow
	for _, row := range rows {
		out = append(out, model.HomePublicRoomRow{
			ID:            row.ID,
			Name:          row.Name,
			Description:   row.Description,
			MemberCount:   int(row.MemberCount),
			LastMessageAt: nullTimeToStringPtr(row.LastMessageAt),
		})
	}

	return out, nil
}
