package theory

import (
	"context"

	"fmt"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
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
		userRepo       repository.UserRepository
		authz          authz.Service
		blockSvc       block.Service
		notifService   notification.Service
		settingsSvc    settings.Service
		credibilitySvc *credibility.Service
		quoteClient    *quotefinder.Client
		contentFilter  *contentfilter.Manager
	}
)

func NewService(
	repo repository.TheoryRepository,
	userRepo repository.UserRepository,
	authzService authz.Service,
	blockSvc block.Service,
	notifService notification.Service,
	settingsSvc settings.Service,
	credibilitySvc *credibility.Service,
	quoteClient *quotefinder.Client,
	contentFilter *contentfilter.Manager,
) Service {
	return &service{
		repo:           repo,
		userRepo:       userRepo,
		authz:          authzService,
		blockSvc:       blockSvc,
		notifService:   notifService,
		settingsSvc:    settingsSvc,
		credibilitySvc: credibilitySvc,
		quoteClient:    quoteClient,
		contentFilter:  contentFilter,
	}
}

func (s *service) filterTexts(ctx context.Context, texts ...string) error {
	if s.contentFilter == nil {
		return nil
	}
	return s.contentFilter.Check(ctx, texts...)
}

func evidenceNotes(evidence []dto.EvidenceInput) []string {
	out := make([]string, 0, len(evidence))
	for _, e := range evidence {
		out = append(out, e.Note)
	}
	return out
}

func (s *service) actorName(ctx context.Context, userID uuid.UUID) string {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return "Someone"
	}
	return u.DisplayName
}

func (s *service) CreateTheory(ctx context.Context, userID uuid.UUID, req dto.CreateTheoryRequest) (uuid.UUID, error) {
	logger.Log.Debug().Str("user_id", userID.String()).Str("title", req.Title).Msg("creating theory")

	if err := s.filterTexts(ctx, append([]string{req.Title, req.Body}, evidenceNotes(req.Evidence)...)...); err != nil {
		return uuid.Nil, err
	}

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
	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, userID)
	theories, total, err := s.repo.List(ctx, p, userID, blockedIDs)
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
	if err := s.filterTexts(ctx, append([]string{req.Title, req.Body}, evidenceNotes(req.Evidence)...)...); err != nil {
		return err
	}
	if s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		if err := s.repo.UpdateAsAdmin(ctx, id, req); err != nil {
			return err
		}
		go s.notifyContentEdited(ctx, id, "theory", id, userID)
		return nil
	}
	return s.repo.Update(ctx, id, userID, req)
}

func (s *service) notifyContentEdited(ctx context.Context, contentID uuid.UUID, contentType string, referenceID uuid.UUID, editorID uuid.UUID) {
	authorID, err := s.repo.GetTheoryAuthorID(ctx, contentID)
	if err != nil {
		return
	}
	notification.SendEditNotification(ctx, s.userRepo, s.settingsSvc, s.notifService, notification.EditNotifyParams{
		AuthorID:      authorID,
		EditorID:      editorID,
		ContentType:   contentType,
		ReferenceID:   referenceID,
		ReferenceType: contentType,
		LinkPath:      fmt.Sprintf("/theory/%s", referenceID),
	})
}

func (s *service) DeleteTheory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyTheory) {
		return s.repo.DeleteAsAdmin(ctx, id)
	}
	return s.repo.Delete(ctx, id, userID)
}

func (s *service) CreateResponse(ctx context.Context, theoryID uuid.UUID, userID uuid.UUID, req dto.CreateResponseRequest) (uuid.UUID, error) {
	logger.Log.Debug().Str("theory_id", theoryID.String()).Str("user_id", userID.String()).Str("side", req.Side).Msg("creating response")

	if err := s.filterTexts(ctx, append([]string{req.Body}, evidenceNotes(req.Evidence)...)...); err != nil {
		return uuid.Nil, err
	}

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

	theoryAuthorID, err := s.repo.GetTheoryAuthorID(ctx, theoryID)
	if err != nil {
		return uuid.Nil, err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, theoryAuthorID); blocked {
		return uuid.Nil, block.ErrUserBlocked
	}

	if req.ParentID == nil {
		if theoryAuthorID == userID {
			return uuid.Nil, ErrCannotRespondToOwnTheory
		}
	}

	id, err := s.repo.CreateResponse(ctx, theoryID, userID, req)
	if err != nil {
		return uuid.Nil, err
	}

	go func() {
		s.resolveEvidenceWeights(ctx, theoryID, id)
		s.credibilitySvc.Recalculate(ctx, theoryID)
	}()

	go func() {
		authorID, err := s.repo.GetTheoryAuthorID(ctx, theoryID)
		if err != nil {
			return
		}
		title, _ := s.repo.GetTheoryTitle(ctx, theoryID)
		baseURL := s.settingsSvc.Get(ctx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/theory/%s#response-%s", baseURL, theoryID, id)
		subject, body := notification.NotifEmail(s.actorName(ctx, userID), "responded to your theory", title, linkURL)
		if err := s.notifService.Notify(ctx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifTheoryResponse,
			ReferenceID:   theoryID,
			ReferenceType: "theory",
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     body,
		}); err != nil {
			logger.Log.Warn().Err(err).Msg("notify theory response failed")
		}
	}()

	if req.ParentID != nil {
		go func() {
			recipientID, _, err := s.repo.GetResponseInfo(ctx, *req.ParentID)
			if err != nil {
				return
			}
			title, _ := s.repo.GetTheoryTitle(ctx, theoryID)
			baseURL := s.settingsSvc.Get(ctx, config.SettingBaseURL)
			linkURL := fmt.Sprintf("%s/theory/%s#response-%s", baseURL, theoryID, id)
			subject, body := notification.NotifEmail(s.actorName(ctx, userID), "replied to your response", title, linkURL)
			if err := s.notifService.Notify(ctx, dto.NotifyParams{
				RecipientID:   recipientID,
				Type:          dto.NotifResponseReply,
				ReferenceID:   theoryID,
				ReferenceType: "theory",
				ActorID:       userID,
				EmailSubject:  subject,
				EmailBody:     body,
			}); err != nil {
				logger.Log.Warn().Err(err).Msg("notify response reply failed")
			}
		}()
	}

	return id, nil
}

