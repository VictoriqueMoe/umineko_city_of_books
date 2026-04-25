package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type (
	Invite struct {
		Code      string
		CreatedBy uuid.UUID
		UsedBy    *uuid.UUID
		UsedAt    *string
		CreatedAt string
	}

	InviteRepository interface {
		Create(ctx context.Context, code string, createdBy uuid.UUID) error
		GetByCode(ctx context.Context, code string) (*Invite, error)
		MarkUsed(ctx context.Context, code string, usedBy uuid.UUID) error
		List(ctx context.Context, limit, offset int) ([]Invite, int, error)
		Delete(ctx context.Context, code string) error
	}

	inviteRepository struct {
		db *sql.DB
	}
)

func (r *inviteRepository) Create(ctx context.Context, code string, createdBy uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO invites (code, created_by) VALUES ($1, $2)`, code, createdBy,
	)
	if err != nil {
		return fmt.Errorf("create invite: %w", err)
	}
	return nil
}

func (r *inviteRepository) GetByCode(ctx context.Context, code string) (*Invite, error) {
	var inv Invite
	err := r.db.QueryRowContext(ctx,
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

func (r *inviteRepository) MarkUsed(ctx context.Context, code string, usedBy uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE invites SET used_by = $1, used_at = NOW() WHERE code = $2`, usedBy, code,
	)
	if err != nil {
		return fmt.Errorf("mark invite used: %w", err)
	}
	return nil
}

func (r *inviteRepository) List(ctx context.Context, limit, offset int) ([]Invite, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM invites`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count invites: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT code, created_by, used_by, used_at, created_at FROM invites ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list invites: %w", err)
	}
	defer rows.Close()

	var invites []Invite
	for rows.Next() {
		var inv Invite
		if err := rows.Scan(&inv.Code, &inv.CreatedBy, &inv.UsedBy, &inv.UsedAt, &inv.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan invite: %w", err)
		}
		invites = append(invites, inv)
	}
	return invites, total, rows.Err()
}

func (r *inviteRepository) Delete(ctx context.Context, code string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM invites WHERE code = $1`, code)
	if err != nil {
		return fmt.Errorf("delete invite: %w", err)
	}
	return nil
}
