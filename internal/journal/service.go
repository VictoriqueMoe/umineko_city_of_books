package journal

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/journal/params"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/utils"

	"github.com/google/uuid"
)

const archiveAfter = 7 * 24 * time.Hour

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

type (
	Service interface {
		CreateJournal(ctx context.Context, userID uuid.UUID, req dto.CreateJournalRequest) (uuid.UUID, error)
		GetJournalDetail(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.JournalDetailResponse, error)
		ListJournals(ctx context.Context, p params.ListParams, viewerID uuid.UUID) (*dto.JournalListResponse, error)
		ListJournalsByUser(ctx context.Context, authorID uuid.UUID, viewerID uuid.UUID, limit, offset int) (*dto.JournalListResponse, error)
		ListFollowedByUser(ctx context.Context, followerID uuid.UUID, viewerID uuid.UUID, page bounds.Page) (*dto.JournalListResponse, error)
		UpdateJournal(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateJournalRequest) error
		DeleteJournal(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

		CreateEntry(ctx context.Context, journalID uuid.UUID, userID uuid.UUID, req dto.CreateJournalEntryRequest) (uuid.UUID, int, error)
		GetEntry(ctx context.Context, journalID uuid.UUID, entryNumber int, viewerID uuid.UUID) (*dto.JournalEntryResponse, []dto.JournalCommentResponse, error)
		UpdateEntry(ctx context.Context, entryID uuid.UUID, userID uuid.UUID, req dto.UpdateJournalEntryRequest) error
		DeleteEntry(ctx context.Context, entryID uuid.UUID, userID uuid.UUID) error

		CreateComment(ctx context.Context, journalID uuid.UUID, userID uuid.UUID, entryID *uuid.UUID, parentID *uuid.UUID, body string) (uuid.UUID, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		LikeComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		UnlikeComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error)
		UploadEntryMedia(ctx context.Context, entryID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error)
		DeleteEntryMedia(ctx context.Context, entryID uuid.UUID, mediaID int64, userID uuid.UUID) error

		FollowJournal(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		UnfollowJournal(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

		ArchiveStale(ctx context.Context) (int, error)
	}

	service struct {
		repo          repository.JournalRepository
		userRepo      repository.UserRepository
		auditRepo     repository.AuditLogRepository
		authz         authz.Service
		blockSvc      block.Service
		notifService  notification.Service
		settingsSvc   settings.Service
		uploadSvc     upload.Service
		uploader      *media.Uploader
		contentFilter *contentfilter.Manager
	}
)

func NewService(
	repo repository.JournalRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditLogRepository,
	authzService authz.Service,
	blockSvc block.Service,
	notifService notification.Service,
	uploadSvc upload.Service,
	mediaProc *media.Processor,
	settingsSvc settings.Service,
	contentFilter *contentfilter.Manager,
) Service {
	return &service{
		repo:          repo,
		userRepo:      userRepo,
		auditRepo:     auditRepo,
		authz:         authzService,
		blockSvc:      blockSvc,
		notifService:  notifService,
		settingsSvc:   settingsSvc,
		uploadSvc:     uploadSvc,
		uploader:      media.NewUploader(uploadSvc, settingsSvc, mediaProc),
		contentFilter: contentFilter,
	}
}

func (s *service) filterTexts(ctx context.Context, texts ...string) error {
	if s.contentFilter == nil {
		return nil
	}
	return s.contentFilter.Check(ctx, texts...)
}

func (s *service) writeAudit(ctx context.Context, entry repository.NewAuditEntry) {
	if err := s.auditRepo.Create(ctx, entry); err != nil {
		logger.Log.Error().Err(err).Str("action", string(entry.Action)).Msg("failed to write audit log")
	}
}

func (s *service) actorName(ctx context.Context, userID uuid.UUID) string {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return "Someone"
	}
	return u.DisplayLabel()
}

func countWords(html string) int {
	text := htmlTagRe.ReplaceAllString(html, " ")
	return len(strings.Fields(text))
}

func sameOptional(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}

	return *a == *b
}

func changedDetails(changed []string) string {
	if len(changed) == 0 {
		return "changed=none"
	}

	return "changed=" + strings.Join(changed, ",")
}

func journalUpdateDetails(before *dto.JournalResponse, req dto.CreateJournalRequest) string {
	if before == nil {
		return ""
	}

	var changed []string

	if before.Title != req.Title {
		changed = append(changed, "title")
	}

	if before.Work != req.Work {
		changed = append(changed, "work")
	}

	return changedDetails(changed)
}

func journalEntryUpdateDetails(before *repository.JournalEntryRow, title *string, body string, isDraft bool) string {
	var changed []string

	if !sameOptional(before.Title, title) {
		changed = append(changed, "title")
	}

	if before.Body != body {
		changed = append(changed, "body")
	}

	if before.IsDraft != isDraft {
		changed = append(changed, "is_draft")
	}

	return changedDetails(changed)
}

func (s *service) CreateJournal(ctx context.Context, userID uuid.UUID, req dto.CreateJournalRequest) (uuid.UUID, error) {
	if strings.TrimSpace(req.Title) == "" {
		return uuid.Nil, ErrEmptyTitle
	}
	if err := s.filterTexts(ctx, req.Title); err != nil {
		return uuid.Nil, err
	}

	limit := s.settingsSvc.GetInt(ctx, config.SettingMaxJournalsPerDay)
	if limit > 0 {
		count, err := s.repo.CountUserJournalsToday(ctx, userID)
		if err != nil {
			return uuid.Nil, err
		}
		if count >= limit {
			return uuid.Nil, ErrRateLimited
		}
	}

	created, err := s.repo.Create(ctx, userID, req)
	if err != nil {
		return uuid.Nil, err
	}

	return created.ID, nil
}

func (s *service) GetJournalDetail(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.JournalDetailResponse, error) {
	journal, err := s.repo.GetByID(ctx, id, viewerID)
	if err != nil || journal == nil {
		if journal == nil && err == nil {
			return nil, ErrNotFound
		}
		return nil, err
	}

	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)
	commentRows, _, err := s.repo.GetComments(ctx, id, viewerID, 500, 0, blockedIDs)
	if err != nil {
		return nil, err
	}

	commentIDs := make([]uuid.UUID, len(commentRows))
	for i := range commentRows {
		commentIDs[i] = commentRows[i].ID
	}
	mediaMap, _ := s.repo.GetCommentMediaBatch(ctx, commentIDs)

	flatComments := make([]dto.JournalCommentResponse, len(commentRows))
	for i, c := range commentRows {
		flatComments[i] = repository.JournalCommentToDTO(c, mediaMap[c.ID], journal.Author.ID)
	}

	tree := utils.BuildTree(flatComments,
		func(c dto.JournalCommentResponse) uuid.UUID { return c.ID },
		func(c dto.JournalCommentResponse) *uuid.UUID { return c.ParentID },
		func(c *dto.JournalCommentResponse, replies []dto.JournalCommentResponse) { c.Replies = replies },
	)

	entryRows, err := s.repo.ListEntries(ctx, id)
	if err != nil {
		return nil, err
	}
	isAuthor := viewerID != uuid.Nil && viewerID == journal.Author.ID
	entries := make([]dto.JournalEntrySummary, 0, len(entryRows))
	for _, e := range entryRows {
		if e.IsDraft && !isAuthor {
			continue
		}
		entries = append(entries, repository.JournalEntrySummaryToDTO(e))
	}

	var latestEntry *dto.JournalEntryResponse
	if journal.LatestEntryNumber != nil {
		entry, err := s.repo.GetEntry(ctx, id, *journal.LatestEntryNumber)
		if err != nil {
			return nil, err
		}
		if entry != nil {
			entryMediaMap, _ := s.repo.GetMediaBatch(ctx, []uuid.UUID{entry.ID})
			latestEntry = new(repository.JournalEntryToDTO(entry, entryMediaMap[entry.ID]))
		}
	}

	return &dto.JournalDetailResponse{
		JournalResponse: *journal,
		Entries:         entries,
		LatestEntry:     latestEntry,
		Comments:        tree,
	}, nil
}

