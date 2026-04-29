package oc

import (
	"context"
	"fmt"
	"io"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/utils"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

const (
	seriesUmineko   = "umineko"
	seriesHigurashi = "higurashi"
	seriesCiconia   = "ciconia"
	seriesCustom    = "custom"
)

type (
	Service interface {
		CreateOC(ctx context.Context, userID uuid.UUID, req dto.CreateOCRequest) (uuid.UUID, error)
		GetOC(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.OCDetailResponse, error)
		UpdateOC(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateOCRequest) error
		DeleteOC(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		ListOCs(
			ctx context.Context,
			viewerID uuid.UUID,
			sort string,
			crackOCsOnly bool,
			series string,
			customSeriesName string,
			ownerID uuid.UUID,
			limit, offset int,
		) (*dto.OCListResponse, error)
		ListOCsByUser(
			ctx context.Context,
			userID uuid.UUID,
			viewerID uuid.UUID,
			limit, offset int,
		) (*dto.OCListResponse, error)
		ListOCSummariesByUser(ctx context.Context, userID uuid.UUID) ([]dto.OCSummary, error)
		UploadOCImage(
			ctx context.Context,
			ocID uuid.UUID,
			userID uuid.UUID,
			contentType string,
			fileSize int64,
			reader io.Reader,
		) (string, error)
		AddGalleryImage(
			ctx context.Context,
			ocID uuid.UUID,
			userID uuid.UUID,
			caption string,
			contentType string,
			fileSize int64,
			reader io.Reader,
		) (*dto.OCImage, error)
		UpdateGalleryImage(ctx context.Context, ocID uuid.UUID, imageID int64, userID uuid.UUID, req dto.UpdateOCImageRequest) error
		DeleteGalleryImage(ctx context.Context, ocID uuid.UUID, imageID int64, userID uuid.UUID) error

		Vote(ctx context.Context, userID uuid.UUID, ocID uuid.UUID, value int) error
		ToggleFavourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID) (bool, error)

		CreateComment(ctx context.Context, ocID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateCommentRequest) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UploadCommentMedia(
			ctx context.Context,
			commentID uuid.UUID,
			userID uuid.UUID,
			contentType string,
			fileSize int64,
			reader io.Reader,
		) (*dto.PostMediaResponse, error)
	}

	service struct {
		ocRepo        repository.OCRepository
		userRepo      repository.UserRepository
		authz         authz.Service
		blockSvc      block.Service
		notifService  notification.Service
		uploadSvc     upload.Service
		hub           *ws.Hub
		uploader      *media.Uploader
		settingsSvc   settings.Service
		contentFilter *contentfilter.Manager
	}
)

func NewService(
	ocRepo repository.OCRepository,
	userRepo repository.UserRepository,
	authzService authz.Service,
	blockSvc block.Service,
	notifService notification.Service,
	uploadSvc upload.Service,
	mediaProc *media.Processor,
	settingsSvc settings.Service,
	hub *ws.Hub,
	contentFilter *contentfilter.Manager,
) Service {
	return &service{
		ocRepo:        ocRepo,
		userRepo:      userRepo,
		authz:         authzService,
		blockSvc:      blockSvc,
		notifService:  notifService,
		uploadSvc:     uploadSvc,
		hub:           hub,
		uploader:      media.NewUploader(uploadSvc, settingsSvc, mediaProc),
		settingsSvc:   settingsSvc,
		contentFilter: contentFilter,
	}
}

func (s *service) filterTexts(ctx context.Context, texts ...string) error {
	if s.contentFilter == nil {
		return nil
	}
	return s.contentFilter.Check(ctx, texts...)
}

func validateSeries(series string, customSeriesName string) (string, string, error) {
	series = strings.ToLower(strings.TrimSpace(series))
	customSeriesName = strings.TrimSpace(customSeriesName)
	switch series {
	case seriesUmineko, seriesHigurashi, seriesCiconia:
		return series, "", nil
	case seriesCustom:
		if customSeriesName == "" {
			return "", "", ErrEmptyCustomSeries
		}
		return series, customSeriesName, nil
	default:
		return "", "", ErrInvalidSeries
	}
}

func (s *service) sendOwnerOCEvent(ownerID uuid.UUID, action string, oc *model.OCRow) {
	if s.hub == nil {
		return
	}
	var summary dto.OCSummary
	if oc != nil {
		summary = dto.OCSummary{
			ID:               oc.ID,
			Name:             oc.Name,
			Series:           oc.Series,
			CustomSeriesName: oc.CustomSeriesName,
			ThumbnailURL:     oc.ThumbnailURL,
		}
		if summary.ThumbnailURL == "" {
			summary.ThumbnailURL = oc.ImageURL
		}
	}
	s.hub.SendToUser(ownerID, ws.Message{
		Type: "user_ocs_changed",
		Data: map[string]interface{}{
			"action": action,
			"oc":     summary,
		},
	})
}

func (s *service) sendOwnerOCDeleted(ownerID uuid.UUID, ocID uuid.UUID) {
	if s.hub == nil {
		return
	}
	s.hub.SendToUser(ownerID, ws.Message{
		Type: "user_ocs_changed",
		Data: map[string]interface{}{
			"action": "deleted",
			"oc":     dto.OCSummary{ID: ocID},
		},
	})
}

func (s *service) CreateOC(ctx context.Context, userID uuid.UUID, req dto.CreateOCRequest) (uuid.UUID, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return uuid.Nil, ErrEmptyName
	}
	series, customSeriesName, err := validateSeries(req.Series, req.CustomSeriesName)
	if err != nil {
		return uuid.Nil, err
	}
	if err := s.filterTexts(ctx, name, req.Description, customSeriesName); err != nil {
		return uuid.Nil, err
	}

	exists, err := s.ocRepo.HasOC(ctx, userID, name)
	if err != nil {
		return uuid.Nil, err
	}
	if exists {
		return uuid.Nil, ErrDuplicateName
	}

	id := uuid.New()
	description := strings.TrimSpace(req.Description)
	if err := s.ocRepo.Create(ctx, id, userID, name, description, series, customSeriesName); err != nil {
		return uuid.Nil, err
	}

	row, _ := s.ocRepo.GetByID(ctx, id, userID)
	s.sendOwnerOCEvent(userID, "created", row)
	return id, nil
}

