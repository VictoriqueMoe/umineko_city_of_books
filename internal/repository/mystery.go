package repository

import (
	"context"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	MysteryRepository interface {
		Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, body string, difficulty string, freeForAll bool, keepOpenAfterSolve bool) error
		AddClue(ctx context.Context, mysteryID uuid.UUID, body string, truthType string, sortOrder int, playerID *uuid.UUID) error
		Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, body string, difficulty string) error
		UpdateAsAdmin(ctx context.Context, id uuid.UUID, title string, body string, difficulty string, freeForAll bool, keepOpenAfterSolve bool) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		GetByID(ctx context.Context, id uuid.UUID) (*MysteryRow, error)
		List(ctx context.Context, sort string, solved *bool, limit, offset int, excludeUserIDs []uuid.UUID) ([]MysteryRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]MysteryRow, int, error)
		GetClues(ctx context.Context, mysteryID uuid.UUID) ([]dto.MysteryClue, error)
		DeleteClues(ctx context.Context, mysteryID uuid.UUID) error
		DeleteClue(ctx context.Context, clueID int) error
		UpdateClue(ctx context.Context, clueID int, body string) error
		GetAuthorID(ctx context.Context, mysteryID uuid.UUID) (uuid.UUID, error)

		CreateAttempt(ctx context.Context, id uuid.UUID, mysteryID uuid.UUID, userID uuid.UUID, parentID *uuid.UUID, body string) error
		DeleteAttempt(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAttemptAsAdmin(ctx context.Context, id uuid.UUID) error
		GetAttempts(ctx context.Context, mysteryID uuid.UUID, viewerID uuid.UUID) ([]MysteryAttemptRow, error)
		GetAttemptAuthorID(ctx context.Context, attemptID uuid.UUID) (uuid.UUID, error)
		GetAttemptMysteryID(ctx context.Context, attemptID uuid.UUID) (uuid.UUID, error)

		VoteAttempt(ctx context.Context, userID uuid.UUID, attemptID uuid.UUID, value int) error

		MarkSolved(ctx context.Context, mysteryID uuid.UUID, attemptID uuid.UUID, lockMystery bool) error
		MarkPermanentlySolved(ctx context.Context, mysteryID uuid.UUID) error
		UserHasWinningAttempt(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID) (bool, error)
		GetSolverIDs(ctx context.Context, mysteryID uuid.UUID) ([]uuid.UUID, error)
		IsSolved(ctx context.Context, mysteryID uuid.UUID) (bool, error)
		IsPaused(ctx context.Context, mysteryID uuid.UUID) (bool, error)
		SetPaused(ctx context.Context, mysteryID uuid.UUID, paused bool) error
		SetGmAway(ctx context.Context, mysteryID uuid.UUID, away bool) error

		GetLeaderboard(ctx context.Context, limit int) ([]LeaderboardEntry, error)
		GetTopDetectiveIDs(ctx context.Context) ([]string, error)
		GetGMLeaderboard(ctx context.Context, limit int) ([]GMLeaderboardEntry, error)
		GetTopGMIDs(ctx context.Context) ([]string, error)

		CountAttempts(ctx context.Context, mysteryID uuid.UUID) (int, error)
		CountClues(ctx context.Context, mysteryID uuid.UUID) (int, error)
		GetPlayerIDs(ctx context.Context, mysteryID uuid.UUID) ([]uuid.UUID, error)

		CreateComment(ctx context.Context, id uuid.UUID, mysteryID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, mysteryID uuid.UUID, viewerID uuid.UUID, excludeUserIDs []uuid.UUID) ([]MysteryCommentRow, error)
		GetCommentMysteryID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]MysteryCommentMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]MysteryCommentMediaRow, error)

		AddAttachment(ctx context.Context, mysteryID uuid.UUID, fileURL string, fileName string, fileSize int) (int64, error)
		DeleteAttachment(ctx context.Context, id int64, mysteryID uuid.UUID) error
		GetAttachments(ctx context.Context, mysteryID uuid.UUID) ([]dto.MysteryAttachment, error)

		AddMysteryMedia(ctx context.Context, mysteryID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error)
		UpdateMysteryMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateMysteryMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetMysteryMedia(ctx context.Context, mysteryID uuid.UUID) ([]MysteryMediaRow, error)
		DeleteMysteryMedia(ctx context.Context, id int64, mysteryID uuid.UUID) (string, error)
	}

	MysteryRow struct {
		ID                    uuid.UUID
		UserID                uuid.UUID
		Title                 string
		Body                  string
		Difficulty            string
		Solved                bool
		Paused                bool
		GmAway                bool
		FreeForAll            bool
		KeepOpenAfterSolve    bool
		WinnerID              *uuid.UUID
		WinnerUsername        *string
		WinnerDisplayName     *string
		WinnerAvatarURL       *string
		WinnerRole            *string
		SolvedAt              *string
		PausedAt              *string
		PausedDurationSeconds int
		AuthorUsername        string
		AuthorDisplayName     string
		AuthorAvatarURL       string
		AuthorRole            string
		AttemptCount          int
		ClueCount             int
		SolverCount           int
		CreatedAt             string
		UpdatedAt             string
	}

	MysteryAttemptRow struct {
		ID                uuid.UUID
		MysteryID         uuid.UUID
		UserID            uuid.UUID
		ParentID          *uuid.UUID
		Body              string
		IsWinner          bool
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		AuthorRole        string
		VoteScore         int
		UserVote          int
		CreatedAt         string
	}

	MysteryCommentRow struct {
		ID                uuid.UUID
		MysteryID         uuid.UUID
		ParentID          *uuid.UUID
		UserID            uuid.UUID
		Body              string
		CreatedAt         string
		UpdatedAt         *string
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		AuthorRole        string
		LikeCount         int
		UserLiked         bool
	}

	MysteryCommentMediaRow = model.CommentMediaRow

	MysteryMediaRow struct {
		ID           int64
		MysteryID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
	}

	LeaderboardEntry struct {
		UserID          uuid.UUID
		Username        string
		DisplayName     string
		AvatarURL       string
		Role            string
		Score           int
		EasySolved      int
		MediumSolved    int
		HardSolved      int
		NightmareSolved int
		ScoreAdjustment int
	}

	GMLeaderboardEntry struct {
		UserID       uuid.UUID
		Username     string
		DisplayName  string
		AvatarURL    string
		Role         string
		Score        int
		MysteryCount int
		PlayerCount  int
	}
)

