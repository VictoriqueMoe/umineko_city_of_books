package db

import (
	"context"
	"database/sql"
	"fmt"
)

type (
	DBTX interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		PrepareContext(context.Context, string) (*sql.Stmt, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
)

func joinedTx(tx []*sql.Tx) *sql.Tx {
	if len(tx) > 0 && tx[0] != nil {
		return tx[0]
	}

	return nil
}

func TxOrDB(db *sql.DB, tx []*sql.Tx) DBTX {
	if joined := joinedTx(tx); joined != nil {
		return joined
	}

	return db
}

func withTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func WithTx(ctx context.Context, db *sql.DB, tx []*sql.Tx, fn func(*sql.Tx) error) error {
	if joined := joinedTx(tx); joined != nil {
		return fn(joined)
	}

	return withTx(ctx, db, fn)
}
