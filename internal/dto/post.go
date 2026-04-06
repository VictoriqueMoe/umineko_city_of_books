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

	EmbedResponse struct {
		URL      string `json:"url"`
		Type     string `json:"type"`
		Title    string `json:"title,omitempty"`
		Desc     string `json:"description,omitempty"`
		Image    string `json:"image,omitempty"`
		SiteName string `json:"site_name,omitempty"`
		VideoID  string `json:"video_id,omitempty"`
	}

	PostResponse struct {
		ID           uuid.UUID           `json:"id"`
		Author       UserResponse        `json:"author"`
		Body         string              `json:"body"`
		Media        []PostMediaResponse `json:"media"`
		Embeds       []EmbedResponse     `json:"embeds,omitempty"`
		Poll         *PollResponse       `json:"poll,omitempty"`
		LikeCount    int                 `json:"like_count"`
		CommentCount int                 `json:"comment_count"`
		ViewCount    int                 `json:"view_count"`
		UserLiked    bool                `json:"user_liked"`
		CreatedAt    string              `json:"created_at"`
		UpdatedAt    *string             `json:"updated_at,omitempty"`
	}

	UpdatePostRequest struct {
		Body string `json:"body"`
	}

	PostDetailResponse struct {
		PostResponse
		Comments      []PostCommentResponse `json:"comments"`
		LikedBy       []UserResponse        `json:"liked_by"`
		ViewerBlocked bool                  `json:"viewer_blocked"`
	}

	PostCommentResponse struct {
		ID        uuid.UUID             `json:"id"`
		ParentID  *uuid.UUID            `json:"parent_id,omitempty"`
		Author    UserResponse          `json:"author"`
		Body      string                `json:"body"`
		Media     []PostMediaResponse   `json:"media"`
		Embeds    []EmbedResponse       `json:"embeds,omitempty"`
		LikeCount int                   `json:"like_count"`
		UserLiked bool                  `json:"user_liked"`
		Replies   []PostCommentResponse `json:"replies,omitempty"`
		CreatedAt string                `json:"created_at"`
		UpdatedAt *string               `json:"updated_at,omitempty"`
	}

	PostListResponse struct {
		Posts  []PostResponse `json:"posts"`
		Total  int            `json:"total"`
		Limit  int            `json:"limit"`
		Offset int            `json:"offset"`
	}

	CreatePostRequest struct {
		Corner string           `json:"corner"`
		Body   string           `json:"body"`
		Poll   *CreatePollInput `json:"poll,omitempty"`
	}

	CreatePollInput struct {
		Options         []PollOptionInput `json:"options"`
		DurationSeconds int               `json:"duration_seconds"`
	}

	PollOptionInput struct {
		Label string `json:"label"`
	}

	PollResponse struct {
		ID              string               `json:"id"`
		Options         []PollOptionResponse `json:"options"`
		TotalVotes      int                  `json:"total_votes"`
		UserVotedOption *int                 `json:"user_voted_option"`
		Expired         bool                 `json:"expired"`
		ExpiresAt       string               `json:"expires_at"`
		DurationSeconds int                  `json:"duration_seconds"`
	}

	PollOptionResponse struct {
		ID        int     `json:"id"`
		Label     string  `json:"label"`
		VoteCount int     `json:"vote_count"`
		Percent   float64 `json:"percent"`
	}

	VotePollRequest struct {
		OptionID int `json:"option_id"`
	}

	CreateCommentRequest struct {
		ParentID *uuid.UUID `json:"parent_id,omitempty"`
		Body     string     `json:"body"`
	}

	UpdateCommentRequest struct {
		Body string `json:"body"`
	}

	FollowStatsResponse struct {
		FollowerCount  int  `json:"follower_count"`
		FollowingCount int  `json:"following_count"`
		IsFollowing    bool `json:"is_following"`
		FollowsYou     bool `json:"follows_you"`
	}
)
