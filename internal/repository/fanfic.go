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
		GetComments(ctx context.Context, fanficID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error)
	}
)

type fanficRepository struct {
	dao FanficRepository
}

func NewFanficRepo(dao FanficRepository) FanficRepository {
	return &fanficRepository{dao: dao}
}

func (r *fanficRepository) CreateWithDetails(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, summary string, series string, rating string, language string, status string, isOneshot bool, containsLemons bool, genres []string, tags []string, characters []dto.FanficCharacter, isPairing bool) error {
	return r.dao.CreateWithDetails(ctx, id, userID, title, summary, series, rating, language, status, isOneshot, containsLemons, genres, tags, characters, isPairing)
}

func (r *fanficRepository) UpdateWithDetails(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, summary string, series string, rating string, language string, status string, isOneshot bool, containsLemons bool, genres []string, tags []string, characters []dto.FanficCharacter, isPairing bool, asAdmin bool) error {
	return r.dao.UpdateWithDetails(ctx, id, userID, title, summary, series, rating, language, status, isOneshot, containsLemons, genres, tags, characters, isPairing, asAdmin)
}

func (r *fanficRepository) UpdateCoverImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string) error {
	return r.dao.UpdateCoverImage(ctx, id, imageURL, thumbnailURL)
}

func (r *fanficRepository) UpdateWordCount(ctx context.Context, fanficID uuid.UUID) error {
	return r.dao.UpdateWordCount(ctx, fanficID)
}

func (r *fanficRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.Delete(ctx, id, userID)
}

func (r *fanficRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteAsAdmin(ctx, id)
}

func (r *fanficRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.FanficRow, error) {
	return r.dao.GetByID(ctx, id, viewerID)
}

func (r *fanficRepository) GetAuthorID(ctx context.Context, fanficID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetAuthorID(ctx, fanficID)
}

func (r *fanficRepository) List(ctx context.Context, viewerID uuid.UUID, params fanficparams.ListParams, excludeUserIDs []uuid.UUID) ([]model.FanficRow, int, error) {
	return r.dao.List(ctx, viewerID, params, excludeUserIDs)
}

func (r *fanficRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit int, offset int) ([]model.FanficRow, int, error) {
	return r.dao.ListByUser(ctx, userID, viewerID, limit, offset)
}

func (r *fanficRepository) CreateChapter(ctx context.Context, id uuid.UUID, fanficID uuid.UUID, chapterNumber int, title string, body string, wordCount int) error {
	return r.dao.CreateChapter(ctx, id, fanficID, chapterNumber, title, body, wordCount)
}

func (r *fanficRepository) UpdateChapter(ctx context.Context, id uuid.UUID, title string, body string, wordCount int) error {
	return r.dao.UpdateChapter(ctx, id, title, body, wordCount)
}

func (r *fanficRepository) DeleteChapter(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteChapter(ctx, id)
}

func (r *fanficRepository) GetChapter(ctx context.Context, fanficID uuid.UUID, chapterNumber int) (*model.FanficChapterRow, error) {
	return r.dao.GetChapter(ctx, fanficID, chapterNumber)
}

func (r *fanficRepository) ListChapters(ctx context.Context, fanficID uuid.UUID) ([]model.FanficChapterSummaryRow, error) {
	return r.dao.ListChapters(ctx, fanficID)
}

func (r *fanficRepository) GetChapterCount(ctx context.Context, fanficID uuid.UUID) (int, error) {
	return r.dao.GetChapterCount(ctx, fanficID)
}

func (r *fanficRepository) GetNextChapterNumber(ctx context.Context, fanficID uuid.UUID) (int, error) {
	return r.dao.GetNextChapterNumber(ctx, fanficID)
}

func (r *fanficRepository) GetChapterFanficID(ctx context.Context, chapterID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetChapterFanficID(ctx, chapterID)
}

func (r *fanficRepository) GetChapterAuthorID(ctx context.Context, chapterID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetChapterAuthorID(ctx, chapterID)
}

