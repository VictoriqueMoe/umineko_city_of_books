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