func (s *service) ListJournals(ctx context.Context, p params.ListParams, viewerID uuid.UUID) (*dto.JournalListResponse, error) {
	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)
	journals, total, err := s.repo.List(ctx, p, viewerID, blockedIDs)
	if err != nil {
		return nil, err
	}
	return &dto.JournalListResponse{
		Journals: journals,
		Total:    total,
		Limit:    p.Limit,
		Offset:   p.Offset,
	}, nil
}

func (s *service) ListJournalsByUser(ctx context.Context, authorID uuid.UUID, viewerID uuid.UUID, limit, offset int) (*dto.JournalListResponse, error) {
	p := params.NewListParams("new", "", authorID, "", true, limit, offset)
	return s.ListJournals(ctx, p, viewerID)
}

func (s *service) ListFollowedByUser(ctx context.Context, followerID uuid.UUID, viewerID uuid.UUID, page bounds.Page) (*dto.JournalListResponse, error) {
	journals, total, err := s.repo.ListFollowedByUser(ctx, followerID, viewerID, page.Limit(), page.Offset())
	if err != nil {
		return nil, err
	}

	return &dto.JournalListResponse{
		Journals: journals,
		Total:    total,
		Limit:    page.Limit(),
		Offset:   page.Offset(),
	}, nil
}

