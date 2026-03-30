package dto

import "github.com/google/uuid"

type (
	NotificationType = string

	NotificationResponse struct {
		ID          int          `json:"id"`
		Type        string       `json:"type"`
		ReferenceID uuid.UUID    `json:"reference_id"`
		TheoryID    uuid.UUID    `json:"theory_id"`
		TheoryTitle string       `json:"theory_title"`
		Actor       UserResponse `json:"actor"`
		Read        bool         `json:"read"`
		CreatedAt   string       `json:"created_at"`
	}

	NotificationListResponse struct {
		Notifications []NotificationResponse `json:"notifications"`
		Total         int                    `json:"total"`
		Limit         int                    `json:"limit"`
		Offset        int                    `json:"offset"`
	}

	UnreadCountResponse struct {
		Count int `json:"count"`
	}
)

const (
	NotifTheoryResponse = "theory_response"
	NotifResponseReply  = "response_reply"
	NotifTheoryUpvote   = "theory_upvote"
	NotifResponseUpvote = "response_upvote"
)
