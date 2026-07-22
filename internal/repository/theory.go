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

type theoryRepository struct {
	dao TheoryRepository
}

func NewTheoryRepo(dao TheoryRepository) TheoryRepository {
	return &theoryRepository{dao: dao}
}

func (r *theoryRepository) Create(ctx context.Context, userID uuid.UUID, req dto.CreateTheoryRequest) (uuid.UUID, error) {
	return r.dao.Create(ctx, userID, req)
}

func (r *theoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*dto.TheoryDetailResponse, error) {
	return r.dao.GetByID(ctx, id)
}

func (r *theoryRepository) List(ctx context.Context, p params.ListParams, userID uuid.UUID, excludeUserIDs []uuid.UUID) ([]dto.TheoryResponse, int, error) {
	return r.dao.List(ctx, p, userID, excludeUserIDs)
}

func (r *theoryRepository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateTheoryRequest) error {
	return r.dao.Update(ctx, id, userID, req)
}

func (r *theoryRepository) UpdateAsAdmin(ctx context.Context, id uuid.UUID, req dto.CreateTheoryRequest) error {
	return r.dao.UpdateAsAdmin(ctx, id, req)
}

func (r *theoryRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.Delete(ctx, id, userID)
}

func (r *theoryRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteAsAdmin(ctx, id)
}

func (r *theoryRepository) GetEvidence(ctx context.Context, theoryID uuid.UUID) ([]dto.EvidenceResponse, error) {
	return r.dao.GetEvidence(ctx, theoryID)
}

func (r *theoryRepository) CreateResponse(ctx context.Context, theoryID uuid.UUID, userID uuid.UUID, req dto.CreateResponseRequest) (uuid.UUID, error) {
	return r.dao.CreateResponse(ctx, theoryID, userID, req)
}

func (r *theoryRepository) DeleteResponse(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.DeleteResponse(ctx, id, userID)
}

func (r *theoryRepository) DeleteResponseAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteResponseAsAdmin(ctx, id)
}

func (r *theoryRepository) GetResponses(ctx context.Context, theoryID uuid.UUID, userID uuid.UUID) ([]dto.ResponseResponse, error) {
	return r.dao.GetResponses(ctx, theoryID, userID)
}

func (r *theoryRepository) GetResponseEvidence(ctx context.Context, responseID uuid.UUID) ([]dto.EvidenceResponse, error) {
	return r.dao.GetResponseEvidence(ctx, responseID)
}

func (r *theoryRepository) VoteTheory(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID, value int) error {
	return r.dao.VoteTheory(ctx, userID, theoryID, value)
}

func (r *theoryRepository) VoteResponse(ctx context.Context, userID uuid.UUID, responseID uuid.UUID, value int) error {
	return r.dao.VoteResponse(ctx, userID, responseID, value)
}

func (r *theoryRepository) GetUserTheoryVote(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID) (int, error) {
	return r.dao.GetUserTheoryVote(ctx, userID, theoryID)
}

func (r *theoryRepository) GetTheoryAuthorID(ctx context.Context, theoryID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetTheoryAuthorID(ctx, theoryID)
}

func (r *theoryRepository) GetResponseInfo(ctx context.Context, responseID uuid.UUID) (authorID uuid.UUID, theoryID uuid.UUID, err error) {
	return r.dao.GetResponseInfo(ctx, responseID)
}

func (r *theoryRepository) GetRecentActivityByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]dto.ActivityItem, int, error) {
	return r.dao.GetRecentActivityByUser(ctx, userID, limit, offset)
}

func (r *theoryRepository) CountUserTheoriesToday(ctx context.Context, userID uuid.UUID) (int, error) {
	return r.dao.CountUserTheoriesToday(ctx, userID)
}

func (r *theoryRepository) CountUserResponsesToday(ctx context.Context, userID uuid.UUID) (int, error) {
	return r.dao.CountUserResponsesToday(ctx, userID)
}

func (r *theoryRepository) UpdateCredibilityScore(ctx context.Context, theoryID uuid.UUID, score float64) error {
	return r.dao.UpdateCredibilityScore(ctx, theoryID, score)
}

func (r *theoryRepository) GetResponseEvidenceWeights(ctx context.Context, theoryID uuid.UUID) (withLoveSum float64, withoutLoveSum float64, err error) {
	return r.dao.GetResponseEvidenceWeights(ctx, theoryID)
}

func (r *theoryRepository) SetEvidenceTruthWeight(ctx context.Context, evidenceID int, weight float64) error {
	return r.dao.SetEvidenceTruthWeight(ctx, evidenceID, weight)
}

func (r *theoryRepository) GetTheoryTitle(ctx context.Context, theoryID uuid.UUID) (string, error) {
	return r.dao.GetTheoryTitle(ctx, theoryID)
}

func (r *theoryRepository) GetTheorySeries(ctx context.Context, theoryID uuid.UUID) (string, error) {
	return r.dao.GetTheorySeries(ctx, theoryID)
}
