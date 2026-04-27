package journal

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
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

type (
	Service interface {
		CreateJournal(ctx context.Context, userID uuid.UUID, req dto.CreateJournalRequest) (uuid.UUID, error)
		GetJournalDetail(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.JournalDetailResponse, error)
		ListJournals(ctx context.Context, p params.ListParams, viewerID uuid.UUID) (*dto.JournalListResponse, error)
		ListJournalsByUser(ctx context.Context, authorID uuid.UUID, viewerID uuid.UUID, limit, offset int) (*dto.JournalListResponse, error)
		ListFollowedByUser(ctx context.Context, followerID uuid.UUID, viewerID uuid.UUID, limit, offset int) (*dto.JournalListResponse, error)
		UpdateJournal(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateJournalRequest) error
		DeleteJournal(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

		CreateComment(ctx context.Context, journalID uuid.UUID, userID uuid.UUID, parentID *uuid.UUID, body string) (uuid.UUID, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		LikeComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		UnlikeComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error)

		FollowJournal(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		UnfollowJournal(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

		ArchiveStale(ctx context.Context) (int, error)
	}

	service struct {
		repo          repository.JournalRepository
		userRepo      repository.UserRepository
		authz         authz.Service
		blockSvc      block.Service
		notifService  notification.Service
		settingsSvc   settings.Service
		uploader      *media.Uploader
		contentFilter *contentfilter.Manager
	}
)

func NewService(
	repo repository.JournalRepository,
	userRepo repository.UserRepository,
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
		authz:         authzService,
		blockSvc:      blockSvc,
		notifService:  notifService,
		settingsSvc:   settingsSvc,
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

func (s *service) actorName(ctx context.Context, userID uuid.UUID) string {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return "Someone"
	}
	return u.DisplayName
}

func (s *service) CreateJournal(ctx context.Context, userID uuid.UUID, req dto.CreateJournalRequest) (uuid.UUID, error) {
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Body) == "" {
		return uuid.Nil, ErrEmptyBody
	}
	if err := s.filterTexts(ctx, req.Title, req.Body); err != nil {
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

	return s.repo.Create(ctx, userID, req)
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

	return &dto.JournalDetailResponse{
		JournalResponse: *journal,
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

func (s *service) ListFollowedByUser(ctx context.Context, followerID uuid.UUID, viewerID uuid.UUID, limit, offset int) (*dto.JournalListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	journals, total, err := s.repo.ListFollowedByUser(ctx, followerID, viewerID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &dto.JournalListResponse{
		Journals: journals,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}, nil
}

func (s *service) UpdateJournal(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateJournalRequest) error {
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Body) == "" {
		return ErrEmptyBody
	}
	if err := s.filterTexts(ctx, req.Title, req.Body); err != nil {
		return err
	}
	if s.authz.Can(ctx, userID, authz.PermEditAnyJournal) {
		return s.repo.UpdateAsAdmin(ctx, id, req)
	}
	return s.repo.Update(ctx, id, userID, req)
}

func (s *service) DeleteJournal(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyJournal) {
		return s.repo.DeleteAsAdmin(ctx, id)
	}
	return s.repo.Delete(ctx, id, userID)
}

func (s *service) CreateComment(ctx context.Context, journalID uuid.UUID, userID uuid.UUID, parentID *uuid.UUID, body string) (uuid.UUID, error) {
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

	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return uuid.Nil, block.ErrUserBlocked
	}

	id := uuid.New()
	if err := s.repo.CreateComment(ctx, id, journalID, parentID, userID, body); err != nil {
		return uuid.Nil, err
	}

	isAuthorComment := userID == authorID

	go func() {
		bgCtx := context.Background()
		title, _ := s.repo.GetTitle(bgCtx, journalID)
		baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/journals/%s#comment-%s", baseURL, journalID, id)
		actor := s.actorName(bgCtx, userID)

		if isAuthorComment {
			_ = s.repo.UpdateLastAuthorActivity(bgCtx, journalID)

			followerIDs, err := s.repo.GetFollowerIDs(bgCtx, journalID)
			if err != nil {
				logger.Log.Error().Err(err).Msg("get follower ids failed")
				return
			}
			subject, emailBody := notification.NotifEmail(actor, "posted a new update on", title, linkURL)
			blockedSet := make(map[uuid.UUID]struct{})
			if blockedIDs, err := s.blockSvc.GetBlockedIDs(bgCtx, userID); err == nil {
				for i := 0; i < len(blockedIDs); i++ {
					blockedSet[blockedIDs[i]] = struct{}{}
				}
			}
			params := make([]dto.NotifyParams, 0, len(followerIDs))
			for i := 0; i < len(followerIDs); i++ {
				followerID := followerIDs[i]
				if followerID == userID {
					continue
				}
				if _, isBlocked := blockedSet[followerID]; isBlocked {
					continue
				}
				params = append(params, dto.NotifyParams{
					RecipientID:   followerID,
					Type:          dto.NotifJournalUpdate,
					ReferenceID:   journalID,
					ReferenceType: fmt.Sprintf("journal_comment:%s", id),
					ActorID:       userID,
					EmailSubject:  subject,
					EmailBody:     emailBody,
				})
			}
			s.notifService.NotifyMany(bgCtx, params)
		} else {
			subject, emailBody := notification.NotifEmail(actor, "commented on your journal", title, linkURL)
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   authorID,
				Type:          dto.NotifJournalCommented,
				ReferenceID:   journalID,
				ReferenceType: fmt.Sprintf("journal_comment:%s", id),
				ActorID:       userID,
				EmailSubject:  subject,
				EmailBody:     emailBody,
			})
		}

		if parentID != nil {
			parentAuthor, err := s.repo.GetCommentAuthorID(bgCtx, *parentID)
			if err == nil && parentAuthor != userID {
				replySubject, replyBody := notification.NotifEmail(actor, "replied to your comment", title, linkURL)
				_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
					RecipientID:   parentAuthor,
					Type:          dto.NotifJournalCommentReply,
					ReferenceID:   journalID,
					ReferenceType: fmt.Sprintf("journal_comment:%s", id),
					ActorID:       userID,
					EmailSubject:  replySubject,
					EmailBody:     replyBody,
				})
			}
		}
	}()

	return id, nil
}

