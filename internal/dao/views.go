package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/model/spec"
)

type (
	viewDAO struct {
		sqlcSource
		q viewQuerier
	}
)

func newViewDAO(db *sql.DB, q viewQuerier) *viewDAO {
	return &viewDAO{sqlcSource: newSQLCSource(db), q: q}
}

func (v *viewDAO) RecordView(ctx context.Context, s spec.ViewRecord, tx ...*sql.Tx) (bool, error) {
	n, err := v.q.Record(ctx, v.queries(tx), s)
	if err != nil {
		return false, fmt.Errorf("record view: %w", err)
	}

	return n > 0, nil
}
