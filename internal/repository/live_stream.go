package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
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
		SetTitle(ctx context.Context, id uuid.UUID, title string) error
	}
)

var (
	ErrLiveStreamCapacity     = errors.New("live stream capacity reached")
	ErrLiveStreamActiveExists = errors.New("user already has an active live stream")
)

type liveStreamRepository struct {
	dao LiveStreamRepository
}

func NewLiveStreamRepo(dao LiveStreamRepository) LiveStreamRepository {
	return &liveStreamRepository{dao: dao}
}

func (r *liveStreamRepository) Create(ctx context.Context, userID uuid.UUID, title string, maxConcurrent int) (uuid.UUID, error) {
	return r.dao.Create(ctx, userID, title, maxConcurrent)
}

func (r *liveStreamRepository) GetByID(ctx context.Context, id uuid.UUID) (*LiveStreamRow, error) {
	return r.dao.GetByID(ctx, id)
}

func (r *liveStreamRepository) GetByRoom(ctx context.Context, room string) (*LiveStreamRow, error) {
	return r.dao.GetByRoom(ctx, room)
}

func (r *liveStreamRepository) GetActiveByUser(ctx context.Context, userID uuid.UUID) (*LiveStreamRow, error) {
	return r.dao.GetActiveByUser(ctx, userID)
}

func (r *liveStreamRepository) ListLive(ctx context.Context) ([]LiveStreamRow, error) {
	return r.dao.ListLive(ctx)
}

func (r *liveStreamRepository) ListStartingBefore(ctx context.Context, cutoff string) ([]LiveStreamRow, error) {
	return r.dao.ListStartingBefore(ctx, cutoff)
}

func (r *liveStreamRepository) CountActive(ctx context.Context) (int, error) {
	return r.dao.CountActive(ctx)
}

func (r *liveStreamRepository) SetIngress(ctx context.Context, id uuid.UUID, ingressID, room, whipURL, streamKey string) error {
	return r.dao.SetIngress(ctx, id, ingressID, room, whipURL, streamKey)
}

func (r *liveStreamRepository) MarkLive(ctx context.Context, id uuid.UUID) error {
	return r.dao.MarkLive(ctx, id)
}

func (r *liveStreamRepository) MarkOffline(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.dao.MarkOffline(ctx, id)
}

func (r *liveStreamRepository) AdjustViewerCount(ctx context.Context, id uuid.UUID, delta int) (int, bool, error) {
	return r.dao.AdjustViewerCount(ctx, id, delta)
}

func (r *liveStreamRepository) SetThumbnail(ctx context.Context, id uuid.UUID, url string) error {
	return r.dao.SetThumbnail(ctx, id, url)
}

func (r *liveStreamRepository) SetEgress(ctx context.Context, id uuid.UUID, egressID, hlsURL string) error {
	return r.dao.SetEgress(ctx, id, egressID, hlsURL)
}

func (r *liveStreamRepository) SetDefaultMode(ctx context.Context, id uuid.UUID, mode string) error {
	return r.dao.SetDefaultMode(ctx, id, mode)
}

func (r *liveStreamRepository) SetTitle(ctx context.Context, id uuid.UUID, title string) error {
	return r.dao.SetTitle(ctx, id, title)
}
