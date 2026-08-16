package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/db"

	"github.com/google/uuid"
)

type (
	commentDeleteDAO interface {
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
	}

	commentDeleteSpec struct {
		CommentID   uuid.UUID
		UserID      uuid.UUID
		AsAdmin     bool
		OwnAction   AuditAction
		AdminAction AuditAction
		TargetType  AuditTargetType
	}
)

func deleteCommentWithAudit(ctx context.Context, database *sql.DB, dao commentDeleteDAO, audit AuditLogRepository, spec commentDeleteSpec, tx []*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTxOrJoin(ctx, database, tx, func(tx *sql.Tx) error {
		authorID, err := dao.GetCommentAuthorID(ctx, spec.CommentID, tx)
		if err != nil {
			return err
		}

		mediaPaths, err := dao.CollectSingleCommentMediaPaths(ctx, spec.CommentID, tx)
		if err != nil {
			return err
		}

		action := spec.OwnAction
		if authorID != spec.UserID {
			action = spec.AdminAction
		}

		if spec.AsAdmin {
			if err := dao.DeleteCommentAsAdmin(ctx, spec.CommentID, tx); err != nil {
				return err
			}
		} else {
			if err := dao.DeleteComment(ctx, spec.CommentID, spec.UserID, tx); err != nil {
				return err
			}
		}

		entry := NewAuditEntry{
			ActorID:    spec.UserID,
			Action:     action,
			TargetType: spec.TargetType,
			TargetID:   spec.CommentID.String(),
			SubjectID:  authorID,
		}

		if err := audit.Create(ctx, entry, tx); err != nil {
			return fmt.Errorf("audit comment delete: %w", err)
		}

		paths = mediaPaths

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}
