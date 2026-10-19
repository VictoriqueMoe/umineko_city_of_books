package journal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/comment"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/homefeed"
	"umineko_city_of_books/internal/journal/params"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/og"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/sidebar"
	"umineko_city_of_books/internal/upload"

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
		SetJournalPaused(ctx context.Context, id uuid.UUID, userID uuid.UUID, paused bool) error

		CreateEntry(ctx context.Context, journalID uuid.UUID, userID uuid.UUID, req dto.CreateJournalEntryRequest, files []upload.FileUpload) (uuid.UUID, int, error)
		GetEntry(ctx context.Context, journalID uuid.UUID, entryNumber int, viewerID uuid.UUID) (*dto.JournalEntryResponse, []dto.CommentResponse, error)
		UpdateEntry(ctx context.Context, entryID uuid.UUID, userID uuid.UUID, req dto.UpdateJournalEntryRequest) error
		DeleteEntry(ctx context.Context, entryID uuid.UUID, userID uuid.UUID) error

		CreateComment(ctx context.Context, journalID uuid.UUID, userID uuid.UUID, req dto.CreateJournalCommentRequest, files []upload.FileUpload) (uuid.UUID, error)
		UploadEntryMedia(ctx context.Context, entryID uuid.UUID, userID uuid.UUID, contentType string, filename string, fileSize int64, reader io.Reader, isSpoiler bool) (*dto.PostMediaResponse, error)
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
		mentionSvc    mention.Service
		commentSvc    comment.Service
		settingsSvc   settings.Service
		uploadSvc     upload.Service
		uploader      upload.MediaRecorder
		contentFilter *contentfilter.Manager
		ogCache       *og.Resolver
		echoCache     homefeed.Service
		activity      sidebar.ActivityBumper
	}
)

func NewService(
	repo repository.JournalRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditLogRepository,
	authzService authz.Service,
	blockSvc block.Service,
	notifService notification.Service,
	mentionSvc mention.Service,
	commentSvc comment.Service,
	uploadSvc upload.Service,
	mediaProc *media.Processor,
	settingsSvc settings.Service,
	contentFilter *contentfilter.Manager,
	ogCache *og.Resolver,
	echoCache homefeed.Service,
	activity sidebar.ActivityBumper,
) Service {
	return &service{
		repo:          repo,
		userRepo:      userRepo,
		auditRepo:     auditRepo,
		authz:         authzService,
		blockSvc:      blockSvc,
		notifService:  notifService,
		mentionSvc:    mentionSvc,
		commentSvc:    commentSvc,
		settingsSvc:   settingsSvc,
		uploadSvc:     uploadSvc,
		uploader:      media.NewUploader(uploadSvc, settingsSvc, mediaProc),
		contentFilter: contentFilter,
		ogCache:       ogCache,
		echoCache:     echoCache,
		activity:      activity,
	}
}

func (s *service) writeAudit(ctx context.Context, entry audit.NewEntry) {
	if err := s.auditRepo.Create(ctx, entry); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("action", string(entry.Action)).Msg("failed to write audit log")
	}
}

