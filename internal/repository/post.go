package repository

import (
	"context"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type (
	PostRepository interface {
		Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, corner string, body string, sharedContentID *string, sharedContentType *string) error
		UpdatePost(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdatePostAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.PostRow, error)
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		ListAll(ctx context.Context, viewerID uuid.UUID, corner string, search string, sort string, seed int, limit, offset int, excludeUserIDs []uuid.UUID, resolvedFilter string) ([]model.PostRow, int, error)
		ListByFollowing(ctx context.Context, userID uuid.UUID, corner string, sort string, seed int, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.PostRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.PostRow, int, error)

		AddMedia(ctx context.Context, postID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		DeleteMedia(ctx context.Context, id int64, postID uuid.UUID) (string, error)
		UpdateMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetMedia(ctx context.Context, postID uuid.UUID) ([]model.PostMediaRow, error)
		GetMediaBatch(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error)

		Like(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
		Unlike(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
		GetLikedBy(ctx context.Context, postID uuid.UUID, excludeUserIDs []uuid.UUID) ([]model.PostLikeUser, error)
		RecordView(ctx context.Context, postID uuid.UUID, viewerHash string) (bool, error)
		GetPostAuthorID(ctx context.Context, postID uuid.UUID) (uuid.UUID, error)
		GetSharedContentAuthor(ctx context.Context, contentID string, contentType string) (uuid.UUID, error)

		ResolveSuggestion(ctx context.Context, postID uuid.UUID, resolvedBy uuid.UUID, status string) error
		UnresolveSuggestion(ctx context.Context, postID uuid.UUID) error

		CreateComment(ctx context.Context, id uuid.UUID, postID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, postID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error)

		CountUserPostsToday(ctx context.Context, userID uuid.UUID) (int, error)
		GetCornerCounts(ctx context.Context) (map[string]int, error)

		GetShareCount(ctx context.Context, contentID string, contentType string) (int, error)
		GetShareCountsBatch(ctx context.Context, contentIDs []string, contentType string) (map[string]int, error)
		IncrementShareCount(ctx context.Context, contentID string, contentType string) error
		DecrementShareCount(ctx context.Context, contentID string, contentType string) error
		GetSharedContentFields(ctx context.Context, postID uuid.UUID) (*string, *string, error)
		GetSharedContentPreviews(refs []SharedContentRef) map[string]*dto.SharedContentPreview

		CreatePollWithOptions(ctx context.Context, pollID uuid.UUID, postID uuid.UUID, durationSeconds int, expiresAt string, options []string) error
		GetPollByPostID(ctx context.Context, postID uuid.UUID, viewerID uuid.UUID) (*model.PollRow, []model.PollOptionRow, *int, error)
		GetPollsByPostIDs(ctx context.Context, postIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID]*model.PollRow, map[uuid.UUID][]model.PollOptionRow, map[uuid.UUID]*int, error)
		VotePoll(ctx context.Context, pollID uuid.UUID, userID uuid.UUID, optionID int) error

		AddEmbed(ctx context.Context, ownerID string, ownerType string, url string, embedType string, title string, description string, image string, siteName string, videoID string, sortOrder int) error
		DeleteEmbeds(ctx context.Context, ownerID string, ownerType string) error
		UpdateEmbed(ctx context.Context, id int, title string, description string, image string, siteName string) error
		GetEmbeds(ctx context.Context, ownerID string, ownerType string) ([]model.EmbedRow, error)
		GetEmbedsBatch(ctx context.Context, ownerIDs []string, ownerType string) (map[string][]model.EmbedRow, error)
		GetStaleEmbeds(ctx context.Context, olderThan string, limit int) ([]model.EmbedRow, error)
	}

	SharedContentRef struct {
		ID   string
		Type string
	}
)

type postRepository struct {
	dao PostRepository
}

func NewPostRepo(dao PostRepository) PostRepository {
	return &postRepository{dao: dao}
}

func (r *postRepository) Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, corner string, body string, sharedContentID *string, sharedContentType *string) error {
	return r.dao.Create(ctx, id, userID, corner, body, sharedContentID, sharedContentType)
}

func (r *postRepository) UpdatePost(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.UpdatePost(ctx, id, userID, body)
}

func (r *postRepository) UpdatePostAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	return r.dao.UpdatePostAsAdmin(ctx, id, body)
}