func (s *service) UpdateJournal(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateJournalRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return ErrEmptyTitle
	}
	if err := s.filterTexts(ctx, req.Title); err != nil {
		return err
	}

	authorID, err := s.repo.GetAuthorID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	asAdmin := authorID != userID && s.authz.Can(ctx, userID, authz.PermEditAnyJournal)

	var before *dto.JournalResponse
	if asAdmin {
		before, _ = s.repo.GetByID(ctx, id, userID)
	}

	if err := s.repo.Update(ctx, repository.JournalUpdate{
		ID:      id,
		UserID:  userID,
		Title:   req.Title,
		Work:    req.Work,
		AsAdmin: asAdmin,
	}); err != nil {
		return err
	}

	if asAdmin {
		s.writeAudit(ctx, repository.NewAuditEntry{
			ActorID:    userID,
			Action:     repository.AuditActionJournalUpdateAdmin,
			TargetType: repository.AuditTargetJournal,
			TargetID:   id.String(),
			Details:    journalUpdateDetails(before, req),
			SubjectID:  authorID,
		})
	}

	return nil
}

func (s *service) DeleteJournal(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.repo.GetAuthorID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	asAdmin := authorID != userID && s.authz.Can(ctx, userID, authz.PermDeleteAnyJournal)
	title, _ := s.repo.GetTitle(ctx, id)

	paths, err := s.repo.Delete(ctx, id, userID, asAdmin)
	if err != nil {
		return err
	}

	action := repository.AuditActionJournalDelete
	if asAdmin {
		action = repository.AuditActionJournalDeleteAdmin
	}

	details := ""
	if title != "" {
		details = "title=" + title
	}

	s.writeAudit(ctx, repository.NewAuditEntry{
		ActorID:    userID,
		Action:     action,
		TargetType: repository.AuditTargetJournal,
		TargetID:   id.String(),
		Details:    details,
		SubjectID:  authorID,
	})

	s.uploadSvc.Delete(paths...)

	return nil
}

func (s *service) CreateEntry(ctx context.Context, journalID uuid.UUID, userID uuid.UUID, req dto.CreateJournalEntryRequest) (uuid.UUID, int, error) {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return uuid.Nil, 0, ErrEmptyBody
	}
	titleTrim := strings.TrimSpace(req.Title)
	if err := s.filterTexts(ctx, titleTrim, body); err != nil {
		return uuid.Nil, 0, err
	}

	authorID, err := s.repo.GetAuthorID(ctx, journalID)
	if err != nil {
		return uuid.Nil, 0, ErrNotFound
	}

	asAdmin := authorID != userID
	if asAdmin && !s.authz.Can(ctx, userID, authz.PermEditAnyJournal) {
		return uuid.Nil, 0, ErrNotAuthor
	}

	nextNumber, err := s.repo.GetNextEntryNumber(ctx, journalID)
	if err != nil {
		return uuid.Nil, 0, err
	}

	var titlePtr *string
	if titleTrim != "" {
		titlePtr = &titleTrim
	}

	created, err := s.repo.CreateEntry(ctx, repository.NewJournalEntry{
		JournalID:   journalID,
		EntryNumber: nextNumber,
		Title:       titlePtr,
		Body:        body,
		WordCount:   countWords(body),
		IsDraft:     req.IsDraft,
	})
	if err != nil {
		return uuid.Nil, 0, err
	}

	if asAdmin {
		s.writeAudit(ctx, repository.NewAuditEntry{
			ActorID:    userID,
			Action:     repository.AuditActionJournalEntryCreateAdmin,
			TargetType: repository.AuditTargetJournalEntry,
			TargetID:   created.ID.String(),
			Details:    fmt.Sprintf("journal_id=%s,entry_number=%d,is_draft=%t", journalID, nextNumber, req.IsDraft),
			SubjectID:  authorID,
		})
	}

	if !req.IsDraft {
		go s.notifyEntryPublished(journalID, nextNumber, userID)
	}

	return created.ID, nextNumber, nil
}

