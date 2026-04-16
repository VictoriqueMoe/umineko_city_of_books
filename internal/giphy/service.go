package giphy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"umineko_city_of_books/internal/config"
)

type (
	Service interface {
		Enabled() bool
		Search(ctx context.Context, q string, offset, limit int) (*Response, error)
		Trending(ctx context.Context, offset, limit int) (*Response, error)
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
		apiKey             string
		baseURL            string
		httpClient         *http.Client
		cache              *cache
		rateLimitedUntilNs atomic.Int64
	}

	RateLimitError struct {
		ResetAt time.Time
	}
)

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("giphy rate limited until %s", e.ResetAt.UTC().Format(time.RFC3339))
}

const (
	defaultBaseURL       = "https://api.giphy.com/v1"
	defaultLimit         = 24
	maxLimit             = 50
	rating               = "pg-13"
	bundle               = "messaging_non_clips"
	requestTimeout       = 10 * time.Second
	searchTTL            = 1 * time.Hour
	trendingTTL          = 6 * time.Hour
	cacheMaxItems        = 500
	defaultRateLimitHold = 1 * time.Hour
)

var (
	ErrDisabled = errors.New("giphy integration is not configured")
)

func NewService() Service {
	return &service{
		apiKey:  config.Cfg.GiphyAPIKey,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		cache: newCache(cacheMaxItems),
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
	return s.get(ctx, "/gifs/search", params, searchTTL)
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

func (s *service) get(ctx context.Context, path string, params url.Values, ttl time.Duration) (*Response, error) {
	cacheKey := cacheKeyFor(path, params)

	if s.cache != nil && ttl > 0 {
		if cached, ok := s.cache.get(cacheKey); ok {
			return cached, nil
		}
	}

	if resetAt, blocked := s.rateLimitResetAt(); blocked {
		return nil, &RateLimitError{ResetAt: resetAt}
	}

	reqURL := s.baseURL + path + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build giphy request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call giphy: %w", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		resetAt := parseRateLimitReset(resp.Header, time.Now())
		s.rateLimitedUntilNs.Store(resetAt.UnixNano())
		return nil, &RateLimitError{ResetAt: resetAt}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("giphy %d: %s", resp.StatusCode, string(body))
	}

	var out Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode giphy response: %w", err)
	}

	if s.cache != nil && ttl > 0 {
		s.cache.set(cacheKey, &out, ttl)
	}

	return &out, nil
}

func (s *service) rateLimitResetAt() (time.Time, bool) {
	ns := s.rateLimitedUntilNs.Load()
	if ns == 0 {
		return time.Time{}, false
	}
	resetAt := time.Unix(0, ns)
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
