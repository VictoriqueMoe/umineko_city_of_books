package giphy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/giphy/banlist"
	"umineko_city_of_books/internal/httpclient"
	"umineko_city_of_books/internal/logger"
)

type (
	Service interface {
		Enabled() bool
		Search(ctx context.Context, q string, offset, limit int) (*Response, error)
		Trending(ctx context.Context, offset, limit int) (*Response, error)
		UserForGif(ctx context.Context, gifID string) (string, bool)
	}

	Image struct {
		URL    string `json:"url"`
		Width  string `json:"width"`
		Height string `json:"height"`
	}

	Gif struct {
		ID     string           `json:"id"`
		Title  string           `json:"title"`
		URL    string           `json:"url"`
		Images map[string]Image `json:"images"`
		User   *GifUser         `json:"user,omitempty"`
	}

	Pagination struct {
		TotalCount int `json:"total_count"`
		Count      int `json:"count"`
		Offset     int `json:"offset"`
	}

	Response struct {
		Data       []Gif      `json:"data"`
		Pagination Pagination `json:"pagination"`
	}

	service struct {
		apiKey     string
		baseURL    string
		httpClient *httpclient.Client
		cache      *cache.Manager
		banlist    banlist.Service
	}

	gifUserEntry struct {
		Username string `json:"username"`
		Known    bool   `json:"known"`
	}

	GifUser struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	}
)

const (
	defaultBaseURL       = "https://api.giphy.com/v1"
	defaultLimit         = 24
	maxLimit             = 50
	rating               = "pg-13"
	bundle               = "messaging_non_clips"
	requestTimeout       = 10 * time.Second
	defaultRateLimitHold = 1 * time.Hour
	trendingTTL          = 6 * time.Hour
)

func NewService(banlistSvc banlist.Service, cacheMgr *cache.Manager) Service {
	return &service{
		apiKey:     config.Cfg.GiphyAPIKey,
		baseURL:    defaultBaseURL,
		httpClient: httpclient.New(requestTimeout),
		cache:      cacheMgr,
		banlist:    banlistSvc,
	}
}

func (s *service) Enabled() bool {
	return s.apiKey != ""
}

func (s *service) Search(ctx context.Context, q string, offset, limit int) (*Response, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	params := baseParams(s.apiKey, limit, offset)
	params.Set("q", q)
	return s.get(ctx, "/gifs/search", params, cache.GiphyResponse.TTL)
}

func (s *service) Trending(ctx context.Context, offset, limit int) (*Response, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	params := baseParams(s.apiKey, limit, offset)
	return s.get(ctx, "/gifs/trending", params, trendingTTL)
}

func baseParams(apiKey string, limit, offset int) url.Values {
	if limit <= 0 || limit > maxLimit {
		limit = defaultLimit
	}
	if offset < 0 {
		offset = 0
	}
	p := url.Values{}
	p.Set("api_key", apiKey)
	p.Set("limit", fmt.Sprintf("%d", limit))
	p.Set("offset", fmt.Sprintf("%d", offset))
	p.Set("rating", rating)
	p.Set("bundle", bundle)
	return p
}

func (s *service) fetch(ctx context.Context, path string, params url.Values, out any) error {
	if resetAt, blocked := s.rateLimitResetAt(ctx); blocked {
		return &RateLimitError{ResetAt: resetAt}
	}

	err := s.httpClient.JSON(ctx, httpclient.Request{
		Method: http.MethodGet,
		URL:    s.baseURL + path,
		Query:  params,
	}, out)
	if err == nil {
		return nil
	}

	statusErr, ok := errors.AsType[*httpclient.StatusError](err)
	if !ok {
		return fmt.Errorf("call giphy: %w", err)
	}

	if statusErr.StatusCode == http.StatusTooManyRequests {
		resetAt := s.tripRateLimit(ctx, parseRateLimitReset(statusErr.Header, time.Now()))

		return &RateLimitError{ResetAt: resetAt}
	}

	return fmt.Errorf("giphy: %w", statusErr)
}