func (s *service) resolveEvidenceWeights(ctx context.Context, theoryID uuid.UUID, responseID uuid.UUID) {
	seriesStr, err := s.repo.GetTheorySeries(ctx, theoryID)
	if err != nil {
		logger.Log.Error().Err(err).Str("theory_id", theoryID.String()).Msg("failed to get theory series for weight resolution")
		return
	}

	series, err := quotefinder.ParseSeries(seriesStr)
	if err != nil {
		logger.Log.Warn().Err(err).Str("series", seriesStr).Msg("theory has invalid series, defaulting to umineko")
		series = quotefinder.SeriesUmineko
	}

	evidence, err := s.repo.GetResponseEvidence(ctx, responseID)
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to get response evidence for weight resolution")
		return
	}

	for _, ev := range evidence {
		var q *quotefinder.Quote
		if ev.AudioID != "" {
			q, err = s.quoteClient.GetByAudioID(series, ev.AudioID)
		} else if ev.QuoteIndex != nil {
			q, err = s.quoteClient.GetByIndex(series, *ev.QuoteIndex)
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
	authorID, err := s.repo.GetTheoryAuthorID(ctx, theoryID)
	if err != nil {
		return err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return block.ErrUserBlocked
	}

	if err := s.repo.VoteTheory(ctx, userID, theoryID, value); err != nil {
		return err
	}

	if value == 1 {
		go func() {
			authorID, err := s.repo.GetTheoryAuthorID(ctx, theoryID)
			if err != nil {
				return
			}
			title, _ := s.repo.GetTheoryTitle(ctx, theoryID)
			baseURL := s.settingsSvc.Get(ctx, config.SettingBaseURL)
			linkURL := fmt.Sprintf("%s/theory/%s", baseURL, theoryID)
			subject, body := notification.NotifEmail(s.actorName(ctx, userID), "upvoted your theory", title, linkURL)
			if err := s.notifService.Notify(ctx, dto.NotifyParams{
				RecipientID:   authorID,
				Type:          dto.NotifTheoryUpvote,
				ReferenceID:   theoryID,
				ReferenceType: "theory",
				ActorID:       userID,
				EmailSubject:  subject,
				EmailBody:     body,
			}); err != nil {
				logger.Log.Warn().Err(err).Msg("notify theory upvote failed")
			}
		}()
	}

	return nil
}

func (s *service) VoteResponse(ctx context.Context, userID uuid.UUID, responseID uuid.UUID, value int) error {
	respAuthorID, _, err := s.repo.GetResponseInfo(ctx, responseID)
	if err != nil {
		return err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, respAuthorID); blocked {
		return block.ErrUserBlocked
	}

	if err := s.repo.VoteResponse(ctx, userID, responseID, value); err != nil {
		return err
	}

	if value == 1 {
		go func() {
			recipientID, theoryID, err := s.repo.GetResponseInfo(ctx, responseID)
			if err != nil {
				return
			}
			title, _ := s.repo.GetTheoryTitle(ctx, theoryID)
			baseURL := s.settingsSvc.Get(ctx, config.SettingBaseURL)
			linkURL := fmt.Sprintf("%s/theory/%s#response-%s", baseURL, theoryID, responseID)
			subject, body := notification.NotifEmail(s.actorName(ctx, userID), "upvoted your response", title, linkURL)
			if err := s.notifService.Notify(ctx, dto.NotifyParams{
				RecipientID:   recipientID,
				Type:          dto.NotifResponseUpvote,
				ReferenceID:   theoryID,
				ReferenceType: "theory",
				ActorID:       userID,
				EmailSubject:  subject,
				EmailBody:     body,
			}); err != nil {
				logger.Log.Warn().Err(err).Msg("notify response upvote failed")
			}
		}()
	}

	return nil
}