func (s *service) eligibleFollowerIDs(ctx context.Context, journalID uuid.UUID, actorUserID uuid.UUID) ([]uuid.UUID, error) {
	followerIDs, err := s.repo.GetFollowerIDs(ctx, journalID)
	if err != nil {
		return nil, err
	}

	blockedSet := make(map[uuid.UUID]struct{})
	if blockedIDs, err := s.blockSvc.GetBlockedIDs(ctx, actorUserID); err == nil {
		for i := range blockedIDs {
			blockedSet[blockedIDs[i]] = struct{}{}
		}
	}

	eligible := make([]uuid.UUID, 0, len(followerIDs))
	for _, followerID := range followerIDs {
		if followerID == actorUserID {
			continue
		}
		if _, isBlocked := blockedSet[followerID]; isBlocked {
			continue
		}
		eligible = append(eligible, followerID)
	}

	return eligible, nil
}

func (s *service) notifyEntryPublished(journalID uuid.UUID, entryNumber int, actorUserID uuid.UUID) {
	bgCtx := context.Background()
	title, _ := s.repo.GetTitle(bgCtx, journalID)
	actor := s.actorName(bgCtx, actorUserID)

	followerIDs, err := s.eligibleFollowerIDs(bgCtx, journalID, actorUserID)
	if err != nil {
		logger.Log.Error().Err(err).Msg("get follower ids failed on entry publish")
		return
	}

	notifyParams := make([]dto.NotifyParams, 0, len(followerIDs))
	for _, followerID := range followerIDs {
		notifyParams = append(notifyParams, dto.NotifyParams{
			RecipientID:   followerID,
			Type:          dto.NotifJournalUpdate,
			ReferenceID:   journalID,
			ReferenceType: fmt.Sprintf("journal_entry:%d", entryNumber),
			ActorID:       actorUserID,
			EmailActor:    actor,
			EmailAction:   "posted a new entry on",
			EmailTitle:    title,
			EmailLink:     fmt.Sprintf("/journals/%s/entry/%d", journalID, entryNumber),
		})
	}
	s.notifService.NotifyMany(bgCtx, notifyParams)
}

func (s *service) GetEntry(ctx context.Context, journalID uuid.UUID, entryNumber int, viewerID uuid.UUID) (*dto.JournalEntryResponse, []dto.JournalCommentResponse, error) {
	entry, err := s.repo.GetEntry(ctx, journalID, entryNumber)
	if err != nil {
		return nil, nil, err
	}
	if entry == nil {
		return nil, nil, ErrEntryNotFound
	}

	authorID, err := s.repo.GetAuthorID(ctx, journalID)
	if err != nil {
		return nil, nil, ErrNotFound
	}

	if entry.IsDraft && viewerID != authorID {
		return nil, nil, ErrEntryNotFound
	}

	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)
	commentRows, _, err := s.repo.GetEntryComments(ctx, entry.ID, viewerID, 500, 0, blockedIDs)
	if err != nil {
		return nil, nil, err
	}

	commentIDs := make([]uuid.UUID, len(commentRows))
	for i := range commentRows {
		commentIDs[i] = commentRows[i].ID
	}
	mediaMap, _ := s.repo.GetCommentMediaBatch(ctx, commentIDs)

	flatComments := make([]dto.JournalCommentResponse, len(commentRows))
	for i, c := range commentRows {
		flatComments[i] = repository.JournalCommentToDTO(c, mediaMap[c.ID], authorID)
	}

	tree := utils.BuildTree(flatComments,
		func(c dto.JournalCommentResponse) uuid.UUID { return c.ID },
		func(c dto.JournalCommentResponse) *uuid.UUID { return c.ParentID },
		func(c *dto.JournalCommentResponse, replies []dto.JournalCommentResponse) { c.Replies = replies },
	)

	entryMediaMap, _ := s.repo.GetMediaBatch(ctx, []uuid.UUID{entry.ID})
	return new(repository.JournalEntryToDTO(entry, entryMediaMap[entry.ID])), tree, nil
}