func (s *service) actorName(ctx context.Context, userID uuid.UUID) string {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("user_id", userID.String()).Msg("actor lookup failed, naming them Someone")

		return "Someone"
	}
	if u == nil {
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

func journalEntryUpdateDetails(before *model.JournalEntryRow, title *string, body string, isDraft bool) string {
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
	if err := s.contentFilter.Check(ctx, req.Title); err != nil {
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

	created, err := s.repo.Create(ctx, spec.NewJournal{
		UserID: userID,
		Title:  req.Title,
		Work:   req.Work,
	})
	if err != nil {
		return uuid.Nil, err
	}

	s.mentionSvc.NotifyAsync(ctx, mention.Reference{Kind: mention.KindJournal, EntityID: created.ID}, userID, req.Title)

	s.activity.BumpSidebarActivity(sidebar.KeyJournals)

	return created.ID, nil
}

func (s *service) GetJournalDetail(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.JournalDetailResponse, error) {
	journal, err := s.repo.GetByID(ctx, spec.JournalLookup{ID: id, ViewerID: viewerID})
	if err != nil || journal == nil {
		if journal == nil && err == nil {
			return nil, ErrNotFound
		}
		return nil, err
	}

	comments, err := s.commentSvc.Thread(ctx, spec.CommentTarget{Kind: spec.CommentKindJournal, ID: id}, viewerID)
	if err != nil {
		return nil, fmt.Errorf("journal comments: %w", err)
	}

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
		entries = append(entries, model.JournalEntrySummaryToDTO(e))
	}

	var latestEntry *dto.JournalEntryResponse
	if journal.LatestEntryNumber != nil {
		entry, err := s.repo.GetEntry(ctx, spec.JournalEntryLookup{
			JournalID:   id,
			EntryNumber: *journal.LatestEntryNumber,
		})
		if err != nil {
			return nil, err
		}
		if entry != nil {
			entryMediaMap, err := s.repo.GetMediaBatch(ctx, []uuid.UUID{entry.ID})
			if err != nil {
				return nil, fmt.Errorf("latest entry media: %w", err)
			}
			latestEntry = new(model.JournalEntryToDTO(entry, entryMediaMap[entry.ID]))
		}
	}

	return &dto.JournalDetailResponse{
		JournalResponse: *journal,
		Entries:         entries,
		LatestEntry:     latestEntry,
		Comments:        comments,
	}, nil
}

func (s *service) ListJournals(ctx context.Context, p params.ListParams, viewerID uuid.UUID) (*dto.JournalListResponse, error) {
	blockedIDs, err := s.blockSvc.GetBlockedIDs(ctx, viewerID)
	if err != nil {
		return nil, fmt.Errorf("blocked users: %w", err)
	}

	journals, total, err := s.repo.List(ctx, spec.JournalQuery{
		Sort:            p.Sort,
		Work:            p.Work,
		AuthorID:        p.AuthorID,
		Search:          p.Search,
		IncludeArchived: p.IncludeArchived,
		Limit:           p.Limit,
		Offset:          p.Offset,
		ViewerID:        viewerID,
		ExcludeUserIDs:  blockedIDs,
	})
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
	journals, total, err := s.repo.ListFollowedByUser(ctx, spec.JournalFollowedQuery{
		FollowerID: followerID,
		ViewerID:   viewerID,
		Limit:      page.Limit(),
		Offset:     page.Offset(),
	})
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
	if err := s.contentFilter.Check(ctx, req.Title); err != nil {
		return err
	}

	authorID, err := s.repo.GetAuthorID(ctx, id)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	asAdmin := authorID != userID && s.authz.Can(ctx, userID, authz.PermEditAnyJournal)

	var before *dto.JournalResponse
	if asAdmin {
		before, err = s.repo.GetByID(ctx, spec.JournalLookup{ID: id, ViewerID: userID})
		if err != nil {
			return fmt.Errorf("journal before admin edit: %w", err)
		}
	}

	if err := s.repo.Update(ctx, spec.JournalUpdate{
		ID:      id,
		UserID:  userID,
		Title:   req.Title,
		Work:    req.Work,
		AsAdmin: asAdmin,
	}); err != nil {
		return err
	}

	if asAdmin {
		s.writeAudit(ctx, audit.NewEntry{
			ActorID:    userID,
			Action:     audit.ActionJournalUpdateAdmin,
			TargetType: audit.TargetJournal,
			TargetID:   id.String(),
			Details:    journalUpdateDetails(before, req),
			SubjectID:  authorID,
		})
	}

	return nil
}

func (s *service) DeleteJournal(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.repo.GetAuthorID(ctx, id)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	asAdmin := authorID != userID && s.authz.Can(ctx, userID, authz.PermDeleteAnyJournal)
	title, err := s.repo.GetTitle(ctx, id)
	if err != nil {
		return fmt.Errorf("journal title for the audit: %w", err)
	}

	paths, err := s.repo.DeleteWithMedia(ctx, spec.JournalDeletion{
		ID:      id,
		UserID:  userID,
		AsAdmin: asAdmin,
	})
	if err != nil {
		return err
	}

	action := audit.ActionJournalDelete
	if asAdmin {
		action = audit.ActionJournalDeleteAdmin
	}

	details := ""
	if title != "" {
		details = "title=" + title
	}

	s.writeAudit(ctx, audit.NewEntry{
		ActorID:    userID,
		Action:     action,
		TargetType: audit.TargetJournal,
		TargetID:   id.String(),
		Details:    details,
		SubjectID:  authorID,
	})

	s.uploadSvc.Delete(paths...)

	s.clearPageCache(ctx, id.String())

	return nil
}

func (s *service) clearPageCache(ctx context.Context, id string) {
	if s.ogCache != nil {
		if err := s.ogCache.ClearMetaCache(ctx, og.KindJournal, id); err != nil {
			logger.Ctx(ctx).Warn().Err(err).Str("journal_id", id).Msg("clear og meta cache failed")
		}
	}

	if s.echoCache == nil {
		return
	}

	if err := s.echoCache.ClearEchoCache(ctx); err != nil {
		logger.Ctx(ctx).Warn().Err(err).Msg("clear echo cache failed")
	}
}

func (s *service) SetJournalPaused(ctx context.Context, id uuid.UUID, userID uuid.UUID, paused bool) error {
	authorID, err := s.repo.GetAuthorID(ctx, id)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if authorID != userID {
		return ErrNotAuthor
	}

	return s.repo.SetPaused(ctx, spec.JournalPause{ID: id, UserID: userID, Paused: paused})
}

func (s *service) CreateEntry(ctx context.Context, journalID uuid.UUID, userID uuid.UUID, req dto.CreateJournalEntryRequest, files []upload.FileUpload) (uuid.UUID, int, error) {
	body := strings.TrimSpace(req.Body)
	if body == "" && len(files) == 0 {
		return uuid.Nil, 0, ErrEmptyBody
	}
	titleTrim := strings.TrimSpace(req.Title)
	if err := s.contentFilter.Check(ctx, titleTrim, body); err != nil {
		return uuid.Nil, 0, err
	}

	authorID, err := s.repo.GetAuthorID(ctx, journalID)
	if errors.Is(err, dao.ErrNotFound) {
		return uuid.Nil, 0, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, 0, err
	}

	asAdmin := authorID != userID
	if asAdmin && !s.authz.Can(ctx, userID, authz.PermEditAnyJournal) {
		return uuid.Nil, 0, ErrNotAuthor
	}

	validated, err := upload.ValidateMedia(ctx, s.settingsSvc, files)
	if err != nil {
		return uuid.Nil, 0, err
	}

	nextNumber, err := s.repo.GetNextEntryNumber(ctx, journalID)
	if err != nil {
		return uuid.Nil, 0, err
	}

	var titlePtr *string
	if titleTrim != "" {
		titlePtr = &titleTrim
	}

	hasMedia := len(files) > 0

	created, err := s.repo.CreateEntry(ctx, spec.NewJournalEntry{
		JournalID:            journalID,
		EntryNumber:          nextNumber,
		Title:                titlePtr,
		Body:                 body,
		WordCount:            countWords(body),
		IsDraft:              req.IsDraft,
		RecordAuthorActivity: !hasMedia,
	})
	if err != nil {
		return uuid.Nil, 0, err
	}

	if hasMedia {
		if err := s.attachEntryMedia(ctx, validated, journalID, created.ID); err != nil {
			return uuid.Nil, 0, err
		}
	}

	if asAdmin {
		s.writeAudit(ctx, audit.NewEntry{
			ActorID:    userID,
			Action:     audit.ActionJournalEntryCreateAdmin,
			TargetType: audit.TargetJournalEntry,
			TargetID:   created.ID.String(),
			Details:    fmt.Sprintf("journal_id=%s,entry_number=%d,is_draft=%t", journalID, nextNumber, req.IsDraft),
			SubjectID:  authorID,
		})
	}

	if !req.IsDraft {
		s.notifyEntryMentions(ctx, journalID, nextNumber, authorID, body)

		go s.notifyEntryPublished(journalID, nextNumber, authorID)

		s.activity.BumpSidebarActivity(sidebar.KeyJournals)
	}

	return created.ID, nextNumber, nil
}

func (s *service) attachEntryMedia(ctx context.Context, validated upload.ValidatedMedia, journalID uuid.UUID, entryID uuid.UUID) error {
	if err := validated.SaveToEntity(ctx, s.uploader, s.uploadSvc, "journals", s.repo, entryID); err != nil {
		return errors.Join(err, s.rollbackEntry(ctx, entryID))
	}

	if err := s.repo.UpdateLastAuthorActivity(ctx, journalID); err != nil {
		return errors.Join(fmt.Errorf("record journal author activity: %w", err), s.rollbackEntry(ctx, entryID))
	}

	return nil
}

func (s *service) rollbackEntry(ctx context.Context, entryID uuid.UUID) error {
	paths, err := s.repo.DeleteEntryWithMedia(context.WithoutCancel(ctx), entryID)
	if err != nil {
		return fmt.Errorf("roll back journal entry %s: %w", entryID, err)
	}

	s.uploadSvc.Delete(paths...)

	return nil
}

func (s *service) notifyEntryMentions(ctx context.Context, journalID uuid.UUID, entryNumber int, authorID uuid.UUID, body string) {
	s.mentionSvc.NotifyAsync(ctx, mention.Reference{
		Kind:        mention.KindJournalEntry,
		EntityID:    journalID,
		EntryNumber: entryNumber,
	}, authorID, body)
}

func (s *service) eligibleFollowerIDs(ctx context.Context, journalID uuid.UUID, actorUserID uuid.UUID) ([]uuid.UUID, error) {
	followerIDs, err := s.repo.GetFollowerIDs(ctx, journalID)
	if err != nil {
		return nil, err
	}

	blockedIDs, err := s.blockSvc.GetBlockedIDs(ctx, actorUserID)
	if err != nil {
		return nil, fmt.Errorf("blocked users: %w", err)
	}

	blockedSet := make(map[uuid.UUID]struct{}, len(blockedIDs))
	for _, blockedID := range blockedIDs {
		blockedSet[blockedID] = struct{}{}
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

func (s *service) notifyEntryPublished(journalID uuid.UUID, entryNumber int, authorID uuid.UUID) {
	bgCtx := context.Background()
	title, err := s.repo.GetTitle(bgCtx, journalID)
	if err != nil {
		logger.Ctx(bgCtx).Warn().Err(err).Str("journal_id", journalID.String()).Msg("entry published notifications skipped, title lookup failed")
		return
	}
	actor := s.actorName(bgCtx, authorID)

	followerIDs, err := s.eligibleFollowerIDs(bgCtx, journalID, authorID)
	if err != nil {
		logger.Ctx(bgCtx).Error().Err(err).Msg("get follower ids failed on entry publish")
		return
	}

	notifyParams := make([]dto.NotifyParams, 0, len(followerIDs))
	for _, followerID := range followerIDs {
		notifyParams = append(notifyParams, dto.NotifyParams{
			RecipientID:   followerID,
			Type:          dto.NotifJournalUpdate,
			ReferenceID:   journalID,
			ReferenceType: fmt.Sprintf("journal_entry:%d", entryNumber),
			ActorID:       authorID,
			EmailActor:    actor,
			EmailAction:   "posted a new entry on",
			EmailTitle:    title,
			EmailLink:     fmt.Sprintf("/journals/%s/entry/%d", journalID, entryNumber),
		})
	}
	s.notifService.NotifyMany(bgCtx, notifyParams)
}

func (s *service) GetEntry(ctx context.Context, journalID uuid.UUID, entryNumber int, viewerID uuid.UUID) (*dto.JournalEntryResponse, []dto.CommentResponse, error) {
	entry, err := s.repo.GetEntry(ctx, spec.JournalEntryLookup{
		JournalID:   journalID,
		EntryNumber: entryNumber,
	})
	if err != nil {
		return nil, nil, err
	}
	if entry == nil {
		return nil, nil, ErrEntryNotFound
	}

	authorID, err := s.repo.GetAuthorID(ctx, journalID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	if entry.IsDraft && viewerID != authorID {
		return nil, nil, ErrEntryNotFound
	}

	target := spec.CommentTarget{Kind: spec.CommentKindJournalEntry, ID: journalID, EntryNumber: entryNumber}
	comments, err := s.commentSvc.Thread(ctx, target, viewerID)
	if err != nil {
		return nil, nil, fmt.Errorf("journal entry comments: %w", err)
	}

	entryMediaMap, err := s.repo.GetMediaBatch(ctx, []uuid.UUID{entry.ID})
	if err != nil {
		return nil, nil, fmt.Errorf("entry media: %w", err)
	}

	return new(model.JournalEntryToDTO(entry, entryMediaMap[entry.ID])), comments, nil
}

func (s *service) UpdateEntry(ctx context.Context, entryID uuid.UUID, userID uuid.UUID, req dto.UpdateJournalEntryRequest) error {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return ErrEmptyBody
	}
	titleTrim := strings.TrimSpace(req.Title)
	if err := s.contentFilter.Check(ctx, titleTrim, body); err != nil {
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
	if errors.Is(err, dao.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
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

	firstPublished, err := s.repo.UpdateEntry(ctx, spec.JournalEntryUpdate{
		ID:                   entryID,
		JournalID:            existing.JournalID,
		Title:                titlePtr,
		Body:                 body,
		WordCount:            countWords(body),
		IsDraft:              req.IsDraft,
		RecordAuthorActivity: publishing,
	})
	if err != nil {
		return err
	}

	if asAdmin {
		s.writeAudit(ctx, audit.NewEntry{
			ActorID:    userID,
			Action:     audit.ActionJournalEntryUpdateAdmin,
			TargetType: audit.TargetJournalEntry,
			TargetID:   entryID.String(),
			Details:    journalEntryUpdateDetails(existing, titlePtr, body, req.IsDraft),
			SubjectID:  authorID,
		})
	}

	if firstPublished {
		s.notifyEntryMentions(ctx, existing.JournalID, existing.EntryNumber, authorID, body)

		go s.notifyEntryPublished(existing.JournalID, existing.EntryNumber, authorID)

		s.activity.BumpSidebarActivity(sidebar.KeyJournals)
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

	paths, err := s.repo.DeleteEntryWithMedia(ctx, entryID)
	if err != nil {
		return err
	}

	action := audit.ActionJournalEntryDelete
	if asAdmin {
		action = audit.ActionJournalEntryDeleteAdmin
	}

	s.writeAudit(ctx, audit.NewEntry{
		ActorID:    userID,
		Action:     action,
		TargetType: audit.TargetJournalEntry,
		TargetID:   entryID.String(),
		SubjectID:  authorID,
	})

	s.uploadSvc.Delete(paths...)

	return nil
}

func (s *service) CreateComment(ctx context.Context, journalID uuid.UUID, userID uuid.UUID, req dto.CreateJournalCommentRequest, files []upload.FileUpload) (uuid.UUID, error) {
	authorID, err := s.repo.GetAuthorID(ctx, journalID)
	if errors.Is(err, dao.ErrNotFound) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}

	archived, err := s.repo.IsArchived(ctx, journalID)
	if err != nil {
		return uuid.Nil, err
	}
	if archived {
		return uuid.Nil, ErrArchived
	}

	target := spec.CommentTarget{Kind: spec.CommentKindJournal, ID: journalID}
	if req.EntryID != nil {
		entry, err := s.repo.GetEntryByID(ctx, *req.EntryID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("comment entry: %w", err)
		}
		if entry == nil {
			return uuid.Nil, ErrEntryNotFound
		}
		if entry.JournalID != journalID {
			return uuid.Nil, ErrEntryMismatch
		}

		target.Kind = spec.CommentKindJournalEntry
		target.EntryNumber = entry.EntryNumber
	}

	commentID, err := s.commentSvc.Create(ctx, target, userID, dto.CreateCommentRequest{ParentID: req.ParentID, Body: req.Body}, files)
	if err != nil {
		return uuid.Nil, err
	}

	if userID != authorID {
		return commentID, nil
	}

	if err := s.repo.UpdateLastAuthorActivity(ctx, journalID); err != nil {
		activityErr := fmt.Errorf("record journal author activity: %w", err)
		ref := spec.CommentRef{Kind: target.Kind, CommentID: commentID}
		if deleteErr := s.commentSvc.Delete(context.WithoutCancel(ctx), ref, userID); deleteErr != nil {
			return uuid.Nil, errors.Join(activityErr, fmt.Errorf("roll back comment %s: %w", commentID, deleteErr))
		}

		return uuid.Nil, activityErr
	}

	go s.notifyAuthorComment(context.WithoutCancel(ctx), target, commentID, userID)

	return commentID, nil
}

func (s *service) notifyAuthorComment(ctx context.Context, target spec.CommentTarget, commentID uuid.UUID, authorID uuid.UUID) {
	title, err := s.repo.GetTitle(ctx, target.ID)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("journal_id", target.ID.String()).Msg("journal update notifications skipped, title lookup failed")
		return
	}

	followerIDs, err := s.eligibleFollowerIDs(ctx, target.ID, authorID)
	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("journal_id", target.ID.String()).Msg("get follower ids failed on author comment")
		return
	}

	refType := fmt.Sprintf("journal_comment:%s", commentID)
	link := fmt.Sprintf("/journals/%s#comment-%s", target.ID, commentID)
	if target.Kind == spec.CommentKindJournalEntry {
		refType = fmt.Sprintf("journal_entry_comment:%d:%s", target.EntryNumber, commentID)
		link = fmt.Sprintf("/journals/%s/entry/%d#comment-%s", target.ID, target.EntryNumber, commentID)
	}

	actor := s.actorName(ctx, authorID)
	notifyParams := make([]dto.NotifyParams, 0, len(followerIDs))
	for _, followerID := range followerIDs {
		notifyParams = append(notifyParams, dto.NotifyParams{
			RecipientID:   followerID,
			Type:          dto.NotifJournalUpdate,
			ReferenceID:   target.ID,
			ReferenceType: refType,
			ActorID:       authorID,
			EmailActor:    actor,
			EmailAction:   "posted a new update on",
			EmailTitle:    title,
			EmailLink:     link,
		})
	}
	s.notifService.NotifyMany(ctx, notifyParams)
}

func (s *service) UploadEntryMedia(ctx context.Context, entryID uuid.UUID, userID uuid.UUID, contentType string, filename string, fileSize int64, reader io.Reader, isSpoiler bool) (*dto.PostMediaResponse, error) {
	authorID, err := s.repo.GetEntryAuthorID(ctx, entryID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if authorID != userID {
		return nil, ErrNotAuthor
	}

	return s.uploader.SaveAndRecord(
		ctx,
		"journals",
		contentType,
		filename,
		fileSize,
		reader,
		isSpoiler,
		func(mediaURL, mediaType, thumbURL, filename string, sortOrder int) (int64, error) {
			return s.repo.AddMedia(ctx, spec.NewMedia{
				TargetID:     entryID,
				MediaURL:     mediaURL,
				MediaType:    mediaType,
				ThumbnailURL: thumbURL,
				Filename:     filename,
				SortOrder:    sortOrder,
				IsSpoiler:    isSpoiler,
			})
		},
		s.repo.UpdateMediaURL,
		s.repo.UpdateMediaThumbnail,
	)
}

func (s *service) DeleteEntryMedia(ctx context.Context, entryID uuid.UUID, mediaID int64, userID uuid.UUID) error {
	authorID, err := s.repo.GetEntryAuthorID(ctx, entryID)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if authorID != userID {
		return ErrNotAuthor
	}

	mediaURL, err := s.repo.DeleteMedia(ctx, spec.MediaDeletion{ID: mediaID, TargetID: entryID})
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(mediaURL)
	return nil
}

func (s *service) FollowJournal(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.repo.GetAuthorID(ctx, id)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if authorID == userID {
		return ErrCannotFollowOwn
	}

	blocked, err := s.blockSvc.IsBlockedEither(ctx, userID, authorID)
	if err != nil {
		return fmt.Errorf("block check: %w", err)
	}
	if blocked {
		return block.ErrUserBlocked
	}

	if err := s.repo.Follow(ctx, spec.JournalFollow{UserID: userID, JournalID: id}); err != nil {
		return err
	}

	go func() {
		bgCtx := context.Background()
		title, err := s.repo.GetTitle(bgCtx, id)
		if err != nil {
			logger.Ctx(bgCtx).Warn().Err(err).Str("journal_id", id.String()).Msg("follow notification skipped, title lookup failed")
			return
		}
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
	return s.repo.Unfollow(ctx, spec.JournalFollow{UserID: userID, JournalID: id})
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
			logger.Ctx(ctx).Warn().Err(err).Str("journal_id", id.String()).Msg("archive notification skipped, author lookup failed")
			continue
		}
		title, err := s.repo.GetTitle(ctx, id)
		if err != nil {
			logger.Ctx(ctx).Warn().Err(err).Str("journal_id", id.String()).Msg("archive notification skipped, title lookup failed")
			continue
		}
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
