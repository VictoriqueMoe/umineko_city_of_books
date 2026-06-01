package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/watchparty"

	"github.com/google/uuid"
)

type (
	ChatWatchPartySessionRow struct {
		ID                  uuid.UUID
		RoomID              uuid.UUID
		StartedBy           uuid.UUID
		ControllerID        uuid.UUID
		HyperbeamSessionID  string
		HyperbeamAdminToken string
		EmbedURL            string
		VMBaseURL           string
		Title               string
		Type                string
		StartURL            sql.NullString
		Region              sql.NullString
		Status              string
		StartedAt           string
		EndedAt             sql.NullString
		EndedReason         sql.NullString
	}

	ChatWatchPartyParticipantRow struct {
		SessionID           uuid.UUID
		UserID              uuid.UUID
		Username            string
		DisplayName         string
		AvatarURL           string
		HasControl          bool
		HyperbeamIdentifier string
		JoinedAt            string
		LeftAt              sql.NullString
	}

	ChatWatchPartyMessageRow struct {
		ID                uuid.UUID
		SessionID         uuid.UUID
		Kind              watchparty.MessageKind
		SenderID          uuid.NullUUID
		SenderUsername    sql.NullString
		SenderDisplayName sql.NullString
		SenderAvatarURL   sql.NullString
		Body              string
		CreatedAt         string
	}

	ChatWatchPartyRepository interface {
		CreateSession(ctx context.Context, row ChatWatchPartySessionRow) (uuid.UUID, error)
		GetByID(ctx context.Context, sessionID uuid.UUID) (*ChatWatchPartySessionRow, error)
		ListActiveByRoom(ctx context.Context, roomID uuid.UUID) ([]ChatWatchPartySessionRow, error)
		EndSession(ctx context.Context, sessionID uuid.UUID, reason string) error
		SetControllerID(ctx context.Context, sessionID, controllerID uuid.UUID) error

		UpsertParticipant(ctx context.Context, sessionID, userID uuid.UUID, hasControl bool, identifier string) error
		SetParticipantIdentifier(ctx context.Context, sessionID, userID uuid.UUID, identifier string) error
		MarkParticipantLeft(ctx context.Context, sessionID, userID uuid.UUID) error
		MarkAllParticipantsLeft(ctx context.Context, sessionID uuid.UUID) error
		GetActiveParticipants(ctx context.Context, sessionID uuid.UUID) ([]ChatWatchPartyParticipantRow, error)
		GetParticipant(ctx context.Context, sessionID, userID uuid.UUID) (*ChatWatchPartyParticipantRow, error)
		SetParticipantControl(ctx context.Context, sessionID, userID uuid.UUID, hasControl bool) error
		CountActiveParticipants(ctx context.Context, sessionID uuid.UUID) (int, error)

		InsertMessage(ctx context.Context, id, sessionID, senderID uuid.UUID, body string) error
		InsertSystemMessage(ctx context.Context, id, sessionID uuid.UUID, body string) error
		ListMessages(ctx context.Context, sessionID uuid.UUID, limit int) ([]ChatWatchPartyMessageRow, error)
		GetMessageByID(ctx context.Context, messageID uuid.UUID) (*ChatWatchPartyMessageRow, error)

		ListIdleActiveSessions(ctx context.Context, idleBefore string) ([]ChatWatchPartySessionRow, error)
	}

	chatWatchPartyRepository struct {
		db *sql.DB
	}
)

