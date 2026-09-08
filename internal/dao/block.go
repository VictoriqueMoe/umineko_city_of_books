package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	BlockDAO interface {
		Block(ctx context.Context, s spec.BlockSpec, tx ...*sql.Tx) error
		Unblock(ctx context.Context, s spec.BlockSpec, tx ...*sql.Tx) error
		IsBlocked(ctx context.Context, s spec.BlockSpec, tx ...*sql.Tx) (bool, error)
		IsBlockedEither(ctx context.Context, s spec.BlockPairSpec, tx ...*sql.Tx) (bool, error)
		GetBlockedIDs(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
		GetBlockedUsers(ctx context.Context, blockerID uuid.UUID, tx ...*sql.Tx) ([]model.BlockedUser, error)
	}

	blockDAO struct {
		db *sql.DB
	}

	blockedUserRow = sqlcgen.ListBlockedUsersRow
)

func toBlockedUser(row blockedUserRow) model.BlockedUser {
	return model.BlockedUser{
		ID:          row.ID,
		Username:    row.Username,
		DisplayName: row.DisplayName,
		AvatarURL:   row.AvatarUrl,
		BlockedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (r *blockDAO) Block(ctx context.Context, s spec.BlockSpec, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).CreateBlock(ctx, sqlcgen.CreateBlockParams{
		BlockerID: s.BlockerID,
		BlockedID: s.BlockedID,
	})
	if err != nil {
		return fmt.Errorf("block user: %w", err)
	}

	return nil
}

func (r *blockDAO) Unblock(ctx context.Context, s spec.BlockSpec, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).DeleteBlock(ctx, sqlcgen.DeleteBlockParams{
		BlockerID: s.BlockerID,
		BlockedID: s.BlockedID,
	})
	if err != nil {
		return fmt.Errorf("unblock user: %w", err)
	}

	return nil
}

func (r *blockDAO) IsBlocked(ctx context.Context, s spec.BlockSpec, tx ...*sql.Tx) (bool, error) {
	count, err := genQueries(r.db, tx).CountBlock(ctx, sqlcgen.CountBlockParams{
		BlockerID: s.BlockerID,
		BlockedID: s.BlockedID,
	})
	if err != nil {
		return false, fmt.Errorf("check block: %w", err)
	}

	return count > 0, nil
}

func (r *blockDAO) IsBlockedEither(ctx context.Context, s spec.BlockPairSpec, tx ...*sql.Tx) (bool, error) {
	count, err := genQueries(r.db, tx).CountBlockEither(ctx, sqlcgen.CountBlockEitherParams{
		BlockerID:   s.UserA,
		BlockedID:   s.UserB,
		BlockerID_2: s.UserB,
		BlockedID_2: s.UserA,
	})
	if err != nil {
		return false, fmt.Errorf("check block either: %w", err)
	}

	return count > 0, nil
}

func (r *blockDAO) GetBlockedIDs(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	ids, err := genQueries(r.db, tx).ListBlockedIDs(ctx, sqlcgen.ListBlockedIDsParams{
		BlockerID: userID,
		BlockedID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("get blocked ids: %w", err)
	}

	return ids, nil
}

func (r *blockDAO) GetBlockedUsers(ctx context.Context, blockerID uuid.UUID, tx ...*sql.Tx) ([]model.BlockedUser, error) {
	rows, err := genQueries(r.db, tx).ListBlockedUsers(ctx, blockerID)
	if err != nil {
		return nil, fmt.Errorf("get blocked users: %w", err)
	}

	var users []model.BlockedUser
	for _, row := range rows {
		users = append(users, toBlockedUser(row))
	}

	return users, nil
}
