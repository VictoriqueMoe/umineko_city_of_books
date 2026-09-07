package spec

import (
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/theory/params"

	"github.com/google/uuid"
)

type (
	NewTheory struct {
		UserID   uuid.UUID
		Title    string
		Body     string
		Episode  int
		Series   string
		Evidence []dto.EvidenceInput
	}

	TheoryUpdate struct {
		ID       uuid.UUID
		UserID   uuid.UUID
		Title    string
		Body     string
		Episode  int
		AsAdmin  bool
		Evidence []dto.EvidenceInput
	}

	NewTheoryResponse struct {
		TheoryID uuid.UUID
		UserID   uuid.UUID
		ParentID *uuid.UUID
		Side     string
		Body     string
		Evidence []dto.EvidenceInput
	}

	NewTheoryEvidence struct {
		TheoryID  uuid.UUID
		Evidence  dto.EvidenceInput
		SortOrder int
	}

	NewResponseEvidence struct {
		ResponseID uuid.UUID
		Evidence   dto.EvidenceInput
		SortOrder  int
	}

	TheoryEvidenceReplacement struct {
		TheoryID uuid.UUID
		Evidence []dto.EvidenceInput
	}

	TheoryListFilter struct {
		Params         params.ListParams
		ViewerID       uuid.UUID
		ExcludeUserIDs []uuid.UUID
	}

	TheoryResponseQuery struct {
		TheoryID uuid.UUID
		ViewerID uuid.UUID
	}

	TheoryVoteLookup struct {
		UserID   uuid.UUID
		TheoryID uuid.UUID
	}

	UserActivityQuery struct {
		UserID uuid.UUID
		Limit  int
		Offset int
	}

	TheoryCredibilityUpdate struct {
		TheoryID uuid.UUID
		Score    float64
	}

	TheoryRefutation struct {
		TheoryID   uuid.UUID
		ResponseID uuid.UUID
	}

	EvidenceTruthWeightUpdate struct {
		EvidenceID int
		Weight     float64
	}
)
