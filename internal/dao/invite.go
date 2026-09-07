package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	InviteDAO interface {
		Create(ctx context.Context, s spec.NewInvite, tx ...*sql.Tx) error
		GetByCode(ctx context.Context, code string, tx ...*sql.Tx) (*model.Invite, error)
		MarkUsed(ctx context.Context, s spec.InviteRedemption, tx ...*sql.Tx) error
		List(ctx context.Context, q spec.InviteListQuery, tx ...*sql.Tx) ([]model.Invite, int, error)
		Delete(ctx context.Context, code string, tx ...*sql.Tx) error
	}

	inviteDAO struct {
		db *sql.DB
	}
)

var (
	ErrInviteUnavailable = errors.New("invite code is missing or already used")
)

func (r *inviteDAO) Create(ctx context.Context, s spec.NewInvite, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO invites (code, created_by) VALUES ($1, $2)`, s.Code, s.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("create invite: %w", err)
	}

	return nil
}

func (r *inviteDAO) GetByCode(ctx context.Context, code string, tx ...*sql.Tx) (*model.Invite, error) {
	var inv model.Invite

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

func (r *inviteDAO) MarkUsed(ctx context.Context, s spec.InviteRedemption, tx ...*sql.Tx) error {
	result, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE invites SET used_by = $1, used_at = NOW() WHERE code = $2 AND used_by IS NULL`, s.UsedBy, s.Code,
	)
	if err != nil {
		return fmt.Errorf("mark invite used: %w", err)
	}

	claimed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark invite used: %w", err)
	}

	if claimed == 0 {
		return ErrInviteUnavailable
	}

	return nil
}

func (r *inviteDAO) List(ctx context.Context, q spec.InviteListQuery, tx ...*sql.Tx) ([]model.Invite, int, error) {
	var total int
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM invites`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count invites: %w", err)
	}

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT code, created_by, used_by, used_at, created_at FROM invites ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		q.Limit, q.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list invites: %w", err)
	}
	defer rows.Close()

	var invites []model.Invite
	for rows.Next() {
		var inv model.Invite
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
