package dto

import "github.com/google/uuid"

type (
	FanficCharacter struct {
		Series        string `json:"series"`
		CharacterID   string `json:"character_id,omitempty"`
		CharacterName string `json:"character_name"`
		SortOrder     int    `json:"sort_order"`
	}

	FanficResponse struct {
		ID                uuid.UUID         `json:"id"`
		Author            UserResponse      `json:"author"`
		Title             string            `json:"title"`
		Summary           string            `json:"summary"`
		Series            string            `json:"series"`
		Rating            string            `json:"rating"`
		Language          string            `json:"language"`
		Status            string            `json:"status"`
		IsOneshot         bool              `json:"is_oneshot"`
		ContainsLemons    bool              `json:"contains_lemons"`
		CoverImageURL     string            `json:"cover_image_url,omitempty"`
		CoverThumbnailURL string            `json:"cover_thumbnail_url,omitempty"`
		Genres            []string          `json:"genres"`
		Tags              []string          `json:"tags"`
		Characters        []FanficCharacter `json:"characters"`
		IsPairing         bool              `json:"is_pairing"`
		WordCount         int               `json:"word_count"`
		ChapterCount      int               `json:"chapter_count"`
		FavouriteCount    int               `json:"favourite_count"`
		ViewCount         int               `json:"view_count"`
		CommentCount      int               `json:"comment_count"`
		UserFavourited    bool              `json:"user_favourited"`
		PublishedAt       string            `json:"published_at"`
		CreatedAt         string            `json:"created_at"`
		UpdatedAt         *string           `json:"updated_at,omitempty"`
	}

	FanficDetailResponse struct {
		FanficResponse
		Chapters        []FanficChapterSummary  `json:"chapters"`
		Comments        []FanficCommentResponse `json:"comments"`
		ReadingProgress int                     `json:"reading_progress"`
		ViewerBlocked   bool                    `json:"viewer_blocked"`
	}

	FanficChapterResponse struct {
		ID         uuid.UUID `json:"id"`
		ChapterNum int       `json:"chapter_number"`
		Title      string    `json:"title"`
		Body       string    `json:"body"`
		WordCount  int       `json:"word_count"`
		HasPrev    bool      `json:"has_prev"`
		HasNext    bool      `json:"has_next"`
		CreatedAt  string    `json:"created_at"`
		UpdatedAt  *string   `json:"updated_at,omitempty"`
	}

	FanficChapterSummary struct {
		ID         uuid.UUID `json:"id"`
		ChapterNum int       `json:"chapter_number"`
		Title      string    `json:"title"`
		WordCount  int       `json:"word_count"`
	}

	FanficCommentResponse struct {
		ID        uuid.UUID               `json:"id"`
		ParentID  *uuid.UUID              `json:"parent_id,omitempty"`
		Author    UserResponse            `json:"author"`
		Body      string                  `json:"body"`
		Media     []PostMediaResponse     `json:"media"`
		LikeCount int                     `json:"like_count"`
		UserLiked bool                    `json:"user_liked"`
		Replies   []FanficCommentResponse `json:"replies,omitempty"`
		CreatedAt string                  `json:"created_at"`
		UpdatedAt *string                 `json:"updated_at,omitempty"`
	}

	FanficListResponse struct {
		Fanfics []FanficResponse `json:"fanfics"`
		Total   int              `json:"total"`
		Limit   int              `json:"limit"`
		Offset  int              `json:"offset"`
	}

	CreateFanficRequest struct {
		Title          string            `json:"title"`
		Summary        string            `json:"summary"`
		Series         string            `json:"series"`
		Rating         string            `json:"rating"`
		Language       string            `json:"language"`
		Status         string            `json:"status"`
		IsOneshot      bool              `json:"is_oneshot"`
		ContainsLemons bool              `json:"contains_lemons"`
		Genres         []string          `json:"genres"`
		Tags           []string          `json:"tags"`
		Characters     []FanficCharacter `json:"characters"`
		IsPairing      bool              `json:"is_pairing"`
		Body           string            `json:"body,omitempty"`
	}

	UpdateFanficRequest struct {
		Title          string            `json:"title"`
		Summary        string            `json:"summary"`
		Series         string            `json:"series"`
		Rating         string            `json:"rating"`
		Language       string            `json:"language"`
		Status         string            `json:"status"`
		IsOneshot      bool              `json:"is_oneshot"`
		ContainsLemons bool              `json:"contains_lemons"`
		Genres         []string          `json:"genres"`
		Tags           []string          `json:"tags"`
		Characters     []FanficCharacter `json:"characters"`
		IsPairing      bool              `json:"is_pairing"`
	}

	CreateChapterRequest struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	UpdateChapterRequest struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
)
