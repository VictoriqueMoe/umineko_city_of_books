package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type (
	PostDAO interface {
		Create(ctx context.Context, spec NewPost, tx ...*sql.Tx) (*model.PostRow, error)
		UpdatePost(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error
		UpdatePostAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*model.PostRow, error)
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		ListAll(ctx context.Context, viewerID uuid.UUID, corner string, search string, sort string, seed int, limit, offset int, excludeUserIDs []uuid.UUID, resolvedFilter string, tx ...*sql.Tx) ([]model.PostRow, int, error)
		ListByFollowing(ctx context.Context, userID uuid.UUID, corner string, sort string, seed int, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.PostRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.PostRow, int, error)

		AddMedia(ctx context.Context, spec NewPostMedia, tx ...*sql.Tx) (int64, error)
		DeleteMedia(ctx context.Context, id int64, postID uuid.UUID, tx ...*sql.Tx) (string, error)
		UpdateMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error
		UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error
		GetMedia(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		GetMediaBatch(ctx context.Context, postIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		CollectMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)

		Like(ctx context.Context, userID uuid.UUID, postID uuid.UUID, tx ...*sql.Tx) error
		Unlike(ctx context.Context, userID uuid.UUID, postID uuid.UUID, tx ...*sql.Tx) error
		GetLikedBy(ctx context.Context, postID uuid.UUID, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.PostLikeUser, error)
		RecordView(ctx context.Context, postID uuid.UUID, viewerHash string, tx ...*sql.Tx) (bool, error)
		GetPostAuthorID(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetSharedContentAuthor(ctx context.Context, contentID string, contentType string, tx ...*sql.Tx) (uuid.UUID, error)

		ResolveSuggestion(ctx context.Context, postID uuid.UUID, resolvedBy uuid.UUID, status string, tx ...*sql.Tx) error
		UnresolveSuggestion(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) error

		CreateComment(ctx context.Context, postID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetComments(ctx context.Context, postID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error)
		GetCommentByID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (*CommentRow, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error
		AddCommentMedia(ctx context.Context, spec NewPostCommentMedia, tx ...*sql.Tx) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)

		CountUserPostsToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetCornerCounts(ctx context.Context, tx ...*sql.Tx) (map[string]int, error)

		GetShareCount(ctx context.Context, contentID string, contentType string, tx ...*sql.Tx) (int, error)
		GetShareCountsBatch(ctx context.Context, contentIDs []string, contentType string, tx ...*sql.Tx) (map[string]int, error)
		IncrementShareCount(ctx context.Context, contentID string, contentType string, tx ...*sql.Tx) error
		DecrementShareCount(ctx context.Context, contentID string, contentType string, tx ...*sql.Tx) error
		GetSharedContentFields(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) (*string, *string, error)
		GetSharedContentPreviews(refs []SharedContentRef, tx ...*sql.Tx) map[string]*dto.SharedContentPreview

		CreatePoll(ctx context.Context, postID uuid.UUID, durationSeconds int, expiresAt string, tx ...*sql.Tx) (*model.PollRow, error)
		AddPollOption(ctx context.Context, pollID string, label string, sortOrder int, tx ...*sql.Tx) error
		GetPollByPostID(ctx context.Context, postID uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*model.PollRow, []model.PollOptionRow, *int, error)
		GetPollsByPostIDs(ctx context.Context, postIDs []uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID]*model.PollRow, map[uuid.UUID][]model.PollOptionRow, map[uuid.UUID]*int, error)
		VotePoll(ctx context.Context, pollID uuid.UUID, userID uuid.UUID, optionID int, tx ...*sql.Tx) error

		AddEmbed(ctx context.Context, spec NewEmbed, tx ...*sql.Tx) error
		DeleteEmbeds(ctx context.Context, ownerID string, ownerType string, tx ...*sql.Tx) error
		UpdateEmbed(ctx context.Context, spec EmbedUpdate, tx ...*sql.Tx) error
		GetEmbeds(ctx context.Context, ownerID string, ownerType string, tx ...*sql.Tx) ([]model.EmbedRow, error)
		GetEmbedsBatch(ctx context.Context, ownerIDs []string, ownerType string, tx ...*sql.Tx) (map[string][]model.EmbedRow, error)
		GetStaleEmbeds(ctx context.Context, olderThan string, limit int, tx ...*sql.Tx) ([]model.EmbedRow, error)
	}

	PostRepository interface {
		PostDAO

		CreateWithDetails(ctx context.Context, spec NewPost, tx ...*sql.Tx) (*model.PostRow, error)
		UpdateWithDetails(ctx context.Context, spec PostUpdate, tx ...*sql.Tx) error
		DeleteWithSharedContent(ctx context.Context, spec PostDelete, tx ...*sql.Tx) (*SharedContentRef, []string, error)
		UpdateCommentWithDetails(ctx context.Context, spec PostCommentUpdate, tx ...*sql.Tx) error
		DeleteCommentWithAudit(ctx context.Context, spec PostCommentDelete, tx ...*sql.Tx) ([]string, error)
		CreatePollWithOptions(ctx context.Context, postID uuid.UUID, durationSeconds int, expiresAt string, options []string, tx ...*sql.Tx) (*model.PollRow, error)
	}

	SharedContentRef struct {
		ID   string
		Type string
	}

	NewPost struct {
		UserID        uuid.UUID
		Corner        string
		Body          string
		SharedContent *SharedContentRef
		Poll          *NewPoll
	}

	NewPoll struct {
		DurationSeconds int
		ExpiresAt       string
		Options         []string
	}

	PostUpdate struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		Body    string
		AsAdmin bool
	}

	PostDelete struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		AsAdmin bool
		Audit   NewAuditEntry
	}

	PostCommentUpdate struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		Body    string
		AsAdmin bool
	}

	PostCommentDelete struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		AsAdmin bool
		Audit   NewAuditEntry
	}

	NewPostMedia struct {
		PostID       uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
	}

	NewPostCommentMedia struct {
		CommentID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
	}

	NewEmbed struct {
		OwnerID     string
		OwnerType   string
		URL         string
		EmbedType   string
		Title       string
		Description string
		Image       string
		SiteName    string
		VideoID     string
		SortOrder   int
	}

	EmbedUpdate struct {
		ID          int
		Title       string
		Description string
		Image       string
		SiteName    string
	}
)

