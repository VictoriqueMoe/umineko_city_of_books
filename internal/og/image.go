package og

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/storage"
	"umineko_city_of_books/internal/storage/engine"
)

type (
	ImageService struct {
		cache   *cache.Manager
		storage storage.Service
	}
)

func NewImageService(cacheMgr *cache.Manager, storageSvc storage.Service) *ImageService {
	return &ImageService{cache: cacheMgr, storage: storageSvc}
}

func (s *ImageService) JPEG(ctx context.Context, info engine.ObjectInfo, maxPixels int) ([]byte, error) {
	load := func(ctx context.Context) ([]byte, error) {
		tmp, err := os.CreateTemp("", "ogsource-*.webp")
		if err != nil {
			return nil, fmt.Errorf("create temp file: %w", err)
		}

		localPath := tmp.Name()
		_ = tmp.Close()
		defer os.Remove(localPath)

		if err := s.storage.Fetch(ctx, info, localPath); err != nil {
			return nil, err
		}

		return media.WebPToJPEG(ctx, localPath, maxPixels)
	}

	return s.cache.Load(ctx, cache.OGImage, load, info.Key, fingerprint(info))
}

func fingerprint(info engine.ObjectInfo) string {
	return strconv.FormatInt(info.ModTime.UnixNano(), 10) + "-" + strconv.FormatInt(info.Size, 10)
}
