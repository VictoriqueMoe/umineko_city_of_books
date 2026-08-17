package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	MysteryDAO interface {
		Create(ctx context.Context, userID uuid.UUID, title string, body string, difficulty string, freeForAll bool, keepOpenAfterSolve bool, knox dto.KnoxContract, tx ...*sql.Tx) (*MysteryRow, error)
		AddClue(ctx context.Context, mysteryID uuid.UUID, spec NewClue, tx ...*sql.Tx) (*dto.MysteryClue, error)
		Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, body string, difficulty string, tx ...*sql.Tx) error
		UpdateAsAdmin(ctx context.Context, id uuid.UUID, title string, body string, difficulty string, freeForAll bool, keepOpenAfterSolve bool, knox dto.KnoxContract, tx ...*sql.Tx) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*MysteryRow, error)
		List(ctx context.Context, sort string, solved *bool, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]MysteryRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]MysteryRow, int, error)
		GetClues(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]dto.MysteryClue, error)
		DeleteClues(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) error
		DeleteClue(ctx context.Context, clueID int, tx ...*sql.Tx) error
		UpdateClue(ctx context.Context, clueID int, body string, tx ...*sql.Tx) error
		GetAuthorID(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)

		CreateAttempt(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, parentID *uuid.UUID, body string, tx ...*sql.Tx) (*MysteryAttemptRow, error)
		DeleteAttempt(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteAttemptAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetAttempts(ctx context.Context, mysteryID uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) ([]MysteryAttemptRow, error)
		GetAttemptAuthorID(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetAttemptMysteryID(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)

		VoteAttempt(ctx context.Context, userID uuid.UUID, attemptID uuid.UUID, value int, tx ...*sql.Tx) error

		GetAttemptOwner(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, uuid.UUID, error)
		SetMysteryWinner(ctx context.Context, mysteryID uuid.UUID, winnerID uuid.UUID, tx ...*sql.Tx) error
		SetAttemptWinner(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) error
		MarkPermanentlySolved(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) error
		UserHasWinningAttempt(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) (bool, error)
		GetSolverIDs(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
		IsSolved(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (bool, error)
		IsPaused(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (bool, error)
		SetPaused(ctx context.Context, mysteryID uuid.UUID, paused bool, tx ...*sql.Tx) error
		SetGmAway(ctx context.Context, mysteryID uuid.UUID, away bool, tx ...*sql.Tx) error

		GetLeaderboard(ctx context.Context, limit int, tx ...*sql.Tx) ([]LeaderboardEntry, error)
		GetTopDetectiveIDs(ctx context.Context, tx ...*sql.Tx) ([]string, error)
		GetGMLeaderboard(ctx context.Context, limit int, tx ...*sql.Tx) ([]GMLeaderboardEntry, error)
		GetTopGMIDs(ctx context.Context, tx ...*sql.Tx) ([]string, error)

		CountAttempts(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (int, error)
		CountClues(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetPlayerIDs(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)

		CreateComment(ctx context.Context, mysteryID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetComments(ctx context.Context, mysteryID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error
		AddCommentMedia(ctx context.Context, spec NewMysteryCommentMedia, tx ...*sql.Tx) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)

		AddAttachment(ctx context.Context, mysteryID uuid.UUID, fileURL string, fileName string, fileSize int, tx ...*sql.Tx) (int64, error)
		DeleteAttachment(ctx context.Context, id int64, mysteryID uuid.UUID, tx ...*sql.Tx) error
		GetAttachments(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]dto.MysteryAttachment, error)

		AddMedia(ctx context.Context, spec NewMysteryMedia, tx ...*sql.Tx) (int64, error)
		UpdateMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error
		UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error
		GetMedia(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		DeleteMedia(ctx context.Context, id int64, mysteryID uuid.UUID, tx ...*sql.Tx) (string, error)

		GetAttachmentPaths(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)
	}

	MysteryRepository interface {
		MysteryDAO

		MarkSolved(ctx context.Context, mysteryID uuid.UUID, attemptID uuid.UUID, lockMystery bool, tx ...*sql.Tx) error
		CreateWithClues(ctx context.Context, spec NewMystery, tx ...*sql.Tx) (*MysteryRow, error)
		UpdateWithClues(ctx context.Context, spec MysteryUpdate, tx ...*sql.Tx) error
		DeleteWithFiles(ctx context.Context, spec MysteryDelete, tx ...*sql.Tx) ([]string, error)
		DeleteCommentWithAudit(ctx context.Context, spec MysteryCommentDelete, tx ...*sql.Tx) ([]string, error)
	}

	NewMystery struct {
		UserID             uuid.UUID
		Title              string
		Body               string
		Difficulty         string
		FreeForAll         bool
		KeepOpenAfterSolve bool
		Knox               dto.KnoxContract
		Clues              []NewClue
	}

	MysteryUpdate struct {
		ID                 uuid.UUID
		Title              string
		Body               string
		Difficulty         string
		FreeForAll         bool
		KeepOpenAfterSolve bool
		Knox               dto.KnoxContract
		Clues              []NewClue
	}

	NewClue struct {
		Body      string
		TruthType string
		SortOrder int
		PlayerID  *uuid.UUID
	}

	NewMysteryMedia struct {
		MysteryID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
	}

	NewMysteryCommentMedia struct {
		CommentID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
	}

	MysteryDelete struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		AsAdmin bool
	}

	MysteryCommentDelete struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		AsAdmin bool
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
		Knox                  dto.KnoxContract
		KnoxPublished         bool
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
	db    *sql.DB
	dao   MysteryDAO
	audit AuditLogRepository
	cache *cache.Manager
}

func NewMysteryRepo(database *sql.DB, dao MysteryDAO, audit AuditLogRepository, c *cache.Manager) MysteryRepository {
	return &mysteryRepository{db: database, dao: dao, audit: audit, cache: c}
}

func (r *mysteryRepository) Create(ctx context.Context, userID uuid.UUID, title string, body string, difficulty string, freeForAll bool, keepOpenAfterSolve bool, knox dto.KnoxContract, tx ...*sql.Tx) (*MysteryRow, error) {
	return r.dao.Create(ctx, userID, title, body, difficulty, freeForAll, keepOpenAfterSolve, knox, tx...)
}

func (r *mysteryRepository) CreateWithClues(ctx context.Context, spec NewMystery, tx ...*sql.Tx) (*MysteryRow, error) {
	var created *MysteryRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.dao.Create(ctx, spec.UserID, spec.Title, spec.Body, spec.Difficulty, spec.FreeForAll, spec.KeepOpenAfterSolve, spec.Knox, tx)
		if err != nil {
			return err
		}

		return r.addClues(ctx, created.ID, spec.Clues, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *mysteryRepository) UpdateWithClues(ctx context.Context, spec MysteryUpdate, tx ...*sql.Tx) error {
	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.UpdateAsAdmin(ctx, spec.ID, spec.Title, spec.Body, spec.Difficulty, spec.FreeForAll, spec.KeepOpenAfterSolve, spec.Knox, tx); err != nil {
			return err
		}

		if err := r.dao.DeleteClues(ctx, spec.ID, tx); err != nil {
			return err
		}

		return r.addClues(ctx, spec.ID, spec.Clues, tx)
	})
	if err != nil {
		return err
	}

	r.invalidateLeaderboards(ctx)

	return nil
}

func (r *mysteryRepository) addClues(ctx context.Context, mysteryID uuid.UUID, clues []NewClue, tx *sql.Tx) error {
	for i := range clues {
		if _, err := r.dao.AddClue(ctx, mysteryID, clues[i], tx); err != nil {
			return err
		}
	}

	return nil
}

func (r *mysteryRepository) AddClue(ctx context.Context, mysteryID uuid.UUID, spec NewClue, tx ...*sql.Tx) (*dto.MysteryClue, error) {
	return r.dao.AddClue(ctx, mysteryID, spec, tx...)
}

func (r *mysteryRepository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, body string, difficulty string, tx ...*sql.Tx) error {
	if err := r.dao.Update(ctx, id, userID, title, body, difficulty, tx...); err != nil {
		return err
	}

	r.invalidateLeaderboards(ctx)

	return nil
}

func (r *mysteryRepository) UpdateAsAdmin(ctx context.Context, id uuid.UUID, title string, body string, difficulty string, freeForAll bool, keepOpenAfterSolve bool, knox dto.KnoxContract, tx ...*sql.Tx) error {
	if err := r.dao.UpdateAsAdmin(ctx, id, title, body, difficulty, freeForAll, keepOpenAfterSolve, knox, tx...); err != nil {
		return err
	}

	r.invalidateLeaderboards(ctx)

	return nil
}

func (r *mysteryRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	if err := r.dao.Delete(ctx, id, userID, tx...); err != nil {
		return err
	}

	r.invalidateLeaderboards(ctx)

	return nil
}

func (r *mysteryRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	if err := r.dao.DeleteAsAdmin(ctx, id, tx...); err != nil {
		return err
	}

	r.invalidateLeaderboards(ctx)

	return nil
}

func (r *mysteryRepository) DeleteWithFiles(ctx context.Context, spec MysteryDelete, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		row, err := r.dao.GetByID(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		collected, err := r.collectFilePaths(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		paths = dedupePaths(collected)

		if spec.AsAdmin {
			if err := r.dao.DeleteAsAdmin(ctx, spec.ID, tx); err != nil {
				return err
			}
		} else {
			if err := r.dao.Delete(ctx, spec.ID, spec.UserID, tx); err != nil {
				return err
			}
		}

		if row == nil {
			return nil
		}

		action := AuditActionMysteryDelete
		if row.UserID != spec.UserID {
			action = AuditActionMysteryDeleteAdmin
		}

		if err := r.audit.Create(ctx, NewAuditEntry{
			ActorID:    spec.UserID,
			Action:     action,
			TargetType: AuditTargetMystery,
			TargetID:   spec.ID.String(),
			Details:    fmt.Sprintf("title=%q attempts=%d", row.Title, row.AttemptCount),
			SubjectID:  row.UserID,
		}, tx); err != nil {
			return fmt.Errorf("audit mystery delete: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	r.invalidateLeaderboards(ctx)

	return paths, nil
}

func (r *mysteryRepository) collectFilePaths(ctx context.Context, mysteryID uuid.UUID, tx *sql.Tx) ([]string, error) {
	mediaPaths, err := r.dao.CollectMediaPaths(ctx, mysteryID, tx)
	if err != nil {
		return nil, err
	}

	attachmentPaths, err := r.dao.GetAttachmentPaths(ctx, mysteryID, tx)
	if err != nil {
		return nil, err
	}

	commentMediaPaths, err := r.dao.CollectCommentMediaPaths(ctx, mysteryID, tx)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(mediaPaths)+len(attachmentPaths)+len(commentMediaPaths))
	paths = append(paths, mediaPaths...)
	paths = append(paths, attachmentPaths...)
	paths = append(paths, commentMediaPaths...)

	return dedupePaths(paths), nil
}

func (r *mysteryRepository) invalidateLeaderboards(ctx context.Context) {
	if err := r.cache.Del(ctx, cache.MysteryTopDetectives.Key(), cache.MysteryTopGMs.Key()); err != nil {
		logger.Log.Error().Err(err).Msg("failed to invalidate mystery leaderboard caches after write")
	}
}

func (r *mysteryRepository) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*MysteryRow, error) {
	return r.dao.GetByID(ctx, id, tx...)
}

func (r *mysteryRepository) List(ctx context.Context, sort string, solved *bool, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]MysteryRow, int, error) {
	return r.dao.List(ctx, sort, solved, limit, offset, excludeUserIDs, tx...)
}

func (r *mysteryRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]MysteryRow, int, error) {
	return r.dao.ListByUser(ctx, userID, limit, offset, tx...)
}

func (r *mysteryRepository) GetClues(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]dto.MysteryClue, error) {
	return r.dao.GetClues(ctx, mysteryID, tx...)
}

func (r *mysteryRepository) DeleteClues(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteClues(ctx, mysteryID, tx...)
}

func (r *mysteryRepository) DeleteClue(ctx context.Context, clueID int, tx ...*sql.Tx) error {
	return r.dao.DeleteClue(ctx, clueID, tx...)
}

func (r *mysteryRepository) UpdateClue(ctx context.Context, clueID int, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateClue(ctx, clueID, body, tx...)
}

func (r *mysteryRepository) GetAuthorID(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetAuthorID(ctx, mysteryID, tx...)
}

func (r *mysteryRepository) CreateAttempt(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, parentID *uuid.UUID, body string, tx ...*sql.Tx) (*MysteryAttemptRow, error) {
	return r.dao.CreateAttempt(ctx, mysteryID, userID, parentID, body, tx...)
}

func (r *mysteryRepository) DeleteAttempt(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	if err := r.dao.DeleteAttempt(ctx, id, userID, tx...); err != nil {
		return err
	}

	r.invalidateLeaderboards(ctx)

	return nil
}

func (r *mysteryRepository) DeleteAttemptAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	if err := r.dao.DeleteAttemptAsAdmin(ctx, id, tx...); err != nil {
		return err
	}

	r.invalidateLeaderboards(ctx)

	return nil
}

func (r *mysteryRepository) GetAttempts(ctx context.Context, mysteryID uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) ([]MysteryAttemptRow, error) {
	return r.dao.GetAttempts(ctx, mysteryID, viewerID, tx...)
}

func (r *mysteryRepository) GetAttemptAuthorID(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetAttemptAuthorID(ctx, attemptID, tx...)
}

func (r *mysteryRepository) GetAttemptMysteryID(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetAttemptMysteryID(ctx, attemptID, tx...)
}

func (r *mysteryRepository) GetAttemptOwner(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, uuid.UUID, error) {
	return r.dao.GetAttemptOwner(ctx, attemptID, tx...)
}

func (r *mysteryRepository) SetMysteryWinner(ctx context.Context, mysteryID uuid.UUID, winnerID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.SetMysteryWinner(ctx, mysteryID, winnerID, tx...)
}

func (r *mysteryRepository) SetAttemptWinner(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.SetAttemptWinner(ctx, attemptID, tx...)
}

func (r *mysteryRepository) VoteAttempt(ctx context.Context, userID uuid.UUID, attemptID uuid.UUID, value int, tx ...*sql.Tx) error {
	return r.dao.VoteAttempt(ctx, userID, attemptID, value, tx...)
}

func (r *mysteryRepository) MarkSolved(ctx context.Context, mysteryID uuid.UUID, attemptID uuid.UUID, lockMystery bool, tx ...*sql.Tx) error {
	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		attemptUserID, attemptMysteryID, err := r.dao.GetAttemptOwner(ctx, attemptID, tx)
		if err != nil {
			return err
		}

		if attemptMysteryID != mysteryID {
			return fmt.Errorf("attempt does not belong to mystery")
		}

		if lockMystery {
			if err := r.dao.SetMysteryWinner(ctx, mysteryID, attemptUserID, tx); err != nil {
				return err
			}
		}

		return r.dao.SetAttemptWinner(ctx, attemptID, tx)
	})
	if err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.MysteryTopDetectives.Key(), cache.MysteryTopGMs.Key())
}

