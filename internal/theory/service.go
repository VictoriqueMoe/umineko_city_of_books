package theory

import (
	"context"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/theory/params"
)

type (
	Service interface {
		CreateTheory(ctx context.Context, userID int, req dto.CreateTheoryRequest) (int64, error)
		GetTheoryDetail(ctx context.Context, id int, userID int) (*dto.TheoryDetailResponse, error)
		ListTheories(ctx context.Context, p params.ListParams, userID int) (*dto.TheoryListResponse, error)
		UpdateTheory(ctx context.Context, id, userID int, title, body string, episode int) error
		DeleteTheory(ctx context.Context, id, userID int) error
		CreateResponse(ctx context.Context, theoryID, userID int, req dto.CreateResponseRequest) (int64, error)
		DeleteResponse(ctx context.Context, id, userID int) error
		VoteTheory(ctx context.Context, userID, theoryID, value int) error
		VoteResponse(ctx context.Context, userID, responseID, value int) error
	}

	service struct {
		repo repository.TheoryRepository
	}
)

func NewService(repo repository.TheoryRepository) Service {
	return &service{repo: repo}
}

func (s *service) CreateTheory(ctx context.Context, userID int, req dto.CreateTheoryRequest) (int64, error) {
	return s.repo.Create(ctx, userID, req)
}

func (s *service) GetTheoryDetail(ctx context.Context, id int, userID int) (*dto.TheoryDetailResponse, error) {
	detail, err := s.repo.GetByID(ctx, id)
	if err != nil || detail == nil {
		return detail, err
	}

	evidence, err := s.repo.GetEvidence(ctx, id)
	if err != nil {
		return nil, err
	}
	detail.Evidence = evidence

	responses, err := s.repo.GetResponses(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	detail.Responses = responses

	if userID > 0 {
		vote, _ := s.repo.GetUserTheoryVote(ctx, userID, id)
		detail.UserVote = vote
	}

	return detail, nil
}

func (s *service) ListTheories(ctx context.Context, p params.ListParams, userID int) (*dto.TheoryListResponse, error) {
	theories, total, err := s.repo.List(ctx, p, userID)
	if err != nil {
		return nil, err
	}
	return &dto.TheoryListResponse{
		Theories: theories,
		Total:    total,
		Limit:    p.Limit,
		Offset:   p.Offset,
	}, nil
}

func (s *service) UpdateTheory(ctx context.Context, id, userID int, title, body string, episode int) error {
	return s.repo.Update(ctx, id, userID, title, body, episode)
}

func (s *service) DeleteTheory(ctx context.Context, id, userID int) error {
	return s.repo.Delete(ctx, id, userID)
}

func (s *service) CreateResponse(ctx context.Context, theoryID, userID int, req dto.CreateResponseRequest) (int64, error) {
	return s.repo.CreateResponse(ctx, theoryID, userID, req)
}

func (s *service) DeleteResponse(ctx context.Context, id, userID int) error {
	return s.repo.DeleteResponse(ctx, id, userID)
}

func (s *service) VoteTheory(ctx context.Context, userID, theoryID, value int) error {
	return s.repo.VoteTheory(ctx, userID, theoryID, value)
}

func (s *service) VoteResponse(ctx context.Context, userID, responseID, value int) error {
	return s.repo.VoteResponse(ctx, userID, responseID, value)
}