const (
	postEmbedOwnerType    = "post"
	commentEmbedOwnerType = "comment"
)

type postRepository struct {
	db    *sql.DB
	dao   PostDAO
	audit AuditLogRepository
}

func NewPostRepo(database *sql.DB, dao PostDAO, audit AuditLogRepository) PostRepository {
	return &postRepository{db: database, dao: dao, audit: audit}
}

func (r *postRepository) CreateWithDetails(ctx context.Context, spec NewPost, tx ...*sql.Tx) (*model.PostRow, error) {
	var created *model.PostRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.dao.Create(ctx, spec, tx)
		if err != nil {
			return err
		}

		if spec.Poll == nil {
			return nil
		}

		_, err = r.createPoll(ctx, created.ID, *spec.Poll, tx)

		return err
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *postRepository) UpdateWithDetails(ctx context.Context, spec PostUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		if spec.AsAdmin {
			err = r.dao.UpdatePostAsAdmin(ctx, spec.ID, spec.Body, tx)
		} else {
			err = r.dao.UpdatePost(ctx, spec.ID, spec.UserID, spec.Body, tx)
		}
		if err != nil {
			return err
		}

		return r.dao.DeleteEmbeds(ctx, spec.ID.String(), postEmbedOwnerType, tx)
	})
}

