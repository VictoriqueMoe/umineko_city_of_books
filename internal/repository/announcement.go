package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	AnnouncementRepository interface {
		dao.AnnouncementDAO

		UpdateCommentBody(ctx context.Context, update spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteCommentWithAudit(ctx context.Context, deletion spec.CommentDeletion, tx ...*sql.Tx) ([]string, error)
		DeleteWithMedia(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) ([]string, error)
	}

	announcementRepository struct {
		db *sql.DB
		dao.AnnouncementDAO
		audit AuditLogRepository
	}
)

func NewAnnouncementRepo(database *sql.DB, announcements dao.AnnouncementDAO, auditLog AuditLogRepository) AnnouncementRepository {
	return &announcementRepository{db: database, AnnouncementDAO: announcements, audit: auditLog}
}

func (r *announcementRepository) UpdateCommentBody(ctx context.Context, update spec.CommentUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		authorID, err := r.AnnouncementDAO.GetCommentAuthorID(ctx, update.CommentID, tx)
		if err != nil {
			return err
		}

		if err := r.AnnouncementDAO.UpdateComment(ctx, update, tx); err != nil {
			return err
		}

		if authorID == update.UserID {
			return nil
		}

		entry := audit.NewEntry{
			ActorID:    update.UserID,
			Action:     audit.ActionAnnouncementCommentUpdateAdmin,
			TargetType: audit.TargetAnnouncementComment,
			TargetID:   update.CommentID.String(),
			SubjectID:  authorID,
		}

		if err := r.audit.Create(ctx, entry, tx); err != nil {
			return fmt.Errorf("audit comment update: %w", err)
		}

		return nil
	})
}

func (r *announcementRepository) DeleteCommentWithAudit(ctx context.Context, deletion spec.CommentDeletion, tx ...*sql.Tx) ([]string, error) {
	return deleteCommentWithAudit(ctx, r.db, r.AnnouncementDAO, r.audit, commentDeleteSpec{
		CommentDeletion: deletion,
		OwnAction:       audit.ActionAnnouncementCommentDelete,
		AdminAction:     audit.ActionAnnouncementCommentDeleteAdmin,
		TargetType:      audit.TargetAnnouncementComment,
	}, tx)
}

func (r *announcementRepository) DeleteWithMedia(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		mediaPaths, err := r.AnnouncementDAO.CollectCommentMediaPaths(ctx, id, tx)
		if err != nil {
			return err
		}

		if err := r.AnnouncementDAO.Delete(ctx, id, tx); err != nil {
			return err
		}

		paths = mediaPaths

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}
