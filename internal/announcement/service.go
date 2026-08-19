package announcement

import (
	"context"
	"errors"
	"fmt"
	"io"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/utils"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

const deletedAuthorName = "Deleted user"

type (
	Service interface {
		List(ctx context.Context, page bounds.Page) (*dto.AnnouncementListResponse, error)
		GetDetail(ctx context.Context, id, viewerID uuid.UUID) (*dto.AnnouncementDetailResponse, error)
		GetLatest(ctx context.Context) (*dto.AnnouncementResponse, error)
		Create(ctx context.Context, userID uuid.UUID, title, body string) (uuid.UUID, error)
		Update(ctx context.Context, actorID uuid.UUID, id uuid.UUID, title, body string) error
		Delete(ctx context.Context, actorID uuid.UUID, id uuid.UUID) error
		SetPinned(ctx context.Context, actorID uuid.UUID, id uuid.UUID, pinned bool) error

		CreateComment(ctx context.Context, announcementID, userID uuid.UUID, parentID *uuid.UUID, body string) (uuid.UUID, error)
		UpdateComment(ctx context.Context, id, userID uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id, userID uuid.UUID) error
		LikeComment(ctx context.Context, userID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID, commentID uuid.UUID) error
		UploadCommentMedia(ctx context.Context, commentID, userID uuid.UUID, contentType string, filename string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error)
	}

	service struct {
		repo         repository.AnnouncementRepository
		userRepo     repository.UserRepository
		auditRepo    repository.AuditLogRepository
		blockSvc     block.Service
		notifService notification.Service
		settingsSvc  settings.Service
		authzSvc     authz.Service
		hub          *ws.Hub
		uploader     *media.Uploader
		uploadSvc    upload.Service
	}
)

func NewService(
	repo repository.AnnouncementRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditLogRepository,
	blockSvc block.Service,
	notifService notification.Service,
	settingsSvc settings.Service,
	authzSvc authz.Service,
	hub *ws.Hub,
	uploader *media.Uploader,
	uploadSvc upload.Service,
) Service {
	return &service{
		repo:         repo,
		userRepo:     userRepo,
		auditRepo:    auditRepo,
		blockSvc:     blockSvc,
		notifService: notifService,
		settingsSvc:  settingsSvc,
		authzSvc:     authzSvc,
		hub:          hub,
		uploader:     uploader,
		uploadSvc:    uploadSvc,
	}
}

func rowToResponse(r repository.AnnouncementRow) dto.AnnouncementResponse {
	if r.AuthorID == uuid.Nil {
		r.AuthorDisplayName = deletedAuthorName
	}

	return dto.AnnouncementResponse{
		ID:        r.ID,
		Title:     r.Title,
		Body:      r.Body,
		Pinned:    r.Pinned,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
		Author: dto.UserResponse{
			ID:          r.AuthorID,
			Username:    r.AuthorUsername,
			DisplayName: r.AuthorDisplayName,
			AvatarURL:   r.AuthorAvatarURL,
			Role:        role.Role(r.AuthorRole),
		},
	}
}

func announcementCommentToResponse(c repository.CommentRow, media []model.PostMediaRow) dto.AnnouncementCommentResponse {
	return dto.AnnouncementCommentResponse{
		ID:       c.ID,
		ParentID: c.ParentID,
		Author: dto.UserResponse{
			ID:          c.UserID,
			Username:    c.AuthorUsername,
			DisplayName: c.AuthorDisplayName,
			AvatarURL:   c.AuthorAvatarURL,
			Role:        role.Role(c.AuthorRole),
		},
		Body:      c.Body,
		Media:     model.MediaRowsToResponse(media),
		LikeCount: c.LikeCount,
		UserLiked: c.UserLiked,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (s *service) List(ctx context.Context, page bounds.Page) (*dto.AnnouncementListResponse, error) {
	rows, total, err := s.repo.List(ctx, page.Limit(), page.Offset())
	if err != nil {
		return nil, err
	}
	items := make([]dto.AnnouncementResponse, len(rows))
	for i, r := range rows {
		items[i] = rowToResponse(r)
	}
	return &dto.AnnouncementListResponse{
		Announcements: items,
		Total:         total,
		Limit:         page.Limit(),
		Offset:        page.Offset(),
	}, nil
}

func (s *service) GetDetail(ctx context.Context, id, viewerID uuid.UUID) (*dto.AnnouncementDetailResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}

	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)
	commentRows, _, _ := s.repo.GetComments(ctx, id, viewerID, 500, 0, blockedIDs)

	commentIDs := make([]uuid.UUID, len(commentRows))
	for i, c := range commentRows {
		commentIDs[i] = c.ID
	}
	mediaMap, _ := s.repo.GetCommentMediaBatch(ctx, commentIDs)

	flat := make([]dto.AnnouncementCommentResponse, len(commentRows))
	for i, c := range commentRows {
		flat[i] = announcementCommentToResponse(c, mediaMap[c.ID])
	}
	tree := utils.BuildTree(flat,
		func(c dto.AnnouncementCommentResponse) uuid.UUID { return c.ID },
		func(c dto.AnnouncementCommentResponse) *uuid.UUID { return c.ParentID },
		func(c *dto.AnnouncementCommentResponse, replies []dto.AnnouncementCommentResponse) {
			c.Replies = replies
		},
	)

	return &dto.AnnouncementDetailResponse{
		AnnouncementResponse: rowToResponse(*row),
		Comments:             tree,
	}, nil
}

func (s *service) GetLatest(ctx context.Context) (*dto.AnnouncementResponse, error) {
	row, err := s.repo.GetLatest(ctx)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return new(rowToResponse(*row)), nil
}

func (s *service) audit(ctx context.Context, actorID uuid.UUID, action repository.AuditAction, id, subjectID uuid.UUID, details string) {
	if err := s.auditRepo.Create(ctx, repository.NewAuditEntry{
		ActorID:    actorID,
		Action:     action,
		TargetType: repository.AuditTargetAnnouncement,
		TargetID:   id.String(),
		Details:    details,
		SubjectID:  subjectID,
	}); err != nil {
		logger.Log.Error().Err(err).Str("action", string(action)).Msg("failed to write audit log")
	}
}

func (s *service) Create(ctx context.Context, userID uuid.UUID, title, body string) (uuid.UUID, error) {
	if title == "" || body == "" {
		return uuid.Nil, ErrEmptyTitleOrBody
	}
	created, err := s.repo.Create(ctx, userID, title, body)
	if err != nil {
		return uuid.Nil, err
	}

	s.hub.Broadcast(ws.Message{
		Type: "new_announcement",
		Data: map[string]interface{}{
			"id":        created.ID,
			"title":     title,
			"author_id": userID,
		},
	})

	s.audit(ctx, userID, repository.AuditActionAnnouncementCreate, created.ID, userID, fmt.Sprintf("title=%s", title))

	return created.ID, nil
}

func (s *service) Update(ctx context.Context, actorID uuid.UUID, id uuid.UUID, title, body string) error {
	if title == "" || body == "" {
		return ErrEmptyTitleOrBody
	}

	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if current == nil {
		return ErrNotFound
	}

	if err := s.repo.Update(ctx, id, title, body); err != nil {
		return err
	}

	details := fmt.Sprintf("title=%s", title)
	if current.Title != title {
		details = fmt.Sprintf("title=%s -> %s", current.Title, title)
	}

	s.audit(ctx, actorID, repository.AuditActionAnnouncementUpdate, id, current.AuthorID, details)

	return nil
}

func (s *service) Delete(ctx context.Context, actorID uuid.UUID, id uuid.UUID) error {
	doomed, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if doomed == nil {
		return ErrNotFound
	}

	paths, err := s.repo.DeleteWithMedia(ctx, id)
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(paths...)

	s.audit(ctx, actorID, repository.AuditActionAnnouncementDelete, id, doomed.AuthorID, fmt.Sprintf("title=%s", doomed.Title))

	return nil
}

func (s *service) SetPinned(ctx context.Context, actorID uuid.UUID, id uuid.UUID, pinned bool) error {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if current == nil {
		return ErrNotFound
	}

	if err := s.repo.SetPinned(ctx, id, pinned); err != nil {
		return err
	}

	s.audit(ctx, actorID, repository.AuditActionAnnouncementPin, id, current.AuthorID, fmt.Sprintf("title=%s pinned=%t", current.Title, pinned))

	return nil
}

func (s *service) CreateComment(ctx context.Context, announcementID, userID uuid.UUID, parentID *uuid.UUID, body string) (uuid.UUID, error) {
	if body == "" {
		return uuid.Nil, ErrEmptyBody
	}

	ann, err := s.repo.GetByID(ctx, announcementID)
	if err != nil || ann == nil {
		return uuid.Nil, ErrNotFound
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, ann.AuthorID); blocked {
		return uuid.Nil, ErrBlocked
	}

	created, err := s.repo.CreateComment(ctx, announcementID, parentID, userID, body)
	if err != nil {
		logger.Log.Error().Err(err).
			Str("announcement_id", announcementID.String()).
			Str("user_id", userID.String()).
			Msg("failed to create announcement comment")
		return uuid.Nil, err
	}

	go s.notifyCommentCreated(ann, announcementID, created.ID, userID, parentID)

	return created.ID, nil
}

func (s *service) notifyCommentCreated(ann *repository.AnnouncementRow, announcementID, commentID, actorID uuid.UUID, parentID *uuid.UUID) {
	if ann.AuthorID == uuid.Nil {
		return
	}

	bgCtx := context.Background()
	actor, err := s.userRepo.GetByID(bgCtx, actorID)
	if err != nil || actor == nil {
		return
	}
	_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
		RecipientID:   ann.AuthorID,
		Type:          dto.NotifAnnouncementCommented,
		ReferenceID:   announcementID,
		ReferenceType: fmt.Sprintf("announcement_comment:%s", commentID),
		ActorID:       actorID,
		EmailActor:    actor.DisplayName,
		EmailAction:   "commented on your announcement",
		EmailTitle:    ann.Title,
		EmailLink:     fmt.Sprintf("/announcements/%s#comment-%s", announcementID, commentID),
	})

	if parentID != nil {
		parentAuthor, err := s.repo.GetCommentAuthorID(bgCtx, *parentID)
		if err == nil && parentAuthor != ann.AuthorID {
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   parentAuthor,
				Type:          dto.NotifAnnouncementCommentReply,
				ReferenceID:   announcementID,
				ReferenceType: fmt.Sprintf("announcement_comment:%s", commentID),
				ActorID:       actorID,
				EmailActor:    actor.DisplayName,
				EmailAction:   "replied to your comment",
				EmailTitle:    ann.Title,
				EmailLink:     fmt.Sprintf("/announcements/%s#comment-%s", announcementID, commentID),
			})
		}
	}
}