func (r *mysteryRepository) MarkPermanentlySolved(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) error {
	if err := r.dao.MarkPermanentlySolved(ctx, mysteryID, tx...); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.MysteryTopDetectives.Key(), cache.MysteryTopGMs.Key())
}

func (r *mysteryRepository) UserHasWinningAttempt(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.UserHasWinningAttempt(ctx, mysteryID, userID, tx...)
}

func (r *mysteryRepository) GetSolverIDs(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	return r.dao.GetSolverIDs(ctx, mysteryID, tx...)
}

func (r *mysteryRepository) IsSolved(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.IsSolved(ctx, mysteryID, tx...)
}

func (r *mysteryRepository) IsPaused(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.IsPaused(ctx, mysteryID, tx...)
}

func (r *mysteryRepository) SetPaused(ctx context.Context, mysteryID uuid.UUID, paused bool, tx ...*sql.Tx) error {
	return r.dao.SetPaused(ctx, mysteryID, paused, tx...)
}

func (r *mysteryRepository) SetGmAway(ctx context.Context, mysteryID uuid.UUID, away bool, tx ...*sql.Tx) error {
	return r.dao.SetGmAway(ctx, mysteryID, away, tx...)
}

func (r *mysteryRepository) GetLeaderboard(ctx context.Context, limit int, tx ...*sql.Tx) ([]LeaderboardEntry, error) {
	return r.dao.GetLeaderboard(ctx, limit, tx...)
}

