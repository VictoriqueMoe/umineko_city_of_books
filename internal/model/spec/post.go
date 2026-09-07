package spec

import (
	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/model"

	"github.com/google/uuid"
)

type (
	NewPost struct {
		UserID        uuid.UUID
		Corner        string
		Body          string
		SharedContent *model.SharedContentRef
		Poll          *NewPoll
	}

	NewPoll struct {
		PostID          uuid.UUID
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
		Audit   audit.NewEntry
	}

	PostCommentDelete struct {
		CommentDeletion
		Audit audit.NewEntry
	}

	PostLookup struct {
		ID       uuid.UUID
		ViewerID uuid.UUID
	}

	PostUserPage struct {
		UserID   uuid.UUID
		ViewerID uuid.UUID
		Limit    int
		Offset   int
	}

	PostFeedQuery struct {
		ViewerID       uuid.UUID
		Corner         string
		Search         string
		Sort           string
		Seed           int
		Limit          int
		Offset         int
		ExcludeUserIDs []uuid.UUID
		ResolvedFilter string
	}

	PostFollowingFeedQuery struct {
		UserID         uuid.UUID
		Corner         string
		Sort           string
		Seed           int
		Limit          int
		Offset         int
		ExcludeUserIDs []uuid.UUID
	}

	SuggestionResolution struct {
		PostID     uuid.UUID
		ResolvedBy uuid.UUID
		Status     string
	}

	SharedContentBatchRef struct {
		ContentIDs  []string
		ContentType string
	}

	NewPostPoll struct {
		PostID          uuid.UUID
		DurationSeconds int
		ExpiresAt       string
	}

	NewPostPollOption struct {
		PollID    string
		Label     string
		SortOrder int
	}

	PostPollQuery struct {
		PostID   uuid.UUID
		ViewerID uuid.UUID
	}

	PostPollBatchQuery struct {
		PostIDs  []uuid.UUID
		ViewerID uuid.UUID
	}

	PostPollVote struct {
		PollID   uuid.UUID
		UserID   uuid.UUID
		OptionID int
	}
)
