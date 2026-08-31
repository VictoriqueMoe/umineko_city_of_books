package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type (
	CommentDAO[K comparable] interface {
		CreateComment(ctx context.Context, entityID K, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error)
	}

	CommentDAOs struct {
		ByID    map[string]CommentDAO[uuid.UUID]
		BySlug  map[string]CommentDAO[string]
		Journal JournalCommentWriter
	}

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