func (s *service) get(ctx context.Context, path string, params url.Values, ttl time.Duration) (*Response, error) {
	key := cache.GiphyResponse.Key(cacheKeyFor(path, params)).WithTTL(ttl)

	if cached, err := s.cache.Get[Response](ctx, key); err == nil {
		return &cached, nil
	}

	var out Response
	if err := s.fetch(ctx, path, params, &out); err != nil {
		return nil, err
	}

	s.filterBannedGifs(&out)

	_ = s.cache.Set(ctx, key, out)

	return &out, nil
}

func (s *service) UserForGif(ctx context.Context, gifID string) (string, bool) {
	if gifID == "" {
		return "", false
	}
	key := cache.GiphyGifUser.Key(gifID)
	if cached, err := s.cache.Get[gifUserEntry](ctx, key); err == nil {
		return cached.Username, cached.Known
	}

	if !s.Enabled() {
		return "", false
	}

	params := url.Values{}
	params.Set("api_key", s.apiKey)

	var payload struct {
		Data Gif `json:"data"`
	}

	if err := s.fetch(ctx, "/gifs/"+gifID, params, &payload); err != nil {
		if statusErr, ok := errors.AsType[*httpclient.StatusError](err); ok && statusErr.StatusCode == http.StatusNotFound {
			_ = s.cache.Set(ctx, key, gifUserEntry{})
		}

		return "", false
	}

	username := ""
	if payload.Data.User != nil {
		username = payload.Data.User.Username
	}

	_ = s.cache.Set(ctx, key, gifUserEntry{Username: username, Known: username != ""})

	return username, username != ""
}

func (s *service) filterBannedGifs(resp *Response) {
	if s.banlist == nil || resp == nil || len(resp.Data) == 0 {
		return
	}
	kept := resp.Data[:0]
	for _, g := range resp.Data {
		if s.banlist.ContainsGif(g.ID) {
			continue
		}
		if g.User != nil && s.banlist.ContainsUser(g.User.Username) {
			continue
		}
		kept = append(kept, g)
	}
	resp.Data = kept
	resp.Pagination.Count = len(kept)
}

func (s *service) tripRateLimit(ctx context.Context, resetAt time.Time) time.Time {
	if !resetAt.After(time.Now()) {
		return resetAt
	}

	err := s.cache.Set(ctx, cache.GiphyRateLimitedUntil.Key(), resetAt)
	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Time("reset_at", resetAt).Msg("giphy could not record its upstream rate limit, later calls will reach giphy until it is stored")
	}

	return resetAt
}

func (s *service) rateLimitResetAt(ctx context.Context) (time.Time, bool) {
	resetAt, err := s.cache.Get[time.Time](ctx, cache.GiphyRateLimitedUntil.Key())
	if errors.Is(err, cache.ErrMiss) {
		return time.Time{}, false
	}

	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Msg("giphy could not read its upstream rate limit, letting this call through")

		return time.Time{}, false
	}

	if time.Now().After(resetAt) {
		return time.Time{}, false
	}

	return resetAt, true
}

func parseRateLimitReset(h http.Header, now time.Time) time.Time {
	if v := h.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return now.Add(time.Duration(secs) * time.Second)
		}
		if t, err := http.ParseTime(v); err == nil {
			return t
		}
	}
	if v := h.Get("X-RateLimit-Reset"); v != "" {
		if epoch, err := strconv.ParseInt(v, 10, 64); err == nil {
			return time.Unix(epoch, 0)
		}
	}
	return now.Add(defaultRateLimitHold)
}

func cacheKeyFor(path string, params url.Values) string {
	stripped := url.Values{}
	for k, v := range params {
		if k == "api_key" {
			continue
		}
		stripped[k] = v
	}
	return path + "?" + stripped.Encode()
}
