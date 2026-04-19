package dto

import "github.com/google/uuid"

type (
	SecretSummary struct {
		ID             string        `json:"id"`
		Title          string        `json:"title"`
		Description    string        `json:"description"`
		TotalPieces    int           `json:"total_pieces"`
		Solved         bool          `json:"solved"`
		Solver         *UserResponse `json:"solver,omitempty"`
		SolvedAt       string        `json:"solved_at,omitempty"`
		ViewerProgress int           `json:"viewer_progress"`
		CommentCount   int           `json:"comment_count"`
	}

	SecretLeaderboardEntry struct {
		User   UserResponse `json:"user"`
		Pieces int          `json:"pieces_collected"`
		Solved bool         `json:"solved"`
	}

	SecretDetailResponse struct {
		SecretSummary
		Riddle      string                   `json:"riddle"`
		Leaderboard []SecretLeaderboardEntry `json:"leaderboard"`
		Comments    []SecretCommentResponse  `json:"comments"`
	}

	SecretCommentResponse struct {
		ID        uuid.UUID               `json:"id"`
		ParentID  *uuid.UUID              `json:"parent_id,omitempty"`
		Author    UserResponse            `json:"author"`
		Body      string                  `json:"body"`
		Media     []PostMediaResponse     `json:"media"`
		LikeCount int                     `json:"like_count"`
		UserLiked bool                    `json:"user_liked"`
		Replies   []SecretCommentResponse `json:"replies,omitempty"`
		CreatedAt string                  `json:"created_at"`
		UpdatedAt *string                 `json:"updated_at,omitempty"`
	}

	SecretListResponse struct {
		Secrets            []SecretSummary     `json:"secrets"`
		SolversLeaderboard []SecretSolverEntry `json:"solvers_leaderboard"`
	}

	SecretSolverEntry struct {
		User       UserResponse `json:"user"`
		Solved     int          `json:"solved_count"`
		LastSolved string       `json:"last_solved_at"`
	}

	CreateSecretCommentRequest struct {
		Body     string     `json:"body"`
		ParentID *uuid.UUID `json:"parent_id,omitempty"`
	}

	UpdateSecretCommentRequest struct {
		Body string `json:"body"`
	}

	SecretProgressEvent struct {
		SecretID    string       `json:"secret_id"`
		User        UserResponse `json:"user"`
		Pieces      int          `json:"pieces_collected"`
		TotalPieces int          `json:"total_pieces"`
	}

	SecretSolvedEvent struct {
		SecretID string       `json:"secret_id"`
		Solver   UserResponse `json:"solver"`
		SolvedAt string       `json:"solved_at"`
	}
)