func (r *chatWatchPartyRepository) CreateSession(ctx context.Context, row ChatWatchPartySessionRow) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO chat_watch_party_sessions
		    (room_id, started_by, controller_id, hyperbeam_session_id, hyperbeam_admin_token, embed_url, vm_base_url, title, type, start_url, region, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'active')
		 RETURNING id`,
		row.RoomID, row.StartedBy, row.ControllerID, row.HyperbeamSessionID, row.HyperbeamAdminToken, row.EmbedURL,
		row.VMBaseURL, row.Title, row.Type, row.StartURL, row.Region,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create watch party session: %w", err)
	}
	return id, nil
}

func (r *chatWatchPartyRepository) ListActiveByRoom(ctx context.Context, roomID uuid.UUID) ([]ChatWatchPartySessionRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, room_id, started_by, controller_id, hyperbeam_session_id, hyperbeam_admin_token, embed_url, vm_base_url,
		        title, type, start_url, region, status, started_at, ended_at, ended_reason
		 FROM chat_watch_party_sessions
		 WHERE room_id = $1 AND status = 'active'
		 ORDER BY started_at ASC`,
		roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("list active watch parties: %w", err)
	}
	defer rows.Close()
	var result []ChatWatchPartySessionRow
	for rows.Next() {
		var s ChatWatchPartySessionRow
		if err := rows.Scan(&s.ID, &s.RoomID, &s.StartedBy, &s.ControllerID, &s.HyperbeamSessionID, &s.HyperbeamAdminToken,
			&s.EmbedURL, &s.VMBaseURL, &s.Title, &s.Type, &s.StartURL, &s.Region, &s.Status, &s.StartedAt, &s.EndedAt, &s.EndedReason); err != nil {
			return nil, fmt.Errorf("scan active watch party: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *chatWatchPartyRepository) GetByID(ctx context.Context, sessionID uuid.UUID) (*ChatWatchPartySessionRow, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, room_id, started_by, controller_id, hyperbeam_session_id, hyperbeam_admin_token, embed_url, vm_base_url,
		        title, type, start_url, region, status, started_at, ended_at, ended_reason
		 FROM chat_watch_party_sessions
		 WHERE id = $1`,
		sessionID,
	)
	return scanSessionRow(row)
}

func scanSessionRow(row *sql.Row) (*ChatWatchPartySessionRow, error) {
	var s ChatWatchPartySessionRow
	err := row.Scan(&s.ID, &s.RoomID, &s.StartedBy, &s.ControllerID, &s.HyperbeamSessionID, &s.HyperbeamAdminToken,
		&s.EmbedURL, &s.VMBaseURL, &s.Title, &s.Type, &s.StartURL, &s.Region, &s.Status, &s.StartedAt, &s.EndedAt, &s.EndedReason)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan watch party session: %w", err)
	}
	return &s, nil
}

func (r *chatWatchPartyRepository) EndSession(ctx context.Context, sessionID uuid.UUID, reason string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_watch_party_sessions
		    SET status = 'ended', ended_at = NOW(), ended_reason = $2
		  WHERE id = $1 AND status = 'active'`,
		sessionID, reason,
	)
	if err != nil {
		return fmt.Errorf("end watch party session: %w", err)
	}
	return nil
}

func (r *chatWatchPartyRepository) SetControllerID(ctx context.Context, sessionID, controllerID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_watch_party_sessions SET controller_id = $2 WHERE id = $1`,
		sessionID, controllerID,
	)
	if err != nil {
		return fmt.Errorf("set watch party controller: %w", err)
	}
	return nil
}

func (r *chatWatchPartyRepository) UpsertParticipant(ctx context.Context, sessionID, userID uuid.UUID, hasControl bool, identifier string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO chat_watch_party_participants (session_id, user_id, has_control, hyperbeam_identifier)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (session_id, user_id) DO UPDATE SET
		     has_control = EXCLUDED.has_control,
		     hyperbeam_identifier = EXCLUDED.hyperbeam_identifier,
		     left_at = NULL,
		     joined_at = NOW()`,
		sessionID, userID, hasControl, identifier,
	)
	if err != nil {
		return fmt.Errorf("upsert watch party participant: %w", err)
	}
	return nil
}

func (r *chatWatchPartyRepository) SetParticipantIdentifier(ctx context.Context, sessionID, userID uuid.UUID, identifier string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_watch_party_participants SET hyperbeam_identifier = $3 WHERE session_id = $1 AND user_id = $2`,
		sessionID, userID, identifier,
	)
	if err != nil {
		return fmt.Errorf("set watch party participant identifier: %w", err)
	}
	return nil
}

func (r *chatWatchPartyRepository) MarkParticipantLeft(ctx context.Context, sessionID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_watch_party_participants
		    SET left_at = NOW(), has_control = FALSE
		  WHERE session_id = $1 AND user_id = $2 AND left_at IS NULL`,
		sessionID, userID,
	)
	if err != nil {
		return fmt.Errorf("mark watch party participant left: %w", err)
	}
	return nil
}

func (r *chatWatchPartyRepository) MarkAllParticipantsLeft(ctx context.Context, sessionID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_watch_party_participants
		    SET left_at = NOW(), has_control = FALSE
		  WHERE session_id = $1 AND left_at IS NULL`,
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("mark all watch party participants left: %w", err)
	}
	return nil
}

func (r *chatWatchPartyRepository) GetActiveParticipants(ctx context.Context, sessionID uuid.UUID) ([]ChatWatchPartyParticipantRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT p.session_id, p.user_id, u.username, u.display_name, u.avatar_url, p.has_control, p.hyperbeam_identifier, p.joined_at, p.left_at
		   FROM chat_watch_party_participants p
		   JOIN users u ON u.id = p.user_id
		  WHERE p.session_id = $1 AND p.left_at IS NULL
		  ORDER BY p.joined_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list watch party participants: %w", err)
	}
	defer rows.Close()

	var result []ChatWatchPartyParticipantRow
	for rows.Next() {
		var p ChatWatchPartyParticipantRow
		if err := rows.Scan(&p.SessionID, &p.UserID, &p.Username, &p.DisplayName, &p.AvatarURL, &p.HasControl, &p.HyperbeamIdentifier, &p.JoinedAt, &p.LeftAt); err != nil {
			return nil, fmt.Errorf("scan watch party participant: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *chatWatchPartyRepository) GetParticipant(ctx context.Context, sessionID, userID uuid.UUID) (*ChatWatchPartyParticipantRow, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT p.session_id, p.user_id, u.username, u.display_name, u.avatar_url, p.has_control, p.hyperbeam_identifier, p.joined_at, p.left_at
		   FROM chat_watch_party_participants p
		   JOIN users u ON u.id = p.user_id
		  WHERE p.session_id = $1 AND p.user_id = $2
		  LIMIT 1`,
		sessionID, userID,
	)
	var p ChatWatchPartyParticipantRow
	err := row.Scan(&p.SessionID, &p.UserID, &p.Username, &p.DisplayName, &p.AvatarURL, &p.HasControl, &p.HyperbeamIdentifier, &p.JoinedAt, &p.LeftAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan watch party participant: %w", err)
	}
	return &p, nil
}

