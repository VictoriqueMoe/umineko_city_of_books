package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	ownedDAO struct {
		sqlcSource
		q      ownedQuerier
		entity string
	}
)

func newOwnedDAO(db *sql.DB, q ownedQuerier, entity string) *ownedDAO {
	return &ownedDAO{sqlcSource: newSQLCSource(db), q: q, entity: entity}
}

func (o *ownedDAO) Delete(ctx context.Context, s spec.OwnedDeletion, tx ...*sql.Tx) error {
	n, err := o.q.Delete(ctx, o.queries(tx), s)
	if err != nil {
		return fmt.Errorf("delete %s: %w", o.entity, err)
	}

	if n == 0 {
		return fmt.Errorf("%s not found or not owned", o.entity)
	}

	return nil
}

func (o *ownedDAO) DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	if err := o.q.DeleteAsAdmin(ctx, o.queries(tx), id); err != nil {
		return fmt.Errorf("admin delete %s: %w", o.entity, err)
	}

	return nil
}

func (o *ownedDAO) GetAuthorID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	userID, err := o.q.AuthorID(ctx, o.queries(tx), id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get %s author: %w", o.entity, err)
	}

	return userID, nil
}

func (o *ownedDAO) IncrementViewCount(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	if err := o.q.IncrementViewCount(ctx, o.queries(tx), id); err != nil {
		return fmt.Errorf("increment %s view count: %w", o.entity, err)
	}

	return nil
}
