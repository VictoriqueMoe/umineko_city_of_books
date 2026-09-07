package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	commentDeleteDAO interface {
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		DeleteComment(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) error
	}

	commentDeleteSpec struct {
		spec.CommentDeletion
		OwnAction   audit.Action
		AdminAction audit.Action
		TargetType  audit.TargetType
	}
)

func deleteCommentWithAudit(ctx context.Context, database *sql.DB, comments commentDeleteDAO, auditRepo AuditLogRepository, s commentDeleteSpec, tx []*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, database, tx, func(tx *sql.Tx) error {
		authorID, err := comments.GetCommentAuthorID(ctx, s.CommentID, tx)
		if err != nil {
			return err
		}

		mediaPaths, err := comments.CollectSingleCommentMediaPaths(ctx, s.CommentID, tx)
		if err != nil {
			return err
		}

		action := s.OwnAction
		if authorID != s.UserID {
			action = s.AdminAction
		}

		if err := comments.DeleteComment(ctx, s.CommentDeletion, tx); err != nil {
			return err
		}

		entry := audit.NewEntry{
			ActorID:    s.UserID,
			Action:     action,
			TargetType: s.TargetType,
			TargetID:   s.CommentID.String(),
			SubjectID:  authorID,
		}

		if err := auditRepo.Create(ctx, entry, tx); err != nil {
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
