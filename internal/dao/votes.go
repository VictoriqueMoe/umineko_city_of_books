package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type (
	voteDAO struct {
		db     *sql.DB
		table  string
		fk     string
		action string
	}
)

func newVoteDAO(db *sql.DB, table, fk, action string) *voteDAO {
	return &voteDAO{db: db, table: table, fk: fk, action: action}
}

func (v *voteDAO) Vote(ctx context.Context, userID uuid.UUID, entityID uuid.UUID, value int, tx ...*sql.Tx) error {
	if value == 0 {
		_, err := txOrDB(v.db, tx).ExecContext(ctx,
			`DELETE FROM `+v.table+` WHERE user_id = $1 AND `+v.fk+` = $2`,
			userID, entityID,
		)
		return err
	}

	_, err := txOrDB(v.db, tx).ExecContext(ctx,
		`INSERT INTO `+v.table+` (user_id, `+v.fk+`, value) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, `+v.fk+`) DO UPDATE SET value = EXCLUDED.value`,
		userID, entityID, value,
	)
	if err != nil && v.action != "" {
		return fmt.Errorf("%s: %w", v.action, err)
	}

	return err
}
