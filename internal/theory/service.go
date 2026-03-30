package theory

import (
	"context"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/credibility"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/quotefinder"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/theory/params"

	"github.com/google/uuid"
)

type (
	Service interface {
		CreateTheory(ctx context.Context, userID uuid.UUID, req dto.CreateTheoryRequest) (uuid.UUID, error)
		GetTheoryDetail(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*dto.TheoryDetailResponse, error)
		ListTheories(ctx context.Context, p params.ListParams, userID uuid.UUID) (*dto.TheoryListResponse, error)
		UpdateTheory(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateTheoryRequest) error
		DeleteTheory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		CreateResponse(ctx context.Context, theoryID uuid.UUID, userID uuid.UUID, req dto.CreateResponseRequest) (uuid.UUID, error)
		DeleteResponse(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		VoteTheory(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID, value int) error
		VoteResponse(ctx context.Context, userID uuid.UUID, responseID uuid.UUID, value int) error
	}

	service struct {
		repo           repository.TheoryRepository
		authz          authz.Service
		notifService   notification.Service
		settingsSvc    settings.Service
		credibilitySvc *credibility.Service
		quoteClient    *quotefinder.Client
	}
)

func NewService(
	repo repository.TheoryRepository,
	authzService authz.Service,
	notifService notification.Service,
	settingsSvc settings.Service,
	credibilitySvc *credibility.Service,
	quoteClient *quotefinder.Client,
) Service {
	return &service{
		repo:           repo,
		authz:          authzService,
		notifService:   notifService,
		settingsSvc:    settingsSvc,
		credibilitySvc: credibilitySvc,
		quoteClient:    quoteClient,
	}
}

func (s *service) CreateTheory(ctx context.Context, userID uuid.UUID, req dto.CreateTheoryRequest) (uuid.UUID, error) {
	logger.Log.Debug().Str("user_id", userID.String()).Str("title", req.Title).Msg("creating theory")

	limit := s.settingsSvc.GetInt(ctx, config.SettingMaxTheoriesPerDay)
	if limit > 0 {
		count, err := s.repo.CountUserTheoriesToday(ctx, userID)
		if err != nil {
			return uuid.Nil, err
		}
		if count >= limit {
			return uuid.Nil, ErrRateLimited
		}
	}

	return s.repo.Create(ctx, userID, req)
}

func (s *service) GetTheoryDetail(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*dto.TheoryDetailResponse, error) {
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

	if userID != uuid.Nil {
		vote, err := s.repo.GetUserTheoryVote(ctx, userID, id)
		if err != nil {
			logger.Log.Error().Err(err).Str("theory_id", id.String()).Msg("failed to get user theory vote")
		}
		detail.UserVote = vote
	}

	return detail, nil
}

func (s *service) ListTheories(ctx context.Context, p params.ListParams, userID uuid.UUID) (*dto.TheoryListResponse, error) {
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

func (s *service) UpdateTheory(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateTheoryRequest) error {
	return s.repo.Update(ctx, id, userID, req)
}

func (s *service) DeleteTheory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyTheory) {
		return s.repo.DeleteAsAdmin(ctx, id)
	}
	return s.repo.Delete(ctx, id, userID)
}

func (s *service) CreateResponse(ctx context.Context, theoryID uuid.UUID, userID uuid.UUID, req dto.CreateResponseRequest) (uuid.UUID, error) {
	logger.Log.Debug().Str("theory_id", theoryID.String()).Str("user_id", userID.String()).Str("side", req.Side).Msg("creating response")

	limit := s.settingsSvc.GetInt(ctx, config.SettingMaxResponsesPerDay)
	if limit > 0 {
		count, err := s.repo.CountUserResponsesToday(ctx, userID)
		if err != nil {
			return uuid.Nil, err
		}
		if count >= limit {
			return uuid.Nil, ErrRateLimited
		}
	}

	if req.ParentID == nil {
		authorID, err := s.repo.GetTheoryAuthorID(ctx, theoryID)
		if err != nil {
			return uuid.Nil, err
		}
		if authorID == userID {
			return uuid.Nil, ErrCannotRespondToOwnTheory
		}
	}

	id, err := s.repo.CreateResponse(ctx, theoryID, userID, req)
	if err != nil {
		return uuid.Nil, err
	}

	go func() {
		s.resolveEvidenceWeights(ctx, id)
		s.credibilitySvc.Recalculate(ctx, theoryID)
	}()

	go func() {
		if err := s.notifService.NotifyTheoryResponse(ctx, theoryID, userID); err != nil {
			logger.Log.Warn().Err(err).Msg("notify theory response failed")
		}
	}()

	if req.ParentID != nil {
		go func() {
			if err := s.notifService.NotifyResponseReply(ctx, *req.ParentID, theoryID, userID); err != nil {
				logger.Log.Warn().Err(err).Msg("notify response reply failed")
			}
		}()
	}

	return id, nil
}

func (s *service) resolveEvidenceWeights(ctx context.Context, responseID uuid.UUID) {
	evidence, err := s.repo.GetResponseEvidence(ctx, responseID)
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to get response evidence for weight resolution")
		return
	}

	for _, ev := range evidence {
		var q *quotefinder.Quote
		if ev.AudioID != "" {
			q, err = s.quoteClient.GetByAudioID(ev.AudioID)
		} else if ev.QuoteIndex != nil {
			q, err = s.quoteClient.GetByIndex(*ev.QuoteIndex)
		}
		if err != nil {
			logger.Log.Warn().Err(err).Int("evidence_id", ev.ID).Msg("failed to resolve quote for truth weight")
			continue
		}

		weight := quotefinder.TruthWeight(q)
		if weight != 1.0 {
			if err := s.repo.SetEvidenceTruthWeight(ctx, ev.ID, weight); err != nil {
				logger.Log.Error().Err(err).Int("evidence_id", ev.ID).Msg("failed to set truth weight")
			}
		}
	}
}

func (s *service) DeleteResponse(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	_, theoryID, _ := s.repo.GetResponseInfo(ctx, id)

	var err error
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyResponse) {
		err = s.repo.DeleteResponseAsAdmin(ctx, id)
	} else {
		err = s.repo.DeleteResponse(ctx, id, userID)
	}

	if err == nil && theoryID != uuid.Nil {
		go s.credibilitySvc.Recalculate(ctx, theoryID)
	}

	return err
}

func (s *service) VoteTheory(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID, value int) error {
	if err := s.repo.VoteTheory(ctx, userID, theoryID, value); err != nil {
		return err
	}

	if value == 1 {
		go func() {
			if err := s.notifService.NotifyTheoryUpvote(ctx, theoryID, userID); err != nil {
				logger.Log.Warn().Err(err).Msg("notify theory upvote failed")
			}
		}()
	}

	return nil
}

func (s *service) VoteResponse(ctx context.Context, userID uuid.UUID, responseID uuid.UUID, value int) error {
	if err := s.repo.VoteResponse(ctx, userID, responseID, value); err != nil {
		return err
	}

	if value == 1 {
		go func() {
			_, theoryID, err := s.repo.GetResponseInfo(ctx, responseID)
			if err != nil {
				logger.Log.Warn().Err(err).Msg("get response info for upvote notification failed")
				return
			}
			if err := s.notifService.NotifyResponseUpvote(ctx, responseID, theoryID, userID); err != nil {
				logger.Log.Warn().Err(err).Msg("notify response upvote failed")
			}
		}()
	}

	return nil
}
