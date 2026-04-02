package dto

import "github.com/google/uuid"

type (
	ArtResponse struct {
		ID           uuid.UUID    `json:"id"`
		Author       UserResponse `json:"author"`
		Corner       string       `json:"corner"`
		Title        string       `json:"title"`
		Description  string       `json:"description"`
		ImageURL     string       `json:"image_url"`
		ThumbnailURL string       `json:"thumbnail_url"`
		GalleryID    *uuid.UUID   `json:"gallery_id,omitempty"`
		Tags         []string     `json:"tags"`
		LikeCount    int          `json:"like_count"`
		CommentCount int          `json:"comment_count"`
		ViewCount    int          `json:"view_count"`
		UserLiked    bool         `json:"user_liked"`
		CreatedAt    string       `json:"created_at"`
		UpdatedAt    *string      `json:"updated_at,omitempty"`
	}

	ArtDetailResponse struct {
		ArtResponse
		Comments []ArtCommentResponse `json:"comments"`
		LikedBy  []UserResponse       `json:"liked_by"`
	}

	CreateArtRequest struct {
		Title       string     `json:"title"`
		Description string     `json:"description"`
		Corner      string     `json:"corner"`
		Tags        []string   `json:"tags"`
		GalleryID   *uuid.UUID `json:"gallery_id,omitempty"`
	}

	UpdateArtRequest struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	}

	ArtCommentResponse struct {
		ID        uuid.UUID            `json:"id"`
		ParentID  *uuid.UUID           `json:"parent_id,omitempty"`
		Author    UserResponse         `json:"author"`
		Body      string               `json:"body"`
		Media     []PostMediaResponse  `json:"media"`
		Embeds    []EmbedResponse      `json:"embeds,omitempty"`
		LikeCount int                  `json:"like_count"`
		UserLiked bool                 `json:"user_liked"`
		Replies   []ArtCommentResponse `json:"replies,omitempty"`
		CreatedAt string               `json:"created_at"`
		UpdatedAt *string              `json:"updated_at,omitempty"`
	}

	ArtListResponse struct {
		Art    []ArtResponse `json:"art"`
		Total  int           `json:"total"`
		Limit  int           `json:"limit"`
		Offset int           `json:"offset"`
	}

	TagCountResponse struct {
		Tag   string `json:"tag"`
		Count int    `json:"count"`
	}

	GalleryResponse struct {
		ID                uuid.UUID    `json:"id"`
		Author            UserResponse `json:"author"`
		Name              string       `json:"name"`
		Description       string       `json:"description"`
		CoverImageURL     string       `json:"cover_image_url"`
		CoverThumbnailURL string       `json:"cover_thumbnail_url"`
		ArtCount          int          `json:"art_count"`
		CreatedAt         string       `json:"created_at"`
		UpdatedAt         *string      `json:"updated_at,omitempty"`
	}

	CreateGalleryRequest struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	UpdateGalleryRequest struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
)
