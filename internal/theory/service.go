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
	"umineko_city_of_books/internal/social"
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
		RefuteTheory(ctx context.Context, theoryID uuid.UUID, userID uuid.UUID, responseID uuid.UUID) error
		VoteTheory(ctx context.Context, userID uuid.UUID, theoryID uuid.UUID, value int) error
		VoteResponse(ctx context.Context, userID uuid.UUID, responseID uuid.UUID, value int) error
	}

	service struct {
		repo           repository.TheoryRepository
		userRepo       repository.UserRepository
		followRepo     repository.FollowRepository
		auditRepo      repository.AuditLogRepository
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
	followRepo repository.FollowRepository,
	auditRepo repository.AuditLogRepository,
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
		followRepo:     followRepo,
		auditRepo:      auditRepo,
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

func (s *service) audit(ctx context.Context, entry repository.NewAuditEntry) {
	if err := s.auditRepo.Create(ctx, entry); err != nil {
		logger.Log.Error().Err(err).Str("action", string(entry.Action)).Msg("failed to write audit log")
	}
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
	return u.DisplayLabel()
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

	created, err := s.repo.Create(ctx, repository.NewTheory{
		UserID:   userID,
		Title:    req.Title,
		Body:     req.Body,
		Episode:  req.Episode,
		Series:   req.Series,
		Evidence: req.Evidence,
	})
	if err != nil {
		return uuid.Nil, err
	}

	go social.ProcessMentions(s.userRepo, s.blockSvc, s.notifService, s.settingsSvc, userID, req.Body, created.ID, "theory", fmt.Sprintf("/theory/%s", created.ID))

	go notification.SendFollowerNotification(context.Background(), s.followRepo, s.notifService, notification.FollowerNotifyParams{
		ActorID:       userID,
		Type:          dto.NotifTheoryCreated,
		ReferenceID:   created.ID,
		ReferenceType: "theory",
		Action:        "posted a new theory",
		Title:         req.Title,
	})

	return created.ID, nil
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

	authorID, err := s.repo.GetTheoryAuthorID(ctx, id)
	if err != nil {
		return err
	}

	asAdmin := authorID != userID && s.authz.Can(ctx, userID, authz.PermEditAnyTheory)

	if err := s.repo.Update(ctx, repository.TheoryUpdate{
		ID:       id,
		UserID:   userID,
		Title:    req.Title,
		Body:     req.Body,
		Episode:  req.Episode,
		AsAdmin:  asAdmin,
		Evidence: req.Evidence,
	}); err != nil {
		return err
	}

	if asAdmin {
		go s.notifyContentEdited(ctx, id, "theory", id, userID)

		s.audit(ctx, repository.NewAuditEntry{
			ActorID:    userID,
			Action:     repository.AuditActionTheoryUpdateAdmin,
			TargetType: repository.AuditTargetTheory,
			TargetID:   id.String(),
			Details:    fmt.Sprintf("title=%q", req.Title),
			SubjectID:  authorID,
		})
	}

	return nil
}

func (s *service) notifyContentEdited(ctx context.Context, contentID uuid.UUID, contentType string, referenceID uuid.UUID, editorID uuid.UUID) {
	authorID, err := s.repo.GetTheoryAuthorID(ctx, contentID)
	if err != nil {
		return
	}
	notification.SendEditNotification(ctx, s.userRepo, s.notifService, notification.EditNotifyParams{
		AuthorID:      authorID,
		EditorID:      editorID,
		ContentType:   contentType,
		ReferenceID:   referenceID,
		ReferenceType: contentType,
		LinkPath:      fmt.Sprintf("/theory/%s", referenceID),
	})
}

func (s *service) DeleteTheory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.repo.GetTheoryAuthorID(ctx, id)
	if err != nil {
		return err
	}

	title, err := s.repo.GetTheoryTitle(ctx, id)
	if err != nil {
		return err
	}

	if s.authz.Can(ctx, userID, authz.PermDeleteAnyTheory) {
		err = s.repo.DeleteAsAdmin(ctx, id)
	} else {
		err = s.repo.Delete(ctx, id, userID)
	}
	if err != nil {
		return err
	}

	action := repository.AuditActionTheoryDelete
	if authorID != userID {
		action = repository.AuditActionTheoryDeleteAdmin
	}

	s.audit(ctx, repository.NewAuditEntry{
		ActorID:    userID,
		Action:     action,
		TargetType: repository.AuditTargetTheory,
		TargetID:   id.String(),
		Details:    fmt.Sprintf("title=%q", title),
		SubjectID:  authorID,
	})

	return nil
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

	created, err := s.repo.CreateResponse(ctx, repository.NewTheoryResponse{
		TheoryID: theoryID,
		UserID:   userID,
		ParentID: req.ParentID,
		Side:     req.Side,
		Body:     req.Body,
		Evidence: req.Evidence,
	})
	if err != nil {
		return uuid.Nil, err
	}

	go func() {
		s.resolveEvidenceWeights(ctx, theoryID, created.ID)
		s.credibilitySvc.Recalculate(ctx, theoryID)
		if err := s.repo.RecomputeStatus(ctx, theoryID); err != nil {
			logger.Log.Warn().Err(err).Str("theory_id", theoryID.String()).Msg("recompute theory status failed")
		}
	}()

	go social.ProcessMentions(s.userRepo, s.blockSvc, s.notifService, s.settingsSvc, userID, req.Body, theoryID, fmt.Sprintf("theory_response:%s", created.ID), fmt.Sprintf("/theory/%s#response-%s", theoryID, created.ID))

	go func() {
		authorID, err := s.repo.GetTheoryAuthorID(ctx, theoryID)
		if err != nil {
			return
		}
		title, _ := s.repo.GetTheoryTitle(ctx, theoryID)
		if err := s.notifService.Notify(ctx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifTheoryResponse,
			ReferenceID:   theoryID,
			ReferenceType: "theory",
			ActorID:       userID,
			EmailActor:    s.actorName(ctx, userID),
			EmailAction:   "responded to your theory",
			EmailTitle:    title,
			EmailLink:     fmt.Sprintf("/theory/%s#response-%s", theoryID, created.ID),
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
			if err := s.notifService.Notify(ctx, dto.NotifyParams{
				RecipientID:   recipientID,
				Type:          dto.NotifResponseReply,
				ReferenceID:   theoryID,
				ReferenceType: "theory",
				ActorID:       userID,
				EmailActor:    s.actorName(ctx, userID),
				EmailAction:   "replied to your response",
				EmailTitle:    title,
				EmailLink:     fmt.Sprintf("/theory/%s#response-%s", theoryID, created.ID),
			}); err != nil {
				logger.Log.Warn().Err(err).Msg("notify response reply failed")
			}
		}()
	}

	return created.ID, nil
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
	responseAuthorID, theoryID, _ := s.repo.GetResponseInfo(ctx, id)

	var err error
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyResponse) {
		err = s.repo.DeleteResponseAsAdmin(ctx, id)
	} else {
		err = s.repo.DeleteResponse(ctx, id, userID)
	}
	if err != nil {
		return err
	}

	if responseAuthorID != uuid.Nil && responseAuthorID != userID {
		s.audit(ctx, repository.NewAuditEntry{
			ActorID:    userID,
			Action:     repository.AuditActionTheoryResponseDeleteAdmin,
			TargetType: repository.AuditTargetTheoryResponse,
			TargetID:   id.String(),
			Details:    fmt.Sprintf("theory=%s", theoryID),
			SubjectID:  responseAuthorID,
		})
	}

	if theoryID != uuid.Nil {
		go func() {
			s.credibilitySvc.Recalculate(ctx, theoryID)
			if err := s.repo.RecomputeStatus(ctx, theoryID); err != nil {
				logger.Log.Warn().Err(err).Str("theory_id", theoryID.String()).Msg("recompute theory status failed")
			}
		}()
	}

	return nil
}

