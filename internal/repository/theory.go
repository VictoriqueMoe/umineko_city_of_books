package repository

import (
	"context"
	"database/sql"
	"errors"

	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/theory/params"

	"github.com/google/uuid"
)

var (
	ErrRefutationRejected = errors.New("the response cannot refute this theory")
)

type (
	ResponseMeta struct {
		AuthorID uuid.UUID
		TheoryID uuid.UUID
		Side     string
		ParentID *uuid.UUID
	}

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

	TheoryDAO interface {
		InsertTheory(ctx context.Context, spec NewTheory, tx ...*sql.Tx) (*dto.TheoryDetailResponse, error)
		InsertTheoryEvidence(ctx context.Context, theoryID uuid.UUID, ev dto.EvidenceInput, sortOrder int, tx ...*sql.Tx) (*dto.EvidenceResponse, error)
		GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*dto.TheoryDetailResponse, error)
		List(ctx context.Context, p params.ListParams, userID uuid.UUID, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]dto.TheoryResponse, int, error)
		UpdateTheory(ctx context.Context, spec TheoryUpdate, tx ...*sql.Tx) error
		ReplaceTheoryEvidence(ctx context.Context, theoryID uuid.UUID, evidence []dto.EvidenceInput, tx ...*sql.Tx) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetEvidence(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) ([]dto.EvidenceResponse, error)
		InsertResponse(ctx context.Context, spec NewTheoryResponse, tx ...*sql.Tx) (*dto.ResponseResponse, error)
		InsertResponseEvidence(ctx context.Context, responseID uuid.UUID, ev dto.EvidenceInput, sortOrder int, tx ...*sql.Tx) (*dto.EvidenceResponse, error)
		DeleteResponse(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteResponseAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetResponses(ctx context.Context, theoryID uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) ([]dto.ResponseResponse, error)
		GetResponseEvidence(ctx context.Context, responseID uuid.UUID, tx ...*sql.Tx) ([]dto.EvidenceResponse, error)
		VoteTheory(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID, value int, tx ...*sql.Tx) error
		VoteResponse(ctx context.Context, userID uuid.UUID, responseID uuid.UUID, value int, tx ...*sql.Tx) error
		GetUserTheoryVote(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetTheoryAuthorID(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetResponseInfo(ctx context.Context, responseID uuid.UUID, tx ...*sql.Tx) (authorID uuid.UUID, theoryID uuid.UUID, err error)
		GetRecentActivityByUser(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]dto.ActivityItem, int, error)
		CountUserTheoriesToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		CountUserResponsesToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		UpdateCredibilityScore(ctx context.Context, theoryID uuid.UUID, score float64, tx ...*sql.Tx) error
		GetResponseEvidenceWeights(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (withLoveSum float64, withoutLoveSum float64, err error)
		RecomputeStatus(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) error
		MarkRefuted(ctx context.Context, theoryID uuid.UUID, responseID uuid.UUID, tx ...*sql.Tx) error
		GetResponseMeta(ctx context.Context, responseID uuid.UUID, tx ...*sql.Tx) (ResponseMeta, error)
		SetEvidenceTruthWeight(ctx context.Context, evidenceID int, weight float64, tx ...*sql.Tx) error
		GetTheoryTitle(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (string, error)
		GetTheorySeries(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (string, error)
	}

	TheoryRepository interface {
		TheoryDAO

		Create(ctx context.Context, spec NewTheory, tx ...*sql.Tx) (*dto.TheoryDetailResponse, error)
		Update(ctx context.Context, spec TheoryUpdate, tx ...*sql.Tx) error
		CreateResponse(ctx context.Context, spec NewTheoryResponse, tx ...*sql.Tx) (*dto.ResponseResponse, error)
	}
)

type theoryRepository struct {
	db    *sql.DB
	dao   TheoryDAO
	audit AuditLogRepository
}

func NewTheoryRepo(database *sql.DB, dao TheoryDAO, audit AuditLogRepository) TheoryRepository {
	return &theoryRepository{db: database, dao: dao, audit: audit}
}

func (r *theoryRepository) Create(ctx context.Context, spec NewTheory, tx ...*sql.Tx) (*dto.TheoryDetailResponse, error) {
	var created *dto.TheoryDetailResponse

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error
		created, err = r.dao.InsertTheory(ctx, spec, tx)
		if err != nil {
			return err
		}

		for i, ev := range spec.Evidence {
			stored, err := r.dao.InsertTheoryEvidence(ctx, created.ID, ev, i, tx)
			if err != nil {
				return err
			}

			created.Evidence = append(created.Evidence, *stored)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *theoryRepository) InsertTheory(ctx context.Context, spec NewTheory, tx ...*sql.Tx) (*dto.TheoryDetailResponse, error) {
	return r.dao.InsertTheory(ctx, spec, tx...)
}

func (r *theoryRepository) InsertTheoryEvidence(ctx context.Context, theoryID uuid.UUID, ev dto.EvidenceInput, sortOrder int, tx ...*sql.Tx) (*dto.EvidenceResponse, error) {
	return r.dao.InsertTheoryEvidence(ctx, theoryID, ev, sortOrder, tx...)
}

func (r *theoryRepository) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*dto.TheoryDetailResponse, error) {
	return r.dao.GetByID(ctx, id, tx...)
}

func (r *theoryRepository) List(ctx context.Context, p params.ListParams, userID uuid.UUID, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]dto.TheoryResponse, int, error) {
	return r.dao.List(ctx, p, userID, excludeUserIDs, tx...)
}

func (r *theoryRepository) Update(ctx context.Context, spec TheoryUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.UpdateTheory(ctx, spec, tx); err != nil {
			return err
		}

		return r.dao.ReplaceTheoryEvidence(ctx, spec.ID, spec.Evidence, tx)
	})
}

