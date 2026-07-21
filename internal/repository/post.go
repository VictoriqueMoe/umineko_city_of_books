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
		GetComments(ctx context.Context, postID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.PostCommentRow, int, error)
		GetCommentPostID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
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
