package dto

import "github.com/google/uuid"

const CrackshipThreshold = -3

type (
	ShipCharacter struct {
		Series        string `json:"series"`
		CharacterID   string `json:"character_id,omitempty"`
		CharacterName string `json:"character_name"`
		SortOrder     int    `json:"sort_order"`
	}

	ShipResponse struct {
		ID           uuid.UUID       `json:"id"`
		Author       UserResponse    `json:"author"`
		Title        string          `json:"title"`
		Description  string          `json:"description"`
		ImageURL     string          `json:"image_url,omitempty"`
		ThumbnailURL string          `json:"thumbnail_url,omitempty"`
		Characters   []ShipCharacter `json:"characters"`
		VoteScore    int             `json:"vote_score"`
		UserVote     int             `json:"user_vote,omitempty"`
		CommentCount int             `json:"comment_count"`
		IsCrackship  bool            `json:"is_crackship"`
		CreatedAt    string          `json:"created_at"`
		UpdatedAt    *string         `json:"updated_at,omitempty"`
	}

	ShipDetailResponse struct {
		ShipResponse
		Comments      []ShipCommentResponse `json:"comments"`
		ViewerBlocked bool                  `json:"viewer_blocked"`
	}

	ShipCommentResponse struct {
		ID        uuid.UUID             `json:"id"`
		ParentID  *uuid.UUID            `json:"parent_id,omitempty"`
		Author    UserResponse          `json:"author"`
		Body      string                `json:"body"`
		Media     []PostMediaResponse   `json:"media"`
		LikeCount int                   `json:"like_count"`
		UserLiked bool                  `json:"user_liked"`
		Replies   []ShipCommentResponse `json:"replies,omitempty"`
		CreatedAt string                `json:"created_at"`
		UpdatedAt *string               `json:"updated_at,omitempty"`
	}

	ShipListResponse struct {
		Ships  []ShipResponse `json:"ships"`
		Total  int            `json:"total"`
		Limit  int            `json:"limit"`
		Offset int            `json:"offset"`
	}

	CreateShipRequest struct {
		Title       string          `json:"title"`
		Description string          `json:"description"`
		Characters  []ShipCharacter `json:"characters"`
	}

	UpdateShipRequest struct {
		Title       string          `json:"title"`
		Description string          `json:"description"`
		Characters  []ShipCharacter `json:"characters"`
	}

	CharacterListResponse struct {
		Series     string               `json:"series"`
		Characters []CharacterListEntry `json:"characters"`
	}

	CharacterListEntry struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
)