func (s *service) UpdateComment(ctx context.Context, id, userID uuid.UUID, body string) error {
	if body == "" {
		return ErrEmptyBody
	}

	asAdmin := s.authzSvc.Can(ctx, userID, authz.PermEditAnyComment)

	spec := repository.AnnouncementCommentUpdate{
		CommentID: id,
		UserID:    userID,
		Body:      body,
		AsAdmin:   asAdmin,
	}

	if err := s.repo.UpdateCommentBody(ctx, spec); err != nil {
		if asAdmin {
			return err
		}

		return ErrForbidden
	}

	return nil
}

func (s *service) DeleteComment(ctx context.Context, id, userID uuid.UUID) error {
	asAdmin := s.authzSvc.Can(ctx, userID, authz.PermDeleteAnyComment)

	spec := repository.AnnouncementCommentDeletion{
		CommentID: id,
		UserID:    userID,
		AsAdmin:   asAdmin,
	}

	paths, err := s.repo.DeleteCommentWithAudit(ctx, spec)
	if err != nil {
		if asAdmin {
			return err
		}

		return ErrForbidden
	}

	s.uploadSvc.Delete(paths...)

	return nil
}

func (s *service) LikeComment(ctx context.Context, userID, commentID uuid.UUID) error {
	commentAuthorID, err := s.repo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return ErrCommentNotFound
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, commentAuthorID); blocked {
		return ErrBlocked
	}
	if err := s.repo.LikeComment(ctx, userID, commentID); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return ErrBlocked
		}
		return err
	}

	go s.notifyCommentLiked(commentID, commentAuthorID, userID)

	return nil
}