func (s *service) GetOC(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.OCDetailResponse, error) {
	row, err := s.ocRepo.GetByID(ctx, id, viewerID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}

	gallery, _ := s.ocRepo.GetGallery(ctx, id)
	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)

	comments, _, _ := s.ocRepo.GetComments(ctx, id, viewerID, 500, 0, blockedIDs)

	commentIDs := make([]uuid.UUID, len(comments))
	for i, c := range comments {
		commentIDs[i] = c.ID
	}
	commentMediaMap, _ := s.ocRepo.GetCommentMediaBatch(ctx, commentIDs)

	flatComments := make([]dto.OCCommentResponse, len(comments))
	for i, c := range comments {
		flatComments[i] = c.ToResponse(commentMediaMap[c.ID])
	}
	threaded := utils.BuildTree(flatComments,
		func(c dto.OCCommentResponse) uuid.UUID { return c.ID },
		func(c dto.OCCommentResponse) *uuid.UUID { return c.ParentID },
		func(c *dto.OCCommentResponse, replies []dto.OCCommentResponse) { c.Replies = replies },
	)

	viewerBlocked := false
	if viewerID != uuid.Nil {
		viewerBlocked, _ = s.blockSvc.IsBlockedEither(ctx, viewerID, row.UserID)
	}

	return &dto.OCDetailResponse{
		OCResponse:    row.ToResponse(gallery),
		Comments:      threaded,
		ViewerBlocked: viewerBlocked,
	}, nil
}

