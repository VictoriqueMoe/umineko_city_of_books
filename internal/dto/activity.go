package dto

import "github.com/google/uuid"

type (
	ActivityItem struct {
		Type        string    `json:"type"`
		TheoryID    uuid.UUID `json:"theory_id"`
		TheoryTitle string    `json:"theory_title"`
		Side        string    `json:"side,omitempty"`
		Body        string    `json:"body"`
		CreatedAt   string    `json:"created_at"`
	}

	ActivityListResponse struct {
		Items  []ActivityItem `json:"items"`
		Total  int            `json:"total"`
		Limit  int            `json:"limit"`
		Offset int            `json:"offset"`
	}
)
