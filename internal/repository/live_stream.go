package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type (
	LiveStreamRow struct {
		ID             uuid.UUID
		UserID         uuid.UUID
		Title          string
		Status         string
		LivekitRoom    string
		IngressID      string
		WhipURL        string
		StreamKey      string
		ViewerCount    int
		StartedAt      sql.NullString
		EndedAt        sql.NullString
		CreatedAt      string
		ThumbnailURL   string
		EgressID       string
		HLSPlaylistURL string
		DefaultMode    string
		Username       string
		DisplayName    string
		AvatarURL      string
	}

	LiveStreamRepository interface {
		Create(ctx context.Context, userID uuid.UUID, title string, maxConcurrent int) (uuid.UUID, error)
		GetByID(ctx context.Context, id uuid.UUID) (*LiveStreamRow, error)
		GetByRoom(ctx context.Context, room string) (*LiveStreamRow, error)
		GetActiveByUser(ctx context.Context, userID uuid.UUID) (*LiveStreamRow, error)
		ListLive(ctx context.Context) ([]LiveStreamRow, error)
		ListStartingBefore(ctx context.Context, cutoff string) ([]LiveStreamRow, error)
		CountActive(ctx context.Context) (int, error)
		SetIngress(ctx context.Context, id uuid.UUID, ingressID, room, whipURL, streamKey string) error
		MarkLive(ctx context.Context, id uuid.UUID) error
		MarkOffline(ctx context.Context, id uuid.UUID) (bool, error)
		AdjustViewerCount(ctx context.Context, id uuid.UUID, delta int) (int, bool, error)
		SetThumbnail(ctx context.Context, id uuid.UUID, url string) error
		SetEgress(ctx context.Context, id uuid.UUID, egressID, hlsURL string) error
		SetDefaultMode(ctx context.Context, id uuid.UUID, mode string) error
	}

	liveStreamRepository struct {
		db *sql.DB
	}
)

var (
	ErrLiveStreamCapacity     = errors.New("live stream capacity reached")
	ErrLiveStreamActiveExists = errors.New("user already has an active live stream")
)

const liveStreamSelectColumns = `s.id, s.user_id, s.title, s.status, s.livekit_room, s.ingress_id,
	s.whip_url, s.stream_key, s.viewer_count, s.started_at, s.ended_at, s.created_at, s.thumbnail_url,
	s.egress_id, s.hls_playlist_url, s.default_mode,
	u.username, u.display_name, u.avatar_url`

func scanLiveStreamRow(scan func(dest ...any) error) (*LiveStreamRow, error) {
	var s LiveStreamRow
	err := scan(&s.ID, &s.UserID, &s.Title, &s.Status, &s.LivekitRoom, &s.IngressID,
		&s.WhipURL, &s.StreamKey, &s.ViewerCount, &s.StartedAt, &s.EndedAt, &s.CreatedAt, &s.ThumbnailURL,
		&s.EgressID, &s.HLSPlaylistURL, &s.DefaultMode,
		&s.Username, &s.DisplayName, &s.AvatarURL)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan live stream: %w", err)
	}

	return &s, nil
}