func (r *theoryRepository) UpdateTheory(ctx context.Context, spec TheoryUpdate, tx ...*sql.Tx) error {
	return r.dao.UpdateTheory(ctx, spec, tx...)
}

func (r *theoryRepository) ReplaceTheoryEvidence(ctx context.Context, theoryID uuid.UUID, evidence []dto.EvidenceInput, tx ...*sql.Tx) error {
	return r.dao.ReplaceTheoryEvidence(ctx, theoryID, evidence, tx...)
}

func (r *theoryRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Delete(ctx, id, userID, tx...)
}

func (r *theoryRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteAsAdmin(ctx, id, tx...)
}

func (r *theoryRepository) GetEvidence(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) ([]dto.EvidenceResponse, error) {
	return r.dao.GetEvidence(ctx, theoryID, tx...)
}

func (r *theoryRepository) CreateResponse(ctx context.Context, spec NewTheoryResponse, tx ...*sql.Tx) (*dto.ResponseResponse, error) {
	var created *dto.ResponseResponse

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error
		created, err = r.dao.InsertResponse(ctx, spec, tx)
		if err != nil {
			return err
		}

		for i, ev := range spec.Evidence {
			stored, err := r.dao.InsertResponseEvidence(ctx, created.ID, ev, i, tx)
			if err != nil {
				return err
			}

			created.Evidence = append(created.Evidence, *stored)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *theoryRepository) InsertResponse(ctx context.Context, spec NewTheoryResponse, tx ...*sql.Tx) (*dto.ResponseResponse, error) {
	return r.dao.InsertResponse(ctx, spec, tx...)
}

func (r *theoryRepository) InsertResponseEvidence(ctx context.Context, responseID uuid.UUID, ev dto.EvidenceInput, sortOrder int, tx ...*sql.Tx) (*dto.EvidenceResponse, error) {
	return r.dao.InsertResponseEvidence(ctx, responseID, ev, sortOrder, tx...)
}

func (r *theoryRepository) DeleteResponse(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteResponse(ctx, id, userID, tx...)
}

func (r *theoryRepository) DeleteResponseAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteResponseAsAdmin(ctx, id, tx...)
}

func (r *theoryRepository) GetResponses(ctx context.Context, theoryID uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) ([]dto.ResponseResponse, error) {
	return r.dao.GetResponses(ctx, theoryID, userID, tx...)
}

func (r *theoryRepository) GetResponseEvidence(ctx context.Context, responseID uuid.UUID, tx ...*sql.Tx) ([]dto.EvidenceResponse, error) {
	return r.dao.GetResponseEvidence(ctx, responseID, tx...)
}

func (r *theoryRepository) VoteTheory(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID, value int, tx ...*sql.Tx) error {
	return r.dao.VoteTheory(ctx, userID, theoryID, value, tx...)
}

func (r *theoryRepository) VoteResponse(ctx context.Context, userID uuid.UUID, responseID uuid.UUID, value int, tx ...*sql.Tx) error {
	return r.dao.VoteResponse(ctx, userID, responseID, value, tx...)
}

func (r *theoryRepository) GetUserTheoryVote(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.GetUserTheoryVote(ctx, userID, theoryID, tx...)
}

func (r *theoryRepository) GetTheoryAuthorID(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetTheoryAuthorID(ctx, theoryID, tx...)
}

func (r *theoryRepository) GetResponseInfo(ctx context.Context, responseID uuid.UUID, tx ...*sql.Tx) (authorID uuid.UUID, theoryID uuid.UUID, err error) {
	return r.dao.GetResponseInfo(ctx, responseID, tx...)
}

func (r *theoryRepository) GetRecentActivityByUser(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]dto.ActivityItem, int, error) {
	return r.dao.GetRecentActivityByUser(ctx, userID, limit, offset, tx...)
}

func (r *theoryRepository) CountUserTheoriesToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.CountUserTheoriesToday(ctx, userID, tx...)
}

func (r *theoryRepository) CountUserResponsesToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.CountUserResponsesToday(ctx, userID, tx...)
}

func (r *theoryRepository) UpdateCredibilityScore(ctx context.Context, theoryID uuid.UUID, score float64, tx ...*sql.Tx) error {
	return r.dao.UpdateCredibilityScore(ctx, theoryID, score, tx...)
}

func (r *theoryRepository) GetResponseEvidenceWeights(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (withLoveSum float64, withoutLoveSum float64, err error) {
	return r.dao.GetResponseEvidenceWeights(ctx, theoryID, tx...)
}

func (r *theoryRepository) SetEvidenceTruthWeight(ctx context.Context, evidenceID int, weight float64, tx ...*sql.Tx) error {
	return r.dao.SetEvidenceTruthWeight(ctx, evidenceID, weight, tx...)
}

func (r *theoryRepository) RecomputeStatus(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.RecomputeStatus(ctx, theoryID, tx...)
}

func (r *theoryRepository) MarkRefuted(ctx context.Context, theoryID uuid.UUID, responseID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.MarkRefuted(ctx, theoryID, responseID, tx...)
}

func (r *theoryRepository) GetResponseMeta(ctx context.Context, responseID uuid.UUID, tx ...*sql.Tx) (ResponseMeta, error) {
	return r.dao.GetResponseMeta(ctx, responseID, tx...)
}

func (r *theoryRepository) GetTheoryTitle(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (string, error) {
	return r.dao.GetTheoryTitle(ctx, theoryID, tx...)
}

func (r *theoryRepository) GetTheorySeries(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (string, error) {
	return r.dao.GetTheorySeries(ctx, theoryID, tx...)
}
