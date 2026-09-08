package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"umineko_city_of_books/internal/dao/sqlcgen"
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

func toInvite(row sqlcgen.Invite) model.Invite {
	return model.Invite{
		Code:      row.Code,
		CreatedBy: row.CreatedBy,
		UsedBy:    row.UsedBy,
		UsedAt:    nullTimeToStringPtr(row.UsedAt),
		CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (r *inviteDAO) Create(ctx context.Context, s spec.NewInvite, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).CreateInvite(ctx, sqlcgen.CreateInviteParams{
		Code:      s.Code,
		CreatedBy: s.CreatedBy,
	})
	if err != nil {
		return fmt.Errorf("create invite: %w", err)
	}

	return nil
}

func (r *inviteDAO) GetByCode(ctx context.Context, code string, tx ...*sql.Tx) (*model.Invite, error) {
	row, err := genQueries(r.db, tx).GetInviteByCode(ctx, code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get invite: %w", err)
	}

	return new(toInvite(row)), nil
}

func (r *inviteDAO) MarkUsed(ctx context.Context, s spec.InviteRedemption, tx ...*sql.Tx) error {
	claimed, err := genQueries(r.db, tx).MarkInviteUsed(ctx, sqlcgen.MarkInviteUsedParams{
		UsedBy: &s.UsedBy,
		Code:   s.Code,
	})
	if err != nil {
		return fmt.Errorf("mark invite used: %w", err)
	}

	if claimed == 0 {
		return ErrInviteUnavailable
	}

	return nil
}

func (r *inviteDAO) List(ctx context.Context, q spec.InviteListQuery, tx ...*sql.Tx) ([]model.Invite, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountInvites(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count invites: %w", err)
	}

	rows, err := queries.ListInvites(ctx, sqlcgen.ListInvitesParams{
		Limit:  int32(q.Limit),
		Offset: int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list invites: %w", err)
	}

	var invites []model.Invite
	for _, row := range rows {
		invites = append(invites, toInvite(row))
	}

	return invites, int(total), nil
}

func (r *inviteDAO) Delete(ctx context.Context, code string, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteInvite(ctx, code); err != nil {
		return fmt.Errorf("delete invite: %w", err)
	}

	return nil
}
