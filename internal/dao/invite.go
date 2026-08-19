package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/repository"
)

type (
	inviteDAO struct {
		db *sql.DB
	}
)

func (r *inviteDAO) Create(ctx context.Context, code string, createdBy uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO invites (code, created_by) VALUES ($1, $2)`, code, createdBy,
	)
	if err != nil {
		return fmt.Errorf("create invite: %w", err)
	}
	return nil
}

func (r *inviteDAO) GetByCode(ctx context.Context, code string, tx ...*sql.Tx) (*repository.Invite, error) {
	var inv repository.Invite
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT code, created_by, used_by, used_at, created_at FROM invites WHERE code = $1`, code,
	).Scan(&inv.Code, &inv.CreatedBy, &inv.UsedBy, &inv.UsedAt, &inv.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get invite: %w", err)
	}
	return &inv, nil
}

func (r *inviteDAO) MarkUsed(ctx context.Context, code string, usedBy uuid.UUID, tx ...*sql.Tx) error {
	result, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE invites SET used_by = $1, used_at = NOW() WHERE code = $2 AND used_by IS NULL`, usedBy, code,
	)
	if err != nil {
		return fmt.Errorf("mark invite used: %w", err)
	}

	claimed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark invite used: %w", err)
	}

	if claimed == 0 {
		return repository.ErrInviteUnavailable
	}

	return nil
}

func (r *inviteDAO) List(ctx context.Context, limit, offset int, tx ...*sql.Tx) ([]repository.Invite, int, error) {
	var total int
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM invites`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count invites: %w", err)
	}

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT code, created_by, used_by, used_at, created_at FROM invites ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list invites: %w", err)
	}
	defer rows.Close()

	var invites []repository.Invite
	for rows.Next() {
		var inv repository.Invite
		if err := rows.Scan(&inv.Code, &inv.CreatedBy, &inv.UsedBy, &inv.UsedAt, &inv.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan invite: %w", err)
		}
		invites = append(invites, inv)
	}
	return invites, total, rows.Err()
}

func (r *inviteDAO) Delete(ctx context.Context, code string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM invites WHERE code = $1`, code)
	if err != nil {
		return fmt.Errorf("delete invite: %w", err)
	}
	return nil
}
