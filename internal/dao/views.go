package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/model/spec"
)

type (
	viewDAO struct {
		db         *sql.DB
		viewsTable string
		fk         string
	}
)

func newViewDAO(db *sql.DB, viewsTable string, fk string) *viewDAO {
	return &viewDAO{db: db, viewsTable: viewsTable, fk: fk}
}

func (v *viewDAO) RecordView(ctx context.Context, s spec.ViewRecord, tx ...*sql.Tx) (bool, error) {
	res, err := txOrDB(v.db, tx).ExecContext(ctx,
		`INSERT INTO `+v.viewsTable+` (`+v.fk+`, viewer_hash) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		s.TargetID, s.ViewerHash,
	)
	if err != nil {
		return false, fmt.Errorf("record view in %s: %w", v.viewsTable, err)
	}

	n, _ := res.RowsAffected()

	return n > 0, nil
}
