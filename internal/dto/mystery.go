package dto

import "github.com/google/uuid"

type (
	MysteryResponse struct {
		ID                    uuid.UUID     `json:"id"`
		Title                 string        `json:"title"`
		Body                  string        `json:"body"`
		Difficulty            string        `json:"difficulty"`
		Author                UserResponse  `json:"author"`
		Solved                bool          `json:"solved"`
		Paused                bool          `json:"paused"`
		GmAway                bool          `json:"gm_away"`
		FreeForAll            bool          `json:"free_for_all"`
		Winner                *UserResponse `json:"winner,omitempty"`
		SolvedAt              *string       `json:"solved_at,omitempty"`
		PausedAt              *string       `json:"paused_at,omitempty"`
		PausedDurationSeconds int           `json:"paused_duration_seconds"`
		AttemptCount          int           `json:"attempt_count"`
		ClueCount             int           `json:"clue_count"`
		CreatedAt             string        `json:"created_at"`
	}

	MysteryDetailResponse struct {
		ID                    uuid.UUID                `json:"id"`
		Title                 string                   `json:"title"`
		Body                  string                   `json:"body"`
		Difficulty            string                   `json:"difficulty"`
		Author                UserResponse             `json:"author"`
		Solved                bool                     `json:"solved"`
		Paused                bool                     `json:"paused"`
		GmAway                bool                     `json:"gm_away"`
		FreeForAll            bool                     `json:"free_for_all"`
		Winner                *UserResponse            `json:"winner,omitempty"`
		SolvedAt              *string                  `json:"solved_at,omitempty"`
		PausedAt              *string                  `json:"paused_at,omitempty"`
		PausedDurationSeconds int                      `json:"paused_duration_seconds"`
		Clues                 []MysteryClue            `json:"clues"`
		Attempts              []MysteryAttempt         `json:"attempts"`
		Comments              []MysteryCommentResponse `json:"comments"`
		Attachments           []MysteryAttachment      `json:"attachments"`
		PlayerCount           int                      `json:"player_count"`
		CreatedAt             string                   `json:"created_at"`
	}

	MysteryAttachment struct {
		ID       int    `json:"id"`
		FileURL  string `json:"file_url"`
		FileName string `json:"file_name"`
		FileSize int    `json:"file_size"`
	}

	MysteryCommentResponse struct {
		ID        uuid.UUID                `json:"id"`
		ParentID  *uuid.UUID               `json:"parent_id,omitempty"`
		Author    UserResponse             `json:"author"`
		Body      string                   `json:"body"`
		Media     []PostMediaResponse      `json:"media"`
		LikeCount int                      `json:"like_count"`
		UserLiked bool                     `json:"user_liked"`
		Replies   []MysteryCommentResponse `json:"replies,omitempty"`
		CreatedAt string                   `json:"created_at"`
		UpdatedAt *string                  `json:"updated_at,omitempty"`
	}

	MysteryClue struct {
		ID        int        `json:"id"`
		Body      string     `json:"body"`
		TruthType string     `json:"truth_type"`
		SortOrder int        `json:"sort_order"`
		PlayerID  *uuid.UUID `json:"player_id,omitempty"`
	}

	MysteryAttempt struct {
		ID        uuid.UUID        `json:"id"`
		ParentID  *uuid.UUID       `json:"parent_id,omitempty"`
		Author    UserResponse     `json:"author"`
		Body      string           `json:"body"`
		IsWinner  bool             `json:"is_winner"`
		VoteScore int              `json:"vote_score"`
		UserVote  int              `json:"user_vote,omitempty"`
		Replies   []MysteryAttempt `json:"replies,omitempty"`
		CreatedAt string           `json:"created_at"`
	}

	MysteryListResponse struct {
		Mysteries []MysteryResponse `json:"mysteries"`
		Total     int               `json:"total"`
		Limit     int               `json:"limit"`
		Offset    int               `json:"offset"`
	}

	CreateMysteryRequest struct {
		Title      string              `json:"title"`
		Body       string              `json:"body"`
		Difficulty string              `json:"difficulty"`
		FreeForAll bool                `json:"free_for_all"`
		Clues      []CreateClueRequest `json:"clues"`
	}

	CreateClueRequest struct {
		Body      string     `json:"body"`
		TruthType string     `json:"truth_type"`
		PlayerID  *uuid.UUID `json:"player_id,omitempty"`
	}

	CreateAttemptRequest struct {
		ParentID *uuid.UUID `json:"parent_id,omitempty"`
		Body     string     `json:"body"`
	}

	MysteryLeaderboardEntry struct {
		User            UserResponse `json:"user"`
		Score           int          `json:"score"`
		EasySolved      int          `json:"easy_solved"`
		MediumSolved    int          `json:"medium_solved"`
		HardSolved      int          `json:"hard_solved"`
		NightmareSolved int          `json:"nightmare_solved"`
		ScoreAdjustment int          `json:"score_adjustment"`
	}

	MysteryLeaderboardResponse struct {
		Entries []MysteryLeaderboardEntry `json:"entries"`
	}

	GMLeaderboardEntry struct {
		User         UserResponse `json:"user"`
		Score        int          `json:"score"`
		MysteryCount int          `json:"mystery_count"`
		PlayerCount  int          `json:"player_count"`
	}

	GMLeaderboardResponse struct {
		Entries []GMLeaderboardEntry `json:"entries"`
	}
)
