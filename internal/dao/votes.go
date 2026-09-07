package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/model/spec"
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

func (v *voteDAO) Vote(ctx context.Context, s spec.Vote, tx ...*sql.Tx) error {
	if s.Value == 0 {
		_, err := txOrDB(v.db, tx).ExecContext(ctx,
			`DELETE FROM `+v.table+` WHERE user_id = $1 AND `+v.fk+` = $2`,
			s.UserID, s.TargetID,
		)
		return err
	}

	_, err := txOrDB(v.db, tx).ExecContext(ctx,
		`INSERT INTO `+v.table+` (user_id, `+v.fk+`, value) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, `+v.fk+`) DO UPDATE SET value = EXCLUDED.value`,
		s.UserID, s.TargetID, s.Value,
	)
	if err != nil && v.action != "" {
		return fmt.Errorf("%s: %w", v.action, err)
	}

	return err
}
