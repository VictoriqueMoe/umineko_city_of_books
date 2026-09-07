package spec

import (
	"github.com/google/uuid"
)

type (
	NewComment[K comparable] struct {
		TargetID K
		ParentID *uuid.UUID
		UserID   uuid.UUID
		Body     string
	}

	CommentQuery[K comparable] struct {
		TargetID       K
		ViewerID       uuid.UUID
		Limit          int
		Offset         int
		ExcludeUserIDs []uuid.UUID
	}

	CommentUpdate struct {
		CommentID uuid.UUID
		UserID    uuid.UUID
		Body      string
		AsAdmin   bool
	}

	CommentDeletion struct {
		CommentID uuid.UUID
		UserID    uuid.UUID
		AsAdmin   bool
	}

	CommentLike struct {
		UserID    uuid.UUID
		CommentID uuid.UUID
	}
)
