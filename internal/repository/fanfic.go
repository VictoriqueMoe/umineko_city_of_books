package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/dto"
	fanficparams "umineko_city_of_books/internal/fanfic/params"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type (
	FanficDAO interface {
		Create(ctx context.Context, userID uuid.UUID, title string, summary string, series string, rating string, language string, status string, isOneshot bool, containsLemons bool, isPairing bool, tx ...*sql.Tx) (*model.FanficRow, error)
		Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, summary string, series string, rating string, language string, status string, isOneshot bool, containsLemons bool, asAdmin bool, tx ...*sql.Tx) error
		UpdateCoverImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string, tx ...*sql.Tx) error
		UpdateWordCount(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*model.FanficRow, error)
		GetAuthorID(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCoverImagePaths(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]string, error)

		List(ctx context.Context, viewerID uuid.UUID, params fanficparams.ListParams, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.FanficRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit int, offset int, tx ...*sql.Tx) ([]model.FanficRow, int, error)

		CreateChapter(ctx context.Context, fanficID uuid.UUID, spec NewChapter, tx ...*sql.Tx) (*model.FanficChapterRow, error)
		UpdateChapter(ctx context.Context, id uuid.UUID, title string, body string, wordCount int, tx ...*sql.Tx) error
		DeleteChapter(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetChapter(ctx context.Context, fanficID uuid.UUID, chapterNumber int, tx ...*sql.Tx) (*model.FanficChapterRow, error)
		ListChapters(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]model.FanficChapterSummaryRow, error)
		GetChapterCount(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetNextChapterNumber(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetChapterFanficID(ctx context.Context, chapterID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetChapterAuthorID(ctx context.Context, chapterID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)

		GetGenres(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		GetGenresBatch(ctx context.Context, fanficIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error)
		AddGenres(ctx context.Context, fanficID uuid.UUID, genres []string, tx ...*sql.Tx) error
		DeleteGenres(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error
		GetTags(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		GetTagsBatch(ctx context.Context, fanficIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error)
		AddTags(ctx context.Context, fanficID uuid.UUID, tags []string, tx ...*sql.Tx) error
		DeleteTags(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error
		GetCharacters(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]model.FanficCharacterRow, error)
		GetCharactersBatch(ctx context.Context, fanficIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.FanficCharacterRow, error)
		AddCharacters(ctx context.Context, fanficID uuid.UUID, characters []dto.FanficCharacter, isPairing bool, tx ...*sql.Tx) error
		DeleteCharacters(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error

		RegisterOCCharacter(ctx context.Context, name string, creatorID uuid.UUID, tx ...*sql.Tx) error
		SearchOCCharacters(ctx context.Context, query string, tx ...*sql.Tx) ([]string, error)
		GetLanguages(ctx context.Context, tx ...*sql.Tx) ([]string, error)
		RegisterLanguage(ctx context.Context, name string, tx ...*sql.Tx) error
		GetSeries(ctx context.Context, tx ...*sql.Tx) ([]string, error)
		RegisterSeries(ctx context.Context, name string, tx ...*sql.Tx) error

		Favourite(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID, tx ...*sql.Tx) error
		Unfavourite(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID, tx ...*sql.Tx) error
		RecordView(ctx context.Context, fanficID uuid.UUID, viewerHash string, tx ...*sql.Tx) (bool, error)
		GetReadingProgress(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID, tx ...*sql.Tx) (int, error)
		SetReadingProgress(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID, chapterNumber int, tx ...*sql.Tx) error
		ListFavourites(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.FanficRow, int, error)

		CreateComment(ctx context.Context, fanficID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetComments(ctx context.Context, fanficID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error
		AddCommentMedia(ctx context.Context, spec NewFanficCommentMedia, tx ...*sql.Tx) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)
	}

	FanficRepository interface {
		FanficDAO

		CreateWithDetails(ctx context.Context, spec NewFanfic, tx ...*sql.Tx) (*model.FanficRow, error)
		UpdateWithDetails(ctx context.Context, spec FanficUpdate, tx ...*sql.Tx) error
		DeleteFanfic(ctx context.Context, spec FanficDelete, tx ...*sql.Tx) ([]string, error)

		CreateChapterWithCount(ctx context.Context, fanficID uuid.UUID, spec NewChapter, tx ...*sql.Tx) (*model.FanficChapterRow, error)
		UpdateChapterWithCount(ctx context.Context, spec ChapterUpdate, tx ...*sql.Tx) error
		DeleteChapterWithCount(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error

		UpdateCommentBody(ctx context.Context, spec FanficCommentUpdate, tx ...*sql.Tx) error
		DeleteCommentWithAudit(ctx context.Context, spec FanficCommentDelete, tx ...*sql.Tx) ([]string, error)
	}

	NewFanfic struct {
		UserID         uuid.UUID
		Title          string
		Summary        string
		Series         string
		Rating         string
		Language       string
		Status         string
		IsOneshot      bool
		ContainsLemons bool
		IsPairing      bool
		Genres         []string
		Tags           []string
		Characters     []dto.FanficCharacter
		FirstChapter   *NewChapter
	}

	FanficUpdate struct {
		ID             uuid.UUID
		UserID         uuid.UUID
		Title          string
		Summary        string
		Series         string
		Rating         string
		Language       string
		Status         string
		IsOneshot      bool
		ContainsLemons bool
		IsPairing      bool
		AsAdmin        bool
		Genres         []string
		Tags           []string
		Characters     []dto.FanficCharacter
	}

	NewChapter struct {
		Number    int
		Title     string
		Body      string
		WordCount int
	}

	ChapterUpdate struct {
		ID        uuid.UUID
		Title     string
		Body      string
		WordCount int
	}

	FanficDelete struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		AsAdmin bool
	}

	FanficCommentUpdate struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		Body    string
		AsAdmin bool
	}

	FanficCommentDelete struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		AsAdmin bool
		Audit   NewAuditEntry
	}

	NewFanficCommentMedia struct {
		CommentID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
	}
)

type fanficRepository struct {
	db    *sql.DB
	dao   FanficDAO
	audit AuditLogRepository
}

func NewFanficRepo(database *sql.DB, dao FanficDAO, audit AuditLogRepository) FanficRepository {
	return &fanficRepository{db: database, dao: dao, audit: audit}
}

func (r *fanficRepository) CreateWithDetails(ctx context.Context, spec NewFanfic, tx ...*sql.Tx) (*model.FanficRow, error) {
	var created *model.FanficRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.RegisterSeries(ctx, spec.Series, tx); err != nil {
			return err
		}

		if err := r.dao.RegisterLanguage(ctx, spec.Language, tx); err != nil {
			return err
		}

		var err error

		created, err = r.dao.Create(ctx, spec.UserID, spec.Title, spec.Summary, spec.Series, spec.Rating, spec.Language, spec.Status, spec.IsOneshot, spec.ContainsLemons, spec.IsPairing, tx)
		if err != nil {
			return err
		}

		if err := r.dao.AddGenres(ctx, created.ID, spec.Genres, tx); err != nil {
			return err
		}

		if err := r.dao.AddTags(ctx, created.ID, spec.Tags, tx); err != nil {
			return err
		}

		if err := r.dao.AddCharacters(ctx, created.ID, spec.Characters, spec.IsPairing, tx); err != nil {
			return err
		}

		if err := r.registerOCCharacters(ctx, spec.UserID, spec.Characters, tx); err != nil {
			return err
		}

		if spec.FirstChapter == nil {
			return nil
		}

		if _, err := r.dao.CreateChapter(ctx, created.ID, *spec.FirstChapter, tx); err != nil {
			return err
		}

		return r.dao.UpdateWordCount(ctx, created.ID, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *fanficRepository) registerOCCharacters(ctx context.Context, userID uuid.UUID, characters []dto.FanficCharacter, tx *sql.Tx) error {
	for i := range characters {
		name := strings.TrimSpace(characters[i].CharacterName)
		if characters[i].CharacterID != "" || name == "" {
			continue
		}

		if err := r.dao.RegisterOCCharacter(ctx, name, userID, tx); err != nil {
			return err
		}
	}

	return nil
}

func (r *fanficRepository) UpdateWithDetails(ctx context.Context, spec FanficUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.RegisterSeries(ctx, spec.Series, tx); err != nil {
			return err
		}

		if err := r.dao.RegisterLanguage(ctx, spec.Language, tx); err != nil {
			return err
		}

		if err := r.dao.Update(ctx, spec.ID, spec.UserID, spec.Title, spec.Summary, spec.Series, spec.Rating, spec.Language, spec.Status, spec.IsOneshot, spec.ContainsLemons, spec.AsAdmin, tx); err != nil {
			return err
		}

		if err := r.dao.DeleteGenres(ctx, spec.ID, tx); err != nil {
			return err
		}

		if err := r.dao.DeleteTags(ctx, spec.ID, tx); err != nil {
			return err
		}

		if err := r.dao.DeleteCharacters(ctx, spec.ID, tx); err != nil {
			return err
		}

		if err := r.dao.AddGenres(ctx, spec.ID, spec.Genres, tx); err != nil {
			return err
		}

		if err := r.dao.AddTags(ctx, spec.ID, spec.Tags, tx); err != nil {
			return err
		}

		if err := r.dao.AddCharacters(ctx, spec.ID, spec.Characters, spec.IsPairing, tx); err != nil {
			return err
		}

		return r.registerOCCharacters(ctx, spec.UserID, spec.Characters, tx)
	})
}

func (r *fanficRepository) DeleteFanfic(ctx context.Context, spec FanficDelete, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		coverPaths, err := r.dao.GetCoverImagePaths(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		commentPaths, err := r.dao.CollectCommentMediaPaths(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		collected := append(coverPaths, commentPaths...)

		if spec.AsAdmin {
			err = r.dao.DeleteAsAdmin(ctx, spec.ID, tx)
		} else {
			err = r.dao.Delete(ctx, spec.ID, spec.UserID, tx)
		}
		if err != nil {
			return err
		}

		paths = dedupePaths(collected)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *fanficRepository) CreateChapterWithCount(ctx context.Context, fanficID uuid.UUID, spec NewChapter, tx ...*sql.Tx) (*model.FanficChapterRow, error) {
	var created *model.FanficChapterRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.dao.CreateChapter(ctx, fanficID, spec, tx)
		if err != nil {
			return err
		}

		return r.dao.UpdateWordCount(ctx, fanficID, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *fanficRepository) UpdateChapterWithCount(ctx context.Context, spec ChapterUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.UpdateChapter(ctx, spec.ID, spec.Title, spec.Body, spec.WordCount, tx); err != nil {
			return err
		}

		fanficID, err := r.dao.GetChapterFanficID(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		return r.dao.UpdateWordCount(ctx, fanficID, tx)
	})
}

func (r *fanficRepository) DeleteChapterWithCount(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		fanficID, err := r.dao.GetChapterFanficID(ctx, id, tx)
		if err != nil {
			return err
		}

		if err := r.dao.DeleteChapter(ctx, id, tx); err != nil {
			return err
		}

		return r.dao.UpdateWordCount(ctx, fanficID, tx)
	})
}

func (r *fanficRepository) UpdateCommentBody(ctx context.Context, spec FanficCommentUpdate, tx ...*sql.Tx) error {
	if spec.AsAdmin {
		return r.dao.UpdateCommentAsAdmin(ctx, spec.ID, spec.Body, tx...)
	}

	return r.dao.UpdateComment(ctx, spec.ID, spec.UserID, spec.Body, tx...)
}

func (r *fanficRepository) DeleteCommentWithAudit(ctx context.Context, spec FanficCommentDelete, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		mediaPaths, err := r.dao.CollectSingleCommentMediaPaths(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		if spec.AsAdmin {
			err = r.dao.DeleteCommentAsAdmin(ctx, spec.ID, tx)
		} else {
			err = r.dao.DeleteComment(ctx, spec.ID, spec.UserID, tx)
		}
		if err != nil {
			return err
		}

		if err := r.audit.Create(ctx, spec.Audit, tx); err != nil {
			return fmt.Errorf("audit comment delete: %w", err)
		}

		paths = mediaPaths

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *fanficRepository) Create(ctx context.Context, userID uuid.UUID, title string, summary string, series string, rating string, language string, status string, isOneshot bool, containsLemons bool, isPairing bool, tx ...*sql.Tx) (*model.FanficRow, error) {
	return r.dao.Create(ctx, userID, title, summary, series, rating, language, status, isOneshot, containsLemons, isPairing, tx...)
}

func (r *fanficRepository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, summary string, series string, rating string, language string, status string, isOneshot bool, containsLemons bool, asAdmin bool, tx ...*sql.Tx) error {
	return r.dao.Update(ctx, id, userID, title, summary, series, rating, language, status, isOneshot, containsLemons, asAdmin, tx...)
}

func (r *fanficRepository) UpdateCoverImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCoverImage(ctx, id, imageURL, thumbnailURL, tx...)
}

func (r *fanficRepository) UpdateWordCount(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.UpdateWordCount(ctx, fanficID, tx...)
}

func (r *fanficRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Delete(ctx, id, userID, tx...)
}

func (r *fanficRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteAsAdmin(ctx, id, tx...)
}

func (r *fanficRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*model.FanficRow, error) {
	return r.dao.GetByID(ctx, id, viewerID, tx...)
}

func (r *fanficRepository) GetAuthorID(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetAuthorID(ctx, fanficID, tx...)
}

func (r *fanficRepository) GetCoverImagePaths(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.GetCoverImagePaths(ctx, fanficID, tx...)
}

func (r *fanficRepository) List(ctx context.Context, viewerID uuid.UUID, params fanficparams.ListParams, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.FanficRow, int, error) {
	return r.dao.List(ctx, viewerID, params, excludeUserIDs, tx...)
}

func (r *fanficRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit int, offset int, tx ...*sql.Tx) ([]model.FanficRow, int, error) {
	return r.dao.ListByUser(ctx, userID, viewerID, limit, offset, tx...)
}

func (r *fanficRepository) CreateChapter(ctx context.Context, fanficID uuid.UUID, spec NewChapter, tx ...*sql.Tx) (*model.FanficChapterRow, error) {
	return r.dao.CreateChapter(ctx, fanficID, spec, tx...)
}

func (r *fanficRepository) UpdateChapter(ctx context.Context, id uuid.UUID, title string, body string, wordCount int, tx ...*sql.Tx) error {
	return r.dao.UpdateChapter(ctx, id, title, body, wordCount, tx...)
}

func (r *fanficRepository) DeleteChapter(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteChapter(ctx, id, tx...)
}

func (r *fanficRepository) GetChapter(ctx context.Context, fanficID uuid.UUID, chapterNumber int, tx ...*sql.Tx) (*model.FanficChapterRow, error) {
	return r.dao.GetChapter(ctx, fanficID, chapterNumber, tx...)
}

func (r *fanficRepository) ListChapters(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]model.FanficChapterSummaryRow, error) {
	return r.dao.ListChapters(ctx, fanficID, tx...)
}

func (r *fanficRepository) GetChapterCount(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.GetChapterCount(ctx, fanficID, tx...)
}

func (r *fanficRepository) GetNextChapterNumber(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.GetNextChapterNumber(ctx, fanficID, tx...)
}

func (r *fanficRepository) GetChapterFanficID(ctx context.Context, chapterID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetChapterFanficID(ctx, chapterID, tx...)
}

func (r *fanficRepository) GetChapterAuthorID(ctx context.Context, chapterID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetChapterAuthorID(ctx, chapterID, tx...)
}

func (r *fanficRepository) GetGenres(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.GetGenres(ctx, fanficID, tx...)
}

func (r *fanficRepository) GetGenresBatch(ctx context.Context, fanficIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error) {
	return r.dao.GetGenresBatch(ctx, fanficIDs, tx...)
}

func (r *fanficRepository) AddGenres(ctx context.Context, fanficID uuid.UUID, genres []string, tx ...*sql.Tx) error {
	return r.dao.AddGenres(ctx, fanficID, genres, tx...)
}

func (r *fanficRepository) DeleteGenres(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteGenres(ctx, fanficID, tx...)
}

func (r *fanficRepository) GetTags(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.GetTags(ctx, fanficID, tx...)
}

func (r *fanficRepository) GetTagsBatch(ctx context.Context, fanficIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error) {
	return r.dao.GetTagsBatch(ctx, fanficIDs, tx...)
}

func (r *fanficRepository) AddTags(ctx context.Context, fanficID uuid.UUID, tags []string, tx ...*sql.Tx) error {
	return r.dao.AddTags(ctx, fanficID, tags, tx...)
}

func (r *fanficRepository) DeleteTags(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteTags(ctx, fanficID, tx...)
}

func (r *fanficRepository) GetCharacters(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]model.FanficCharacterRow, error) {
	return r.dao.GetCharacters(ctx, fanficID, tx...)
}

func (r *fanficRepository) GetCharactersBatch(ctx context.Context, fanficIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.FanficCharacterRow, error) {
	return r.dao.GetCharactersBatch(ctx, fanficIDs, tx...)
}

func (r *fanficRepository) AddCharacters(ctx context.Context, fanficID uuid.UUID, characters []dto.FanficCharacter, isPairing bool, tx ...*sql.Tx) error {
	return r.dao.AddCharacters(ctx, fanficID, characters, isPairing, tx...)
}

func (r *fanficRepository) DeleteCharacters(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteCharacters(ctx, fanficID, tx...)
}

func (r *fanficRepository) RegisterOCCharacter(ctx context.Context, name string, creatorID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.RegisterOCCharacter(ctx, name, creatorID, tx...)
}

func (r *fanficRepository) SearchOCCharacters(ctx context.Context, query string, tx ...*sql.Tx) ([]string, error) {
	return r.dao.SearchOCCharacters(ctx, query, tx...)
}

func (r *fanficRepository) GetLanguages(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	return r.dao.GetLanguages(ctx, tx...)
}

func (r *fanficRepository) RegisterLanguage(ctx context.Context, name string, tx ...*sql.Tx) error {
	return r.dao.RegisterLanguage(ctx, name, tx...)
}

func (r *fanficRepository) GetSeries(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	return r.dao.GetSeries(ctx, tx...)
}

func (r *fanficRepository) RegisterSeries(ctx context.Context, name string, tx ...*sql.Tx) error {
	return r.dao.RegisterSeries(ctx, name, tx...)
}

func (r *fanficRepository) Favourite(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Favourite(ctx, userID, fanficID, tx...)
}

func (r *fanficRepository) Unfavourite(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Unfavourite(ctx, userID, fanficID, tx...)
}

func (r *fanficRepository) RecordView(ctx context.Context, fanficID uuid.UUID, viewerHash string, tx ...*sql.Tx) (bool, error) {
	return r.dao.RecordView(ctx, fanficID, viewerHash, tx...)
}

func (r *fanficRepository) GetReadingProgress(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.GetReadingProgress(ctx, userID, fanficID, tx...)
}

func (r *fanficRepository) SetReadingProgress(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID, chapterNumber int, tx ...*sql.Tx) error {
	return r.dao.SetReadingProgress(ctx, userID, fanficID, chapterNumber, tx...)
}

func (r *fanficRepository) ListFavourites(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.FanficRow, int, error) {
	return r.dao.ListFavourites(ctx, userID, viewerID, limit, offset, tx...)
}

func (r *fanficRepository) CreateComment(ctx context.Context, fanficID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error) {
	return r.dao.CreateComment(ctx, fanficID, parentID, userID, body, tx...)
}

func (r *fanficRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateComment(ctx, id, userID, body, tx...)
}

func (r *fanficRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body, tx...)
}

func (r *fanficRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteComment(ctx, id, userID, tx...)
}

func (r *fanficRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id, tx...)
}

func (r *fanficRepository) GetComments(ctx context.Context, fanficID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, fanficID, viewerID, limit, offset, excludeUserIDs, tx...)
}

func (r *fanficRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentEntityID(ctx, commentID, tx...)
}

func (r *fanficRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID, tx...)
}

func (r *fanficRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.LikeComment(ctx, userID, commentID, tx...)
}

func (r *fanficRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.UnlikeComment(ctx, userID, commentID, tx...)
}

func (r *fanficRepository) AddCommentMedia(ctx context.Context, spec NewFanficCommentMedia, tx ...*sql.Tx) (int64, error) {
	return r.dao.AddCommentMedia(ctx, spec, tx...)
}

func (r *fanficRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL, tx...)
}

func (r *fanficRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL, tx...)
}

func (r *fanficRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error) {
	return r.dao.GetCommentMedia(ctx, commentID, tx...)
}

func (r *fanficRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs, tx...)
}

func (r *fanficRepository) CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectCommentMediaPaths(ctx, entityID, tx...)
}

func (r *fanficRepository) CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectSingleCommentMediaPaths(ctx, commentID, tx...)
}
