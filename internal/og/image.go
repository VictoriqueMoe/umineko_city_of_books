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
	load := func(ctx context.Context) ([]byte, error) {
		return media.WebPToJPEG(ctx, fullPath, maxPixels)
	}

	return s.cache.Load(ctx, cache.OGImage, load, rel, fingerprint(info))
}

func fingerprint(info os.FileInfo) string {
	return strconv.FormatInt(info.ModTime().UnixNano(), 10) + "-" + strconv.FormatInt(info.Size(), 10)
}