func (s *service) notifyCommentLiked(commentID, recipientID, actorID uuid.UUID) {
	bgCtx := context.Background()
	announcementID, err := s.repo.GetCommentEntityID(bgCtx, commentID)
	if err != nil {
		return
	}
	actor, err := s.userRepo.GetByID(bgCtx, actorID)
	if err != nil || actor == nil {
		return
	}
	_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
		RecipientID:   recipientID,
		Type:          dto.NotifAnnouncementCommentLiked,
		ReferenceID:   announcementID,
		ReferenceType: fmt.Sprintf("announcement_comment:%s", commentID),
		ActorID:       actorID,
		EmailActor:    actor.DisplayName,
		EmailAction:   "liked your comment",
		EmailLink:     fmt.Sprintf("/announcements/%s#comment-%s", announcementID, commentID),
	})
}

func (s *service) UnlikeComment(ctx context.Context, userID, commentID uuid.UUID) error {
	return s.repo.UnlikeComment(ctx, userID, commentID)
}

func (s *service) UploadCommentMedia(ctx context.Context, commentID, userID uuid.UUID, contentType string, filename string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error) {
	authorID, err := s.repo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return nil, ErrCommentNotFound
	}
	if authorID != userID {
		return nil, ErrForbidden
	}

	return s.uploader.SaveAndRecord(ctx, "announcements", contentType, filename, fileSize, reader,
		func(mediaURL, mediaType, thumbURL, filename string, sortOrder int) (int64, error) {
			return s.repo.AddCommentMedia(ctx, repository.NewAnnouncementCommentMedia{
				CommentID:    commentID,
				MediaURL:     mediaURL,
				MediaType:    mediaType,
				ThumbnailURL: thumbURL,
				Filename:     filename,
				SortOrder:    sortOrder,
			})
		},
		s.repo.UpdateCommentMediaURL,
		s.repo.UpdateCommentMediaThumbnail,
	)
}
