package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	ChatWatchPartyDAO interface {
		CreateSession(ctx context.Context, row model.ChatWatchPartySessionRow, tx ...*sql.Tx) (*model.ChatWatchPartySessionRow, error)
		GetByID(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) (*model.ChatWatchPartySessionRow, error)
		ListActiveByRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatWatchPartySessionRow, error)
		EndSession(ctx context.Context, s spec.WatchPartySessionEnd, tx ...*sql.Tx) error
		SetControllerID(ctx context.Context, s spec.WatchPartyControllerUpdate, tx ...*sql.Tx) error

		UpsertParticipant(ctx context.Context, s spec.WatchPartyParticipantUpsert, tx ...*sql.Tx) error
		SetParticipantIdentifier(ctx context.Context, s spec.WatchPartyIdentifierUpdate, tx ...*sql.Tx) error
		MarkParticipantLeft(ctx context.Context, s spec.WatchPartyParticipantRef, tx ...*sql.Tx) error
		MarkAllParticipantsLeft(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) error
		GetActiveParticipants(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) ([]model.ChatWatchPartyParticipantRow, error)
		GetParticipant(ctx context.Context, s spec.WatchPartyParticipantRef, tx ...*sql.Tx) (*model.ChatWatchPartyParticipantRow, error)
		SetParticipantControl(ctx context.Context, s spec.WatchPartyControlUpdate, tx ...*sql.Tx) error
		CountActiveParticipants(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) (int, error)

		ListIdleActiveSessions(ctx context.Context, idleBefore string, tx ...*sql.Tx) ([]model.ChatWatchPartySessionRow, error)
	}

	chatWatchPartyDAO struct {
		db *sql.DB
	}
)

func (r *chatWatchPartyDAO) CreateSession(ctx context.Context, row model.ChatWatchPartySessionRow, tx ...*sql.Tx) (*model.ChatWatchPartySessionRow, error) {
	created, err := scanSessionRow(txOrDB(r.db, tx).QueryRowContext(ctx,
		`INSERT INTO chat_watch_party_sessions
		    (room_id, started_by, controller_id, hyperbeam_session_id, hyperbeam_admin_token, embed_url, vm_base_url, title, type, start_url, region, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'active')
		 RETURNING id, room_id, started_by, controller_id, hyperbeam_session_id, hyperbeam_admin_token, embed_url, vm_base_url,
		           title, type, start_url, region, status, started_at, ended_at, ended_reason`,
		row.RoomID, row.StartedBy, row.ControllerID, row.HyperbeamSessionID, row.HyperbeamAdminToken, row.EmbedURL,
		row.VMBaseURL, row.Title, row.Type, row.StartURL, row.Region,
	))
	if err != nil {
		return nil, fmt.Errorf("create watch party session: %w", err)
	}

	return created, nil
}

func (r *chatWatchPartyDAO) ListActiveByRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatWatchPartySessionRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
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

	var result []model.ChatWatchPartySessionRow
	for rows.Next() {
		var s model.ChatWatchPartySessionRow
		if err := rows.Scan(&s.ID, &s.RoomID, &s.StartedBy, &s.ControllerID, &s.HyperbeamSessionID, &s.HyperbeamAdminToken,
			&s.EmbedURL, &s.VMBaseURL, &s.Title, &s.Type, &s.StartURL, &s.Region, &s.Status, &s.StartedAt, &s.EndedAt, &s.EndedReason); err != nil {
			return nil, fmt.Errorf("scan active watch party: %w", err)
		}

		result = append(result, s)
	}

	return result, rows.Err()
}

func (r *chatWatchPartyDAO) GetByID(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) (*model.ChatWatchPartySessionRow, error) {
	row := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT id, room_id, started_by, controller_id, hyperbeam_session_id, hyperbeam_admin_token, embed_url, vm_base_url,
		        title, type, start_url, region, status, started_at, ended_at, ended_reason
		 FROM chat_watch_party_sessions
		 WHERE id = $1`,
		sessionID,
	)

	return scanSessionRow(row)
}

func scanSessionRow(row *sql.Row) (*model.ChatWatchPartySessionRow, error) {
	var s model.ChatWatchPartySessionRow

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

func (r *chatWatchPartyDAO) EndSession(ctx context.Context, s spec.WatchPartySessionEnd, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_watch_party_sessions
		    SET status = 'ended', ended_at = NOW(), ended_reason = $2
		  WHERE id = $1 AND status = 'active'`,
		s.SessionID, s.Reason,
	)
	if err != nil {
		return fmt.Errorf("end watch party session: %w", err)
	}

	return nil
}