func (s *service) UpdateEntry(ctx context.Context, entryID uuid.UUID, userID uuid.UUID, req dto.UpdateJournalEntryRequest) error {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return ErrEmptyBody
	}
	titleTrim := strings.TrimSpace(req.Title)
	if err := s.filterTexts(ctx, titleTrim, body); err != nil {
		return err
	}

	existing, err := s.repo.GetEntryByID(ctx, entryID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrEntryNotFound
	}

	authorID, err := s.repo.GetAuthorID(ctx, existing.JournalID)
	if err != nil {
		return ErrNotFound
	}

	asAdmin := authorID != userID
	if asAdmin && !s.authz.Can(ctx, userID, authz.PermEditAnyJournal) {
		return ErrNotAuthor
	}

	var titlePtr *string
	if titleTrim != "" {
		titlePtr = &titleTrim
	}

	publishing := existing.IsDraft && !req.IsDraft

	if err := s.repo.UpdateEntry(ctx, repository.JournalEntryUpdate{
		ID:                   entryID,
		JournalID:            existing.JournalID,
		Title:                titlePtr,
		Body:                 body,
		WordCount:            countWords(body),
		IsDraft:              req.IsDraft,
		RecordAuthorActivity: publishing,
	}); err != nil {
		return err
	}

	if asAdmin {
		s.writeAudit(ctx, repository.NewAuditEntry{
			ActorID:    userID,
			Action:     repository.AuditActionJournalEntryUpdateAdmin,
			TargetType: repository.AuditTargetJournalEntry,
			TargetID:   entryID.String(),
			Details:    journalEntryUpdateDetails(existing, titlePtr, body, req.IsDraft),
			SubjectID:  authorID,
		})
	}

	if publishing {
		go s.notifyEntryPublished(existing.JournalID, existing.EntryNumber, userID)
	}

	return nil
}

func (s *service) DeleteEntry(ctx context.Context, entryID uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.repo.GetEntryAuthorID(ctx, entryID)
	if err != nil {
		return ErrEntryNotFound
	}

	asAdmin := authorID != userID
	if asAdmin && !s.authz.Can(ctx, userID, authz.PermDeleteAnyJournal) {
		return ErrNotAuthor
	}

	paths, err := s.repo.DeleteEntry(ctx, entryID)
	if err != nil {
		return err
	}

	action := repository.AuditActionJournalEntryDelete
	if asAdmin {
		action = repository.AuditActionJournalEntryDeleteAdmin
	}

	s.writeAudit(ctx, repository.NewAuditEntry{
		ActorID:    userID,
		Action:     action,
		TargetType: repository.AuditTargetJournalEntry,
		TargetID:   entryID.String(),
		SubjectID:  authorID,
	})

	s.uploadSvc.Delete(paths...)

	return nil
}

