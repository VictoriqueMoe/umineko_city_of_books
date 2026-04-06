package dto

import "github.com/google/uuid"

type (
	TheoryResponse struct {
		ID               uuid.UUID    `json:"id"`
		Title            string       `json:"title"`
		Body             string       `json:"body"`
		Episode          int          `json:"episode"`
		Series           string       `json:"series"`
		Author           UserResponse `json:"author"`
		VoteScore        int          `json:"vote_score"`
		WithLoveCount    int          `json:"with_love_count"`
		WithoutLoveCount int          `json:"without_love_count"`
		UserVote         int          `json:"user_vote,omitempty"`
		CredibilityScore float64      `json:"credibility_score"`
		CreatedAt        string       `json:"created_at"`
	}

	TheoryDetailResponse struct {
		ID               uuid.UUID          `json:"id"`
		Title            string             `json:"title"`
		Body             string             `json:"body"`
		Episode          int                `json:"episode"`
		Series           string             `json:"series"`
		Author           UserResponse       `json:"author"`
		Evidence         []EvidenceResponse `json:"evidence"`
		Responses        []ResponseResponse `json:"responses"`
		VoteScore        int                `json:"vote_score"`
		WithLoveCount    int                `json:"with_love_count"`
		WithoutLoveCount int                `json:"without_love_count"`
		UserVote         int                `json:"user_vote,omitempty"`
		CredibilityScore float64            `json:"credibility_score"`
		CreatedAt        string             `json:"created_at"`
	}

	EvidenceResponse struct {
		ID         int    `json:"id"`
		AudioID    string `json:"audio_id,omitempty"`
		QuoteIndex *int   `json:"quote_index,omitempty"`
		Note       string `json:"note"`
		Lang       string `json:"lang"`
		SortOrder  int    `json:"sort_order"`
	}

	TheoryListResponse struct {
		Theories []TheoryResponse `json:"theories"`
		Total    int              `json:"total"`
		Limit    int              `json:"limit"`
		Offset   int              `json:"offset"`
	}

	EvidenceInput struct {
		AudioID    string `json:"audio_id,omitempty"`
		QuoteIndex *int   `json:"quote_index,omitempty"`
		Note       string `json:"note"`
		Lang       string `json:"lang,omitempty"`
	}

	CreateTheoryRequest struct {
		Title    string          `json:"title"`
		Body     string          `json:"body"`
		Episode  int             `json:"episode"`
		Series   string          `json:"series"`
		Evidence []EvidenceInput `json:"evidence"`
	}

	VoteRequest struct {
		Value int `json:"value"`
	}
)
