package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type (
	ownedDAO struct {
		db     *sql.DB
		table  string
		entity string
	}
)

func newOwnedDAO(db *sql.DB, table, entity string) *ownedDAO {
	return &ownedDAO{db: db, table: table, entity: entity}
}

func (o *ownedDAO) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	res, err := txOrDB(o.db, tx).ExecContext(ctx, `DELETE FROM `+o.table+` WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete %s: %w", o.entity, err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%s not found or not owned", o.entity)
	}

	return nil
}

func (o *ownedDAO) DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(o.db, tx).ExecContext(ctx, `DELETE FROM `+o.table+` WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin delete %s: %w", o.entity, err)
	}

	return nil
}

func (o *ownedDAO) GetAuthorID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	var userID uuid.UUID

	err := txOrDB(o.db, tx).QueryRowContext(ctx, `SELECT user_id FROM `+o.table+` WHERE id = $1`, id).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get %s author: %w", o.entity, err)
	}

	return userID, nil
}