func (s *service) UpdateOC(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateOCRequest) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ErrEmptyName
	}
	series, customSeriesName, err := validateSeries(req.Series, req.CustomSeriesName)
	if err != nil {
		return err
	}
	if err := s.filterTexts(ctx, name, req.Description, customSeriesName); err != nil {
		return err
	}

	description := strings.TrimSpace(req.Description)
	asAdmin := s.authz.Can(ctx, userID, authz.PermEditAnyPost)
	if err := s.ocRepo.Update(ctx, id, userID, name, description, series, customSeriesName, asAdmin); err != nil {
		return err
	}

	ownerID, err := s.ocRepo.GetAuthorID(ctx, id)
	if err == nil {
		row, _ := s.ocRepo.GetByID(ctx, id, ownerID)
		s.sendOwnerOCEvent(ownerID, "updated", row)
	}
	return nil
}

func (s *service) DeleteOC(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	ownerID, err := s.ocRepo.GetAuthorID(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyPost) {
		if err := s.ocRepo.DeleteAsAdmin(ctx, id); err != nil {
			return err
		}
	} else {
		if err := s.ocRepo.Delete(ctx, id, userID); err != nil {
			return err
		}
	}
	s.sendOwnerOCDeleted(ownerID, id)
	return nil
}

func (s *service) ListOCs(
	ctx context.Context,
	viewerID uuid.UUID,
	sort string,
	crackOCsOnly bool,
	series string,
	customSeriesName string,
	ownerID uuid.UUID,
	limit, offset int,
) (*dto.OCListResponse, error) {
	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)

	rows, total, err := s.ocRepo.List(ctx, viewerID, sort, crackOCsOnly, series, customSeriesName, ownerID, limit, offset, blockedIDs)
	if err != nil {
		return nil, err
	}

	ocIDs := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ocIDs[i] = r.ID
	}
	galleryMap, _ := s.ocRepo.GetGalleryBatch(ctx, ocIDs)

	ocs := make([]dto.OCResponse, len(rows))
	for i, r := range rows {
		ocs[i] = r.ToResponse(galleryMap[r.ID])
	}

	return &dto.OCListResponse{
		OCs:    ocs,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *service) ListOCsByUser(
	ctx context.Context,
	userID uuid.UUID,
	viewerID uuid.UUID,
	limit, offset int,
) (*dto.OCListResponse, error) {
	rows, total, err := s.ocRepo.ListByUser(ctx, userID, viewerID, limit, offset)
	if err != nil {
		return nil, err
	}

	ocIDs := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ocIDs[i] = r.ID
	}
	galleryMap, _ := s.ocRepo.GetGalleryBatch(ctx, ocIDs)

	ocs := make([]dto.OCResponse, len(rows))
	for i, r := range rows {
		ocs[i] = r.ToResponse(galleryMap[r.ID])
	}

	return &dto.OCListResponse{
		OCs:    ocs,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *service) ListOCSummariesByUser(ctx context.Context, userID uuid.UUID) ([]dto.OCSummary, error) {
	rows, err := s.ocRepo.ListSummariesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.OCSummary, len(rows))
	for i := range rows {
		out[i] = rows[i].ToResponse()
	}
	return out, nil
}

