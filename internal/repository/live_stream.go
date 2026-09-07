package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model/spec"
)

type (
	LiveStreamRepository interface {
		dao.LiveStreamDAO

		Activate(ctx context.Context, s spec.LiveStreamActivation, tx ...*sql.Tx) error
	}

	liveStreamRepository struct {
		db *sql.DB
		dao.LiveStreamDAO
	}
)

func NewLiveStreamRepo(database *sql.DB, d dao.LiveStreamDAO) LiveStreamRepository {
	return &liveStreamRepository{db: database, LiveStreamDAO: d}
}

func (r *liveStreamRepository) Activate(ctx context.Context, s spec.LiveStreamActivation, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.LiveStreamDAO.SetIngress(ctx, s.Ingress, tx); err != nil {
			return err
		}

		return r.LiveStreamDAO.SetDefaultMode(ctx, spec.LiveStreamDefaultModeUpdate{ID: s.Ingress.ID, DefaultMode: s.DefaultMode}, tx)
	})
}
