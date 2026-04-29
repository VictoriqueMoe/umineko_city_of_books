package dto

import "github.com/google/uuid"

const CrackOCThreshold = -3

type (
	OCImage struct {
		ID           int64  `json:"id"`
		ImageURL     string `json:"image_url"`
		ThumbnailURL string `json:"thumbnail_url,omitempty"`
		Caption      string `json:"caption,omitempty"`
		SortOrder    int    `json:"sort_order"`
	}

	OCResponse struct {
		ID               uuid.UUID    `json:"id"`
		Author           UserResponse `json:"author"`
		Name             string       `json:"name"`
		Description      string       `json:"description"`
		Series           string       `json:"series"`
		CustomSeriesName string       `json:"custom_series_name,omitempty"`
		ImageURL         string       `json:"image_url,omitempty"`
		ThumbnailURL     string       `json:"thumbnail_url,omitempty"`
		Gallery          []OCImage    `json:"gallery"`
		VoteScore        int          `json:"vote_score"`
		UserVote         int          `json:"user_vote,omitempty"`
		FavouriteCount   int          `json:"favourite_count"`
		UserFavourited   bool         `json:"user_favourited"`
		CommentCount     int          `json:"comment_count"`
		IsCrackOC        bool         `json:"is_crack_oc"`
		CreatedAt        string       `json:"created_at"`
		UpdatedAt        *string      `json:"updated_at,omitempty"`
	}

	OCDetailResponse struct {
		OCResponse
		Comments      []OCCommentResponse `json:"comments"`
		ViewerBlocked bool                `json:"viewer_blocked"`
	}

	OCCommentResponse struct {
		ID        uuid.UUID           `json:"id"`
		ParentID  *uuid.UUID          `json:"parent_id,omitempty"`
		Author    UserResponse        `json:"author"`
		Body      string              `json:"body"`
		Media     []PostMediaResponse `json:"media"`
		LikeCount int                 `json:"like_count"`
		UserLiked bool                `json:"user_liked"`
		Replies   []OCCommentResponse `json:"replies,omitempty"`
		CreatedAt string              `json:"created_at"`
		UpdatedAt *string             `json:"updated_at,omitempty"`
	}

	OCListResponse struct {
		OCs    []OCResponse `json:"ocs"`
		Total  int          `json:"total"`
		Limit  int          `json:"limit"`
		Offset int          `json:"offset"`
	}

	OCSummary struct {
		ID               uuid.UUID `json:"id"`
		Name             string    `json:"name"`
		Series           string    `json:"series"`
		CustomSeriesName string    `json:"custom_series_name,omitempty"`
		ThumbnailURL     string    `json:"thumbnail_url,omitempty"`
	}

	CreateOCRequest struct {
		Name             string `json:"name"`
		Description      string `json:"description"`
		Series           string `json:"series"`
		CustomSeriesName string `json:"custom_series_name"`
	}

	UpdateOCRequest struct {
		Name             string `json:"name"`
		Description      string `json:"description"`
		Series           string `json:"series"`
		CustomSeriesName string `json:"custom_series_name"`
	}

	UpdateOCImageRequest struct {
		Caption   *string `json:"caption,omitempty"`
		SortOrder *int    `json:"sort_order,omitempty"`
	}
)