func (r *MysteryRow) ToResponse() dto.MysteryResponse {
	resp := dto.MysteryResponse{
		ID:                    r.ID,
		Title:                 r.Title,
		Body:                  r.Body,
		Difficulty:            r.Difficulty,
		Solved:                r.Solved,
		Paused:                r.Paused,
		GmAway:                r.GmAway,
		FreeForAll:            r.FreeForAll,
		KeepOpenAfterSolve:    r.KeepOpenAfterSolve,
		SolverCount:           r.SolverCount,
		SolvedAt:              r.SolvedAt,
		PausedAt:              r.PausedAt,
		PausedDurationSeconds: r.PausedDurationSeconds,
		Author: dto.UserResponse{
			ID:          r.UserID,
			Username:    r.AuthorUsername,
			DisplayName: r.AuthorDisplayName,
			AvatarURL:   r.AuthorAvatarURL,
			Role:        role.Role(r.AuthorRole),
		},
		AttemptCount: r.AttemptCount,
		ClueCount:    r.ClueCount,
		CreatedAt:    r.CreatedAt,
	}
	if r.WinnerID != nil && r.WinnerUsername != nil {
		resp.Winner = &dto.UserResponse{
			ID:          *r.WinnerID,
			Username:    *r.WinnerUsername,
			DisplayName: *r.WinnerDisplayName,
			AvatarURL:   *r.WinnerAvatarURL,
			Role:        role.Role(*r.WinnerRole),
		}
	}
	return resp
}

func (r *MysteryCommentRow) ToResponse(media []MysteryCommentMediaRow) dto.MysteryCommentResponse {
	mediaList := model.CommentMediaRowsToResponse(media)
	return dto.MysteryCommentResponse{
		ID:       r.ID,
		ParentID: r.ParentID,
		Author: dto.UserResponse{
			ID:          r.UserID,
			Username:    r.AuthorUsername,
			DisplayName: r.AuthorDisplayName,
			AvatarURL:   r.AuthorAvatarURL,
			Role:        role.Role(r.AuthorRole),
		},
		Body:      r.Body,
		Media:     mediaList,
		LikeCount: r.LikeCount,
		UserLiked: r.UserLiked,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