func (r *mysteryRepository) GetTopDetectiveIDs(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	key := cache.MysteryTopDetectives.Key()

	if v, err := cache.Get[[]string](ctx, r.cache, key); err == nil {
		return v, nil
	}

	v, err := r.dao.GetTopDetectiveIDs(ctx, tx...)
	if err != nil {
		return nil, err
	}

	_ = cache.Set(ctx, r.cache, key, v, cache.MysteryTopDetectives.TTL)
	return v, nil
}

func (r *mysteryRepository) GetGMLeaderboard(ctx context.Context, limit int, tx ...*sql.Tx) ([]GMLeaderboardEntry, error) {
	return r.dao.GetGMLeaderboard(ctx, limit, tx...)
}

func (r *mysteryRepository) GetTopGMIDs(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	key := cache.MysteryTopGMs.Key()

	if v, err := cache.Get[[]string](ctx, r.cache, key); err == nil {
		return v, nil
	}

	v, err := r.dao.GetTopGMIDs(ctx, tx...)
	if err != nil {
		return nil, err
	}

	_ = cache.Set(ctx, r.cache, key, v, cache.MysteryTopGMs.TTL)
	return v, nil
}

func (r *mysteryRepository) CountAttempts(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.CountAttempts(ctx, mysteryID, tx...)
}

