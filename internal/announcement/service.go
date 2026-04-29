package announcement

import (
	"context"
	"errors"
	"fmt"
	"io"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/utils"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

type (
	Service interface {
		List(ctx context.Context, limit, offset int) (*dto.AnnouncementListResponse, error)
		GetDetail(ctx context.Context, id, viewerID uuid.UUID) (*dto.AnnouncementDetailResponse, error)
		GetLatest(ctx context.Context) (*dto.AnnouncementResponse, error)
		Create(ctx context.Context, userID uuid.UUID, title, body string) (uuid.UUID, error)
		Update(ctx context.Context, id uuid.UUID, title, body string) error
		Delete(ctx context.Context, id uuid.UUID) error
		SetPinned(ctx context.Context, id uuid.UUID, pinned bool) error

		CreateComment(ctx context.Context, announcementID, userID uuid.UUID, parentID *uuid.UUID, body string) (uuid.UUID, error)
		UpdateComment(ctx context.Context, id, userID uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id, userID uuid.UUID) error
		LikeComment(ctx context.Context, userID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID, commentID uuid.UUID) error
		UploadCommentMedia(ctx context.Context, commentID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error)
	}

	service struct {
		repo         repository.AnnouncementRepository
		userRepo     repository.UserRepository
		blockSvc     block.Service
		notifService notification.Service
		settingsSvc  settings.Service
		authzSvc     authz.Service
		hub          *ws.Hub
		uploader     *media.Uploader
	}
)

func NewService(
	repo repository.AnnouncementRepository,
	userRepo repository.UserRepository,
	blockSvc block.Service,
	notifService notification.Service,
	settingsSvc settings.Service,
	authzSvc authz.Service,
	hub *ws.Hub,
	uploader *media.Uploader,
) Service {
	return &service{
		repo:         repo,
		userRepo:     userRepo,
		blockSvc:     blockSvc,
		notifService: notifService,
		settingsSvc:  settingsSvc,
		authzSvc:     authzSvc,
		hub:          hub,
		uploader:     uploader,
	}
}

func rowToResponse(r repository.AnnouncementRow) dto.AnnouncementResponse {
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

func commentRowToResponse(c repository.AnnouncementCommentRow, mediaRows []repository.AnnouncementCommentMediaRow) dto.AnnouncementCommentResponse {
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
		Media:     model.CommentMediaRowsToResponse(mediaRows),
		LikeCount: c.LikeCount,
		UserLiked: c.UserLiked,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (s *service) List(ctx context.Context, limit, offset int) (*dto.AnnouncementListResponse, error) {
	rows, total, err := s.repo.List(ctx, limit, offset)
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
		Limit:         limit,
		Offset:        offset,
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
		flat[i] = commentRowToResponse(c, mediaMap[c.ID])
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
	resp := rowToResponse(*row)
	return &resp, nil
}

func (s *service) Create(ctx context.Context, userID uuid.UUID, title, body string) (uuid.UUID, error) {
	if title == "" || body == "" {
		return uuid.Nil, ErrEmptyTitleOrBody
	}
	id := uuid.New()
	if err := s.repo.Create(ctx, id, userID, title, body); err != nil {
		return uuid.Nil, err
	}
	s.hub.Broadcast(ws.Message{
		Type: "new_announcement",
		Data: map[string]interface{}{
			"id":        id,
			"title":     title,
			"author_id": userID,
		},
	})
	return id, nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, title, body string) error {
	if title == "" || body == "" {
		return ErrEmptyTitleOrBody
	}
	return s.repo.Update(ctx, id, title, body)
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) SetPinned(ctx context.Context, id uuid.UUID, pinned bool) error {
	return s.repo.SetPinned(ctx, id, pinned)
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

	id := uuid.New()
	if err := s.repo.CreateComment(ctx, id, announcementID, parentID, userID, body); err != nil {
		logger.Log.Error().Err(err).
			Str("announcement_id", announcementID.String()).
			Str("user_id", userID.String()).
			Msg("failed to create announcement comment")
		return uuid.Nil, err
	}

	go s.notifyCommentCreated(ann, announcementID, id, userID, parentID)

	return id, nil
}

func (s *service) notifyCommentCreated(ann *repository.AnnouncementRow, announcementID, commentID, actorID uuid.UUID, parentID *uuid.UUID) {
	bgCtx := context.Background()
	actor, err := s.userRepo.GetByID(bgCtx, actorID)
	if err != nil || actor == nil {
		return
	}
	baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
	linkURL := fmt.Sprintf("%s/announcements/%s#comment-%s", baseURL, announcementID, commentID)

	subject, emailBody := notification.NotifEmail(actor.DisplayName, "commented on your announcement", ann.Title, linkURL)
	_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
		RecipientID:   ann.AuthorID,
		Type:          dto.NotifAnnouncementCommented,
		ReferenceID:   announcementID,
		ReferenceType: fmt.Sprintf("announcement_comment:%s", commentID),
		ActorID:       actorID,
		EmailSubject:  subject,
		EmailBody:     emailBody,
	})

	if parentID != nil {
		parentAuthor, err := s.repo.GetCommentAuthorID(bgCtx, *parentID)
		if err == nil && parentAuthor != ann.AuthorID {
			replySubject, replyBody := notification.NotifEmail(actor.DisplayName, "replied to your comment", ann.Title, linkURL)
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   parentAuthor,
				Type:          dto.NotifAnnouncementCommentReply,
				ReferenceID:   announcementID,
				ReferenceType: fmt.Sprintf("announcement_comment:%s", commentID),
				ActorID:       actorID,
				EmailSubject:  replySubject,
				EmailBody:     replyBody,
			})
		}
	}
}

