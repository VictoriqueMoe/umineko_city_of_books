package spec

import (
	"umineko_city_of_books/internal/audit"

	"github.com/google/uuid"
)

const (
	CommentKindPost         CommentKind = "post_comment"
	CommentKindArt          CommentKind = "art_comment"
	CommentKindAnnouncement CommentKind = "announcement_comment"
	CommentKindMystery      CommentKind = "mystery_comment"
	CommentKindShip         CommentKind = "ship_comment"
	CommentKindOC           CommentKind = "oc_comment"
	CommentKindFanfic       CommentKind = "fanfic_comment"
	CommentKindJournal      CommentKind = "journal_comment"
	CommentKindJournalEntry CommentKind = "journal_entry_comment"
	CommentKindSecret       CommentKind = "secret_comment"
)

type (
	CommentKind string

	CommentTarget struct {
		Kind        CommentKind
		ID          uuid.UUID
		Key         string
		EntryNumber int
	}

	CommentRef struct {
		Kind      CommentKind
		CommentID uuid.UUID
	}

	NewComment struct {
		Target   CommentTarget
		ParentID *uuid.UUID
		UserID   uuid.UUID
		Body     string
	}

	CommentQuery struct {
		Target         CommentTarget
		ViewerID       uuid.UUID
		Limit          int
		Offset         int
		ExcludeUserIDs []uuid.UUID
	}

	CommentUpdate struct {
		CommentRef
		UserID  uuid.UUID
		Body    string
		AsAdmin bool
	}

	CommentDeletion struct {
		CommentRef
		UserID  uuid.UUID
		AsAdmin bool
	}

	AuditedCommentUpdate struct {
		CommentUpdate
		Audit audit.NewEntry
	}

	AuditedCommentDeletion struct {
		CommentDeletion
		Audit audit.NewEntry
	}

	CommentLike struct {
		CommentRef
		UserID uuid.UUID
	}

	NewCommentMedia struct {
		Kind CommentKind
		NewMedia
	}

	CommentMediaURLUpdate struct {
		Kind CommentKind
		MediaURLUpdate
	}

	CommentMediaBatch struct {
		Kind       CommentKind
		CommentIDs []uuid.UUID
	}
)