func (r *postRepository) DeleteWithSharedContent(ctx context.Context, spec PostDelete, tx ...*sql.Tx) (*SharedContentRef, []string, error) {
	var (
		shared *SharedContentRef
		paths  []string
	)

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		contentID, contentType, err := r.dao.GetSharedContentFields(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		if contentID != nil && contentType != nil {
			shared = &SharedContentRef{ID: *contentID, Type: *contentType}
		}

		mediaPaths, err := r.dao.CollectMediaPaths(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		commentPaths, err := r.dao.CollectCommentMediaPaths(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		paths = append(mediaPaths, commentPaths...)

		if spec.AsAdmin {
			err = r.dao.DeleteAsAdmin(ctx, spec.ID, tx)
		} else {
			err = r.dao.Delete(ctx, spec.ID, spec.UserID, tx)
		}
		if err != nil {
			return err
		}

		if err := r.audit.Create(ctx, spec.Audit, tx); err != nil {
			return fmt.Errorf("audit post delete: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return shared, paths, nil
}

func (r *postRepository) UpdateCommentWithDetails(ctx context.Context, spec PostCommentUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		if spec.AsAdmin {
			err = r.dao.UpdateCommentAsAdmin(ctx, spec.ID, spec.Body, tx)
		} else {
			err = r.dao.UpdateComment(ctx, spec.ID, spec.UserID, spec.Body, tx)
		}
		if err != nil {
			return err
		}

		return r.dao.DeleteEmbeds(ctx, spec.ID.String(), commentEmbedOwnerType, tx)
	})
}

func (r *postRepository) DeleteCommentWithAudit(ctx context.Context, spec PostCommentDelete, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		mediaPaths, err := r.dao.CollectSingleCommentMediaPaths(ctx, spec.ID, tx)
		if err != nil {
			return err
		}

		paths = mediaPaths

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

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *postRepository) Create(ctx context.Context, spec NewPost, tx ...*sql.Tx) (*model.PostRow, error) {
	return r.dao.Create(ctx, spec, tx...)
}

func (r *postRepository) UpdatePost(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdatePost(ctx, id, userID, body, tx...)
}

func (r *postRepository) UpdatePostAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdatePostAsAdmin(ctx, id, body, tx...)
}

func (r *postRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*model.PostRow, error) {
	return r.dao.GetByID(ctx, id, viewerID, tx...)
}

func (r *postRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Delete(ctx, id, userID, tx...)
}

func (r *postRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteAsAdmin(ctx, id, tx...)
}

func (r *postRepository) ListAll(ctx context.Context, viewerID uuid.UUID, corner string, search string, sort string, seed int, limit, offset int, excludeUserIDs []uuid.UUID, resolvedFilter string, tx ...*sql.Tx) ([]model.PostRow, int, error) {
	return r.dao.ListAll(ctx, viewerID, corner, search, sort, seed, limit, offset, excludeUserIDs, resolvedFilter, tx...)
}

func (r *postRepository) ListByFollowing(ctx context.Context, userID uuid.UUID, corner string, sort string, seed int, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.PostRow, int, error) {
	return r.dao.ListByFollowing(ctx, userID, corner, sort, seed, limit, offset, excludeUserIDs, tx...)
}

func (r *postRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.PostRow, int, error) {
	return r.dao.ListByUser(ctx, userID, viewerID, limit, offset, tx...)
}

func (r *postRepository) AddMedia(ctx context.Context, spec NewPostMedia, tx ...*sql.Tx) (int64, error) {
	return r.dao.AddMedia(ctx, spec, tx...)
}

func (r *postRepository) DeleteMedia(ctx context.Context, id int64, postID uuid.UUID, tx ...*sql.Tx) (string, error) {
	return r.dao.DeleteMedia(ctx, id, postID, tx...)
}

func (r *postRepository) UpdateMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateMediaURL(ctx, id, mediaURL, tx...)
}

func (r *postRepository) UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateMediaThumbnail(ctx, id, thumbnailURL, tx...)
}

func (r *postRepository) GetMedia(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error) {
	return r.dao.GetMedia(ctx, postID, tx...)
}

func (r *postRepository) GetMediaBatch(ctx context.Context, postIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetMediaBatch(ctx, postIDs, tx...)
}

func (r *postRepository) CollectMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectMediaPaths(ctx, entityID, tx...)
}

func (r *postRepository) Like(ctx context.Context, userID uuid.UUID, postID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Like(ctx, userID, postID, tx...)
}