func (s *service) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return ErrEmptyBody
	}
	if err := s.filterTexts(ctx, body); err != nil {
		return err
	}
	if s.authz.Can(ctx, userID, authz.PermEditAnyComment) {
		return s.repo.UpdateCommentAsAdmin(ctx, id, body)
	}
	return s.repo.UpdateComment(ctx, id, userID, body)
}

func (s *service) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyComment) {
		return s.repo.DeleteCommentAsAdmin(ctx, id)
	}
	return s.repo.DeleteComment(ctx, id, userID)
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
		journalID, err := s.repo.GetCommentJournalID(bgCtx, id)
		if err != nil {
			return
		}
		title, _ := s.repo.GetTitle(bgCtx, journalID)
		baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/journals/%s#comment-%s", baseURL, journalID, id)
		subject, emailBody := notification.NotifEmail(s.actorName(bgCtx, userID), "liked your comment", title, linkURL)
		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   commentAuthorID,
			Type:          dto.NotifJournalCommentLiked,
			ReferenceID:   journalID,
			ReferenceType: fmt.Sprintf("journal_comment:%s", id),
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     emailBody,
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
			return s.repo.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbURL, sortOrder)
		},
		s.repo.UpdateCommentMediaURL,
		s.repo.UpdateCommentMediaThumbnail,
	)
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
		baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/journals/%s", baseURL, id)
		subject, emailBody := notification.NotifEmail(s.actorName(bgCtx, userID), "started following your journal", title, linkURL)
		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifJournalFollowed,
			ReferenceID:   id,
			ReferenceType: "journal",
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     emailBody,
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

	baseURL := s.settingsSvc.Get(ctx, config.SettingBaseURL)
	for _, id := range ids {
		authorID, err := s.repo.GetAuthorID(ctx, id)
		if err != nil {
			continue
		}
		title, _ := s.repo.GetTitle(ctx, id)
		linkURL := fmt.Sprintf("%s/journals/%s", baseURL, id)
		subject, emailBody := notification.NotifEmail("The Scribe", "archived your inactive journal", title, linkURL)
		_ = s.notifService.Notify(ctx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifJournalArchived,
			ReferenceID:   id,
			ReferenceType: "journal",
			ActorID:       authorID,
			EmailSubject:  subject,
			EmailBody:     emailBody,
		})
	}

	return len(ids), nil
}
