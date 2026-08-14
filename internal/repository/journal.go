package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/journal/params"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

const (
	journalCommentTargetType        = "journal_comment"
	journalCommentDeleteAction      = "journal_comment_delete"
	journalCommentDeleteAdminAction = "journal_comment_delete_admin"
)

type (
	JournalDAO interface {
		Create(ctx context.Context, userID uuid.UUID, req dto.CreateJournalRequest, tx ...*sql.Tx) (*dto.JournalResponse, error)
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*dto.JournalResponse, error)
		List(ctx context.Context, p params.ListParams, viewerID uuid.UUID, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]dto.JournalResponse, int, error)
		Update(ctx context.Context, spec JournalUpdate, tx ...*sql.Tx) error
		UpdateAsAdmin(ctx context.Context, spec JournalUpdate, tx ...*sql.Tx) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		CollectMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		ListEntryIDs(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
		ListEntryCommentIDs(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
		GetAuthorID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetTitle(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (string, error)
		IsArchived(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (bool, error)
		CountUserJournalsToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		UpdateLastAuthorActivity(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		ArchiveStale(ctx context.Context, cutoff time.Time, tx ...*sql.Tx) ([]uuid.UUID, error)

		Follow(ctx context.Context, userID uuid.UUID, journalID uuid.UUID, tx ...*sql.Tx) error
		Unfollow(ctx context.Context, userID uuid.UUID, journalID uuid.UUID, tx ...*sql.Tx) error
		IsFollower(ctx context.Context, userID uuid.UUID, journalID uuid.UUID, tx ...*sql.Tx) (bool, error)
		GetFollowerIDs(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
		GetFollowerCount(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) (int, error)
		ListFollowedByUser(ctx context.Context, followerID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]dto.JournalResponse, int, error)

		CreateEntry(ctx context.Context, spec NewJournalEntry, tx ...*sql.Tx) (*JournalEntryRow, error)
		UpdateEntry(ctx context.Context, spec JournalEntryUpdate, tx ...*sql.Tx) error
		DeleteEntry(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetEntry(ctx context.Context, journalID uuid.UUID, entryNumber int, tx ...*sql.Tx) (*JournalEntryRow, error)
		GetEntryByID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (*JournalEntryRow, error)
		ListEntries(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) ([]JournalEntrySummaryRow, error)
		GetNextEntryNumber(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetEntryJournalID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetEntryAuthorID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)

		CreateComment(ctx context.Context, spec NewJournalComment, tx ...*sql.Tx) (*CommentRow, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string, tx ...*sql.Tx) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetComments(ctx context.Context, journalID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error)
		GetEntryComments(ctx context.Context, entryID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentEntryNumber(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (*int, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error

		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int, tx ...*sql.Tx) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)

		AddMedia(ctx context.Context, entityID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int, tx ...*sql.Tx) (int64, error)
		UpdateMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error
		UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error
		GetMediaBatch(ctx context.Context, entityIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		DeleteMedia(ctx context.Context, id int64, entityID uuid.UUID, tx ...*sql.Tx) (string, error)
	}

	JournalRepository interface {
		Create(ctx context.Context, userID uuid.UUID, req dto.CreateJournalRequest, tx ...*sql.Tx) (*dto.JournalResponse, error)
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*dto.JournalResponse, error)
		List(ctx context.Context, p params.ListParams, viewerID uuid.UUID, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]dto.JournalResponse, int, error)
		Update(ctx context.Context, spec JournalUpdate, tx ...*sql.Tx) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, asAdmin bool, tx ...*sql.Tx) ([]string, error)
		GetAuthorID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetTitle(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (string, error)
		IsArchived(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (bool, error)
		CountUserJournalsToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		UpdateLastAuthorActivity(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		ArchiveStale(ctx context.Context, cutoff time.Time, tx ...*sql.Tx) ([]uuid.UUID, error)

		Follow(ctx context.Context, userID uuid.UUID, journalID uuid.UUID, tx ...*sql.Tx) error
		Unfollow(ctx context.Context, userID uuid.UUID, journalID uuid.UUID, tx ...*sql.Tx) error
		IsFollower(ctx context.Context, userID uuid.UUID, journalID uuid.UUID, tx ...*sql.Tx) (bool, error)
		GetFollowerIDs(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
		GetFollowerCount(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) (int, error)
		ListFollowedByUser(ctx context.Context, followerID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]dto.JournalResponse, int, error)

		CreateEntry(ctx context.Context, spec NewJournalEntry, tx ...*sql.Tx) (*JournalEntryRow, error)
		UpdateEntry(ctx context.Context, spec JournalEntryUpdate, tx ...*sql.Tx) error
		DeleteEntry(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) ([]string, error)
		GetEntry(ctx context.Context, journalID uuid.UUID, entryNumber int, tx ...*sql.Tx) (*JournalEntryRow, error)
		GetEntryByID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (*JournalEntryRow, error)
		ListEntries(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) ([]JournalEntrySummaryRow, error)
		GetNextEntryNumber(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetEntryJournalID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetEntryAuthorID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)

		CreateComment(ctx context.Context, spec NewJournalComment, tx ...*sql.Tx) (*CommentRow, error)
		UpdateComment(ctx context.Context, spec JournalCommentUpdate, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, asAdmin bool, tx ...*sql.Tx) ([]string, error)
		GetComments(ctx context.Context, journalID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error)
		GetEntryComments(ctx context.Context, entryID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentEntryNumber(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (*int, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error

		AddCommentMedia(ctx context.Context, spec NewJournalCommentMedia, tx ...*sql.Tx) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)

		AddMedia(ctx context.Context, spec NewJournalEntryMedia, tx ...*sql.Tx) (int64, error)
		UpdateMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error
		UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error
		GetMediaBatch(ctx context.Context, entityIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		DeleteMedia(ctx context.Context, id int64, entityID uuid.UUID, tx ...*sql.Tx) (string, error)
		CollectMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
	}

	JournalUpdate struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		Title   string
		Work    string
		AsAdmin bool
	}

	NewJournalEntry struct {
		JournalID   uuid.UUID
		EntryNumber int
		Title       *string
		Body        string
		WordCount   int
		IsDraft     bool
	}

	JournalEntryUpdate struct {
		ID                   uuid.UUID
		JournalID            uuid.UUID
		Title                *string
		Body                 string
		WordCount            int
		IsDraft              bool
		RecordAuthorActivity bool
	}

	NewJournalComment struct {
		JournalID            uuid.UUID
		EntryID              *uuid.UUID
		ParentID             *uuid.UUID
		UserID               uuid.UUID
		Body                 string
		RecordAuthorActivity bool
	}

	JournalCommentUpdate struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		Body    string
		AsAdmin bool
	}

	NewJournalEntryMedia struct {
		EntryID      uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
	}

	NewJournalCommentMedia struct {
		CommentID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
	}

	JournalEntryRow struct {
		ID          uuid.UUID
		JournalID   uuid.UUID
		EntryNumber int
		Title       *string
		Body        string
		WordCount   int
		IsDraft     bool
		HasPrev     bool
		HasNext     bool
		CreatedAt   string
		UpdatedAt   *string
	}

	JournalEntrySummaryRow struct {
		ID          uuid.UUID
		EntryNumber int
		Title       *string
		WordCount   int
		IsDraft     bool
		CreatedAt   string
	}
)

func JournalCommentToDTO(c CommentRow, media []model.PostMediaRow, authorID uuid.UUID) dto.JournalCommentResponse {
	return dto.JournalCommentResponse{
		ID:       c.ID,
		ParentID: c.ParentID,
		EntryID:  c.EntryID,
		Author: dto.UserResponse{
			ID:          c.UserID,
			Username:    c.AuthorUsername,
			DisplayName: c.AuthorDisplayName,
			AvatarURL:   c.AuthorAvatarURL,
			Role:        role.Role(c.AuthorRole),
		},
		Body:      c.Body,
		Media:     model.MediaRowsToResponse(media),
		LikeCount: c.LikeCount,
		UserLiked: c.UserLiked,
		IsAuthor:  c.UserID == authorID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func JournalEntryToDTO(e *JournalEntryRow, media []model.PostMediaRow) dto.JournalEntryResponse {
	mediaList := model.MediaRowsToResponse(media)

	return dto.JournalEntryResponse{
		ID:          e.ID,
		JournalID:   e.JournalID,
		EntryNumber: e.EntryNumber,
		Title:       e.Title,
		Body:        e.Body,
		WordCount:   e.WordCount,
		IsDraft:     e.IsDraft,
		HasPrev:     e.HasPrev,
		HasNext:     e.HasNext,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		Media:       mediaList,
	}
}

func JournalEntrySummaryToDTO(s JournalEntrySummaryRow) dto.JournalEntrySummary {
	return dto.JournalEntrySummary{
		ID:          s.ID,
		EntryNumber: s.EntryNumber,
		Title:       s.Title,
		WordCount:   s.WordCount,
		IsDraft:     s.IsDraft,
		CreatedAt:   s.CreatedAt,
	}
}

type journalRepository struct {
	db    *sql.DB
	dao   JournalDAO
	audit AuditLogRepository
}

func NewJournalRepo(database *sql.DB, dao JournalDAO, audit AuditLogRepository) JournalRepository {
	return &journalRepository{db: database, dao: dao, audit: audit}
}

func (r *journalRepository) Create(ctx context.Context, userID uuid.UUID, req dto.CreateJournalRequest, tx ...*sql.Tx) (*dto.JournalResponse, error) {
	return r.dao.Create(ctx, userID, req, tx...)
}

func (r *journalRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*dto.JournalResponse, error) {
	return r.dao.GetByID(ctx, id, viewerID, tx...)
}

func (r *journalRepository) List(ctx context.Context, p params.ListParams, viewerID uuid.UUID, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]dto.JournalResponse, int, error) {
	return r.dao.List(ctx, p, viewerID, excludeUserIDs, tx...)
}

func (r *journalRepository) Update(ctx context.Context, spec JournalUpdate, tx ...*sql.Tx) error {
	if spec.AsAdmin {
		return r.dao.UpdateAsAdmin(ctx, spec, tx...)
	}

	return r.dao.Update(ctx, spec, tx...)
}

func (r *journalRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, asAdmin bool, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTxOrJoin(ctx, r.db, tx, func(tx *sql.Tx) error {
		entryIDs, err := r.dao.ListEntryIDs(ctx, id, tx)
		if err != nil {
			return err
		}

		var collected []string
		for _, entryID := range entryIDs {
			entryPaths, err := r.dao.CollectMediaPaths(ctx, entryID, tx)
			if err != nil {
				return err
			}

			collected = append(collected, entryPaths...)
		}

		commentPaths, err := r.dao.CollectCommentMediaPaths(ctx, id, tx)
		if err != nil {
			return err
		}

		collected = append(collected, commentPaths...)

		if asAdmin {
			if err := r.dao.DeleteAsAdmin(ctx, id, tx); err != nil {
				return err
			}
		} else {
			if err := r.dao.Delete(ctx, id, userID, tx); err != nil {
				return err
			}
		}

		paths = dedupePaths(collected)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *journalRepository) GetAuthorID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetAuthorID(ctx, id, tx...)
}

func (r *journalRepository) GetTitle(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (string, error) {
	return r.dao.GetTitle(ctx, id, tx...)
}

func (r *journalRepository) IsArchived(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.IsArchived(ctx, id, tx...)
}

func (r *journalRepository) CountUserJournalsToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.CountUserJournalsToday(ctx, userID, tx...)
}

func (r *journalRepository) UpdateLastAuthorActivity(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.UpdateLastAuthorActivity(ctx, id, tx...)
}

func (r *journalRepository) ArchiveStale(ctx context.Context, cutoff time.Time, tx ...*sql.Tx) ([]uuid.UUID, error) {
	return r.dao.ArchiveStale(ctx, cutoff, tx...)
}

func (r *journalRepository) Follow(ctx context.Context, userID uuid.UUID, journalID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Follow(ctx, userID, journalID, tx...)
}

func (r *journalRepository) Unfollow(ctx context.Context, userID uuid.UUID, journalID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Unfollow(ctx, userID, journalID, tx...)
}

func (r *journalRepository) IsFollower(ctx context.Context, userID uuid.UUID, journalID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.IsFollower(ctx, userID, journalID, tx...)
}

func (r *journalRepository) GetFollowerIDs(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	return r.dao.GetFollowerIDs(ctx, journalID, tx...)
}

func (r *journalRepository) GetFollowerCount(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.GetFollowerCount(ctx, journalID, tx...)
}

func (r *journalRepository) ListFollowedByUser(ctx context.Context, followerID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]dto.JournalResponse, int, error) {
	return r.dao.ListFollowedByUser(ctx, followerID, viewerID, limit, offset, tx...)
}

func (r *journalRepository) CreateEntry(ctx context.Context, spec NewJournalEntry, tx ...*sql.Tx) (*JournalEntryRow, error) {
	var created *JournalEntryRow

	err := db.WithTxOrJoin(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.dao.CreateEntry(ctx, spec, tx)
		if err != nil {
			return err
		}

		return r.dao.UpdateLastAuthorActivity(ctx, spec.JournalID, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *journalRepository) UpdateEntry(ctx context.Context, spec JournalEntryUpdate, tx ...*sql.Tx) error {
	return db.WithTxOrJoin(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.UpdateEntry(ctx, spec, tx); err != nil {
			return err
		}

		if !spec.RecordAuthorActivity {
			return nil
		}

		return r.dao.UpdateLastAuthorActivity(ctx, spec.JournalID, tx)
	})
}

func (r *journalRepository) DeleteEntry(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTxOrJoin(ctx, r.db, tx, func(tx *sql.Tx) error {
		collected, err := r.dao.CollectMediaPaths(ctx, id, tx)
		if err != nil {
			return err
		}

		commentIDs, err := r.dao.ListEntryCommentIDs(ctx, id, tx)
		if err != nil {
			return err
		}

		for _, commentID := range commentIDs {
			commentPaths, err := r.dao.CollectSingleCommentMediaPaths(ctx, commentID, tx)
			if err != nil {
				return err
			}

			collected = append(collected, commentPaths...)
		}

		if err := r.dao.DeleteEntry(ctx, id, tx); err != nil {
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

func (r *journalRepository) GetEntry(ctx context.Context, journalID uuid.UUID, entryNumber int, tx ...*sql.Tx) (*JournalEntryRow, error) {
	return r.dao.GetEntry(ctx, journalID, entryNumber, tx...)
}

func (r *journalRepository) GetEntryByID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (*JournalEntryRow, error) {
	return r.dao.GetEntryByID(ctx, entryID, tx...)
}

func (r *journalRepository) ListEntries(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) ([]JournalEntrySummaryRow, error) {
	return r.dao.ListEntries(ctx, journalID, tx...)
}

func (r *journalRepository) GetNextEntryNumber(ctx context.Context, journalID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.GetNextEntryNumber(ctx, journalID, tx...)
}

func (r *journalRepository) GetEntryJournalID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetEntryJournalID(ctx, entryID, tx...)
}

func (r *journalRepository) GetEntryAuthorID(ctx context.Context, entryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetEntryAuthorID(ctx, entryID, tx...)
}

func (r *journalRepository) CreateComment(ctx context.Context, spec NewJournalComment, tx ...*sql.Tx) (*CommentRow, error) {
	var created *CommentRow

	err := db.WithTxOrJoin(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.dao.CreateComment(ctx, spec, tx)
		if err != nil {
			return err
		}

		if !spec.RecordAuthorActivity {
			return nil
		}

		return r.dao.UpdateLastAuthorActivity(ctx, spec.JournalID, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *journalRepository) UpdateComment(ctx context.Context, spec JournalCommentUpdate, tx ...*sql.Tx) error {
	if spec.AsAdmin {
		return r.dao.UpdateCommentAsAdmin(ctx, spec.ID, spec.Body, tx...)
	}

	return r.dao.UpdateComment(ctx, spec.ID, spec.UserID, spec.Body, tx...)
}

func (r *journalRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, asAdmin bool, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTxOrJoin(ctx, r.db, tx, func(tx *sql.Tx) error {
		mediaPaths, err := r.dao.CollectSingleCommentMediaPaths(ctx, id, tx)
		if err != nil {
			return err
		}

		action := journalCommentDeleteAction

		if asAdmin {
			if err := r.dao.DeleteCommentAsAdmin(ctx, id, tx); err != nil {
				return err
			}

			action = journalCommentDeleteAdminAction
		} else {
			if err := r.dao.DeleteComment(ctx, id, userID, tx); err != nil {
				return err
			}
		}

		if err := r.audit.Create(ctx, NewAuditEntry{
			ActorID:    userID,
			Action:     action,
			TargetType: journalCommentTargetType,
			TargetID:   id.String(),
		}, tx); err != nil {
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

func (r *journalRepository) GetComments(ctx context.Context, journalID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, journalID, viewerID, limit, offset, excludeUserIDs, tx...)
}

func (r *journalRepository) GetEntryComments(ctx context.Context, entryID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]CommentRow, int, error) {
	return r.dao.GetEntryComments(ctx, entryID, viewerID, limit, offset, excludeUserIDs, tx...)
}

func (r *journalRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentEntityID(ctx, commentID, tx...)
}

func (r *journalRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID, tx...)
}

func (r *journalRepository) GetCommentEntryNumber(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (*int, error) {
	return r.dao.GetCommentEntryNumber(ctx, commentID, tx...)
}

func (r *journalRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.LikeComment(ctx, userID, commentID, tx...)
}

func (r *journalRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.UnlikeComment(ctx, userID, commentID, tx...)
}

func (r *journalRepository) AddCommentMedia(ctx context.Context, spec NewJournalCommentMedia, tx ...*sql.Tx) (int64, error) {
	return r.dao.AddCommentMedia(ctx, spec.CommentID, spec.MediaURL, spec.MediaType, spec.ThumbnailURL, spec.SortOrder, tx...)
}

func (r *journalRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL, tx...)
}

func (r *journalRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL, tx...)
}

func (r *journalRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs, tx...)
}

func (r *journalRepository) CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectCommentMediaPaths(ctx, entityID, tx...)
}

func (r *journalRepository) CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectSingleCommentMediaPaths(ctx, commentID, tx...)
}

func (r *journalRepository) AddMedia(ctx context.Context, spec NewJournalEntryMedia, tx ...*sql.Tx) (int64, error) {
	return r.dao.AddMedia(ctx, spec.EntryID, spec.MediaURL, spec.MediaType, spec.ThumbnailURL, spec.SortOrder, tx...)
}

func (r *journalRepository) UpdateMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateMediaURL(ctx, id, mediaURL, tx...)
}

func (r *journalRepository) UpdateMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateMediaThumbnail(ctx, id, thumbnailURL, tx...)
}

func (r *journalRepository) GetMediaBatch(ctx context.Context, entityIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetMediaBatch(ctx, entityIDs, tx...)
}

func (r *journalRepository) DeleteMedia(ctx context.Context, id int64, entityID uuid.UUID, tx ...*sql.Tx) (string, error) {
	return r.dao.DeleteMedia(ctx, id, entityID, tx...)
}

func (r *journalRepository) CollectMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.CollectMediaPaths(ctx, entityID, tx...)
}
