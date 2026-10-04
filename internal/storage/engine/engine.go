package engine

import (
	"context"
	"io"
	"iter"
	"time"

	"umineko_city_of_books/internal/config"
)

type (
	Engine interface {
		ID() config.StorageBackend
		Enabled() bool
		Get(ctx context.Context, location string, rng *ByteRange) (io.ReadCloser, error)
		PutFile(ctx context.Context, key string, localPath string) (string, error)
		Head(ctx context.Context, location string) (ObjectInfo, error)
		Delete(ctx context.Context, locations ...string) error
		List(ctx context.Context, prefix string) iter.Seq2[ObjectInfo, error]
	}

	ObjectInfo struct {
		Key      string
		Location string
		Size     int64
		ModTime  time.Time
		Backend  config.StorageBackend
	}

	ByteRange struct {
		Start int64
		End   int64
	}
)

func (r ByteRange) Length() int64 {
	return r.End - r.Start + 1
}
