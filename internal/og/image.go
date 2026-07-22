package og

import (
	"context"
	"os"
	"strconv"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/media"
)

type (
	ImageService struct {
		cache *cache.Manager
	}
)

func NewImageService(cacheMgr *cache.Manager) *ImageService {
	return &ImageService{cache: cacheMgr}
}

func (s *ImageService) JPEG(ctx context.Context, rel, fullPath string, info os.FileInfo, maxPixels int) ([]byte, error) {
	key := cache.OGImage.Key(rel, fingerprint(info))

	if data, err := cache.Get[[]byte](ctx, s.cache, key); err == nil {
		return data, nil
	}

	data, err := media.WebPToJPEG(ctx, fullPath, maxPixels)
	if err != nil {
		return nil, err
	}

	_ = cache.Set(ctx, s.cache, key, data, cache.OGImage.TTL)

	return data, nil
}

func fingerprint(info os.FileInfo) string {
	return strconv.FormatInt(info.ModTime().UnixNano(), 10) + "-" + strconv.FormatInt(info.Size(), 10)
}
