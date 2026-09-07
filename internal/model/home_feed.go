package model

import (
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
)