func (r *postRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.PostRow, error) {
	return r.dao.GetByID(ctx, id, viewerID)
}

func (r *postRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.Delete(ctx, id, userID)
}

func (r *postRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteAsAdmin(ctx, id)
}

func (r *postRepository) ListAll(ctx context.Context, viewerID uuid.UUID, corner string, search string, sort string, seed int, limit, offset int, excludeUserIDs []uuid.UUID, resolvedFilter string) ([]model.PostRow, int, error) {
	return r.dao.ListAll(ctx, viewerID, corner, search, sort, seed, limit, offset, excludeUserIDs, resolvedFilter)
}

func (r *postRepository) ListByFollowing(ctx context.Context, userID uuid.UUID, corner string, sort string, seed int, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.PostRow, int, error) {
	return r.dao.ListByFollowing(ctx, userID, corner, sort, seed, limit, offset, excludeUserIDs)
}

func (r *postRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.PostRow, int, error) {
	return r.dao.ListByUser(ctx, userID, viewerID, limit, offset)
}

func (r *postRepository) AddMedia(ctx context.Context, postID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	return r.dao.AddMedia(ctx, postID, mediaURL, mediaType, thumbnailURL, sortOrder)
}

func (r *postRepository) DeleteMedia(ctx context.Context, id int64, postID uuid.UUID) (string, error) {
	return r.dao.DeleteMedia(ctx, id, postID)
}

func (r *postRepository) UpdateMediaURL(ctx context.Context, id int64, mediaURL string) error {
	return r.dao.UpdateMediaURL(ctx, id, mediaURL)
}

func (r *postRepository) UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	return r.dao.UpdateMediaThumbnail(ctx, id, thumbnailURL)
}

func (r *postRepository) GetMedia(ctx context.Context, postID uuid.UUID) ([]model.PostMediaRow, error) {
	return r.dao.GetMedia(ctx, postID)
}

func (r *postRepository) GetMediaBatch(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetMediaBatch(ctx, postIDs)
}

func (r *postRepository) Like(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error {
	return r.dao.Like(ctx, userID, postID)
}

func (r *postRepository) Unlike(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error {
	return r.dao.Unlike(ctx, userID, postID)
}

func (r *postRepository) GetLikedBy(ctx context.Context, postID uuid.UUID, excludeUserIDs []uuid.UUID) ([]model.PostLikeUser, error) {
	return r.dao.GetLikedBy(ctx, postID, excludeUserIDs)
}

func (r *postRepository) RecordView(ctx context.Context, postID uuid.UUID, viewerHash string) (bool, error) {
	return r.dao.RecordView(ctx, postID, viewerHash)
}

func (r *postRepository) GetPostAuthorID(ctx context.Context, postID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetPostAuthorID(ctx, postID)
}

func (r *postRepository) GetSharedContentAuthor(ctx context.Context, contentID string, contentType string) (uuid.UUID, error) {
	return r.dao.GetSharedContentAuthor(ctx, contentID, contentType)
}

func (r *postRepository) ResolveSuggestion(ctx context.Context, postID uuid.UUID, resolvedBy uuid.UUID, status string) error {
	return r.dao.ResolveSuggestion(ctx, postID, resolvedBy, status)
}

func (r *postRepository) UnresolveSuggestion(ctx context.Context, postID uuid.UUID) error {
	return r.dao.UnresolveSuggestion(ctx, postID)
}

func (r *postRepository) CreateComment(ctx context.Context, id uuid.UUID, postID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.CreateComment(ctx, id, postID, parentID, userID, body)
}

func (r *postRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.UpdateComment(ctx, id, userID, body)
}

func (r *postRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body)
}

func (r *postRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.DeleteComment(ctx, id, userID)
}

func (r *postRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id)
}

func (r *postRepository) GetComments(ctx context.Context, postID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, postID, viewerID, limit, offset, excludeUserIDs)
}

func (r *postRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentEntityID(ctx, commentID)
}

func (r *postRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID)
}

func (r *postRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.LikeComment(ctx, userID, commentID)
}

func (r *postRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.UnlikeComment(ctx, userID, commentID)
}

