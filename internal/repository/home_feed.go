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