func (s *service) CreateComment(ctx context.Context, journalID uuid.UUID, userID uuid.UUID, entryID *uuid.UUID, parentID *uuid.UUID, body string) (uuid.UUID, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return uuid.Nil, ErrEmptyBody
	}
	if err := s.filterTexts(ctx, body); err != nil {
		return uuid.Nil, err
	}

	authorID, err := s.repo.GetAuthorID(ctx, journalID)
	if err != nil {
		return uuid.Nil, ErrNotFound
	}

	archived, err := s.repo.IsArchived(ctx, journalID)
	if err != nil {
		return uuid.Nil, err
	}
	if archived {
		return uuid.Nil, ErrArchived
	}

	if entryID != nil {
		entryJournalID, err := s.repo.GetEntryJournalID(ctx, *entryID)
		if err != nil {
			return uuid.Nil, ErrEntryNotFound
		}
		if entryJournalID != journalID {
			return uuid.Nil, ErrEntryMismatch
		}
	}

	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return uuid.Nil, block.ErrUserBlocked
	}

	var entryNumber *int
	if entryID != nil {
		if entry, err := s.repo.GetEntryByID(ctx, *entryID); err == nil && entry != nil {
			entryNumber = new(entry.EntryNumber)
		}
	}

	isAuthorComment := userID == authorID

	created, err := s.repo.CreateComment(ctx, repository.NewJournalComment{
		JournalID:            journalID,
		EntryID:              entryID,
		ParentID:             parentID,
		UserID:               userID,
		Body:                 body,
		RecordAuthorActivity: isAuthorComment,
	})
	if err != nil {
		return uuid.Nil, err
	}

	refType := journalCommentRefType(entryNumber, created.ID)

	go func() {
		bgCtx := context.Background()
		title, _ := s.repo.GetTitle(bgCtx, journalID)
		linkURL := commentLinkURL("", journalID, entryNumber, created.ID)
		actor := s.actorName(bgCtx, userID)

		if isAuthorComment {
			followerIDs, err := s.eligibleFollowerIDs(bgCtx, journalID, userID)
			if err != nil {
				logger.Log.Error().Err(err).Msg("get follower ids failed")
				return
			}

			notifyParams := make([]dto.NotifyParams, 0, len(followerIDs))
			for _, followerID := range followerIDs {
				notifyParams = append(notifyParams, dto.NotifyParams{
					RecipientID:   followerID,
					Type:          dto.NotifJournalUpdate,
					ReferenceID:   journalID,
					ReferenceType: refType,
					ActorID:       userID,
					EmailActor:    actor,
					EmailAction:   "posted a new update on",
					EmailTitle:    title,
					EmailLink:     linkURL,
				})
			}
			s.notifService.NotifyMany(bgCtx, notifyParams)
		} else {
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   authorID,
				Type:          dto.NotifJournalCommented,
				ReferenceID:   journalID,
				ReferenceType: refType,
				ActorID:       userID,
				EmailActor:    actor,
				EmailAction:   "commented on your journal",
				EmailTitle:    title,
				EmailLink:     linkURL,
			})
		}

		if parentID != nil {
			parentAuthor, err := s.repo.GetCommentAuthorID(bgCtx, *parentID)
			if err == nil && parentAuthor != userID {
				_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
					RecipientID:   parentAuthor,
					Type:          dto.NotifJournalCommentReply,
					ReferenceID:   journalID,
					ReferenceType: refType,
					ActorID:       userID,
					EmailActor:    actor,
					EmailAction:   "replied to your comment",
					EmailTitle:    title,
					EmailLink:     linkURL,
				})
			}
		}
	}()

	return created.ID, nil
}

func journalCommentRefType(entryNumber *int, commentID uuid.UUID) string {
	if entryNumber != nil {
		return fmt.Sprintf("journal_entry_comment:%d:%s", *entryNumber, commentID)
	}
	return fmt.Sprintf("journal_comment:%s", commentID)
}

func commentLinkURL(baseURL string, journalID uuid.UUID, entryNumber *int, commentID uuid.UUID) string {
	if entryNumber != nil {
		return fmt.Sprintf("%s/journals/%s/entry/%d#comment-%s", baseURL, journalID, *entryNumber, commentID)
	}
	return fmt.Sprintf("%s/journals/%s#comment-%s", baseURL, journalID, commentID)
}

func (s *service) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return ErrEmptyBody
	}
	if err := s.filterTexts(ctx, body); err != nil {
		return err
	}

	authorID, err := s.repo.GetCommentAuthorID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	asAdmin := authorID != userID && s.authz.Can(ctx, userID, authz.PermEditAnyComment)

	if err := s.repo.UpdateComment(ctx, repository.JournalCommentUpdate{
		ID:      id,
		UserID:  userID,
		Body:    body,
		AsAdmin: asAdmin,
	}); err != nil {
		return err
	}

	if asAdmin {
		s.writeAudit(ctx, repository.NewAuditEntry{
			ActorID:    userID,
			Action:     repository.AuditActionJournalCommentUpdateAdmin,
			TargetType: repository.AuditTargetJournalComment,
			TargetID:   id.String(),
			SubjectID:  authorID,
		})
	}

	return nil
}

func (s *service) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.repo.GetCommentAuthorID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	asAdmin := authorID != userID && s.authz.Can(ctx, userID, authz.PermDeleteAnyComment)

	paths, err := s.repo.DeleteComment(ctx, id, userID, asAdmin)
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(paths...)

	return nil
}

