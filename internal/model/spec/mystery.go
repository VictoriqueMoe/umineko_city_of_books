package spec

import (
	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
)

type (
	NewMystery struct {
		UserID             uuid.UUID
		Title              string
		Body               string
		Difficulty         string
		FreeForAll         bool
		KeepOpenAfterSolve bool
		Knox               dto.KnoxContract
	}

	NewMysteryWithClues struct {
		NewMystery
		Clues []NewClue
	}

	MysteryUpdate struct {
		ID                 uuid.UUID
		Title              string
		Body               string
		Difficulty         string
		FreeForAll         bool
		KeepOpenAfterSolve bool
		Knox               dto.KnoxContract
	}

	MysteryUpdateWithClues struct {
		MysteryUpdate
		Clues []NewClue
	}

	MysteryOwnerUpdate struct {
		ID         uuid.UUID
		UserID     uuid.UUID
		Title      string
		Body       string
		Difficulty string
	}

	MysteryListFilter struct {
		Sort           string
		Solved         *bool
		Limit          int
		Offset         int
		ExcludeUserIDs []uuid.UUID
	}

	MysteryUserListFilter struct {
		UserID uuid.UUID
		Limit  int
		Offset int
	}

	NewClue struct {
		Body      string
		TruthType string
		SortOrder int
		PlayerID  *uuid.UUID
	}

	NewMysteryClue struct {
		MysteryID uuid.UUID
		NewClue
	}

	MysteryClueUpdate struct {
		ClueID int
		Body   string
	}

	NewMysteryAttempt struct {
		MysteryID uuid.UUID
		UserID    uuid.UUID
		ParentID  *uuid.UUID
		Body      string
	}

	MysteryAttemptDeletion struct {
		ID     uuid.UUID
		UserID uuid.UUID
	}

	MysteryAttemptQuery struct {
		MysteryID uuid.UUID
		ViewerID  uuid.UUID
	}

	MysteryWinner struct {
		MysteryID uuid.UUID
		WinnerID  uuid.UUID
	}

	MysterySolverQuery struct {
		MysteryID uuid.UUID
		UserID    uuid.UUID
	}

	MysterySolve struct {
		MysteryID   uuid.UUID
		AttemptID   uuid.UUID
		LockMystery bool
	}

	MysteryPauseUpdate struct {
		MysteryID uuid.UUID
		Paused    bool
	}

	MysteryGmAwayUpdate struct {
		MysteryID uuid.UUID
		Away      bool
	}

	NewMysteryAttachment struct {
		MysteryID uuid.UUID
		FileURL   string
		FileName  string
		FileSize  int
	}

	MysteryAttachmentDeletion struct {
		ID        int64
		MysteryID uuid.UUID
	}

	MysteryDelete struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		AsAdmin bool
	}
)
