package dto

import "github.com/google/uuid"

type (
	AnnouncementResponse struct {
		ID        uuid.UUID    `json:"id"`
		Title     string       `json:"title"`
		Body      string       `json:"body"`
		Pinned    bool         `json:"pinned"`
		CreatedAt string       `json:"created_at"`
		UpdatedAt string       `json:"updated_at"`
		Author    UserResponse `json:"author"`
	}

	AnnouncementDetailResponse struct {
		AnnouncementResponse
		Comments []AnnouncementCommentResponse `json:"comments"`
	}

	AnnouncementListResponse struct {
		Announcements []AnnouncementResponse `json:"announcements"`
		Total         int                    `json:"total"`
		Limit         int                    `json:"limit"`
		Offset        int                    `json:"offset"`
	}

	AnnouncementLatestResponse struct {
		Announcement *AnnouncementResponse `json:"announcement"`
	}

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
