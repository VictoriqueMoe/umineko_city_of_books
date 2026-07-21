package repository

import (
	"context"

	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type (
	AnnouncementRepository interface {
		Create(ctx context.Context, id uuid.UUID, authorID uuid.UUID, title string, body string) error
		Update(ctx context.Context, id uuid.UUID, title string, body string) error
		Delete(ctx context.Context, id uuid.UUID) error
		GetByID(ctx context.Context, id uuid.UUID) (*AnnouncementRow, error)
		List(ctx context.Context, limit, offset int) ([]AnnouncementRow, int, error)
		GetLatest(ctx context.Context) (*AnnouncementRow, error)
		SetPinned(ctx context.Context, id uuid.UUID, pinned bool) error

		CreateComment(ctx context.Context, id uuid.UUID, announcementID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, announcementID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]AnnouncementCommentRow, int, error)
		GetCommentAnnouncementID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error

		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]AnnouncementCommentMediaRow, error)
	}

	AnnouncementCommentRow struct {
		ID                uuid.UUID
		AnnouncementID    uuid.UUID
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

	AnnouncementCommentMediaRow = model.CommentMediaRow

	AnnouncementRow struct {
		ID                uuid.UUID
		Title             string
		Body              string
		AuthorID          uuid.UUID
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		AuthorRole        string
		Pinned            bool
		CreatedAt         string
		UpdatedAt         string
	}
)
