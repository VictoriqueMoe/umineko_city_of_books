package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	ChatRoomBanDAO interface {
		Ban(ctx context.Context, s spec.NewChatRoomBan, tx ...*sql.Tx) error
		Unban(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) error
		IsBanned(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error)
		ListForRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatRoomBanRow, error)
		BannedRoomIDsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
	}

	chatRoomBanDAO struct {
		db *sql.DB
	}

	chatRoomBanJoinRow = sqlcgen.ListChatRoomBansRow
)

func toChatRoomBanRow(row chatRoomBanJoinRow) model.ChatRoomBanRow {
	return model.ChatRoomBanRow{
		RoomID:            row.RoomID,
		UserID:            row.UserID,
		Username:          row.Username,
		DisplayName:       row.DisplayName,
		AvatarURL:         row.AvatarUrl,
		Role:              row.Role,
		BannedByID:        row.BannedBy,
		BannedByUsername:  row.BannedByUsername,
		BannedByDisplay:   row.BannedByDisplayName,
		BannedByAvatarURL: row.BannedByAvatarUrl,
		Reason:            row.Reason,
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (r *chatRoomBanDAO) Ban(ctx context.Context, s spec.NewChatRoomBan, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).BanFromChatRoom(ctx, sqlcgen.BanFromChatRoomParams{
		RoomID:   s.RoomID,
		UserID:   s.UserID,
		BannedBy: s.BannedBy,
		Reason:   s.Reason,
	})
	if err != nil {
		return fmt.Errorf("ban from room: %w", err)
	}

	return nil
}

func (r *chatRoomBanDAO) Unban(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UnbanFromChatRoom(ctx, sqlcgen.UnbanFromChatRoomParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if err != nil {
		return fmt.Errorf("unban from room: %w", err)
	}

	return nil
}

func (r *chatRoomBanDAO) IsBanned(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error) {
	_, err := genQueries(r.db, tx).IsBannedFromChatRoom(ctx, sqlcgen.IsBannedFromChatRoomParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check room ban: %w", err)
	}

	return true, nil
}

func (r *chatRoomBanDAO) ListForRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatRoomBanRow, error) {
	rows, err := genQueries(r.db, tx).ListChatRoomBans(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("list room bans: %w", err)
	}

	var result []model.ChatRoomBanRow
	for _, row := range rows {
		result = append(result, toChatRoomBanRow(row))
	}

	return result, nil
}

func (r *chatRoomBanDAO) BannedRoomIDsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	ids, err := genQueries(r.db, tx).ListBannedRoomIDsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list banned rooms for user: %w", err)
	}

	return ids, nil
}
