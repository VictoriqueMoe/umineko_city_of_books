package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model/spec"
)

type (
	OCRepository interface {
		dao.OCDAO

		DeleteOC(ctx context.Context, s spec.OCDeletion, tx ...*sql.Tx) ([]string, error)
		DeleteCommentWithMedia(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) ([]string, error)
	}

	ocRepository struct {
		db *sql.DB
		dao.OCDAO
		audit AuditLogRepository
	}
)

func NewOCRepo(database *sql.DB, ocs dao.OCDAO, auditLog AuditLogRepository) OCRepository {
	return &ocRepository{db: database, OCDAO: ocs, audit: auditLog}
}

func (r *ocRepository) DeleteOC(ctx context.Context, s spec.OCDeletion, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		imagePaths, err := r.OCDAO.GetImagePaths(ctx, s.ID, tx)
		if err != nil {
			return err
		}

		galleryPaths, err := r.OCDAO.GetGalleryPaths(ctx, s.ID, tx)
		if err != nil {
			return err
		}

		commentPaths, err := r.OCDAO.CollectCommentMediaPaths(ctx, s.ID, tx)
		if err != nil {
			return err
		}

		collected := append(imagePaths, galleryPaths...)
		collected = append(collected, commentPaths...)

		if s.AsAdmin {
			if err := r.OCDAO.DeleteAsAdmin(ctx, s.ID, tx); err != nil {
				return err
			}
		} else {
			if err := r.OCDAO.Delete(ctx, spec.OwnedDeletion{ID: s.ID, UserID: s.UserID}, tx); err != nil {
				return err
			}
		}

		paths = dedupePaths(collected)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *ocRepository) DeleteCommentWithMedia(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) ([]string, error) {
	return deleteCommentWithAudit(ctx, r.db, r.OCDAO, r.audit, commentDeleteSpec{
		CommentDeletion: s,
		OwnAction:       audit.ActionOCCommentDelete,
		AdminAction:     audit.ActionOCCommentDeleteAdmin,
		TargetType:      audit.TargetOCComment,
	}, tx)
}
