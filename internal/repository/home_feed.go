package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type (
	HomeActivityRow struct {
		Kind        string
		ID          uuid.UUID
		Title       string
		Body        string
		Corner      string
		CreatedAt   string
		AuthorID    uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
	}

	HomeEchoRow struct {
		Kind        string
		ID          uuid.UUID
		Title       string
		Body        string
		Corner      string
		Episode     int
		IsSpoiler   bool
		CreatedAt   string
		AuthorID    uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
	}

	HomeMemberRow struct {
		ID          uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
		CreatedAt   string
	}

	HomePublicRoomRow struct {
		ID            uuid.UUID
		Name          string
		Description   string
		MemberCount   int
		LastMessageAt *string
	}

	HomeCornerActivityRow struct {
		Corner        string
		PostCount     int
		UniquePosters int
		LastPostAt    *string
	}

	SidebarActivityEntry struct {
		Key      string
		LatestAt string
	}

	HomeFeedRepository interface {
		ListRecentActivity(ctx context.Context, limit int, tx ...*sql.Tx) ([]HomeActivityRow, error)
		ListEchoes(ctx context.Context, ago string, limit int, tx ...*sql.Tx) ([]HomeEchoRow, error)
		ListRecentMembers(ctx context.Context, limit int, tx ...*sql.Tx) ([]HomeMemberRow, error)
		ListPublicRooms(ctx context.Context, limit int, tx ...*sql.Tx) ([]HomePublicRoomRow, error)
		ListCornerActivity24h(ctx context.Context, tx ...*sql.Tx) ([]HomeCornerActivityRow, error)
		ListSidebarActivity(ctx context.Context, tx ...*sql.Tx) ([]SidebarActivityEntry, error)
	}
)

type homeFeedRepository struct {
	dao HomeFeedRepository
}

func NewHomeFeedRepo(dao HomeFeedRepository) HomeFeedRepository {
	return &homeFeedRepository{dao: dao}
}

func (r *homeFeedRepository) ListEchoes(ctx context.Context, ago string, limit int, tx ...*sql.Tx) ([]HomeEchoRow, error) {
	return r.dao.ListEchoes(ctx, ago, limit, tx...)
}

func (r *homeFeedRepository) ListRecentActivity(ctx context.Context, limit int, tx ...*sql.Tx) ([]HomeActivityRow, error) {
	return r.dao.ListRecentActivity(ctx, limit, tx...)
}

func (r *homeFeedRepository) ListRecentMembers(ctx context.Context, limit int, tx ...*sql.Tx) ([]HomeMemberRow, error) {
	return r.dao.ListRecentMembers(ctx, limit, tx...)
}

func (r *homeFeedRepository) ListPublicRooms(ctx context.Context, limit int, tx ...*sql.Tx) ([]HomePublicRoomRow, error) {
	return r.dao.ListPublicRooms(ctx, limit, tx...)
}

func (r *homeFeedRepository) ListCornerActivity24h(ctx context.Context, tx ...*sql.Tx) ([]HomeCornerActivityRow, error) {
	return r.dao.ListCornerActivity24h(ctx, tx...)
}

func (r *homeFeedRepository) ListSidebarActivity(ctx context.Context, tx ...*sql.Tx) ([]SidebarActivityEntry, error) {
	return r.dao.ListSidebarActivity(ctx, tx...)
}
