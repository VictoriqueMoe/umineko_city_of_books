package dto

import "github.com/google/uuid"

type (
	JournalResponse struct {
		ID                   uuid.UUID    `json:"id"`
		Title                string       `json:"title"`
		Body                 string       `json:"body"`
		Work                 string       `json:"work"`
		Author               UserResponse `json:"author"`
		FollowerCount        int          `json:"follower_count"`
		IsFollowing          bool         `json:"is_following"`
		IsArchived           bool         `json:"is_archived"`
		CommentCount         int          `json:"comment_count"`
		CreatedAt            string       `json:"created_at"`
		UpdatedAt            *string      `json:"updated_at,omitempty"`
		LastAuthorActivityAt string       `json:"last_author_activity_at"`
		ArchivedAt           *string      `json:"archived_at,omitempty"`
	}

	JournalDetailResponse struct {
		JournalResponse
		Comments []JournalCommentResponse `json:"comments"`
	}

	JournalListResponse struct {
		Journals []JournalResponse `json:"journals"`
		Total    int               `json:"total"`
		Limit    int               `json:"limit"`
		Offset   int               `json:"offset"`
	}

	CreateJournalRequest struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Work  string `json:"work"`
	}

	JournalCommentResponse struct {
		ID        uuid.UUID                `json:"id"`
		ParentID  *uuid.UUID               `json:"parent_id,omitempty"`
		Author    UserResponse             `json:"author"`
		Body      string                   `json:"body"`
		Media     []PostMediaResponse      `json:"media"`
		LikeCount int                      `json:"like_count"`
		UserLiked bool                     `json:"user_liked"`
		IsAuthor  bool                     `json:"is_author"`
		Replies   []JournalCommentResponse `json:"replies,omitempty"`
		CreatedAt string                   `json:"created_at"`
		UpdatedAt *string                  `json:"updated_at,omitempty"`
	}
)
