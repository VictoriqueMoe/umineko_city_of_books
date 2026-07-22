package repository

import (
	"context"

	"umineko_city_of_books/internal/cache"
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
		GetComments(ctx context.Context, mysteryID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error)

		AddAttachment(ctx context.Context, mysteryID uuid.UUID, fileURL string, fileName string, fileSize int) (int64, error)
		DeleteAttachment(ctx context.Context, id int64, mysteryID uuid.UUID) error
		GetAttachments(ctx context.Context, mysteryID uuid.UUID) ([]dto.MysteryAttachment, error)

		AddMedia(ctx context.Context, mysteryID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error)
		UpdateMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetMedia(ctx context.Context, mysteryID uuid.UUID) ([]model.PostMediaRow, error)
		DeleteMedia(ctx context.Context, id int64, mysteryID uuid.UUID) (string, error)
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

type mysteryRepository struct {
	dao   MysteryRepository
	cache *cache.Manager
}

func NewMysteryRepo(dao MysteryRepository, c *cache.Manager) MysteryRepository {
	return &mysteryRepository{dao: dao, cache: c}
}

func (r *mysteryRepository) Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, body string, difficulty string, freeForAll bool, keepOpenAfterSolve bool) error {
	return r.dao.Create(ctx, id, userID, title, body, difficulty, freeForAll, keepOpenAfterSolve)
}

func (r *mysteryRepository) AddClue(ctx context.Context, mysteryID uuid.UUID, body string, truthType string, sortOrder int, playerID *uuid.UUID) error {
	return r.dao.AddClue(ctx, mysteryID, body, truthType, sortOrder, playerID)
}

func (r *mysteryRepository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, body string, difficulty string) error {
	return r.dao.Update(ctx, id, userID, title, body, difficulty)
}

func (r *mysteryRepository) UpdateAsAdmin(ctx context.Context, id uuid.UUID, title string, body string, difficulty string, freeForAll bool, keepOpenAfterSolve bool) error {
	return r.dao.UpdateAsAdmin(ctx, id, title, body, difficulty, freeForAll, keepOpenAfterSolve)
}

func (r *mysteryRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.Delete(ctx, id, userID)
}

func (r *mysteryRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteAsAdmin(ctx, id)
}

func (r *mysteryRepository) GetByID(ctx context.Context, id uuid.UUID) (*MysteryRow, error) {
	return r.dao.GetByID(ctx, id)
}

func (r *mysteryRepository) List(ctx context.Context, sort string, solved *bool, limit, offset int, excludeUserIDs []uuid.UUID) ([]MysteryRow, int, error) {
	return r.dao.List(ctx, sort, solved, limit, offset, excludeUserIDs)
}

func (r *mysteryRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]MysteryRow, int, error) {
	return r.dao.ListByUser(ctx, userID, limit, offset)
}

func (r *mysteryRepository) GetClues(ctx context.Context, mysteryID uuid.UUID) ([]dto.MysteryClue, error) {
	return r.dao.GetClues(ctx, mysteryID)
}

func (r *mysteryRepository) DeleteClues(ctx context.Context, mysteryID uuid.UUID) error {
	return r.dao.DeleteClues(ctx, mysteryID)
}

func (r *mysteryRepository) DeleteClue(ctx context.Context, clueID int) error {
	return r.dao.DeleteClue(ctx, clueID)
}

func (r *mysteryRepository) UpdateClue(ctx context.Context, clueID int, body string) error {
	return r.dao.UpdateClue(ctx, clueID, body)
}

func (r *mysteryRepository) GetAuthorID(ctx context.Context, mysteryID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetAuthorID(ctx, mysteryID)
}

func (r *mysteryRepository) CreateAttempt(ctx context.Context, id uuid.UUID, mysteryID uuid.UUID, userID uuid.UUID, parentID *uuid.UUID, body string) error {
	return r.dao.CreateAttempt(ctx, id, mysteryID, userID, parentID, body)
}

func (r *mysteryRepository) DeleteAttempt(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.DeleteAttempt(ctx, id, userID)
}

func (r *mysteryRepository) DeleteAttemptAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteAttemptAsAdmin(ctx, id)
}

func (r *mysteryRepository) GetAttempts(ctx context.Context, mysteryID uuid.UUID, viewerID uuid.UUID) ([]MysteryAttemptRow, error) {
	return r.dao.GetAttempts(ctx, mysteryID, viewerID)
}

func (r *mysteryRepository) GetAttemptAuthorID(ctx context.Context, attemptID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetAttemptAuthorID(ctx, attemptID)
}

func (r *mysteryRepository) GetAttemptMysteryID(ctx context.Context, attemptID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetAttemptMysteryID(ctx, attemptID)
}

func (r *mysteryRepository) VoteAttempt(ctx context.Context, userID uuid.UUID, attemptID uuid.UUID, value int) error {
	return r.dao.VoteAttempt(ctx, userID, attemptID, value)
}

func (r *mysteryRepository) MarkSolved(ctx context.Context, mysteryID uuid.UUID, attemptID uuid.UUID, lockMystery bool) error {
	if err := r.dao.MarkSolved(ctx, mysteryID, attemptID, lockMystery); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.MysteryTopDetectives.Key(), cache.MysteryTopGMs.Key())
}

func (r *mysteryRepository) MarkPermanentlySolved(ctx context.Context, mysteryID uuid.UUID) error {
	if err := r.dao.MarkPermanentlySolved(ctx, mysteryID); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.MysteryTopDetectives.Key(), cache.MysteryTopGMs.Key())
}

func (r *mysteryRepository) UserHasWinningAttempt(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID) (bool, error) {
	return r.dao.UserHasWinningAttempt(ctx, mysteryID, userID)
}