func (r *postRepository) Unlike(ctx context.Context, userID uuid.UUID, postID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Unlike(ctx, userID, postID, tx...)
}

func (r *postRepository) GetLikedBy(ctx context.Context, postID uuid.UUID, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.PostLikeUser, error) {
	return r.dao.GetLikedBy(ctx, postID, excludeUserIDs, tx...)
}

func (r *postRepository) RecordView(ctx context.Context, postID uuid.UUID, viewerHash string, tx ...*sql.Tx) (bool, error) {
	return r.dao.RecordView(ctx, postID, viewerHash, tx...)
}

func (r *postRepository) GetPostAuthorID(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetPostAuthorID(ctx, postID, tx...)
}

func (r *postRepository) GetSharedContentAuthor(ctx context.Context, contentID string, contentType string, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetSharedContentAuthor(ctx, contentID, contentType, tx...)
}

func (r *postRepository) ResolveSuggestion(ctx context.Context, postID uuid.UUID, resolvedBy uuid.UUID, status string, tx ...*sql.Tx) error {
	return r.dao.ResolveSuggestion(ctx, postID, resolvedBy, status, tx...)
}

func (r *postRepository) UnresolveSuggestion(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.UnresolveSuggestion(ctx, postID, tx...)
}

func (r *postRepository) CreateComment(ctx context.Context, postID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) (*CommentRow, error) {
	return r.dao.CreateComment(ctx, postID, parentID, userID, body, tx...)
}

func (r *postRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateComment(ctx, id, userID, body, tx...)
}

func (r *postRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body, tx...)
}

func (r *postRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteComment(ctx, id, userID, tx...)
}

func (r *postRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id, tx...)
}

func (r *postRepository) GetComments(ctx context.Context, postID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, postID, viewerID, limit, offset, excludeUserIDs, tx...)
}

func (r *postRepository) GetCommentByID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (*CommentRow, error) {
	return r.dao.GetCommentByID(ctx, commentID, tx...)
}

func (r *postRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentEntityID(ctx, commentID, tx...)
}

func (r *postRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID, tx...)
}

func (r *postRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.LikeComment(ctx, userID, commentID, tx...)
}

func (r *postRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.UnlikeComment(ctx, userID, commentID, tx...)
}

func (r *postRepository) AddCommentMedia(ctx context.Context, spec NewPostCommentMedia, tx ...*sql.Tx) (int64, error) {
	return r.dao.AddCommentMedia(ctx, spec, tx...)
}

func (r *postRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL, tx...)
}

func (r *postRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL, tx...)
}

func (r *postRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error) {
	return r.dao.GetCommentMedia(ctx, commentID, tx...)
}

func (r *postRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs, tx...)
}

func (r *postRepository) CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectCommentMediaPaths(ctx, entityID, tx...)
}

func (r *postRepository) CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectSingleCommentMediaPaths(ctx, commentID, tx...)
}

func (r *postRepository) CountUserPostsToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.CountUserPostsToday(ctx, userID, tx...)
}

func (r *postRepository) GetCornerCounts(ctx context.Context, tx ...*sql.Tx) (map[string]int, error) {
	return r.dao.GetCornerCounts(ctx, tx...)
}

func (r *postRepository) GetShareCount(ctx context.Context, contentID string, contentType string, tx ...*sql.Tx) (int, error) {
	return r.dao.GetShareCount(ctx, contentID, contentType, tx...)
}

func (r *postRepository) GetShareCountsBatch(ctx context.Context, contentIDs []string, contentType string, tx ...*sql.Tx) (map[string]int, error) {
	return r.dao.GetShareCountsBatch(ctx, contentIDs, contentType, tx...)
}

func (r *postRepository) IncrementShareCount(ctx context.Context, contentID string, contentType string, tx ...*sql.Tx) error {
	return r.dao.IncrementShareCount(ctx, contentID, contentType, tx...)
}

func (r *postRepository) DecrementShareCount(ctx context.Context, contentID string, contentType string, tx ...*sql.Tx) error {
	return r.dao.DecrementShareCount(ctx, contentID, contentType, tx...)
}

