package repository

import (
	"context"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/theory/params"

	"github.com/google/uuid"
)

type (
	TheoryRepository interface {
		Create(ctx context.Context, userID uuid.UUID, req dto.CreateTheoryRequest) (uuid.UUID, error)
		GetByID(ctx context.Context, id uuid.UUID) (*dto.TheoryDetailResponse, error)
		List(ctx context.Context, p params.ListParams, userID uuid.UUID, excludeUserIDs []uuid.UUID) ([]dto.TheoryResponse, int, error)
		Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateTheoryRequest) error
		UpdateAsAdmin(ctx context.Context, id uuid.UUID, req dto.CreateTheoryRequest) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		GetEvidence(ctx context.Context, theoryID uuid.UUID) ([]dto.EvidenceResponse, error)
		CreateResponse(ctx context.Context, theoryID uuid.UUID, userID uuid.UUID, req dto.CreateResponseRequest) (uuid.UUID, error)
		DeleteResponse(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteResponseAsAdmin(ctx context.Context, id uuid.UUID) error
		GetResponses(ctx context.Context, theoryID uuid.UUID, userID uuid.UUID) ([]dto.ResponseResponse, error)
		GetResponseEvidence(ctx context.Context, responseID uuid.UUID) ([]dto.EvidenceResponse, error)
		VoteTheory(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID, value int) error
		VoteResponse(ctx context.Context, userID uuid.UUID, responseID uuid.UUID, value int) error
		GetUserTheoryVote(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID) (int, error)
		GetTheoryAuthorID(ctx context.Context, theoryID uuid.UUID) (uuid.UUID, error)
		GetResponseInfo(ctx context.Context, responseID uuid.UUID) (authorID uuid.UUID, theoryID uuid.UUID, err error)
		GetRecentActivityByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]dto.ActivityItem, int, error)
		CountUserTheoriesToday(ctx context.Context, userID uuid.UUID) (int, error)
		CountUserResponsesToday(ctx context.Context, userID uuid.UUID) (int, error)
		UpdateCredibilityScore(ctx context.Context, theoryID uuid.UUID, score float64) error
		GetResponseEvidenceWeights(ctx context.Context, theoryID uuid.UUID) (withLoveSum float64, withoutLoveSum float64, err error)
		SetEvidenceTruthWeight(ctx context.Context, evidenceID int, weight float64) error
		GetTheoryTitle(ctx context.Context, theoryID uuid.UUID) (string, error)
		GetTheorySeries(ctx context.Context, theoryID uuid.UUID) (string, error)
	}
)
