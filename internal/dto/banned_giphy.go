package dto

import "time"

type (
	BannedGiphyEntry struct {
		Kind      string    `json:"kind"`
		Value     string    `json:"value"`
		Reason    string    `json:"reason"`
		CreatedAt time.Time `json:"created_at"`
		CreatedBy *string   `json:"created_by,omitempty"`
	}

	BannedGiphyListResponse struct {
		Entries []BannedGiphyEntry `json:"entries"`
	}

	AddBannedGiphyRequest struct {
		// Input is a Giphy URL or raw identifier. The server will infer
		// whether it's a gif or a user/channel. If Kind is also supplied it
		// must agree with what the server detects.
		Input  string `json:"input"`
		Kind   string `json:"kind,omitempty"`
		Reason string `json:"reason,omitempty"`
	}

	AddBannedGiphyResponse struct {
		Entry BannedGiphyEntry `json:"entry"`
	}
)