func (r *fanficRepository) GetGenres(ctx context.Context, fanficID uuid.UUID) ([]string, error) {
	return r.dao.GetGenres(ctx, fanficID)
}

func (r *fanficRepository) GetGenresBatch(ctx context.Context, fanficIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	return r.dao.GetGenresBatch(ctx, fanficIDs)
}

func (r *fanficRepository) GetTags(ctx context.Context, fanficID uuid.UUID) ([]string, error) {
	return r.dao.GetTags(ctx, fanficID)
}

func (r *fanficRepository) GetTagsBatch(ctx context.Context, fanficIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	return r.dao.GetTagsBatch(ctx, fanficIDs)
}

func (r *fanficRepository) GetCharacters(ctx context.Context, fanficID uuid.UUID) ([]model.FanficCharacterRow, error) {
	return r.dao.GetCharacters(ctx, fanficID)
}

func (r *fanficRepository) GetCharactersBatch(ctx context.Context, fanficIDs []uuid.UUID) (map[uuid.UUID][]model.FanficCharacterRow, error) {
	return r.dao.GetCharactersBatch(ctx, fanficIDs)
}

func (r *fanficRepository) RegisterOCCharacter(ctx context.Context, name string, creatorID uuid.UUID) error {
	return r.dao.RegisterOCCharacter(ctx, name, creatorID)
}

func (r *fanficRepository) SearchOCCharacters(ctx context.Context, query string) ([]string, error) {
	return r.dao.SearchOCCharacters(ctx, query)
}

func (r *fanficRepository) GetLanguages(ctx context.Context) ([]string, error) {
	return r.dao.GetLanguages(ctx)
}

func (r *fanficRepository) RegisterLanguage(ctx context.Context, name string) error {
	return r.dao.RegisterLanguage(ctx, name)
}

func (r *fanficRepository) GetSeries(ctx context.Context) ([]string, error) {
	return r.dao.GetSeries(ctx)
}

func (r *fanficRepository) RegisterSeries(ctx context.Context, name string) error {
	return r.dao.RegisterSeries(ctx, name)
}

func (r *fanficRepository) Favourite(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID) error {
	return r.dao.Favourite(ctx, userID, fanficID)
}

func (r *fanficRepository) Unfavourite(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID) error {
	return r.dao.Unfavourite(ctx, userID, fanficID)
}

func (r *fanficRepository) RecordView(ctx context.Context, fanficID uuid.UUID, viewerHash string) (bool, error) {
	return r.dao.RecordView(ctx, fanficID, viewerHash)
}

func (r *fanficRepository) GetReadingProgress(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID) (int, error) {
	return r.dao.GetReadingProgress(ctx, userID, fanficID)
}

func (r *fanficRepository) SetReadingProgress(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID, chapterNumber int) error {
	return r.dao.SetReadingProgress(ctx, userID, fanficID, chapterNumber)
}

func (r *fanficRepository) ListFavourites(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.FanficRow, int, error) {
	return r.dao.ListFavourites(ctx, userID, viewerID, limit, offset)
}

func (r *fanficRepository) CreateComment(ctx context.Context, id uuid.UUID, fanficID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.CreateComment(ctx, id, fanficID, parentID, userID, body)
}

func (r *fanficRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.UpdateComment(ctx, id, userID, body)
}

func (r *fanficRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body)
}

func (r *fanficRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.DeleteComment(ctx, id, userID)
}

func (r *fanficRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id)
}

func (r *fanficRepository) GetComments(ctx context.Context, fanficID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, fanficID, viewerID, limit, offset, excludeUserIDs)
}

func (r *fanficRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentEntityID(ctx, commentID)
}

func (r *fanficRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID)
}

func (r *fanficRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.LikeComment(ctx, userID, commentID)
}

func (r *fanficRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.UnlikeComment(ctx, userID, commentID)
}

func (r *fanficRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	return r.dao.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbnailURL, sortOrder)
}

func (r *fanficRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL)
}

func (r *fanficRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL)
}

func (r *fanficRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error) {
	return r.dao.GetCommentMedia(ctx, commentID)
}

func (r *fanficRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs)
}
