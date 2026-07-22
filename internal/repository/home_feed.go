package repository

import (
	"context"

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
		ListRecentActivity(ctx context.Context, limit int) ([]HomeActivityRow, error)
		ListRecentMembers(ctx context.Context, limit int) ([]HomeMemberRow, error)
		ListPublicRooms(ctx context.Context, limit int) ([]HomePublicRoomRow, error)
		ListCornerActivity24h(ctx context.Context) ([]HomeCornerActivityRow, error)
		ListSidebarActivity(ctx context.Context) ([]SidebarActivityEntry, error)
	}
)

type homeFeedRepository struct {
	dao HomeFeedRepository
}

func NewHomeFeedRepo(dao HomeFeedRepository) HomeFeedRepository {
	return &homeFeedRepository{dao: dao}
}

func (r *homeFeedRepository) ListRecentActivity(ctx context.Context, limit int) ([]HomeActivityRow, error) {
	return r.dao.ListRecentActivity(ctx, limit)
}

func (r *homeFeedRepository) ListRecentMembers(ctx context.Context, limit int) ([]HomeMemberRow, error) {
	return r.dao.ListRecentMembers(ctx, limit)
}

func (r *homeFeedRepository) ListPublicRooms(ctx context.Context, limit int) ([]HomePublicRoomRow, error) {
	return r.dao.ListPublicRooms(ctx, limit)
}

func (r *homeFeedRepository) ListCornerActivity24h(ctx context.Context) ([]HomeCornerActivityRow, error) {
	return r.dao.ListCornerActivity24h(ctx)
}

func (r *homeFeedRepository) ListSidebarActivity(ctx context.Context) ([]SidebarActivityEntry, error) {
	return r.dao.ListSidebarActivity(ctx)
}
