package dto

import "github.com/google/uuid"

type (
	JournalResponse struct {
		ID                   uuid.UUID    `json:"id"`
		Title                string       `json:"title"`
		Work                 string       `json:"work"`
		Author               UserResponse `json:"author"`
		FollowerCount        int          `json:"follower_count"`
		IsFollowing          bool         `json:"is_following"`
		IsArchived           bool         `json:"is_archived"`
		CommentCount         int          `json:"comment_count"`
		EntryCount           int          `json:"entry_count"`
		LatestEntryNumber    *int         `json:"latest_entry_number,omitempty"`
		LatestEntryTitle     *string      `json:"latest_entry_title,omitempty"`
		LatestEntryExcerpt   string       `json:"latest_entry_excerpt"`
		LatestEntryAt        *string      `json:"latest_entry_at,omitempty"`
		CreatedAt            string       `json:"created_at"`
		UpdatedAt            *string      `json:"updated_at,omitempty"`
		LastAuthorActivityAt string       `json:"last_author_activity_at"`
		ArchivedAt           *string      `json:"archived_at,omitempty"`
	}

	JournalDetailResponse struct {
		JournalResponse
		Entries     []JournalEntrySummary    `json:"entries"`
		LatestEntry *JournalEntryResponse    `json:"latest_entry,omitempty"`
		Comments    []JournalCommentResponse `json:"comments"`
	}

	JournalListResponse struct {
		Journals []JournalResponse `json:"journals"`
		Total    int               `json:"total"`
		Limit    int               `json:"limit"`
		Offset   int               `json:"offset"`
	}

	CreateJournalRequest struct {
		Title string `json:"title"`
		Work  string `json:"work"`
	}

	JournalEntryResponse struct {
		ID          uuid.UUID           `json:"id"`
		JournalID   uuid.UUID           `json:"journal_id"`
		EntryNumber int                 `json:"entry_number"`
		Title       *string             `json:"title,omitempty"`
		Body        string              `json:"body"`
		WordCount   int                 `json:"word_count"`
		HasPrev     bool                `json:"has_prev"`
		HasNext     bool                `json:"has_next"`
		CreatedAt   string              `json:"created_at"`
		UpdatedAt   *string             `json:"updated_at,omitempty"`
		Media       []PostMediaResponse `json:"media"`
	}

	JournalEntrySummary struct {
		ID          uuid.UUID `json:"id"`
		EntryNumber int       `json:"entry_number"`
		Title       *string   `json:"title,omitempty"`
		WordCount   int       `json:"word_count"`
		CreatedAt   string    `json:"created_at"`
	}

	CreateJournalEntryRequest struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	UpdateJournalEntryRequest struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	JournalCommentResponse struct {
		ID        uuid.UUID                `json:"id"`
		ParentID  *uuid.UUID               `json:"parent_id,omitempty"`
		EntryID   *uuid.UUID               `json:"entry_id,omitempty"`
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
