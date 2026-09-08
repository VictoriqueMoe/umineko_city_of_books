package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	LiveStreamDAO interface {
		Create(ctx context.Context, s spec.NewLiveStream, tx ...*sql.Tx) (*model.LiveStreamRow, error)
		GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.LiveStreamRow, error)
		GetByRoom(ctx context.Context, room string, tx ...*sql.Tx) (*model.LiveStreamRow, error)
		GetActiveByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (*model.LiveStreamRow, error)
		GetActiveByUsername(ctx context.Context, username string, tx ...*sql.Tx) (*model.LiveStreamRow, error)
		ListLive(ctx context.Context, tx ...*sql.Tx) ([]model.LiveStreamRow, error)
		ListStartingBefore(ctx context.Context, cutoff string, tx ...*sql.Tx) ([]model.LiveStreamRow, error)
		CountActive(ctx context.Context, tx ...*sql.Tx) (int, error)
		SetIngress(ctx context.Context, s spec.LiveStreamIngressUpdate, tx ...*sql.Tx) error
		MarkLive(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		MarkOffline(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (bool, error)
		AdjustViewerCount(ctx context.Context, s spec.LiveStreamViewerAdjustment, tx ...*sql.Tx) (int, bool, error)
		SetThumbnail(ctx context.Context, s spec.LiveStreamThumbnailUpdate, tx ...*sql.Tx) error
		SetEgress(ctx context.Context, s spec.LiveStreamEgressUpdate, tx ...*sql.Tx) error
		SetDefaultMode(ctx context.Context, s spec.LiveStreamDefaultModeUpdate, tx ...*sql.Tx) error
		SetTitle(ctx context.Context, s spec.LiveStreamTitleUpdate, tx ...*sql.Tx) error
	}

	liveStreamDAO struct {
		db *sql.DB
	}

	liveStreamJoinRow = sqlcgen.GetLiveStreamByIDRow
)

var (
	ErrLiveStreamCapacity     = errors.New("live stream capacity reached")
	ErrLiveStreamActiveExists = errors.New("user already has an active live stream")
)

func liveStreamNullTime(nt sql.NullTime) sql.NullString {
	if !nt.Valid {
		return sql.NullString{}
	}

	return sql.NullString{String: nt.Time.UTC().Format(time.RFC3339), Valid: true}
}

func toLiveStreamRow(row liveStreamJoinRow) model.LiveStreamRow {
	return model.LiveStreamRow{
		ID:             row.ID,
		UserID:         row.UserID,
		Title:          row.Title,
		Status:         row.Status,
		LivekitRoom:    row.LivekitRoom,
		IngressID:      row.IngressID,
		WhipURL:        row.WhipUrl,
		StreamKey:      row.StreamKey,
		ViewerCount:    int(row.ViewerCount),
		StartedAt:      liveStreamNullTime(row.StartedAt),
		EndedAt:        liveStreamNullTime(row.EndedAt),
		CreatedAt:      row.CreatedAt.UTC().Format(time.RFC3339),
		ThumbnailURL:   row.ThumbnailUrl,
		EgressID:       row.EgressID,
		HLSPlaylistURL: row.HlsPlaylistUrl,
		DefaultMode:    string(row.DefaultMode),
		Username:       row.Username,
		DisplayName:    row.DisplayName,
		AvatarURL:      row.AvatarUrl,
	}
}

func (r *liveStreamDAO) Create(ctx context.Context, s spec.NewLiveStream, tx ...*sql.Tx) (*model.LiveStreamRow, error) {
	created, err := genQueries(r.db, tx).CreateLiveStream(ctx, sqlcgen.CreateLiveStreamParams{
		UserID:        s.UserID,
		Title:         s.Title,
		MaxConcurrent: int32(s.MaxConcurrent),
	})
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		return nil, ErrLiveStreamActiveExists
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrLiveStreamCapacity
	}
	if err != nil {
		return nil, fmt.Errorf("create live stream: %w", err)
	}

	return new(toLiveStreamRow(liveStreamJoinRow(created))), nil
}

func (r *liveStreamDAO) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.LiveStreamRow, error) {
	row, err := genQueries(r.db, tx).GetLiveStreamByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get live stream: %w", err)
	}

	return new(toLiveStreamRow(row)), nil
}

func (r *liveStreamDAO) GetByRoom(ctx context.Context, room string, tx ...*sql.Tx) (*model.LiveStreamRow, error) {
	row, err := genQueries(r.db, tx).GetLiveStreamByRoom(ctx, room)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get live stream by room: %w", err)
	}

	return new(toLiveStreamRow(liveStreamJoinRow(row))), nil
}

func (r *liveStreamDAO) GetActiveByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (*model.LiveStreamRow, error) {
	row, err := genQueries(r.db, tx).GetActiveLiveStreamByUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active live stream by user: %w", err)
	}

	return new(toLiveStreamRow(liveStreamJoinRow(row))), nil
}