func (r *mysteryRepository) CountClues(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.CountClues(ctx, mysteryID, tx...)
}

func (r *mysteryRepository) GetPlayerIDs(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	return r.dao.GetPlayerIDs(ctx, mysteryID, tx...)
}

func (r *mysteryRepository) CreateComment(ctx context.Context, mysteryID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error) {
	return r.dao.CreateComment(ctx, mysteryID, parentID, userID, body, tx...)
}

func (r *mysteryRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateComment(ctx, id, userID, body, tx...)
}

func (r *mysteryRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body, tx...)
}

func (r *mysteryRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteComment(ctx, id, userID, tx...)
}

func (r *mysteryRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id, tx...)
}

func (r *mysteryRepository) DeleteCommentWithAudit(ctx context.Context, spec MysteryCommentDelete, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		authorID, err := r.dao.GetCommentAuthorID(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		collected, err := r.dao.CollectSingleCommentMediaPaths(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		paths = dedupePaths(collected)

		if spec.AsAdmin {
			if err := r.dao.DeleteCommentAsAdmin(ctx, spec.ID, tx); err != nil {
				return err
			}
		} else {
			if err := r.dao.DeleteComment(ctx, spec.ID, spec.UserID, tx); err != nil {
				return err
			}
		}

		action := AuditActionMysteryCommentDelete
		if authorID != spec.UserID {
			action = AuditActionMysteryCommentDeleteAdmin
		}

		if err := r.audit.Create(ctx, NewAuditEntry{
			ActorID:    spec.UserID,
			Action:     action,
			TargetType: AuditTargetMysteryComment,
			TargetID:   spec.ID.String(),
			SubjectID:  authorID,
		}, tx); err != nil {
			return fmt.Errorf("audit comment delete: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *mysteryRepository) GetComments(ctx context.Context, mysteryID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, mysteryID, viewerID, limit, offset, excludeUserIDs, tx...)
}

func (r *mysteryRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentEntityID(ctx, commentID, tx...)
}

func (r *mysteryRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID, tx...)
}

func (r *mysteryRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.LikeComment(ctx, userID, commentID, tx...)
}

func (r *mysteryRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.UnlikeComment(ctx, userID, commentID, tx...)
}

func (r *mysteryRepository) AddCommentMedia(ctx context.Context, spec NewMysteryCommentMedia, tx ...*sql.Tx) (int64, error) {
	return r.dao.AddCommentMedia(ctx, spec, tx...)
}

func (r *mysteryRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL, tx...)
}

func (r *mysteryRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL, tx...)
}

func (r *mysteryRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error) {
	return r.dao.GetCommentMedia(ctx, commentID, tx...)
}

func (r *mysteryRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs, tx...)
}

