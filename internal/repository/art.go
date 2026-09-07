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
	ArtRepository interface {
		dao.ArtDAO

		CreateWithTags(ctx context.Context, s spec.NewArtWithTags, tx ...*sql.Tx) (*model.ArtRow, error)
		UpdateWithTags(ctx context.Context, s spec.ArtUpdateWithTags, tx ...*sql.Tx) error
		DeleteWithImage(ctx context.Context, s spec.ArtDelete, tx ...*sql.Tx) ([]string, error)
		UpdateCommentWithDetails(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteCommentWithAudit(ctx context.Context, s spec.ArtCommentDeletion, tx ...*sql.Tx) ([]string, error)
		DeleteGallery(ctx context.Context, s spec.GalleryRef, tx ...*sql.Tx) ([]string, error)
		RecordView(ctx context.Context, s spec.ViewRecord, tx ...*sql.Tx) (bool, error)
	}

	artRepository struct {
		db *sql.DB
		dao.ArtDAO
		posts PostRepository
		audit AuditLogRepository
	}
)

func NewArtRepo(database *sql.DB, artDAO dao.ArtDAO, posts PostRepository, auditRepo AuditLogRepository) ArtRepository {
	return &artRepository{db: database, ArtDAO: artDAO, posts: posts, audit: auditRepo}
}

func (r *artRepository) CreateWithTags(ctx context.Context, s spec.NewArtWithTags, tx ...*sql.Tx) (*model.ArtRow, error) {
	var created *model.ArtRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.ArtDAO.CreateArt(ctx, s.NewArt, tx)
		if err != nil {
			return err
		}

		return r.ArtDAO.InsertTags(ctx, spec.ArtTagInsert{ArtID: created.ID, Tags: s.Tags}, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *artRepository) UpdateWithTags(ctx context.Context, s spec.ArtUpdateWithTags, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.ArtDAO.UpdateArt(ctx, s.ArtUpdate, tx); err != nil {
			return err
		}

		if err := r.ArtDAO.DeleteTags(ctx, s.ID, tx); err != nil {
			return err
		}

		return r.ArtDAO.InsertTags(ctx, spec.ArtTagInsert{ArtID: s.ID, Tags: s.Tags}, tx)
	})
}

func (r *artRepository) DeleteWithImage(ctx context.Context, s spec.ArtDelete, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		imagePaths, err := r.ArtDAO.GetArtImagePaths(ctx, s.ID, tx)
		if err != nil {
			return err
		}

		commentPaths, err := r.ArtDAO.CollectCommentMediaPaths(ctx, s.ID, tx)
		if err != nil {
			return err
		}

		paths = append(imagePaths, commentPaths...)

		if s.AsAdmin {
			err = r.ArtDAO.DeleteAsAdmin(ctx, s.ID, tx)
		} else {
			err = r.ArtDAO.Delete(ctx, spec.OwnedDeletion{ID: s.ID, UserID: s.UserID}, tx)
		}
		if err != nil {
			return err
		}

		if err := r.audit.Create(ctx, s.Audit, tx); err != nil {
			return fmt.Errorf("audit art delete: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *artRepository) UpdateCommentWithDetails(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		return r.ArtDAO.UpdateComment(ctx, s, tx)
	})
}

func (r *artRepository) DeleteCommentWithAudit(ctx context.Context, s spec.ArtCommentDeletion, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		mediaPaths, err := r.ArtDAO.CollectSingleCommentMediaPaths(ctx, s.CommentID, tx)
		if err != nil {
			return err
		}

		paths = dedupePaths(mediaPaths)

		if err := r.ArtDAO.DeleteComment(ctx, s.CommentDeletion, tx); err != nil {
			return err
		}

		if err := r.audit.Create(ctx, s.Audit, tx); err != nil {
			return fmt.Errorf("audit comment delete: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *artRepository) DeleteGallery(ctx context.Context, s spec.GalleryRef, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		images, err := r.ArtDAO.ListGalleryArtImages(ctx, s, tx)
		if err != nil {
			return err
		}

		var collected []string

		for _, image := range images {
			if image.ImageURL != "" {
				collected = append(collected, image.ImageURL)
			}

			if image.ThumbnailURL != "" {
				collected = append(collected, image.ThumbnailURL)
			}

			commentPaths, commentErr := r.ArtDAO.CollectCommentMediaPaths(ctx, image.ArtID, tx)
			if commentErr != nil {
				return commentErr
			}

			collected = append(collected, commentPaths...)
		}

		paths = dedupePaths(collected)

		if err := r.ArtDAO.DeleteArtInGallery(ctx, s, tx); err != nil {
			return err
		}

		return r.ArtDAO.DeleteGalleryRow(ctx, s, tx)
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *artRepository) RecordView(ctx context.Context, s spec.ViewRecord, tx ...*sql.Tx) (bool, error) {
	var inserted bool

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		ok, err := r.ArtDAO.RecordView(ctx, s, tx)
		if err != nil {
			return err
		}

		if !ok {
			return nil
		}

		if err := r.ArtDAO.IncrementViewCount(ctx, s.TargetID, tx); err != nil {
			return err
		}

		inserted = true

		return nil
	})
	if err != nil {
		return false, err
	}

	return inserted, nil
}