func (r *liveStreamDAO) GetActiveByUsername(ctx context.Context, username string, tx ...*sql.Tx) (*model.LiveStreamRow, error) {
	row, err := genQueries(r.db, tx).GetActiveLiveStreamByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active live stream by username: %w", err)
	}

	return new(toLiveStreamRow(liveStreamJoinRow(row))), nil
}

func (r *liveStreamDAO) ListLive(ctx context.Context, tx ...*sql.Tx) ([]model.LiveStreamRow, error) {
	rows, err := genQueries(r.db, tx).ListLiveStreamsLive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list live streams: %w", err)
	}

	var result []model.LiveStreamRow
	for _, row := range rows {
		result = append(result, toLiveStreamRow(liveStreamJoinRow(row)))
	}

	return result, nil
}

func (r *liveStreamDAO) ListStartingBefore(ctx context.Context, cutoff string, tx ...*sql.Tx) ([]model.LiveStreamRow, error) {
	rows, err := genQueries(r.db, tx).ListLiveStreamsStartingBefore(ctx, cutoff)
	if err != nil {
		return nil, fmt.Errorf("list live streams: %w", err)
	}

	var result []model.LiveStreamRow
	for _, row := range rows {
		result = append(result, toLiveStreamRow(liveStreamJoinRow(row)))
	}

	return result, nil
}

func (r *liveStreamDAO) CountActive(ctx context.Context, tx ...*sql.Tx) (int, error) {
	n, err := genQueries(r.db, tx).CountActiveLiveStreams(ctx)
	if err != nil {
		return 0, fmt.Errorf("count active live streams: %w", err)
	}

	return int(n), nil
}

func (r *liveStreamDAO) SetIngress(ctx context.Context, s spec.LiveStreamIngressUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetLiveStreamIngress(ctx, sqlcgen.SetLiveStreamIngressParams{
		ID:          s.ID,
		IngressID:   s.IngressID,
		LivekitRoom: s.Room,
		WhipUrl:     s.WhipURL,
		StreamKey:   s.StreamKey,
	})
	if err != nil {
		return fmt.Errorf("set live stream ingress: %w", err)
	}

	return nil
}

func (r *liveStreamDAO) MarkLive(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).MarkLiveStreamLive(ctx, id); err != nil {
		return fmt.Errorf("mark live stream live: %w", err)
	}

	return nil
}

func (r *liveStreamDAO) MarkOffline(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (bool, error) {
	affected, err := genQueries(r.db, tx).MarkLiveStreamOffline(ctx, id)
	if err != nil {
		return false, fmt.Errorf("mark live stream offline: %w", err)
	}

	return affected > 0, nil
}

func (r *liveStreamDAO) AdjustViewerCount(ctx context.Context, s spec.LiveStreamViewerAdjustment, tx ...*sql.Tx) (int, bool, error) {
	count, err := genQueries(r.db, tx).AdjustLiveStreamViewerCount(ctx, sqlcgen.AdjustLiveStreamViewerCountParams{
		Delta: int32(s.Delta),
		ID:    s.ID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("adjust live stream viewer count: %w", err)
	}

	return int(count), true, nil
}

func (r *liveStreamDAO) SetThumbnail(ctx context.Context, s spec.LiveStreamThumbnailUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetLiveStreamThumbnail(ctx, sqlcgen.SetLiveStreamThumbnailParams{
		ID:           s.ID,
		ThumbnailUrl: s.URL,
	})
	if err != nil {
		return fmt.Errorf("set live stream thumbnail: %w", err)
	}

	return nil
}

func (r *liveStreamDAO) SetEgress(ctx context.Context, s spec.LiveStreamEgressUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetLiveStreamEgress(ctx, sqlcgen.SetLiveStreamEgressParams{
		ID:             s.ID,
		EgressID:       s.EgressID,
		HlsPlaylistUrl: s.HLSURL,
	})
	if err != nil {
		return fmt.Errorf("set live stream egress: %w", err)
	}

	return nil
}

func (r *liveStreamDAO) SetDefaultMode(ctx context.Context, s spec.LiveStreamDefaultModeUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetLiveStreamDefaultMode(ctx, sqlcgen.SetLiveStreamDefaultModeParams{
		ID:          s.ID,
		DefaultMode: sqlcgen.StreamDefaultMode(s.DefaultMode),
	})
	if err != nil {
		return fmt.Errorf("set live stream default mode: %w", err)
	}

	return nil
}

func (r *liveStreamDAO) SetTitle(ctx context.Context, s spec.LiveStreamTitleUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetLiveStreamTitle(ctx, sqlcgen.SetLiveStreamTitleParams{
		ID:    s.ID,
		Title: s.Title,
	})
	if err != nil {
		return fmt.Errorf("set live stream title: %w", err)
	}

	return nil
}
