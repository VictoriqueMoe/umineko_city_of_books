package quotefinder

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/httpclient"
)

const (
	DefaultBaseURL = "https://quotes.auaurora.moe/api/v1"
)

type (
	Quote struct {
		HasRedTruth    bool `json:"hasRedTruth"`
		HasBlueTruth   bool `json:"hasBlueTruth"`
		HasGoldTruth   bool `json:"hasGoldTruth"`
		HasPurpleTruth bool `json:"hasPurpleTruth"`
	}

	Character struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Group string `json:"group"`
	}

	charactersResponse struct {
		Characters map[string]string `json:"characters"`
		Additional map[string]string `json:"additional"`
	}

	Client struct {
		http    *httpclient.Client
		baseURL string
		cache   *cache.Manager
	}
)

func NewClient(cacheManager *cache.Manager) *Client {
	return NewClientWithBaseURL(DefaultBaseURL, cacheManager)
}

func NewClientWithBaseURL(baseURL string, cacheManager *cache.Manager) *Client {
	return &Client{
		http:    httpclient.New(10 * time.Second),
		baseURL: baseURL,
		cache:   cacheManager,
	}
}

func (c *Client) ListCharacters(ctx context.Context, series Series) ([]Character, error) {
	if !series.Valid() {
		return nil, fmt.Errorf("unsupported series: %s", series)
	}

	return c.cache.Load(ctx, cache.QuoteCharacters, func(ctx context.Context) ([]Character, error) {
		return c.fetchCharacters(ctx, series)
	}, string(series))
}

func (c *Client) fetchCharacters(ctx context.Context, series Series) ([]Character, error) {
	var wrapper charactersResponse
	req := httpclient.Request{Method: http.MethodGet, URL: fmt.Sprintf("%s/%s/characters", c.baseURL, series)}
	if err := c.http.JSON(ctx, req, &wrapper); err != nil {
		return nil, fmt.Errorf("fetch characters: %w", err)
	}

	result := make([]Character, 0, len(wrapper.Characters)+len(wrapper.Additional))
	for id, name := range wrapper.Characters {
		result = append(result, Character{ID: id, Name: name, Group: "main"})
	}
	for id, name := range wrapper.Additional {
		result = append(result, Character{ID: id, Name: name, Group: "additional"})
	}

	return result, nil
}

func (c *Client) GetByAudioID(ctx context.Context, series Series, audioID string) (*Quote, error) {
	if !series.Valid() {
		series = SeriesUmineko
	}

	firstID, _, _ := strings.Cut(audioID, ",")
	firstID = strings.TrimSpace(firstID)
	if firstID == "" {
		return nil, nil
	}

	return c.cache.Load(ctx, cache.QuoteByAudioID, func(ctx context.Context) (*Quote, error) {
		return c.fetchQuote(ctx, fmt.Sprintf("%s/%s/quote/%s", c.baseURL, series, firstID))
	}, string(series), firstID)
}

func (c *Client) GetByIndex(ctx context.Context, series Series, index int) (*Quote, error) {
	if !series.Valid() {
		series = SeriesUmineko
	}

	return c.cache.Load(ctx, cache.QuoteByIndex, func(ctx context.Context) (*Quote, error) {
		return c.fetchQuote(ctx, fmt.Sprintf("%s/%s/quote/index/%d", c.baseURL, series, index))
	}, string(series), strconv.Itoa(index))
}

func (c *Client) fetchQuote(ctx context.Context, quoteURL string) (*Quote, error) {
	var q Quote
	err := c.http.JSON(ctx, httpclient.Request{Method: http.MethodGet, URL: quoteURL}, &q)
	if statusErr, ok := errors.AsType[*httpclient.StatusError](err); ok && statusErr.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fetch quote: %w", err)
	}

	return &q, nil
}

func TruthWeight(q *Quote) float64 {
	switch {
	case q == nil:
		return 1.0
	case q.HasGoldTruth:
		return 3.3
	case q.HasRedTruth:
		return 3.0
	case q.HasPurpleTruth:
		return 2.2
	case q.HasBlueTruth:
		return 2.0
	default:
		return 1.0
	}
}
