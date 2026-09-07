package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model/spec"
)

type (
	SecretRepository interface {
		dao.SecretDAO

		UpdateCommentBody(ctx context.Context, update spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteCommentWithAudit(ctx context.Context, deletion spec.CommentDeletion, tx ...*sql.Tx) ([]string, error)
	}
)

type secretRepository struct {
	db *sql.DB
	dao.SecretDAO
	audit AuditLogRepository
}

func NewSecretRepo(database *sql.DB, d dao.SecretDAO, auditRepo AuditLogRepository) SecretRepository {
	return &secretRepository{db: database, SecretDAO: d, audit: auditRepo}
}

func (r *secretRepository) UpdateCommentBody(ctx context.Context, update spec.CommentUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		authorID, err := r.SecretDAO.GetCommentAuthorID(ctx, update.CommentID, tx)
		if err != nil {
			return err
		}

		if err := r.SecretDAO.UpdateComment(ctx, update, tx); err != nil {
			return err
		}

		if authorID == update.UserID {
			return nil
		}

		entry := audit.NewEntry{
			ActorID:    update.UserID,
			Action:     audit.ActionSecretCommentUpdateAdmin,
			TargetType: audit.TargetSecretComment,
			TargetID:   update.CommentID.String(),
			SubjectID:  authorID,
		}

		if err := r.audit.Create(ctx, entry, tx); err != nil {
			return fmt.Errorf("audit comment update: %w", err)
		}

		return nil
	})
}

func (r *secretRepository) DeleteCommentWithAudit(ctx context.Context, deletion spec.CommentDeletion, tx ...*sql.Tx) ([]string, error) {
	return deleteCommentWithAudit(ctx, r.db, r.SecretDAO, r.audit, commentDeleteSpec{
		CommentDeletion: deletion,
		OwnAction:       audit.ActionSecretCommentDelete,
		AdminAction:     audit.ActionSecretCommentDeleteAdmin,
		TargetType:      audit.TargetSecretComment,
	}, tx)
}
