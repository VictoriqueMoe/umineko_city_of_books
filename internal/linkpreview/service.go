package linkpreview

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
	"time"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"

	"golang.org/x/sync/singleflight"
)

const (
	missTTL      = time.Hour
	maxURLLength = 2048
)

var (
	ErrInvalidURL = errors.New("linkpreview: url must be an absolute http or https url")
)

type (
	Service interface {
		Resolve(ctx context.Context, rawURL string) (dto.LinkPreviewResponse, error)
	}

	service struct {
		cache *cache.Manager
		parse func(rawURL string) *media.Embed
		group singleflight.Group
	}
)

func NewService(cacheMgr *cache.Manager) Service {
	return &service{cache: cacheMgr, parse: media.ParseEmbed}
}

func (s *service) Resolve(ctx context.Context, rawURL string) (dto.LinkPreviewResponse, error) {
	rawURL = strings.TrimSpace(rawURL)
	if err := validate(rawURL); err != nil {
		return dto.LinkPreviewResponse{}, err
	}

	key := cache.LinkPreview.Key(hashURL(rawURL))

	if cached, err := cache.Get[dto.LinkPreviewResponse](ctx, s.cache, key); err == nil {
		return cached, nil
	}

	storeCtx := context.WithoutCancel(ctx)

	resolved, _, _ := s.group.Do(key, func() (any, error) {
		preview := toResponse(rawURL, s.parse(rawURL))

		ttl := cache.LinkPreview.TTL
		if preview.Type == "" {
			ttl = missTTL
		}

		_ = cache.Set(storeCtx, s.cache, key, preview, ttl)

		return preview, nil
	})

	preview, ok := resolved.(dto.LinkPreviewResponse)
	if !ok {
		return dto.LinkPreviewResponse{URL: rawURL}, nil
	}

	return preview, nil
}

func validate(rawURL string) error {
	if rawURL == "" || len(rawURL) > maxURLLength {
		return ErrInvalidURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return ErrInvalidURL
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidURL
	}

	return nil
}

func toResponse(rawURL string, embed *media.Embed) dto.LinkPreviewResponse {
	if embed == nil {
		return dto.LinkPreviewResponse{URL: rawURL}
	}

	return dto.LinkPreviewResponse{
		URL:         rawURL,
		Type:        embed.Type,
		Title:       embed.Title,
		Description: embed.Desc,
		Image:       embed.Image,
		SiteName:    embed.SiteName,
		VideoID:     embed.VideoID,
	}
}

func hashURL(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))

	return hex.EncodeToString(sum[:])
}
