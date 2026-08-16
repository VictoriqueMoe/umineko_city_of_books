package dao

import (
	"context"
	"database/sql"
)

type (
	dbtx interface {
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
		QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
		QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	}
)

func txOrDB(db *sql.DB, tx []*sql.Tx) dbtx {
	if len(tx) > 0 && tx[0] != nil {
		return tx[0]
	}

	return db
}
