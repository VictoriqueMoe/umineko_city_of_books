package dto

import "github.com/google/uuid"

type (
	PostMediaResponse struct {
		ID           int    `json:"id"`
		MediaURL     string `json:"media_url"`
		MediaType    string `json:"media_type"`
		ThumbnailURL string `json:"thumbnail_url,omitempty"`
		SortOrder    int    `json:"sort_order"`
	}

	PostResponse struct {
		ID           uuid.UUID           `json:"id"`
		Author       UserResponse        `json:"author"`
		Body         string              `json:"body"`
		Media        []PostMediaResponse `json:"media"`
		LikeCount    int                 `json:"like_count"`
		CommentCount int                 `json:"comment_count"`
		ViewCount    int                 `json:"view_count"`
		UserLiked    bool                `json:"user_liked"`
		CreatedAt    string              `json:"created_at"`
	}

	PostDetailResponse struct {
		PostResponse
		Comments []PostCommentResponse `json:"comments"`
		LikedBy  []UserResponse        `json:"liked_by"`
	}

	PostCommentResponse struct {
		ID        uuid.UUID           `json:"id"`
		Author    UserResponse        `json:"author"`
		Body      string              `json:"body"`
		Media     []PostMediaResponse `json:"media"`
		CreatedAt string              `json:"created_at"`
	}

	PostListResponse struct {
		Posts  []PostResponse `json:"posts"`
		Total  int            `json:"total"`
		Limit  int            `json:"limit"`
		Offset int            `json:"offset"`
	}

	CreatePostRequest struct {
		Body string `json:"body"`
	}

	CreateCommentRequest struct {
		Body string `json:"body"`
	}

	FollowStatsResponse struct {
		FollowerCount  int  `json:"follower_count"`
		FollowingCount int  `json:"following_count"`
		IsFollowing    bool `json:"is_following"`
	}
)
