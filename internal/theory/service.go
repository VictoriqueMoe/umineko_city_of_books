package theory

import (
	"context"
	"log"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/notification"
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
		repo         repository.TheoryRepository
		notifService notification.Service
	}
)

func NewService(repo repository.TheoryRepository, notifService notification.Service) Service {
	return &service{
		repo:         repo,
		notifService: notifService,
	}
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
	id, err := s.repo.CreateResponse(ctx, theoryID, userID, req)
	if err != nil {
		return 0, err
	}

	go func() {
		if err := s.notifService.NotifyTheoryResponse(ctx, theoryID, userID); err != nil {
			log.Printf("[theory] notify theory response failed: %v", err)
		}
	}()

	if req.ParentID != nil {
		go func() {
			if err := s.notifService.NotifyResponseReply(ctx, *req.ParentID, theoryID, userID); err != nil {
				log.Printf("[theory] notify response reply failed: %v", err)
			}
		}()
	}

	return id, nil
}

func (s *service) DeleteResponse(ctx context.Context, id, userID int) error {
	return s.repo.DeleteResponse(ctx, id, userID)
}

func (s *service) VoteTheory(ctx context.Context, userID, theoryID, value int) error {
	if err := s.repo.VoteTheory(ctx, userID, theoryID, value); err != nil {
		return err
	}

	if value == 1 {
		go func() {
			if err := s.notifService.NotifyTheoryUpvote(ctx, theoryID, userID); err != nil {
				log.Printf("[theory] notify theory upvote failed: %v", err)
			}
		}()
	}

	return nil
}

func (s *service) VoteResponse(ctx context.Context, userID, responseID, value int) error {
	if err := s.repo.VoteResponse(ctx, userID, responseID, value); err != nil {
		return err
	}

	if value == 1 {
		go func() {
			_, theoryID, err := s.repo.GetResponseInfo(ctx, responseID)
			if err != nil {
				log.Printf("[theory] get response info for upvote notification failed: %v", err)
				return
			}
			if err := s.notifService.NotifyResponseUpvote(ctx, responseID, theoryID, userID); err != nil {
				log.Printf("[theory] notify response upvote failed: %v", err)
			}
		}()
	}

	return nil
}
