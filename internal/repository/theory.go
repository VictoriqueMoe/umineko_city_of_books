package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model/spec"
)

type (
	TheoryRepository interface {
		dao.TheoryDAO

		Create(ctx context.Context, s spec.NewTheory, tx ...*sql.Tx) (*dto.TheoryDetailResponse, error)
		Update(ctx context.Context, s spec.TheoryUpdate, tx ...*sql.Tx) error
		CreateResponse(ctx context.Context, s spec.NewTheoryResponse, tx ...*sql.Tx) (*dto.ResponseResponse, error)
	}

	theoryRepository struct {
		db *sql.DB
		dao.TheoryDAO
		audit AuditLogRepository
	}
)

func NewTheoryRepo(database *sql.DB, theoryDAO dao.TheoryDAO, auditRepo AuditLogRepository) TheoryRepository {
	return &theoryRepository{db: database, TheoryDAO: theoryDAO, audit: auditRepo}
}

func (r *theoryRepository) Create(ctx context.Context, s spec.NewTheory, tx ...*sql.Tx) (*dto.TheoryDetailResponse, error) {
	var created *dto.TheoryDetailResponse

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.TheoryDAO.InsertTheory(ctx, s, tx)
		if err != nil {
			return err
		}

		for i, ev := range s.Evidence {
			stored, err := r.TheoryDAO.InsertTheoryEvidence(ctx, spec.NewTheoryEvidence{
				TheoryID:  created.ID,
				Evidence:  ev,
				SortOrder: i,
			}, tx)
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

func (r *theoryRepository) Update(ctx context.Context, s spec.TheoryUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.TheoryDAO.UpdateTheory(ctx, s, tx); err != nil {
			return err
		}

		return r.TheoryDAO.ReplaceTheoryEvidence(ctx, spec.TheoryEvidenceReplacement{
			TheoryID: s.ID,
			Evidence: s.Evidence,
		}, tx)
	})
}

func (r *theoryRepository) CreateResponse(ctx context.Context, s spec.NewTheoryResponse, tx ...*sql.Tx) (*dto.ResponseResponse, error) {
	var created *dto.ResponseResponse

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.TheoryDAO.InsertResponse(ctx, s, tx)
		if err != nil {
			return err
		}

		for i, ev := range s.Evidence {
			stored, err := r.TheoryDAO.InsertResponseEvidence(ctx, spec.NewResponseEvidence{
				ResponseID: created.ID,
				Evidence:   ev,
				SortOrder:  i,
			}, tx)
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