func (r *mysteryRepository) GetSolverIDs(ctx context.Context, mysteryID uuid.UUID) ([]uuid.UUID, error) {
	return r.dao.GetSolverIDs(ctx, mysteryID)
}

func (r *mysteryRepository) IsSolved(ctx context.Context, mysteryID uuid.UUID) (bool, error) {
	return r.dao.IsSolved(ctx, mysteryID)
}

func (r *mysteryRepository) IsPaused(ctx context.Context, mysteryID uuid.UUID) (bool, error) {
	return r.dao.IsPaused(ctx, mysteryID)
}

func (r *mysteryRepository) SetPaused(ctx context.Context, mysteryID uuid.UUID, paused bool) error {
	return r.dao.SetPaused(ctx, mysteryID, paused)
}

func (r *mysteryRepository) SetGmAway(ctx context.Context, mysteryID uuid.UUID, away bool) error {
	return r.dao.SetGmAway(ctx, mysteryID, away)
}

func (r *mysteryRepository) GetLeaderboard(ctx context.Context, limit int) ([]LeaderboardEntry, error) {
	return r.dao.GetLeaderboard(ctx, limit)
}

func (r *mysteryRepository) GetTopDetectiveIDs(ctx context.Context) ([]string, error) {
	key := cache.MysteryTopDetectives.Key()

	if v, err := cache.Get[[]string](ctx, r.cache, key); err == nil {
		return v, nil
	}

	v, err := r.dao.GetTopDetectiveIDs(ctx)
	if err != nil {
		return nil, err
	}

	_ = cache.Set(ctx, r.cache, key, v, cache.MysteryTopDetectives.TTL)
	return v, nil
}

func (r *mysteryRepository) GetGMLeaderboard(ctx context.Context, limit int) ([]GMLeaderboardEntry, error) {
	return r.dao.GetGMLeaderboard(ctx, limit)
}

func (r *mysteryRepository) GetTopGMIDs(ctx context.Context) ([]string, error) {
	key := cache.MysteryTopGMs.Key()

	if v, err := cache.Get[[]string](ctx, r.cache, key); err == nil {
		return v, nil
	}

	v, err := r.dao.GetTopGMIDs(ctx)
	if err != nil {
		return nil, err
	}

	_ = cache.Set(ctx, r.cache, key, v, cache.MysteryTopGMs.TTL)
	return v, nil
}

func (r *mysteryRepository) CountAttempts(ctx context.Context, mysteryID uuid.UUID) (int, error) {
	return r.dao.CountAttempts(ctx, mysteryID)
}

func (r *mysteryRepository) CountClues(ctx context.Context, mysteryID uuid.UUID) (int, error) {
	return r.dao.CountClues(ctx, mysteryID)
}

func (r *mysteryRepository) GetPlayerIDs(ctx context.Context, mysteryID uuid.UUID) ([]uuid.UUID, error) {
	return r.dao.GetPlayerIDs(ctx, mysteryID)
}

func (r *mysteryRepository) CreateComment(ctx context.Context, id uuid.UUID, mysteryID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.CreateComment(ctx, id, mysteryID, parentID, userID, body)
}

func (r *mysteryRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.UpdateComment(ctx, id, userID, body)
}

func (r *mysteryRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body)
}

func (r *mysteryRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.DeleteComment(ctx, id, userID)
}

func (r *mysteryRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id)
}

func (r *mysteryRepository) GetComments(ctx context.Context, mysteryID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, mysteryID, viewerID, limit, offset, excludeUserIDs)
}

func (r *mysteryRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentEntityID(ctx, commentID)
}

func (r *mysteryRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID)
}

func (r *mysteryRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.LikeComment(ctx, userID, commentID)
}

func (r *mysteryRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.UnlikeComment(ctx, userID, commentID)
}

func (r *mysteryRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	return r.dao.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbnailURL, sortOrder)
}

func (r *mysteryRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL)
}

func (r *mysteryRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL)
}

func (r *mysteryRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error) {
	return r.dao.GetCommentMedia(ctx, commentID)
}

func (r *mysteryRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs)
}

func (r *mysteryRepository) AddAttachment(ctx context.Context, mysteryID uuid.UUID, fileURL string, fileName string, fileSize int) (int64, error) {
	return r.dao.AddAttachment(ctx, mysteryID, fileURL, fileName, fileSize)
}

func (r *mysteryRepository) DeleteAttachment(ctx context.Context, id int64, mysteryID uuid.UUID) error {
	return r.dao.DeleteAttachment(ctx, id, mysteryID)
}

func (r *mysteryRepository) GetAttachments(ctx context.Context, mysteryID uuid.UUID) ([]dto.MysteryAttachment, error) {
	return r.dao.GetAttachments(ctx, mysteryID)
}

func (r *mysteryRepository) AddMedia(ctx context.Context, mysteryID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error) {
	return r.dao.AddMedia(ctx, mysteryID, mediaURL, mediaType, thumbnailURL, sortOrder)
}

func (r *mysteryRepository) UpdateMediaURL(ctx context.Context, id int64, mediaURL string) error {
	return r.dao.UpdateMediaURL(ctx, id, mediaURL)
}

func (r *mysteryRepository) UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	return r.dao.UpdateMediaThumbnail(ctx, id, thumbnailURL)
}

func (r *mysteryRepository) GetMedia(ctx context.Context, mysteryID uuid.UUID) ([]model.PostMediaRow, error) {
	return r.dao.GetMedia(ctx, mysteryID)
}

func (r *mysteryRepository) DeleteMedia(ctx context.Context, id int64, mysteryID uuid.UUID) (string, error) {
	return r.dao.DeleteMedia(ctx, id, mysteryID)
}
