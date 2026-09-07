package spec

import (
	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/dto"
	fanficparams "umineko_city_of_books/internal/fanfic/params"

	"github.com/google/uuid"
)

type (
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
	}

	NewFanficWithDetails struct {
		NewFanfic

		Genres       []string
		Tags         []string
		Characters   []dto.FanficCharacter
		FirstChapter *NewChapter
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
		AsAdmin        bool
	}

	FanficUpdateWithDetails struct {
		FanficUpdate

		IsPairing  bool
		Genres     []string
		Tags       []string
		Characters []dto.FanficCharacter
	}

	FanficCoverUpdate struct {
		ID           uuid.UUID
		ImageURL     string
		ThumbnailURL string
	}

	FanficLookup struct {
		ID       uuid.UUID
		ViewerID uuid.UUID
	}

	FanficListFilter struct {
		ViewerID       uuid.UUID
		Params         fanficparams.ListParams
		ExcludeUserIDs []uuid.UUID
	}

	FanficUserListFilter struct {
		UserID   uuid.UUID
		ViewerID uuid.UUID
		Limit    int
		Offset   int
	}

	NewChapter struct {
		FanficID  uuid.UUID
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

	FanficChapterLookup struct {
		FanficID      uuid.UUID
		ChapterNumber int
	}

	FanficGenres struct {
		FanficID uuid.UUID
		Genres   []string
	}

	FanficTags struct {
		FanficID uuid.UUID
		Tags     []string
	}

	FanficCharacters struct {
		FanficID   uuid.UUID
		Characters []dto.FanficCharacter
		IsPairing  bool
	}

	NewFanficOCCharacter struct {
		Name      string
		CreatorID uuid.UUID
	}

	FanficUserRef struct {
		UserID   uuid.UUID
		FanficID uuid.UUID
	}

	FanficReadingProgress struct {
		UserID        uuid.UUID
		FanficID      uuid.UUID
		ChapterNumber int
	}

	FanficDelete struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		AsAdmin bool
	}

	FanficCommentDelete struct {
		CommentDeletion

		Audit audit.NewEntry
	}
)