func (r *chatWatchPartyDAO) SetControllerID(ctx context.Context, s spec.WatchPartyControllerUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_watch_party_sessions SET controller_id = $2 WHERE id = $1`,
		s.SessionID, s.ControllerID,
	)
	if err != nil {
		return fmt.Errorf("set watch party controller: %w", err)
	}

	return nil
}

func (r *chatWatchPartyDAO) UpsertParticipant(ctx context.Context, s spec.WatchPartyParticipantUpsert, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO chat_watch_party_participants (session_id, user_id, has_control, hyperbeam_identifier)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (session_id, user_id) DO UPDATE SET
		     has_control = EXCLUDED.has_control,
		     hyperbeam_identifier = EXCLUDED.hyperbeam_identifier,
		     left_at = NULL,
		     joined_at = NOW()`,
		s.SessionID, s.UserID, s.HasControl, s.Identifier,
	)
	if err != nil {
		return fmt.Errorf("upsert watch party participant: %w", err)
	}

	return nil
}

func (r *chatWatchPartyDAO) SetParticipantIdentifier(ctx context.Context, s spec.WatchPartyIdentifierUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_watch_party_participants SET hyperbeam_identifier = $3 WHERE session_id = $1 AND user_id = $2`,
		s.SessionID, s.UserID, s.Identifier,
	)
	if err != nil {
		return fmt.Errorf("set watch party participant identifier: %w", err)
	}

	return nil
}

func (r *chatWatchPartyDAO) MarkParticipantLeft(ctx context.Context, s spec.WatchPartyParticipantRef, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_watch_party_participants
		    SET left_at = NOW(), has_control = FALSE
		  WHERE session_id = $1 AND user_id = $2 AND left_at IS NULL`,
		s.SessionID, s.UserID,
	)
	if err != nil {
		return fmt.Errorf("mark watch party participant left: %w", err)
	}

	return nil
}

func (r *chatWatchPartyDAO) MarkAllParticipantsLeft(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
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

func (r *chatWatchPartyDAO) GetActiveParticipants(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) ([]model.ChatWatchPartyParticipantRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
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

	var result []model.ChatWatchPartyParticipantRow
	for rows.Next() {
		var p model.ChatWatchPartyParticipantRow
		if err := rows.Scan(&p.SessionID, &p.UserID, &p.Username, &p.DisplayName, &p.AvatarURL, &p.HasControl, &p.HyperbeamIdentifier, &p.JoinedAt, &p.LeftAt); err != nil {
			return nil, fmt.Errorf("scan watch party participant: %w", err)
		}

		result = append(result, p)
	}

	return result, rows.Err()
}

func (r *chatWatchPartyDAO) GetParticipant(ctx context.Context, s spec.WatchPartyParticipantRef, tx ...*sql.Tx) (*model.ChatWatchPartyParticipantRow, error) {
	row := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT p.session_id, p.user_id, u.username, u.display_name, u.avatar_url, p.has_control, p.hyperbeam_identifier, p.joined_at, p.left_at
		   FROM chat_watch_party_participants p
		   JOIN users u ON u.id = p.user_id
		  WHERE p.session_id = $1 AND p.user_id = $2
		  LIMIT 1`,
		s.SessionID, s.UserID,
	)

	var p model.ChatWatchPartyParticipantRow

	err := row.Scan(&p.SessionID, &p.UserID, &p.Username, &p.DisplayName, &p.AvatarURL, &p.HasControl, &p.HyperbeamIdentifier, &p.JoinedAt, &p.LeftAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan watch party participant: %w", err)
	}

	return &p, nil
}

func (r *chatWatchPartyDAO) SetParticipantControl(ctx context.Context, s spec.WatchPartyControlUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_watch_party_participants SET has_control = $3 WHERE session_id = $1 AND user_id = $2 AND left_at IS NULL`,
		s.SessionID, s.UserID, s.HasControl,
	)
	if err != nil {
		return fmt.Errorf("set watch party participant control: %w", err)
	}

	return nil
}

func (r *chatWatchPartyDAO) CountActiveParticipants(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var n int

	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_watch_party_participants WHERE session_id = $1 AND left_at IS NULL`,
		sessionID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count watch party participants: %w", err)
	}

	return n, nil
}

func (r *chatWatchPartyDAO) ListIdleActiveSessions(ctx context.Context, idleBefore string, tx ...*sql.Tx) ([]model.ChatWatchPartySessionRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
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

	var result []model.ChatWatchPartySessionRow
	for rows.Next() {
		var s model.ChatWatchPartySessionRow
		if err := rows.Scan(&s.ID, &s.RoomID, &s.StartedBy, &s.ControllerID, &s.HyperbeamSessionID, &s.HyperbeamAdminToken,
			&s.EmbedURL, &s.VMBaseURL, &s.Title, &s.Type, &s.StartURL, &s.Region, &s.Status, &s.StartedAt, &s.EndedAt, &s.EndedReason); err != nil {
			return nil, fmt.Errorf("scan idle watch party session: %w", err)
		}

		result = append(result, s)
	}

	return result, rows.Err()
}