func (r *liveStreamRepository) Create(ctx context.Context, userID uuid.UUID, title string, maxConcurrent int) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO live_streams (user_id, title, status)
		 SELECT $1, $2, 'starting'
		 WHERE (SELECT COUNT(*) FROM live_streams WHERE status <> 'offline') < $3
		 RETURNING id`,
		userID, title, maxConcurrent,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, ErrLiveStreamCapacity
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return uuid.Nil, ErrLiveStreamActiveExists
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("create live stream: %w", err)
	}

	return id, nil
}

func (r *liveStreamRepository) GetByID(ctx context.Context, id uuid.UUID) (*LiveStreamRow, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+liveStreamSelectColumns+`
		   FROM live_streams s
		   JOIN users u ON u.id = s.user_id
		  WHERE s.id = $1`,
		id,
	)

	return scanLiveStreamRow(row.Scan)
}

func (r *liveStreamRepository) GetByRoom(ctx context.Context, room string) (*LiveStreamRow, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+liveStreamSelectColumns+`
		   FROM live_streams s
		   JOIN users u ON u.id = s.user_id
		  WHERE s.livekit_room = $1`,
		room,
	)

	return scanLiveStreamRow(row.Scan)
}

func (r *liveStreamRepository) GetActiveByUser(ctx context.Context, userID uuid.UUID) (*LiveStreamRow, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+liveStreamSelectColumns+`
		   FROM live_streams s
		   JOIN users u ON u.id = s.user_id
		  WHERE s.user_id = $1 AND s.status <> 'offline'
		  LIMIT 1`,
		userID,
	)

	return scanLiveStreamRow(row.Scan)
}

func (r *liveStreamRepository) listBy(ctx context.Context, where string, args ...any) ([]LiveStreamRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+liveStreamSelectColumns+`
		   FROM live_streams s
		   JOIN users u ON u.id = s.user_id
		  WHERE `+where,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list live streams: %w", err)
	}
	defer rows.Close()

	var result []LiveStreamRow
	for rows.Next() {
		parsed, scanErr := scanLiveStreamRow(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *parsed)
	}

	return result, rows.Err()
}

func (r *liveStreamRepository) ListLive(ctx context.Context) ([]LiveStreamRow, error) {
	return r.listBy(ctx, "s.status = 'live' ORDER BY s.started_at DESC")
}

func (r *liveStreamRepository) ListStartingBefore(ctx context.Context, cutoff string) ([]LiveStreamRow, error) {
	return r.listBy(ctx, "s.status = 'starting' AND s.created_at < $1::timestamptz", cutoff)
}

func (r *liveStreamRepository) CountActive(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM live_streams WHERE status <> 'offline'`,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count active live streams: %w", err)
	}

	return n, nil
}

func (r *liveStreamRepository) SetIngress(ctx context.Context, id uuid.UUID, ingressID, room, whipURL, streamKey string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE live_streams
		    SET ingress_id = $2, livekit_room = $3, whip_url = $4, stream_key = $5
		  WHERE id = $1`,
		id, ingressID, room, whipURL, streamKey,
	)
	if err != nil {
		return fmt.Errorf("set live stream ingress: %w", err)
	}

	return nil
}

func (r *liveStreamRepository) MarkLive(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE live_streams
		    SET status = 'live', started_at = COALESCE(started_at, NOW())
		  WHERE id = $1 AND status <> 'offline'`,
		id,
	)
	if err != nil {
		return fmt.Errorf("mark live stream live: %w", err)
	}

	return nil
}

func (r *liveStreamRepository) MarkOffline(ctx context.Context, id uuid.UUID) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE live_streams
		    SET status = 'offline', ended_at = NOW(), viewer_count = 0
		  WHERE id = $1 AND status <> 'offline'`,
		id,
	)
	if err != nil {
		return false, fmt.Errorf("mark live stream offline: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("mark live stream offline rows: %w", err)
	}

	return affected > 0, nil
}

func (r *liveStreamRepository) AdjustViewerCount(ctx context.Context, id uuid.UUID, delta int) (int, bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`UPDATE live_streams
		    SET viewer_count = GREATEST(0, viewer_count + $2)
		  WHERE id = $1 AND status = 'live'
		  RETURNING viewer_count`,
		id, delta,
	).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("adjust live stream viewer count: %w", err)
	}

	return count, true, nil
}

func (r *liveStreamRepository) SetThumbnail(ctx context.Context, id uuid.UUID, url string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE live_streams SET thumbnail_url = $2 WHERE id = $1`,
		id, url,
	)
	if err != nil {
		return fmt.Errorf("set live stream thumbnail: %w", err)
	}

	return nil
}

func (r *liveStreamRepository) SetEgress(ctx context.Context, id uuid.UUID, egressID, hlsURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE live_streams SET egress_id = $2, hls_playlist_url = $3 WHERE id = $1`,
		id, egressID, hlsURL,
	)
	if err != nil {
		return fmt.Errorf("set live stream egress: %w", err)
	}

	return nil
}

func (r *liveStreamRepository) SetDefaultMode(ctx context.Context, id uuid.UUID, mode string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE live_streams SET default_mode = $2 WHERE id = $1`,
		id, mode,
	)
	if err != nil {
		return fmt.Errorf("set live stream default mode: %w", err)
	}

	return nil
}
