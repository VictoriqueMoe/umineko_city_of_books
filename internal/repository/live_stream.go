package repository

import (
	"context"
	"database/sql"
	"errors"

	"umineko_city_of_books/internal/db"

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

	LiveStreamDAO interface {
		Create(ctx context.Context, userID uuid.UUID, title string, maxConcurrent int, tx ...*sql.Tx) (*LiveStreamRow, error)
		GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*LiveStreamRow, error)
		GetByRoom(ctx context.Context, room string, tx ...*sql.Tx) (*LiveStreamRow, error)
		GetActiveByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (*LiveStreamRow, error)
		ListLive(ctx context.Context, tx ...*sql.Tx) ([]LiveStreamRow, error)
		ListStartingBefore(ctx context.Context, cutoff string, tx ...*sql.Tx) ([]LiveStreamRow, error)
		CountActive(ctx context.Context, tx ...*sql.Tx) (int, error)
		SetIngress(ctx context.Context, spec LiveStreamIngressUpdate, tx ...*sql.Tx) error
		MarkLive(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		MarkOffline(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (bool, error)
		AdjustViewerCount(ctx context.Context, id uuid.UUID, delta int, tx ...*sql.Tx) (int, bool, error)
		SetThumbnail(ctx context.Context, id uuid.UUID, url string, tx ...*sql.Tx) error
		SetEgress(ctx context.Context, id uuid.UUID, egressID, hlsURL string, tx ...*sql.Tx) error
		SetDefaultMode(ctx context.Context, id uuid.UUID, mode string, tx ...*sql.Tx) error
		SetTitle(ctx context.Context, id uuid.UUID, title string, tx ...*sql.Tx) error
	}

	LiveStreamRepository interface {
		LiveStreamDAO

		Activate(ctx context.Context, spec LiveStreamActivation, tx ...*sql.Tx) error
	}

	LiveStreamIngressUpdate struct {
		ID        uuid.UUID
		IngressID string
		Room      string
		WhipURL   string
		StreamKey string
	}

	LiveStreamActivation struct {
		Ingress     LiveStreamIngressUpdate
		DefaultMode string
	}
)

var (
	ErrLiveStreamCapacity     = errors.New("live stream capacity reached")
	ErrLiveStreamActiveExists = errors.New("user already has an active live stream")
)

type liveStreamRepository struct {
	db  *sql.DB
	dao LiveStreamDAO
}

func NewLiveStreamRepo(database *sql.DB, dao LiveStreamDAO) LiveStreamRepository {
	return &liveStreamRepository{db: database, dao: dao}
}

func (r *liveStreamRepository) Activate(ctx context.Context, spec LiveStreamActivation, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.SetIngress(ctx, spec.Ingress, tx); err != nil {
			return err
		}

		return r.dao.SetDefaultMode(ctx, spec.Ingress.ID, spec.DefaultMode, tx)
	})
}

func (r *liveStreamRepository) Create(ctx context.Context, userID uuid.UUID, title string, maxConcurrent int, tx ...*sql.Tx) (*LiveStreamRow, error) {
	return r.dao.Create(ctx, userID, title, maxConcurrent, tx...)
}

func (r *liveStreamRepository) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*LiveStreamRow, error) {
	return r.dao.GetByID(ctx, id, tx...)
}

func (r *liveStreamRepository) GetByRoom(ctx context.Context, room string, tx ...*sql.Tx) (*LiveStreamRow, error) {
	return r.dao.GetByRoom(ctx, room, tx...)
}

func (r *liveStreamRepository) GetActiveByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (*LiveStreamRow, error) {
	return r.dao.GetActiveByUser(ctx, userID, tx...)
}

func (r *liveStreamRepository) ListLive(ctx context.Context, tx ...*sql.Tx) ([]LiveStreamRow, error) {
	return r.dao.ListLive(ctx, tx...)
}

func (r *liveStreamRepository) ListStartingBefore(ctx context.Context, cutoff string, tx ...*sql.Tx) ([]LiveStreamRow, error) {
	return r.dao.ListStartingBefore(ctx, cutoff, tx...)
}

func (r *liveStreamRepository) CountActive(ctx context.Context, tx ...*sql.Tx) (int, error) {
	return r.dao.CountActive(ctx, tx...)
}

func (r *liveStreamRepository) SetIngress(ctx context.Context, spec LiveStreamIngressUpdate, tx ...*sql.Tx) error {
	return r.dao.SetIngress(ctx, spec, tx...)
}

func (r *liveStreamRepository) MarkLive(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.MarkLive(ctx, id, tx...)
}

func (r *liveStreamRepository) MarkOffline(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.MarkOffline(ctx, id, tx...)
}

func (r *liveStreamRepository) AdjustViewerCount(ctx context.Context, id uuid.UUID, delta int, tx ...*sql.Tx) (int, bool, error) {
	return r.dao.AdjustViewerCount(ctx, id, delta, tx...)
}

func (r *liveStreamRepository) SetThumbnail(ctx context.Context, id uuid.UUID, url string, tx ...*sql.Tx) error {
	return r.dao.SetThumbnail(ctx, id, url, tx...)
}

func (r *liveStreamRepository) SetEgress(ctx context.Context, id uuid.UUID, egressID, hlsURL string, tx ...*sql.Tx) error {
	return r.dao.SetEgress(ctx, id, egressID, hlsURL, tx...)
}

func (r *liveStreamRepository) SetDefaultMode(ctx context.Context, id uuid.UUID, mode string, tx ...*sql.Tx) error {
	return r.dao.SetDefaultMode(ctx, id, mode, tx...)
}

func (r *liveStreamRepository) SetTitle(ctx context.Context, id uuid.UUID, title string, tx ...*sql.Tx) error {
	return r.dao.SetTitle(ctx, id, title, tx...)
}
