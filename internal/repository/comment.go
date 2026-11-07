package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	CommentRepository interface {
		dao.CommentDAO

		CreateCommentWithMedia(ctx context.Context, s spec.NewComment, tx ...*sql.Tx) (*model.CommentRow, error)
		AttachCommentMedia(ctx context.Context, s spec.CommentMediaAttachment, tx ...*sql.Tx) (*model.PostMediaRow, error)
		UpdateCommentWithAudit(ctx context.Context, s spec.AuditedCommentUpdate, tx ...*sql.Tx) error
		DeleteCommentWithAudit(ctx context.Context, s spec.AuditedCommentDeletion, tx ...*sql.Tx) ([]string, error)
	}

	commentRepository struct {
		db *sql.DB
		dao.CommentDAO
		pending dao.PendingUploadDAO
		audit   AuditLogRepository
	}
)

func NewCommentRepo(database *sql.DB, comments dao.CommentDAO, pending dao.PendingUploadDAO, auditLog AuditLogRepository) CommentRepository {
	return &commentRepository{db: database, CommentDAO: comments, pending: pending, audit: auditLog}
}

func (r *commentRepository) addMediaFor(kind spec.CommentKind) func(ctx context.Context, m spec.NewMedia, tx ...*sql.Tx) (int64, error) {
	return func(ctx context.Context, m spec.NewMedia, tx ...*sql.Tx) (int64, error) {
		return r.CommentDAO.AddCommentMedia(ctx, spec.NewCommentMedia{Kind: kind, NewMedia: m}, tx...)
	}
}

func (r *commentRepository) CreateCommentWithMedia(ctx context.Context, s spec.NewComment, tx ...*sql.Tx) (*model.CommentRow, error) {
	var created *model.CommentRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.CommentDAO.CreateComment(ctx, s, tx)
		if err != nil {
			return err
		}

		rows, err := claimMediaRows(ctx, r.pending, s.Media, tx)
		if err != nil {
			return err
		}

		_, err = addMediaRows(ctx, r.addMediaFor(s.Target.Kind), created.ID, rows, tx)

		return err
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *commentRepository) AttachCommentMedia(ctx context.Context, s spec.CommentMediaAttachment, tx ...*sql.Tx) (*model.PostMediaRow, error) {
	var attached *model.PostMediaRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error
		attached, err = attachMedia(ctx, r.pending, r.addMediaFor(s.Kind), s.MediaAttachment, tx)

		return err
	})
	if err != nil {
		return nil, err
	}

	return attached, nil
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
