package repository

import (
	"context"

	"umineko_city_of_books/internal/dto"
	fanficparams "umineko_city_of_books/internal/fanfic/params"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type (
	FanficRepository interface {
		CreateWithDetails(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, summary string, series string, rating string, language string, status string, isOneshot bool, containsLemons bool, genres []string, tags []string, characters []dto.FanficCharacter, isPairing bool) error
		UpdateWithDetails(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, summary string, series string, rating string, language string, status string, isOneshot bool, containsLemons bool, genres []string, tags []string, characters []dto.FanficCharacter, isPairing bool, asAdmin bool) error
		UpdateCoverImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string) error
		UpdateWordCount(ctx context.Context, fanficID uuid.UUID) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.FanficRow, error)
		GetAuthorID(ctx context.Context, fanficID uuid.UUID) (uuid.UUID, error)

		List(ctx context.Context, viewerID uuid.UUID, params fanficparams.ListParams, excludeUserIDs []uuid.UUID) ([]model.FanficRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit int, offset int) ([]model.FanficRow, int, error)

		CreateChapter(ctx context.Context, id uuid.UUID, fanficID uuid.UUID, chapterNumber int, title string, body string, wordCount int) error
		UpdateChapter(ctx context.Context, id uuid.UUID, title string, body string, wordCount int) error
		DeleteChapter(ctx context.Context, id uuid.UUID) error
		GetChapter(ctx context.Context, fanficID uuid.UUID, chapterNumber int) (*model.FanficChapterRow, error)
		ListChapters(ctx context.Context, fanficID uuid.UUID) ([]model.FanficChapterSummaryRow, error)
		GetChapterCount(ctx context.Context, fanficID uuid.UUID) (int, error)
		GetNextChapterNumber(ctx context.Context, fanficID uuid.UUID) (int, error)
		GetChapterFanficID(ctx context.Context, chapterID uuid.UUID) (uuid.UUID, error)
		GetChapterAuthorID(ctx context.Context, chapterID uuid.UUID) (uuid.UUID, error)

		GetGenres(ctx context.Context, fanficID uuid.UUID) ([]string, error)
		GetGenresBatch(ctx context.Context, fanficIDs []uuid.UUID) (map[uuid.UUID][]string, error)
		GetTags(ctx context.Context, fanficID uuid.UUID) ([]string, error)
		GetTagsBatch(ctx context.Context, fanficIDs []uuid.UUID) (map[uuid.UUID][]string, error)
		GetCharacters(ctx context.Context, fanficID uuid.UUID) ([]model.FanficCharacterRow, error)
		GetCharactersBatch(ctx context.Context, fanficIDs []uuid.UUID) (map[uuid.UUID][]model.FanficCharacterRow, error)

		RegisterOCCharacter(ctx context.Context, name string, creatorID uuid.UUID) error
		SearchOCCharacters(ctx context.Context, query string) ([]string, error)
		GetLanguages(ctx context.Context) ([]string, error)
		RegisterLanguage(ctx context.Context, name string) error
		GetSeries(ctx context.Context) ([]string, error)
		RegisterSeries(ctx context.Context, name string) error

		Favourite(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID) error
		Unfavourite(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID) error
		RecordView(ctx context.Context, fanficID uuid.UUID, viewerHash string) (bool, error)
		GetReadingProgress(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID) (int, error)
		SetReadingProgress(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID, chapterNumber int) error
		ListFavourites(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.FanficRow, int, error)

		CreateComment(ctx context.Context, id uuid.UUID, fanficID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, fanficID uuid.UUID, viewerID uuid.UUID, excludeUserIDs []uuid.UUID) ([]model.FanficCommentRow, error)
		GetCommentFanficID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.FanficCommentMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.FanficCommentMediaRow, error)
	}
)