func (s *service) UpdateComment(ctx context.Context, id, userID uuid.UUID, body string) error {
	if body == "" {
		return ErrEmptyBody
	}
	if s.authzSvc.Can(ctx, userID, authz.PermEditAnyComment) {
		return s.repo.UpdateCommentAsAdmin(ctx, id, body)
	}
	if err := s.repo.UpdateComment(ctx, id, userID, body); err != nil {
		return ErrForbidden
	}
	return nil
}

func (s *service) DeleteComment(ctx context.Context, id, userID uuid.UUID) error {
	if s.authzSvc.Can(ctx, userID, authz.PermDeleteAnyComment) {
		return s.repo.DeleteCommentAsAdmin(ctx, id)
	}
	if err := s.repo.DeleteComment(ctx, id, userID); err != nil {
		return ErrForbidden
	}
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
	announcementID, err := s.repo.GetCommentAnnouncementID(bgCtx, commentID)
	if err != nil {
		return
	}
	actor, err := s.userRepo.GetByID(bgCtx, actorID)
	if err != nil || actor == nil {
		return
	}
	baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
	linkURL := fmt.Sprintf("%s/announcements/%s#comment-%s", baseURL, announcementID, commentID)
	subject, body := notification.NotifEmail(actor.DisplayName, "liked your comment", "", linkURL)
	_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
		RecipientID:   recipientID,
		Type:          dto.NotifAnnouncementCommentLiked,
		ReferenceID:   announcementID,
		ReferenceType: fmt.Sprintf("announcement_comment:%s", commentID),
		ActorID:       actorID,
		EmailSubject:  subject,
		EmailBody:     body,
	})
}

func (s *service) UnlikeComment(ctx context.Context, userID, commentID uuid.UUID) error {
	return s.repo.UnlikeComment(ctx, userID, commentID)
}

func (s *service) UploadCommentMedia(ctx context.Context, commentID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error) {
	authorID, err := s.repo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return nil, ErrCommentNotFound
	}
	if authorID != userID {
		return nil, ErrForbidden
	}

	return s.uploader.SaveAndRecord(ctx, "announcements", contentType, fileSize, reader,
		func(mediaURL, mediaType, thumbURL string, sortOrder int) (int64, error) {
			return s.repo.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbURL, sortOrder)
		},
		s.repo.UpdateCommentMediaURL,
		s.repo.UpdateCommentMediaThumbnail,
	)
}
