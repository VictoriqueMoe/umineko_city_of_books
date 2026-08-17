package db

import (
	"context"
	"database/sql"
	"fmt"
)

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

func WithTxOrJoin(ctx context.Context, db *sql.DB, tx []*sql.Tx, fn func(*sql.Tx) error) error {
	if len(tx) > 0 && tx[0] != nil {
		return fn(tx[0])
	}

	return withTx(ctx, db, fn)
}