func (s *service) UploadOCImage(ctx context.Context, ocID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (string, error) {
	authorID, err := s.ocRepo.GetAuthorID(ctx, ocID)
	if err != nil {
		return "", ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyPost) {
		return "", fmt.Errorf("not the oc owner")
	}

	mediaID := uuid.New()
	maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
	urlPath, err := s.uploadSvc.SaveImage(ctx, "ocs", mediaID, fileSize, maxSize, reader)
	if err != nil {
		return "", err
	}

	if err := s.ocRepo.UpdateImage(ctx, ocID, urlPath, ""); err != nil {
		return "", err
	}

	row, _ := s.ocRepo.GetByID(ctx, ocID, authorID)
	s.sendOwnerOCEvent(authorID, "updated", row)
	return urlPath, nil
}

func (s *service) AddGalleryImage(
	ctx context.Context,
	ocID uuid.UUID,
	userID uuid.UUID,
	caption string,
	contentType string,
	fileSize int64,
	reader io.Reader,
) (*dto.OCImage, error) {
	authorID, err := s.ocRepo.GetAuthorID(ctx, ocID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyPost) {
		return nil, fmt.Errorf("not the oc owner")
	}

	caption = strings.TrimSpace(caption)
	if err := s.filterTexts(ctx, caption); err != nil {
		return nil, err
	}

	mediaID := uuid.New()
	maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
	urlPath, err := s.uploadSvc.SaveImage(ctx, "ocs", mediaID, fileSize, maxSize, reader)
	if err != nil {
		return nil, err
	}

	existing, _ := s.ocRepo.GetGallery(ctx, ocID)
	id, err := s.ocRepo.AddGalleryImage(ctx, ocID, urlPath, "", caption, len(existing))
	if err != nil {
		return nil, err
	}

	return &dto.OCImage{
		ID:        id,
		ImageURL:  urlPath,
		Caption:   caption,
		SortOrder: len(existing),
	}, nil
}

func (s *service) UpdateGalleryImage(ctx context.Context, ocID uuid.UUID, imageID int64, userID uuid.UUID, req dto.UpdateOCImageRequest) error {
	authorID, err := s.ocRepo.GetAuthorID(ctx, ocID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyPost) {
		return fmt.Errorf("not the oc owner")
	}
	if req.Caption != nil {
		trimmed := strings.TrimSpace(*req.Caption)
		req.Caption = &trimmed
		if err := s.filterTexts(ctx, trimmed); err != nil {
			return err
		}
	}
	return s.ocRepo.UpdateGalleryImage(ctx, imageID, ocID, req.Caption, req.SortOrder)
}

func (s *service) DeleteGalleryImage(ctx context.Context, ocID uuid.UUID, imageID int64, userID uuid.UUID) error {
	authorID, err := s.ocRepo.GetAuthorID(ctx, ocID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyPost) {
		return fmt.Errorf("not the oc owner")
	}
	return s.ocRepo.DeleteGalleryImage(ctx, imageID, ocID)
}

func (s *service) Vote(ctx context.Context, userID uuid.UUID, ocID uuid.UUID, value int) error {
	authorID, err := s.ocRepo.GetAuthorID(ctx, ocID)
	if err != nil {
		return ErrNotFound
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return block.ErrUserBlocked
	}
	return s.ocRepo.Vote(ctx, userID, ocID, value)
}

func (s *service) ToggleFavourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID) (bool, error) {
	authorID, err := s.ocRepo.GetAuthorID(ctx, ocID)
	if err != nil {
		return false, ErrNotFound
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return false, block.ErrUserBlocked
	}

	row, err := s.ocRepo.GetByID(ctx, ocID, userID)
	if err != nil {
		return false, err
	}
	if row == nil {
		return false, ErrNotFound
	}

	if row.UserFavourited {
		if err := s.ocRepo.Unfavourite(ctx, userID, ocID); err != nil {
			return false, err
		}
		return false, nil
	}

	if err := s.ocRepo.Favourite(ctx, userID, ocID); err != nil {
		return false, err
	}

	go func() {
		if authorID == userID {
			return
		}
		bgCtx := context.Background()
		actor, err := s.userRepo.GetByID(bgCtx, userID)
		if err != nil || actor == nil {
			return
		}
		baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/oc/%s", baseURL, ocID)
		subject, emailBody := notification.NotifEmail(actor.DisplayName, "favourited your OC", row.Name, linkURL)
		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifOCFavourited,
			ReferenceID:   ocID,
			ReferenceType: "oc",
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     emailBody,
		})
	}()

	return true, nil
}

