package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type (
	ChatRoomBanRepository interface {
		Ban(ctx context.Context, roomID, userID uuid.UUID, bannedBy *uuid.UUID, reason string) error
		Unban(ctx context.Context, roomID, userID uuid.UUID) error
		IsBanned(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
		ListForRoom(ctx context.Context, roomID uuid.UUID) ([]ChatRoomBanRow, error)
		BannedRoomIDsForUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	}

	chatRoomBanRepository struct {
		db *sql.DB
	}

	ChatRoomBanRow struct {
		RoomID            uuid.UUID
		UserID            uuid.UUID
		Username          string
		DisplayName       string
		AvatarURL         string
		Role              string
		BannedByID        *uuid.UUID
		BannedByUsername  string
		BannedByDisplay   string
		BannedByAvatarURL string
		Reason            string
		CreatedAt         string
	}
)

func (r *chatRoomBanRepository) Ban(ctx context.Context, roomID, userID uuid.UUID, bannedBy *uuid.UUID, reason string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO chat_room_bans (room_id, user_id, banned_by, reason)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (room_id, user_id) DO UPDATE SET
		     banned_by = EXCLUDED.banned_by,
		     reason    = EXCLUDED.reason,
		     created_at = NOW()`,
		roomID, userID, bannedBy, reason,
	)
	if err != nil {
		return fmt.Errorf("ban from room: %w", err)
	}
	return nil
}

func (r *chatRoomBanRepository) Unban(ctx context.Context, roomID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM chat_room_bans WHERE room_id = $1 AND user_id = $2`,
		roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("unban from room: %w", err)
	}
	return nil
}

func (r *chatRoomBanRepository) IsBanned(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx,
		`SELECT 1 FROM chat_room_bans WHERE room_id = $1 AND user_id = $2 LIMIT 1`,
		roomID, userID,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check room ban: %w", err)
	}
	return true, nil
}

func (r *chatRoomBanRepository) ListForRoom(ctx context.Context, roomID uuid.UUID) ([]ChatRoomBanRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT
		     b.room_id, b.user_id,
		     u.username, u.display_name, u.avatar_url, COALESCE(ur.role, ''),
		     b.banned_by,
		     COALESCE(bu.username, ''), COALESCE(bu.display_name, ''), COALESCE(bu.avatar_url, ''),
		     b.reason, b.created_at
		 FROM chat_room_bans b
		 JOIN users u ON b.user_id = u.id
		 LEFT JOIN user_roles ur ON ur.user_id = u.id
		 LEFT JOIN users bu ON b.banned_by = bu.id
		 WHERE b.room_id = $1
		 ORDER BY b.created_at DESC`,
		roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("list room bans: %w", err)
	}
	defer rows.Close()

	var result []ChatRoomBanRow
	for rows.Next() {
		var row ChatRoomBanRow
		if err := rows.Scan(
			&row.RoomID, &row.UserID,
			&row.Username, &row.DisplayName, &row.AvatarURL, &row.Role,
			&row.BannedByID,
			&row.BannedByUsername, &row.BannedByDisplay, &row.BannedByAvatarURL,
			&row.Reason, &row.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan room ban: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *chatRoomBanRepository) BannedRoomIDsForUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT room_id FROM chat_room_bans WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list banned rooms for user: %w", err)
	}
	defer rows.Close()

	var result []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan banned room id: %w", err)
		}
		result = append(result, id)
	}
	return result, rows.Err()
}