func (s *service) LikeComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	commentAuthorID, err := s.repo.GetCommentAuthorID(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, commentAuthorID); blocked {
		return block.ErrUserBlocked
	}
	if err := s.repo.LikeComment(ctx, userID, id); err != nil {
		return err
	}

	if commentAuthorID == userID {
		return nil
	}

	go func() {
		bgCtx := context.Background()
		journalID, err := s.repo.GetCommentEntityID(bgCtx, id)
		if err != nil {
			return
		}
		entryNumber, _ := s.repo.GetCommentEntryNumber(bgCtx, id)
		title, _ := s.repo.GetTitle(bgCtx, journalID)
		linkURL := commentLinkURL("", journalID, entryNumber, id)
		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   commentAuthorID,
			Type:          dto.NotifJournalCommentLiked,
			ReferenceID:   journalID,
			ReferenceType: journalCommentRefType(entryNumber, id),
			ActorID:       userID,
			EmailActor:    s.actorName(bgCtx, userID),
			EmailAction:   "liked your comment",
			EmailTitle:    title,
			EmailLink:     linkURL,
		})
	}()

	return nil
}

func (s *service) UnlikeComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.repo.UnlikeComment(ctx, userID, id)
}

func (s *service) UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error) {
	authorID, err := s.repo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID {
		return nil, ErrNotAuthor
	}

	return s.uploader.SaveAndRecord(
		ctx,
		"journals",
		contentType,
		fileSize,
		reader,
		func(mediaURL, mediaType, thumbURL string, sortOrder int) (int64, error) {
			return s.repo.AddCommentMedia(ctx, repository.NewJournalCommentMedia{
				CommentID:    commentID,
				MediaURL:     mediaURL,
				MediaType:    mediaType,
				ThumbnailURL: thumbURL,
				SortOrder:    sortOrder,
			})
		},
		s.repo.UpdateCommentMediaURL,
		s.repo.UpdateCommentMediaThumbnail,
	)
}

func (s *service) UploadEntryMedia(ctx context.Context, entryID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error) {
	authorID, err := s.repo.GetEntryAuthorID(ctx, entryID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID {
		return nil, ErrNotAuthor
	}

	return s.uploader.SaveAndRecord(
		ctx,
		"journals",
		contentType,
		fileSize,
		reader,
		func(mediaURL, mediaType, thumbURL string, sortOrder int) (int64, error) {
			return s.repo.AddMedia(ctx, repository.NewJournalEntryMedia{
				EntryID:      entryID,
				MediaURL:     mediaURL,
				MediaType:    mediaType,
				ThumbnailURL: thumbURL,
				SortOrder:    sortOrder,
			})
		},
		s.repo.UpdateMediaURL,
		s.repo.UpdateMediaThumbnail,
	)
}

func (s *service) DeleteEntryMedia(ctx context.Context, entryID uuid.UUID, mediaID int64, userID uuid.UUID) error {
	authorID, err := s.repo.GetEntryAuthorID(ctx, entryID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID {
		return ErrNotAuthor
	}

	mediaURL, err := s.repo.DeleteMedia(ctx, mediaID, entryID)
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(mediaURL)
	return nil
}

func (s *service) FollowJournal(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.repo.GetAuthorID(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	if authorID == userID {
		return ErrCannotFollowOwn
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return block.ErrUserBlocked
	}

	if err := s.repo.Follow(ctx, userID, id); err != nil {
		return err
	}

	go func() {
		bgCtx := context.Background()
		title, _ := s.repo.GetTitle(bgCtx, id)
		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifJournalFollowed,
			ReferenceID:   id,
			ReferenceType: "journal",
			ActorID:       userID,
			EmailActor:    s.actorName(bgCtx, userID),
			EmailAction:   "started following your journal",
			EmailTitle:    title,
			EmailLink:     fmt.Sprintf("/journals/%s", id),
		})
	}()

	return nil
}

func (s *service) UnfollowJournal(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.repo.Unfollow(ctx, userID, id)
}

func (s *service) ArchiveStale(ctx context.Context) (int, error) {
	cutoff := time.Now().Add(-archiveAfter)
	ids, err := s.repo.ArchiveStale(ctx, cutoff)
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}

	for _, id := range ids {
		authorID, err := s.repo.GetAuthorID(ctx, id)
		if err != nil {
			continue
		}
		title, _ := s.repo.GetTitle(ctx, id)
		_ = s.notifService.Notify(ctx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifJournalArchived,
			ReferenceID:   id,
			ReferenceType: "journal",
			ActorID:       authorID,
			EmailActor:    "The Scribe",
			EmailAction:   "archived your inactive journal",
			EmailTitle:    title,
			EmailLink:     fmt.Sprintf("/journals/%s", id),
		})
	}

	return len(ids), nil
}
