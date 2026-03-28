package dto

type (
	ResponseResponse struct {
		ID        int                `json:"id"`
		ParentID  *int               `json:"parent_id,omitempty"`
		Author    UserResponse       `json:"author"`
		Side      string             `json:"side"`
		Body      string             `json:"body"`
		Evidence  []EvidenceResponse `json:"evidence"`
		Replies   []ResponseResponse `json:"replies,omitempty"`
		VoteScore int                `json:"vote_score"`
		UserVote  int                `json:"user_vote,omitempty"`
		CreatedAt string             `json:"created_at"`
	}

	CreateResponseRequest struct {
		ParentID *int            `json:"parent_id,omitempty"`
		Side     string          `json:"side"`
		Body     string          `json:"body"`
		Evidence []EvidenceInput `json:"evidence"`
	}
)
