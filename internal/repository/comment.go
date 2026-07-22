package repository

import "github.com/google/uuid"

type (
	CommentRow struct {
		ID                uuid.UUID
		EntityID          string
		EntryID           *uuid.UUID
		ParentID          *uuid.UUID
		UserID            uuid.UUID
		Body              string
		CreatedAt         string
		UpdatedAt         *string
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		AuthorRole        string
		AuthorBanned      bool
		LikeCount         int
		UserLiked         bool
	}
)
