package quotefinder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://quotes.auaurora.moe/api/v1"

type Quote struct {
	HasRedTruth    bool `json:"hasRedTruth"`
	HasBlueTruth   bool `json:"hasBlueTruth"`
	HasGoldTruth   bool `json:"hasGoldTruth"`
	HasPurpleTruth bool `json:"hasPurpleTruth"`
}

type Client struct {
	http *http.Client
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) GetByAudioID(audioID string) (*Quote, error) {
	firstID := strings.Split(audioID, ",")[0]
	firstID = strings.TrimSpace(firstID)
	if firstID == "" {
		return nil, nil
	}

	resp, err := c.http.Get(fmt.Sprintf("%s/quote/%s", baseURL, firstID))
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

func (c *Client) GetByIndex(index int) (*Quote, error) {
	resp, err := c.http.Get(fmt.Sprintf("%s/quote/index/%d", baseURL, index))
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