func (s *service) RefuteTheory(ctx context.Context, theoryID uuid.UUID, userID uuid.UUID, responseID uuid.UUID) error {
	theory, err := s.repo.GetByID(ctx, theoryID)
	if err != nil {
		return err
	}
	if theory == nil {
		return ErrTheoryNotFound
	}

	if theory.Author.ID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return ErrNotAuthor
	}
	if theory.Status == dto.TheoryStatusRefuted {
		return ErrAlreadyRefuted
	}

	meta, err := s.repo.GetResponseMeta(ctx, responseID)
	if err != nil {
		return ErrResponseNotOnTheory
	}
	if meta.TheoryID != theoryID {
		return ErrResponseNotOnTheory
	}
	if meta.ParentID != nil {
		return ErrRefutationMustBeTopLevel
	}
	if meta.Side != "without_love" {
		return ErrRefutationMustOppose
	}
	if meta.AuthorID == theory.Author.ID {
		return ErrCannotRefuteWithOwn
	}

	if err := s.repo.MarkRefuted(ctx, theoryID, responseID); err != nil {
		return err
	}

	by := "author"
	if theory.Author.ID != userID {
		by = "staff"
	}

	s.audit(ctx, repository.NewAuditEntry{
		ActorID:    userID,
		Action:     repository.AuditActionTheoryRefuted,
		TargetType: repository.AuditTargetTheory,
		TargetID:   theoryID.String(),
		Details:    fmt.Sprintf("response=%s by=%s", responseID, by),
		SubjectID:  meta.AuthorID,
	})

	go func() {
		bgCtx := context.Background()
		title, _ := s.repo.GetTheoryTitle(bgCtx, theoryID)
		if err := s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   meta.AuthorID,
			Type:          dto.NotifTheoryRefuted,
			ReferenceID:   theoryID,
			ReferenceType: fmt.Sprintf("theory_response:%s", responseID),
			ActorID:       userID,
			EmailActor:    s.actorName(bgCtx, userID),
			EmailAction:   "accepted your response as the refutation",
			EmailTitle:    title,
			EmailLink:     fmt.Sprintf("/theory/%s#response-%s", theoryID, responseID),
		}); err != nil {
			logger.Log.Warn().Err(err).Msg("notify theory refuted failed")
		}
	}()

	return nil
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
			if err := s.notifService.Notify(ctx, dto.NotifyParams{
				RecipientID:   authorID,
				Type:          dto.NotifTheoryUpvote,
				ReferenceID:   theoryID,
				ReferenceType: "theory",
				ActorID:       userID,
				EmailActor:    s.actorName(ctx, userID),
				EmailAction:   "upvoted your theory",
				EmailTitle:    title,
				EmailLink:     fmt.Sprintf("/theory/%s", theoryID),
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
			if err := s.notifService.Notify(ctx, dto.NotifyParams{
				RecipientID:   recipientID,
				Type:          dto.NotifResponseUpvote,
				ReferenceID:   theoryID,
				ReferenceType: "theory",
				ActorID:       userID,
				EmailActor:    s.actorName(ctx, userID),
				EmailAction:   "upvoted your response",
				EmailTitle:    title,
				EmailLink:     fmt.Sprintf("/theory/%s#response-%s", theoryID, responseID),
			}); err != nil {
				logger.Log.Warn().Err(err).Msg("notify response upvote failed")
			}
		}()
	}

	return nil
}