func (r *postRepository) GetSharedContentFields(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) (*string, *string, error) {
	return r.dao.GetSharedContentFields(ctx, postID, tx...)
}

func (r *postRepository) GetSharedContentPreviews(refs []SharedContentRef, tx ...*sql.Tx) map[string]*dto.SharedContentPreview {
	return r.dao.GetSharedContentPreviews(refs, tx...)
}

func (r *postRepository) CreatePollWithOptions(ctx context.Context, postID uuid.UUID, durationSeconds int, expiresAt string, options []string, tx ...*sql.Tx) (*model.PollRow, error) {
	var created *model.PollRow

	spec := NewPoll{DurationSeconds: durationSeconds, ExpiresAt: expiresAt, Options: options}

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.createPoll(ctx, postID, spec, tx)

		return err
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *postRepository) createPoll(ctx context.Context, postID uuid.UUID, spec NewPoll, tx *sql.Tx) (*model.PollRow, error) {
	created, err := r.dao.CreatePoll(ctx, postID, spec.DurationSeconds, spec.ExpiresAt, tx)
	if err != nil {
		return nil, err
	}

	for i := range spec.Options {
		if err := r.dao.AddPollOption(ctx, created.ID, spec.Options[i], i, tx); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (r *postRepository) CreatePoll(ctx context.Context, postID uuid.UUID, durationSeconds int, expiresAt string, tx ...*sql.Tx) (*model.PollRow, error) {
	return r.dao.CreatePoll(ctx, postID, durationSeconds, expiresAt, tx...)
}

func (r *postRepository) AddPollOption(ctx context.Context, pollID string, label string, sortOrder int, tx ...*sql.Tx) error {
	return r.dao.AddPollOption(ctx, pollID, label, sortOrder, tx...)
}

func (r *postRepository) GetPollByPostID(ctx context.Context, postID uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*model.PollRow, []model.PollOptionRow, *int, error) {
	return r.dao.GetPollByPostID(ctx, postID, viewerID, tx...)
}

func (r *postRepository) GetPollsByPostIDs(ctx context.Context, postIDs []uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID]*model.PollRow, map[uuid.UUID][]model.PollOptionRow, map[uuid.UUID]*int, error) {
	return r.dao.GetPollsByPostIDs(ctx, postIDs, viewerID, tx...)
}

func (r *postRepository) VotePoll(ctx context.Context, pollID uuid.UUID, userID uuid.UUID, optionID int, tx ...*sql.Tx) error {
	return r.dao.VotePoll(ctx, pollID, userID, optionID, tx...)
}

func (r *postRepository) AddEmbed(ctx context.Context, spec NewEmbed, tx ...*sql.Tx) error {
	return r.dao.AddEmbed(ctx, spec, tx...)
}

func (r *postRepository) DeleteEmbeds(ctx context.Context, ownerID string, ownerType string, tx ...*sql.Tx) error {
	return r.dao.DeleteEmbeds(ctx, ownerID, ownerType, tx...)
}

func (r *postRepository) UpdateEmbed(ctx context.Context, spec EmbedUpdate, tx ...*sql.Tx) error {
	return r.dao.UpdateEmbed(ctx, spec, tx...)
}

func (r *postRepository) GetEmbeds(ctx context.Context, ownerID string, ownerType string, tx ...*sql.Tx) ([]model.EmbedRow, error) {
	return r.dao.GetEmbeds(ctx, ownerID, ownerType, tx...)
}

func (r *postRepository) GetEmbedsBatch(ctx context.Context, ownerIDs []string, ownerType string, tx ...*sql.Tx) (map[string][]model.EmbedRow, error) {
	return r.dao.GetEmbedsBatch(ctx, ownerIDs, ownerType, tx...)
}

func (r *postRepository) GetStaleEmbeds(ctx context.Context, olderThan string, limit int, tx ...*sql.Tx) ([]model.EmbedRow, error) {
	return r.dao.GetStaleEmbeds(ctx, olderThan, limit, tx...)
}