func (r *mysteryRepository) AddAttachment(ctx context.Context, mysteryID uuid.UUID, fileURL string, fileName string, fileSize int, tx ...*sql.Tx) (int64, error) {
	return r.dao.AddAttachment(ctx, mysteryID, fileURL, fileName, fileSize, tx...)
}

func (r *mysteryRepository) DeleteAttachment(ctx context.Context, id int64, mysteryID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteAttachment(ctx, id, mysteryID, tx...)
}

func (r *mysteryRepository) GetAttachments(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]dto.MysteryAttachment, error) {
	return r.dao.GetAttachments(ctx, mysteryID, tx...)
}

func (r *mysteryRepository) AddMedia(ctx context.Context, spec NewMysteryMedia, tx ...*sql.Tx) (int64, error) {
	return r.dao.AddMedia(ctx, spec, tx...)
}

func (r *mysteryRepository) UpdateMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateMediaURL(ctx, id, mediaURL, tx...)
}

func (r *mysteryRepository) UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateMediaThumbnail(ctx, id, thumbnailURL, tx...)
}

func (r *mysteryRepository) GetMedia(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error) {
	return r.dao.GetMedia(ctx, mysteryID, tx...)
}

func (r *mysteryRepository) DeleteMedia(ctx context.Context, id int64, mysteryID uuid.UUID, tx ...*sql.Tx) (string, error) {
	return r.dao.DeleteMedia(ctx, id, mysteryID, tx...)
}

func (r *mysteryRepository) GetAttachmentPaths(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.GetAttachmentPaths(ctx, mysteryID, tx...)
}

func (r *mysteryRepository) CollectMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectMediaPaths(ctx, entityID, tx...)
}

func (r *mysteryRepository) CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectCommentMediaPaths(ctx, entityID, tx...)
}

func (r *mysteryRepository) CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectSingleCommentMediaPaths(ctx, commentID, tx...)
}