func (r *postRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	return r.dao.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbnailURL, sortOrder)
}

func (r *postRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL)
}

func (r *postRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL)
}

func (r *postRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error) {
	return r.dao.GetCommentMedia(ctx, commentID)
}

func (r *postRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs)
}

func (r *postRepository) CountUserPostsToday(ctx context.Context, userID uuid.UUID) (int, error) {
	return r.dao.CountUserPostsToday(ctx, userID)
}

func (r *postRepository) GetCornerCounts(ctx context.Context) (map[string]int, error) {
	return r.dao.GetCornerCounts(ctx)
}

func (r *postRepository) GetShareCount(ctx context.Context, contentID string, contentType string) (int, error) {
	return r.dao.GetShareCount(ctx, contentID, contentType)
}

func (r *postRepository) GetShareCountsBatch(ctx context.Context, contentIDs []string, contentType string) (map[string]int, error) {
	return r.dao.GetShareCountsBatch(ctx, contentIDs, contentType)
}

func (r *postRepository) IncrementShareCount(ctx context.Context, contentID string, contentType string) error {
	return r.dao.IncrementShareCount(ctx, contentID, contentType)
}

func (r *postRepository) DecrementShareCount(ctx context.Context, contentID string, contentType string) error {
	return r.dao.DecrementShareCount(ctx, contentID, contentType)
}

func (r *postRepository) GetSharedContentFields(ctx context.Context, postID uuid.UUID) (*string, *string, error) {
	return r.dao.GetSharedContentFields(ctx, postID)
}

func (r *postRepository) GetSharedContentPreviews(refs []SharedContentRef) map[string]*dto.SharedContentPreview {
	return r.dao.GetSharedContentPreviews(refs)
}

func (r *postRepository) CreatePollWithOptions(ctx context.Context, pollID uuid.UUID, postID uuid.UUID, durationSeconds int, expiresAt string, options []string) error {
	return r.dao.CreatePollWithOptions(ctx, pollID, postID, durationSeconds, expiresAt, options)
}

func (r *postRepository) GetPollByPostID(ctx context.Context, postID uuid.UUID, viewerID uuid.UUID) (*model.PollRow, []model.PollOptionRow, *int, error) {
	return r.dao.GetPollByPostID(ctx, postID, viewerID)
}

func (r *postRepository) GetPollsByPostIDs(ctx context.Context, postIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID]*model.PollRow, map[uuid.UUID][]model.PollOptionRow, map[uuid.UUID]*int, error) {
	return r.dao.GetPollsByPostIDs(ctx, postIDs, viewerID)
}

func (r *postRepository) VotePoll(ctx context.Context, pollID uuid.UUID, userID uuid.UUID, optionID int) error {
	return r.dao.VotePoll(ctx, pollID, userID, optionID)
}

func (r *postRepository) AddEmbed(ctx context.Context, ownerID string, ownerType string, url string, embedType string, title string, description string, image string, siteName string, videoID string, sortOrder int) error {
	return r.dao.AddEmbed(ctx, ownerID, ownerType, url, embedType, title, description, image, siteName, videoID, sortOrder)
}

func (r *postRepository) DeleteEmbeds(ctx context.Context, ownerID string, ownerType string) error {
	return r.dao.DeleteEmbeds(ctx, ownerID, ownerType)
}

func (r *postRepository) UpdateEmbed(ctx context.Context, id int, title string, description string, image string, siteName string) error {
	return r.dao.UpdateEmbed(ctx, id, title, description, image, siteName)
}

func (r *postRepository) GetEmbeds(ctx context.Context, ownerID string, ownerType string) ([]model.EmbedRow, error) {
	return r.dao.GetEmbeds(ctx, ownerID, ownerType)
}

func (r *postRepository) GetEmbedsBatch(ctx context.Context, ownerIDs []string, ownerType string) (map[string][]model.EmbedRow, error) {
	return r.dao.GetEmbedsBatch(ctx, ownerIDs, ownerType)
}

func (r *postRepository) GetStaleEmbeds(ctx context.Context, olderThan string, limit int) ([]model.EmbedRow, error) {
	return r.dao.GetStaleEmbeds(ctx, olderThan, limit)
}
