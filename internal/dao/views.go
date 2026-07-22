package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type viewDAO struct {
	db          *sql.DB
	viewsTable  string
	fk          string
	entityTable string
}

func newViewDAO(db *sql.DB, viewsTable string, fk string, entityTable string) *viewDAO {
	return &viewDAO{db: db, viewsTable: viewsTable, fk: fk, entityTable: entityTable}
}

func (v *viewDAO) RecordView(ctx context.Context, entityID uuid.UUID, viewerHash string) (bool, error) {
	res, err := v.db.ExecContext(ctx,
		`INSERT INTO `+v.viewsTable+` (`+v.fk+`, viewer_hash) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		entityID, viewerHash,
	)
	if err != nil {
		return false, fmt.Errorf("record view in %s: %w", v.viewsTable, err)
	}

	n, _ := res.RowsAffected()
	if n > 0 {
		if _, err := v.db.ExecContext(ctx,
			`UPDATE `+v.entityTable+` SET view_count = view_count + 1 WHERE id = $1`, entityID,
		); err != nil {
			return false, fmt.Errorf("increment view count in %s: %w", v.entityTable, err)
		}
	}

	return n > 0, nil
}