func (s *service) CreateComment(ctx context.Context, ocID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error) {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return uuid.Nil, ErrEmptyBody
	}
	if err := s.filterTexts(ctx, body); err != nil {
		return uuid.Nil, err
	}

	authorID, err := s.ocRepo.GetAuthorID(ctx, ocID)
	if err != nil {
		return uuid.Nil, ErrNotFound
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return uuid.Nil, block.ErrUserBlocked
	}

	id := uuid.New()
	if err := s.ocRepo.CreateComment(ctx, id, ocID, req.ParentID, userID, body); err != nil {
		return uuid.Nil, err
	}

	go func() {
		bgCtx := context.Background()
		actor, err := s.userRepo.GetByID(bgCtx, userID)
		if err != nil || actor == nil {
			return
		}
		baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/oc/%s#comment-%s", baseURL, ocID, id)

		subject, emailBody := notification.NotifEmail(actor.DisplayName, "commented on your OC", "", linkURL)
		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifOCCommented,
			ReferenceID:   ocID,
			ReferenceType: fmt.Sprintf("oc_comment:%s", id),
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     emailBody,
		})

		if req.ParentID != nil {
			parentAuthor, err := s.ocRepo.GetCommentAuthorID(bgCtx, *req.ParentID)
			if err == nil && parentAuthor != authorID {
				replySubject, replyBody := notification.NotifEmail(actor.DisplayName, "replied to your comment", "", linkURL)
				_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
					RecipientID:   parentAuthor,
					Type:          dto.NotifOCCommentReply,
					ReferenceID:   ocID,
					ReferenceType: fmt.Sprintf("oc_comment:%s", id),
					ActorID:       userID,
					EmailSubject:  replySubject,
					EmailBody:     replyBody,
				})
			}
		}
	}()

	return id, nil
}

func (s *service) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateCommentRequest) error {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return ErrEmptyBody
	}
	if err := s.filterTexts(ctx, body); err != nil {
		return err
	}
	if s.authz.Can(ctx, userID, authz.PermEditAnyComment) {
		return s.ocRepo.UpdateCommentAsAdmin(ctx, id, body)
	}
	return s.ocRepo.UpdateComment(ctx, id, userID, body)
}

func (s *service) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyComment) {
		return s.ocRepo.DeleteCommentAsAdmin(ctx, id)
	}
	return s.ocRepo.DeleteComment(ctx, id, userID)
}

func (s *service) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	commentAuthorID, err := s.ocRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, commentAuthorID); blocked {
		return block.ErrUserBlocked
	}
	if err := s.ocRepo.LikeComment(ctx, userID, commentID); err != nil {
		return err
	}

	go func() {
		if commentAuthorID == userID {
			return
		}
		bgCtx := context.Background()
		ocID, err := s.ocRepo.GetCommentOCID(bgCtx, commentID)
		if err != nil {
			return
		}
		baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/oc/%s#comment-%s", baseURL, ocID, commentID)
		subject, emailBody := notification.NotifEmail("Someone", "liked your comment", "", linkURL)
		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   commentAuthorID,
			Type:          dto.NotifOCCommentLiked,
			ReferenceID:   ocID,
			ReferenceType: fmt.Sprintf("oc_comment:%s", commentID),
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     emailBody,
		})
	}()

	return nil
}

func (s *service) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return s.ocRepo.UnlikeComment(ctx, userID, commentID)
}

func (s *service) UploadCommentMedia(
	ctx context.Context,
	commentID uuid.UUID,
	userID uuid.UUID,
	contentType string,
	fileSize int64,
	reader io.Reader,
) (*dto.PostMediaResponse, error) {
	authorID, err := s.ocRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID {
		return nil, fmt.Errorf("not the comment author")
	}

	return s.uploader.SaveAndRecord(ctx, "ocs", contentType, fileSize, reader,
		func(mediaURL, mediaType, thumbURL string, sortOrder int) (int64, error) {
			return s.ocRepo.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbURL, sortOrder)
		},
		s.ocRepo.UpdateCommentMediaURL,
		s.ocRepo.UpdateCommentMediaThumbnail,
	)
}