func (r *chatWatchPartyRepository) SetParticipantControl(ctx context.Context, sessionID, userID uuid.UUID, hasControl bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_watch_party_participants SET has_control = $3 WHERE session_id = $1 AND user_id = $2 AND left_at IS NULL`,
		sessionID, userID, hasControl,
	)
	if err != nil {
		return fmt.Errorf("set watch party participant control: %w", err)
	}
	return nil
}

func (r *chatWatchPartyRepository) CountActiveParticipants(ctx context.Context, sessionID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_watch_party_participants WHERE session_id = $1 AND left_at IS NULL`,
		sessionID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count watch party participants: %w", err)
	}
	return n, nil
}

func (r *chatWatchPartyRepository) ListIdleActiveSessions(ctx context.Context, idleBefore string) ([]ChatWatchPartySessionRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT s.id, s.room_id, s.started_by, s.controller_id, s.hyperbeam_session_id, s.hyperbeam_admin_token,
		        s.embed_url, s.vm_base_url, s.title, s.type, s.start_url, s.region, s.status, s.started_at, s.ended_at, s.ended_reason
		   FROM chat_watch_party_sessions s
		  WHERE s.status = 'active'
		    AND s.started_at < $1::timestamptz
		    AND NOT EXISTS (
		        SELECT 1 FROM chat_watch_party_participants p
		         WHERE p.session_id = s.id AND p.left_at IS NULL
		    )`,
		idleBefore,
	)
	if err != nil {
		return nil, fmt.Errorf("list idle watch party sessions: %w", err)
	}
	defer rows.Close()

	var result []ChatWatchPartySessionRow
	for rows.Next() {
		var s ChatWatchPartySessionRow
		if err := rows.Scan(&s.ID, &s.RoomID, &s.StartedBy, &s.ControllerID, &s.HyperbeamSessionID, &s.HyperbeamAdminToken,
			&s.EmbedURL, &s.VMBaseURL, &s.Title, &s.Type, &s.StartURL, &s.Region, &s.Status, &s.StartedAt, &s.EndedAt, &s.EndedReason); err != nil {
			return nil, fmt.Errorf("scan idle watch party session: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *chatWatchPartyRepository) InsertMessage(ctx context.Context, id, sessionID, senderID uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO chat_watch_party_messages (id, session_id, sender_id, body, kind) VALUES ($1, $2, $3, $4, 'user')`,
		id, sessionID, senderID, body,
	)
	if err != nil {
		return fmt.Errorf("insert watch party message: %w", err)
	}
	return nil
}

func (r *chatWatchPartyRepository) InsertSystemMessage(ctx context.Context, id, sessionID uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO chat_watch_party_messages (id, session_id, sender_id, body, kind) VALUES ($1, $2, NULL, $3, 'system')`,
		id, sessionID, body,
	)
	if err != nil {
		return fmt.Errorf("insert watch party system message: %w", err)
	}
	return nil
}

func (r *chatWatchPartyRepository) ListMessages(ctx context.Context, sessionID uuid.UUID, limit int) ([]ChatWatchPartyMessageRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT m.id, m.session_id, m.kind, m.sender_id, u.username, u.display_name, u.avatar_url, m.body, m.created_at
		   FROM chat_watch_party_messages m
		   LEFT JOIN users u ON u.id = m.sender_id
		  WHERE m.session_id = $1
		  ORDER BY m.created_at ASC
		  LIMIT $2`,
		sessionID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list watch party messages: %w", err)
	}
	defer rows.Close()
	var result []ChatWatchPartyMessageRow
	for rows.Next() {
		var m ChatWatchPartyMessageRow
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Kind, &m.SenderID, &m.SenderUsername, &m.SenderDisplayName, &m.SenderAvatarURL, &m.Body, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan watch party message: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *chatWatchPartyRepository) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*ChatWatchPartyMessageRow, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT m.id, m.session_id, m.kind, m.sender_id, u.username, u.display_name, u.avatar_url, m.body, m.created_at
		   FROM chat_watch_party_messages m
		   LEFT JOIN users u ON u.id = m.sender_id
		  WHERE m.id = $1`,
		messageID,
	)
	var m ChatWatchPartyMessageRow
	err := row.Scan(&m.ID, &m.SessionID, &m.Kind, &m.SenderID, &m.SenderUsername, &m.SenderDisplayName, &m.SenderAvatarURL, &m.Body, &m.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan watch party message: %w", err)
	}
	return &m, nil
}
