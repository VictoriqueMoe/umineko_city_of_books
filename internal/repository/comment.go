package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model/spec"
)

type (
	CommentRepository interface {
		dao.CommentDAO

		UpdateCommentWithAudit(ctx context.Context, s spec.AuditedCommentUpdate, tx ...*sql.Tx) error
		DeleteCommentWithAudit(ctx context.Context, s spec.AuditedCommentDeletion, tx ...*sql.Tx) ([]string, error)
	}

	commentRepository struct {
		db *sql.DB
		dao.CommentDAO
		audit AuditLogRepository
	}
)

func NewCommentRepo(database *sql.DB, comments dao.CommentDAO, auditLog AuditLogRepository) CommentRepository {
	return &commentRepository{db: database, CommentDAO: comments, audit: auditLog}
}

func (r *commentRepository) UpdateCommentWithAudit(ctx context.Context, s spec.AuditedCommentUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.CommentDAO.UpdateComment(ctx, s.CommentUpdate, tx); err != nil {
			return err
		}

		if err := r.audit.Create(ctx, s.Audit, tx); err != nil {
			return fmt.Errorf("audit comment update: %w", err)
		}

		return nil
	})
}

func (r *commentRepository) DeleteCommentWithAudit(ctx context.Context, s spec.AuditedCommentDeletion, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		mediaPaths, err := r.CommentDAO.CollectSingleCommentMediaPaths(ctx, s.CommentRef, tx)
		if err != nil {
			return err
		}

		if err := r.CommentDAO.DeleteComment(ctx, s.CommentDeletion, tx); err != nil {
			return err
		}

		if err := r.audit.Create(ctx, s.Audit, tx); err != nil {
			return fmt.Errorf("audit comment delete: %w", err)
		}

		paths = dedupePaths(mediaPaths)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}
