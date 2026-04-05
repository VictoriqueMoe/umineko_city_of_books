package dto

import "github.com/google/uuid"

type (
	AnnouncementCommentResponse struct {
		ID        uuid.UUID                     `json:"id"`
		ParentID  *uuid.UUID                    `json:"parent_id,omitempty"`
		Author    UserResponse                  `json:"author"`
		Body      string                        `json:"body"`
		Media     []PostMediaResponse           `json:"media"`
		LikeCount int                           `json:"like_count"`
		UserLiked bool                          `json:"user_liked"`
		Replies   []AnnouncementCommentResponse `json:"replies,omitempty"`
		CreatedAt string                        `json:"created_at"`
		UpdatedAt *string                       `json:"updated_at,omitempty"`
	}
)
