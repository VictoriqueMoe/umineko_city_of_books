package dto

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
		Comments    []CommentResponse        `json:"comments"`
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
