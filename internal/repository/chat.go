package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type (
	ChatRoomRow struct {
		ID        uuid.UUID
		Name      string
		Type      string
		CreatedBy uuid.UUID
		CreatedAt string
	}

	ChatMessageRow struct {
		ID                uuid.UUID
		RoomID            uuid.UUID
		SenderID          uuid.UUID
		SenderUsername    string
		SenderDisplayName string
		SenderAvatarURL   string
		Body              string
		CreatedAt         string
	}

	ChatRepository interface {
		CreateRoom(ctx context.Context, id uuid.UUID, name, roomType string, createdBy uuid.UUID) error
		AddMember(ctx context.Context, roomID, userID uuid.UUID) error
		RemoveMember(ctx context.Context, roomID, userID uuid.UUID) error
		GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]ChatRoomRow, error)
		GetRoomMembers(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error)
		IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
		FindDMRoom(ctx context.Context, userA, userB uuid.UUID) (uuid.UUID, error)

		InsertMessage(ctx context.Context, id, roomID, senderID uuid.UUID, body string) error
		GetMessages(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]ChatMessageRow, int, error)
		DeleteMessages(ctx context.Context, roomID uuid.UUID) error
	}

	chatRepository struct {
		db *sql.DB
	}
)

func (r *chatRepository) CreateRoom(ctx context.Context, id uuid.UUID, name, roomType string, createdBy uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO chat_rooms (id, name, type, created_by) VALUES (?, ?, ?, ?)`,
		id, name, roomType, createdBy,
	)
	if err != nil {
		return fmt.Errorf("create room: %w", err)
	}
	return nil
}

func (r *chatRepository) AddMember(ctx context.Context, roomID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO chat_room_members (room_id, user_id) VALUES (?, ?)`,
		roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

func (r *chatRepository) RemoveMember(ctx context.Context, roomID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM chat_room_members WHERE room_id = ? AND user_id = ?`, roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}

func (r *chatRepository) GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]ChatRoomRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT cr.id, cr.name, cr.type, cr.created_by, cr.created_at
		 FROM chat_rooms cr
		 JOIN chat_room_members m ON cr.id = m.room_id
		 WHERE m.user_id = ?
		 ORDER BY cr.created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get rooms by user: %w", err)
	}
	defer rows.Close()

	var result []ChatRoomRow
	for rows.Next() {
		var row ChatRoomRow
		if err := rows.Scan(&row.ID, &row.Name, &row.Type, &row.CreatedBy, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan room: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *chatRepository) GetRoomMembers(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id FROM chat_room_members WHERE room_id = ?`, roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("get room members: %w", err)
	}
	defer rows.Close()

	var members []uuid.UUID
	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, uid)
	}
	return members, rows.Err()
}

func (r *chatRepository) IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_room_members WHERE room_id = ? AND user_id = ?`, roomID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}
	return count > 0, nil
}

func (r *chatRepository) FindDMRoom(ctx context.Context, userA, userB uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT cr.id FROM chat_rooms cr
		 JOIN chat_room_members m1 ON cr.id = m1.room_id AND m1.user_id = ?
		 JOIN chat_room_members m2 ON cr.id = m2.room_id AND m2.user_id = ?
		 WHERE cr.type = 'dm'
		 LIMIT 1`,
		userA, userB,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("find dm room: %w", err)
	}
	return id, nil
}

func (r *chatRepository) InsertMessage(ctx context.Context, id, roomID, senderID uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO chat_messages (id, room_id, sender_id, body) VALUES (?, ?, ?, ?)`,
		id, roomID, senderID, body,
	)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	return nil
}

func (r *chatRepository) GetMessages(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]ChatMessageRow, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_messages WHERE room_id = ?`, roomID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count messages: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
		 cm.body, cm.created_at
		 FROM chat_messages cm
		 JOIN users u ON cm.sender_id = u.id
		 WHERE cm.room_id = ?
		 ORDER BY cm.created_at DESC
		 LIMIT ? OFFSET ?`,
		roomID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()

	var messages []ChatMessageRow
	for rows.Next() {
		var msg ChatMessageRow
		if err := rows.Scan(
			&msg.ID, &msg.RoomID, &msg.SenderID,
			&msg.SenderUsername, &msg.SenderDisplayName, &msg.SenderAvatarURL,
			&msg.Body, &msg.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, msg)
	}
	return messages, total, rows.Err()
}

func (r *chatRepository) DeleteMessages(ctx context.Context, roomID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM chat_messages WHERE room_id = ?`, roomID)
	if err != nil {
		return fmt.Errorf("delete messages: %w", err)
	}
	return nil
}
