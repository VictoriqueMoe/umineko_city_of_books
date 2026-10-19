package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	AnnouncementRepository interface {
		dao.AnnouncementDAO

		DeleteWithMedia(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) ([]string, error)
	}

	announcementRepository struct {
		db *sql.DB
		dao.AnnouncementDAO
		comments dao.CommentDAO
	}
)

func NewAnnouncementRepo(database *sql.DB, announcements dao.AnnouncementDAO, comments dao.CommentDAO) AnnouncementRepository {
	return &announcementRepository{db: database, AnnouncementDAO: announcements, comments: comments}
}

func (r *announcementRepository) DeleteWithMedia(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		mediaPaths, err := r.comments.CollectCommentMediaPaths(ctx, spec.CommentTarget{Kind: spec.CommentKindAnnouncement, ID: id}, tx)
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
