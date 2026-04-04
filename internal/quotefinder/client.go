package quotefinder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const baseURL = "https://quotes.auaurora.moe/api/v1"

type (
	Quote struct {
		HasRedTruth    bool `json:"hasRedTruth"`
		HasBlueTruth   bool `json:"hasBlueTruth"`
		HasGoldTruth   bool `json:"hasGoldTruth"`
		HasPurpleTruth bool `json:"hasPurpleTruth"`
	}

	Character struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	charactersResponse struct {
		Characters map[string]string `json:"characters"`
	}

	cachedCharacters struct {
		data      []Character
		expiresAt time.Time
	}

	Client struct {
		http     *http.Client
		charMu   sync.Mutex
		charMemo map[Series]cachedCharacters
	}
)

const characterCacheTTL = 1 * time.Hour

func NewClient() *Client {
	return &Client{
		http:     &http.Client{Timeout: 10 * time.Second},
		charMemo: make(map[Series]cachedCharacters),
	}
}

func (c *Client) ListCharacters(series Series) ([]Character, error) {
	if !series.Valid() {
		return nil, fmt.Errorf("unsupported series: %s", series)
	}

	c.charMu.Lock()
	if entry, ok := c.charMemo[series]; ok && time.Now().Before(entry.expiresAt) {
		c.charMu.Unlock()
		return entry.data, nil
	}
	c.charMu.Unlock()

	resp, err := c.http.Get(fmt.Sprintf("%s/%s/characters", baseURL, series))
	if err != nil {
		return nil, fmt.Errorf("fetch characters: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch characters: status %d", resp.StatusCode)
	}

	var wrapper charactersResponse
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("decode characters: %w", err)
	}

	result := make([]Character, 0, len(wrapper.Characters))
	for id, name := range wrapper.Characters {
		result = append(result, Character{ID: id, Name: name})
	}

	c.charMu.Lock()
	c.charMemo[series] = cachedCharacters{
		data:      result,
		expiresAt: time.Now().Add(characterCacheTTL),
	}
	c.charMu.Unlock()

	return result, nil
}

func (c *Client) GetByAudioID(series Series, audioID string) (*Quote, error) {
	if !series.Valid() {
		series = SeriesUmineko
	}
	firstID := strings.Split(audioID, ",")[0]
	firstID = strings.TrimSpace(firstID)
	if firstID == "" {
		return nil, nil
	}

	resp, err := c.http.Get(fmt.Sprintf("%s/%s/quote/%s", baseURL, series, firstID))
	if err != nil {
		return nil, fmt.Errorf("fetch quote: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var q Quote
	if err := json.NewDecoder(resp.Body).Decode(&q); err != nil {
		return nil, fmt.Errorf("decode quote: %w", err)
	}
	return &q, nil
}

func (c *Client) GetByIndex(series Series, index int) (*Quote, error) {
	if !series.Valid() {
		series = SeriesUmineko
	}
	resp, err := c.http.Get(fmt.Sprintf("%s/%s/quote/index/%d", baseURL, series, index))
	if err != nil {
		return nil, fmt.Errorf("fetch quote by index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var q Quote
	if err := json.NewDecoder(resp.Body).Decode(&q); err != nil {
		return nil, fmt.Errorf("decode quote: %w", err)
	}
	return &q, nil
}

func TruthWeight(q *Quote) float64 {
	if q == nil {
		return 1.0
	}

	if q.HasGoldTruth {
		return 3.3
	}
	if q.HasRedTruth {
		return 3.0
	}
	if q.HasPurpleTruth {
		return 2.2
	}
	if q.HasBlueTruth {
		return 2.0
	}
	return 1.0
}
