package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/model/spec"
)

type (
	voteDAO struct {
		sqlcSource
		q      voteQuerier
		action string
	}
)

func newVoteDAO(db *sql.DB, q voteQuerier, action string) *voteDAO {
	return &voteDAO{sqlcSource: newSQLCSource(db), q: q, action: action}
}

func (v *voteDAO) Vote(ctx context.Context, s spec.Vote, tx ...*sql.Tx) error {
	queries := v.queries(tx)

	if s.Value == 0 {
		return v.q.Clear(ctx, queries, s)
	}

	err := v.q.Upsert(ctx, queries, s)
	if err != nil && v.action != "" {
		return fmt.Errorf("%s: %w", v.action, err)
	}

	return err
}
