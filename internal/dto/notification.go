package dto

import "github.com/google/uuid"

type (
	NotificationType string

	NotifyParams struct {
		RecipientID   uuid.UUID
		Type          NotificationType
		ReferenceID   uuid.UUID
		ReferenceType string
		ActorID       uuid.UUID
		EmailSubject  string
		EmailBody     string
	}

	NotificationResponse struct {
		ID            int          `json:"id"`
		Type          string       `json:"type"`
		ReferenceID   uuid.UUID    `json:"reference_id"`
		ReferenceType string       `json:"reference_type"`
		Actor         UserResponse `json:"actor"`
		Read          bool         `json:"read"`
		CreatedAt     string       `json:"created_at"`
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
	NotifTheoryResponse NotificationType = "theory_response"
	NotifResponseReply  NotificationType = "response_reply"
	NotifTheoryUpvote   NotificationType = "theory_upvote"
	NotifResponseUpvote NotificationType = "response_upvote"
	NotifChatMessage    NotificationType = "chat_message"
	NotifReport         NotificationType = "report"
	NotifNewFollower    NotificationType = "new_follower"
	NotifPostLiked      NotificationType = "post_liked"
	NotifPostCommented  NotificationType = "post_commented"
	NotifMention        NotificationType = "mention"
	NotifArtLiked       NotificationType = "art_liked"
	NotifArtCommented   NotificationType = "art_commented"
)
