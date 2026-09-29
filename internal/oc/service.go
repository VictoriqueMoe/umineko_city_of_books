package oc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/og"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
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
			page bounds.Page,
		) (*dto.OCListResponse, error)
		ListOCsByUser(
			ctx context.Context,
			userID uuid.UUID,
			viewerID uuid.UUID,
			page bounds.Page,
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
			filename string,
			fileSize int64,
			reader io.Reader,
			isSpoiler bool,
		) (*dto.PostMediaResponse, error)
	}

	service struct {
		ocRepo        repository.OCRepository
		userRepo      repository.UserRepository
		auditRepo     repository.AuditLogRepository
		authz         authz.Service
		blockSvc      block.Service
		notifService  notification.Service
		mentionSvc    mention.Service
		uploadSvc     upload.Service
		hub           *ws.Hub
		uploader      *media.Uploader
		settingsSvc   settings.Service
		contentFilter *contentfilter.Manager
		ogCache       *og.Resolver
	}
)

func NewService(
	ocRepo repository.OCRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditLogRepository,
	authzService authz.Service,
	blockSvc block.Service,
	notifService notification.Service,
	mentionSvc mention.Service,
	uploadSvc upload.Service,
	mediaProc *media.Processor,
	settingsSvc settings.Service,
	hub *ws.Hub,
	contentFilter *contentfilter.Manager,
	ogCache *og.Resolver,
) Service {
	return &service{
		ocRepo:        ocRepo,
		userRepo:      userRepo,
		auditRepo:     auditRepo,
		authz:         authzService,
		blockSvc:      blockSvc,
		notifService:  notifService,
		mentionSvc:    mentionSvc,
		uploadSvc:     uploadSvc,
		hub:           hub,
		uploader:      media.NewUploader(uploadSvc, settingsSvc, mediaProc),
		settingsSvc:   settingsSvc,
		contentFilter: contentFilter,
		ogCache:       ogCache,
	}
}

func (s *service) writeAudit(ctx context.Context, entry audit.NewEntry) {
	if err := s.auditRepo.Create(ctx, entry); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("action", string(entry.Action)).Msg("failed to write audit log")
	}
}

func (s *service) ocAuthor(ctx context.Context, ocID uuid.UUID) (uuid.UUID, error) {
	authorID, err := s.ocRepo.GetAuthorID(ctx, ocID)
	if errors.Is(err, dao.ErrNotFound) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}

	return authorID, nil
}

func (s *service) ocCommentAuthor(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	authorID, err := s.ocRepo.GetCommentAuthorID(ctx, commentID)
	if errors.Is(err, dao.ErrNotFound) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}

	return authorID, nil
}

func (s *service) assertNotBlocked(ctx context.Context, userID, authorID uuid.UUID) error {
	blocked, err := s.blockSvc.IsBlockedEither(ctx, userID, authorID)
	if err != nil {
		return fmt.Errorf("block check: %w", err)
	}
	if blocked {
		return block.ErrUserBlocked
	}

	return nil
}

func (s *service) sendOwnerOCUpdated(ctx context.Context, ocID, ownerID uuid.UUID) {
	row, err := s.ocRepo.GetByID(ctx, spec.OCByID{ID: ocID, ViewerID: ownerID})
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("oc_id", ocID.String()).Msg("oc updated event skipped, reload failed")

		return
	}

	s.sendOwnerOCEvent(ownerID, "updated", row)
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
		Data: map[string]any{
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
		Data: map[string]any{
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
	if err := s.contentFilter.Check(ctx, name, req.Description, customSeriesName); err != nil {
		return uuid.Nil, err
	}

	exists, err := s.ocRepo.HasOC(ctx, spec.OCNameLookup{UserID: userID, Name: name})
	if err != nil {
		return uuid.Nil, err
	}
	if exists {
		return uuid.Nil, ErrDuplicateName
	}

	description := strings.TrimSpace(req.Description)

	created, err := s.ocRepo.Create(ctx, spec.NewOC{
		UserID:           userID,
		Name:             name,
		Description:      description,
		Series:           series,
		CustomSeriesName: customSeriesName,
	})
	if err != nil {
		return uuid.Nil, err
	}

	s.sendOwnerOCEvent(userID, "created", created)

	s.mentionSvc.NotifyAsync(ctx, mention.Reference{Kind: mention.KindOC, EntityID: created.ID}, userID, description)

	return created.ID, nil
}

func (s *service) GetOC(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.OCDetailResponse, error) {
	row, err := s.ocRepo.GetByID(ctx, spec.OCByID{ID: id, ViewerID: viewerID})
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}

	gallery, err := s.ocRepo.GetGallery(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("oc gallery: %w", err)
	}

	blockedIDs, err := s.blockSvc.GetBlockedIDs(ctx, viewerID)
	if err != nil {
		return nil, fmt.Errorf("blocked users: %w", err)
	}

	comments, _, err := s.ocRepo.GetComments(ctx, spec.CommentQuery[uuid.UUID]{
		TargetID:       id,
		ViewerID:       viewerID,
		Limit:          500,
		Offset:         0,
		ExcludeUserIDs: blockedIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("oc comments: %w", err)
	}

	commentIDs := make([]uuid.UUID, len(comments))
	for i, c := range comments {
		commentIDs[i] = c.ID
	}
	commentMediaMap, err := s.ocRepo.GetCommentMediaBatch(ctx, commentIDs)
	if err != nil {
		return nil, fmt.Errorf("oc comment media: %w", err)
	}

	flatComments := make([]dto.OCCommentResponse, len(comments))
	for i, c := range comments {
		flatComments[i] = ocCommentToResponse(c, commentMediaMap[c.ID])
	}
	threaded := utils.BuildTree(flatComments,
		func(c dto.OCCommentResponse) uuid.UUID { return c.ID },
		func(c dto.OCCommentResponse) *uuid.UUID { return c.ParentID },
		func(c *dto.OCCommentResponse, replies []dto.OCCommentResponse) { c.Replies = replies },
	)

	viewerBlocked := false
	if viewerID != uuid.Nil {
		viewerBlocked, err = s.blockSvc.IsBlockedEither(ctx, viewerID, row.UserID)
		if err != nil {
			return nil, fmt.Errorf("viewer block check: %w", err)
		}
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
	if err := s.contentFilter.Check(ctx, name, req.Description, customSeriesName); err != nil {
		return err
	}

	description := strings.TrimSpace(req.Description)
	asAdmin := s.authz.Can(ctx, userID, authz.PermEditAnyPost)
	if err := s.ocRepo.Update(ctx, spec.OCUpdate{
		ID:               id,
		UserID:           userID,
		Name:             name,
		Description:      description,
		Series:           series,
		CustomSeriesName: customSeriesName,
		AsAdmin:          asAdmin,
	}); err != nil {
		return err
	}

	ownerID, err := s.ocRepo.GetAuthorID(ctx, id)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("oc_id", id.String()).Msg("oc updated event skipped, owner lookup failed")

		return nil
	}

	s.sendOwnerOCUpdated(ctx, id, ownerID)

	return nil
}

func (s *service) DeleteOC(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	ownerID, err := s.ocAuthor(ctx, id)
	if err != nil {
		return err
	}

	asAdmin := s.authz.Can(ctx, userID, authz.PermDeleteAnyPost)

	paths, err := s.ocRepo.DeleteOC(ctx, spec.OCDeletion{
		ID:      id,
		UserID:  userID,
		AsAdmin: asAdmin,
	})
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(paths...)

	s.sendOwnerOCDeleted(ownerID, id)

	action := audit.ActionOCDelete
	if ownerID != userID {
		action = audit.ActionOCDeleteAdmin
	}

	s.writeAudit(ctx, audit.NewEntry{
		ActorID:    userID,
		Action:     action,
		TargetType: audit.TargetOC,
		TargetID:   id.String(),
		SubjectID:  ownerID,
	})

	if err := s.ogCache.ClearMetaCache(ctx, og.KindOC, id.String()); err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("oc_id", id.String()).Msg("clear og meta cache failed")
	}

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
	page bounds.Page,
) (*dto.OCListResponse, error) {
	blockedIDs, err := s.blockSvc.GetBlockedIDs(ctx, viewerID)
	if err != nil {
		return nil, fmt.Errorf("blocked users: %w", err)
	}

	rows, total, err := s.ocRepo.List(ctx, spec.OCListFilter{
		ViewerID:         viewerID,
		Sort:             sort,
		CrackOCsOnly:     crackOCsOnly,
		Series:           series,
		CustomSeriesName: customSeriesName,
		OwnerID:          ownerID,
		Limit:            page.Limit(),
		Offset:           page.Offset(),
		ExcludeUserIDs:   blockedIDs,
	})
	if err != nil {
		return nil, err
	}

	return s.buildOCList(ctx, rows, total, page.Limit(), page.Offset())
}

func (s *service) ListOCsByUser(
	ctx context.Context,
	userID uuid.UUID,
	viewerID uuid.UUID,
	page bounds.Page,
) (*dto.OCListResponse, error) {
	rows, total, err := s.ocRepo.ListByUser(ctx, spec.OCUserListFilter{
		UserID:   userID,
		ViewerID: viewerID,
		Limit:    page.Limit(),
		Offset:   page.Offset(),
	})
	if err != nil {
		return nil, err
	}

	return s.buildOCList(ctx, rows, total, page.Limit(), page.Offset())
}

func (s *service) buildOCList(ctx context.Context, rows []model.OCRow, total, limit, offset int) (*dto.OCListResponse, error) {
	ocIDs := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ocIDs[i] = r.ID
	}
	galleryMap, err := s.ocRepo.GetGalleryBatch(ctx, ocIDs)
	if err != nil {
		return nil, fmt.Errorf("oc galleries: %w", err)
	}

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
	authorID, err := s.ocAuthor(ctx, ocID)
	if err != nil {
		return "", err
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyPost) {
		return "", ErrNotOwner
	}

	mediaID := uuid.New()
	maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
	urlPath, err := s.uploadSvc.SaveImage(ctx, "ocs", mediaID, fileSize, maxSize, reader)
	if err != nil {
		return "", err
	}

	if err := s.ocRepo.UpdateImage(ctx, spec.OCImageUpdate{ID: ocID, ImageURL: urlPath, ThumbnailURL: ""}); err != nil {
		return "", err
	}

	s.sendOwnerOCUpdated(ctx, ocID, authorID)

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
	authorID, err := s.ocAuthor(ctx, ocID)
	if err != nil {
		return nil, err
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyPost) {
		return nil, ErrNotOwner
	}

	caption = strings.TrimSpace(caption)
	if err := s.contentFilter.Check(ctx, caption); err != nil {
		return nil, err
	}

	existing, err := s.ocRepo.GetGallery(ctx, ocID)
	if err != nil {
		return nil, fmt.Errorf("oc gallery: %w", err)
	}

	mediaID := uuid.New()
	maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
	urlPath, err := s.uploadSvc.SaveImage(ctx, "ocs", mediaID, fileSize, maxSize, reader)
	if err != nil {
		return nil, err
	}

	id, err := s.ocRepo.AddGalleryImage(ctx, spec.NewOCGalleryImage{
		OCID:         ocID,
		ImageURL:     urlPath,
		ThumbnailURL: "",
		Caption:      caption,
		SortOrder:    len(existing),
	})
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
	authorID, err := s.ocAuthor(ctx, ocID)
	if err != nil {
		return err
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyPost) {
		return ErrNotOwner
	}
	if req.Caption != nil {
		trimmed := strings.TrimSpace(*req.Caption)
		req.Caption = &trimmed
		if err := s.contentFilter.Check(ctx, trimmed); err != nil {
			return err
		}
	}
	if req.SortOrder != nil {
		clamped := max(*req.SortOrder, 0)
		req.SortOrder = &clamped
	}

	return s.ocRepo.UpdateGalleryImage(ctx, spec.OCGalleryImageUpdate{
		ID:        imageID,
		OCID:      ocID,
		Caption:   req.Caption,
		SortOrder: req.SortOrder,
	})
}

func (s *service) DeleteGalleryImage(ctx context.Context, ocID uuid.UUID, imageID int64, userID uuid.UUID) error {
	authorID, err := s.ocAuthor(ctx, ocID)
	if err != nil {
		return err
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyPost) {
		return ErrNotOwner
	}
	return s.ocRepo.DeleteGalleryImage(ctx, spec.MediaDeletion{ID: imageID, TargetID: ocID})
}

func (s *service) Vote(ctx context.Context, userID uuid.UUID, ocID uuid.UUID, value int) error {
	authorID, err := s.ocAuthor(ctx, ocID)
	if err != nil {
		return err
	}
	if err := s.assertNotBlocked(ctx, userID, authorID); err != nil {
		return err
	}
	return s.ocRepo.Vote(ctx, spec.Vote{UserID: userID, TargetID: ocID, Value: value})
}

func (s *service) ToggleFavourite(ctx context.Context, userID uuid.UUID, ocID uuid.UUID) (bool, error) {
	authorID, err := s.ocAuthor(ctx, ocID)
	if err != nil {
		return false, err
	}
	if err := s.assertNotBlocked(ctx, userID, authorID); err != nil {
		return false, err
	}

	row, err := s.ocRepo.GetByID(ctx, spec.OCByID{ID: ocID, ViewerID: userID})
	if err != nil {
		return false, err
	}
	if row == nil {
		return false, ErrNotFound
	}

	if row.UserFavourited {
		if err := s.ocRepo.Unfavourite(ctx, spec.Like{UserID: userID, TargetID: ocID}); err != nil {
			return false, err
		}
		return false, nil
	}

	if err := s.ocRepo.Favourite(ctx, spec.Like{UserID: userID, TargetID: ocID}); err != nil {
		return false, err
	}

	go func() {
		if authorID == userID {
			return
		}
		bgCtx := context.Background()
		actor, err := s.userRepo.GetByID(bgCtx, userID)
		if err != nil {
			logger.Ctx(bgCtx).Warn().Err(err).Str("user_id", userID.String()).Msg("oc favourite notification skipped, actor lookup failed")

			return
		}
		if actor == nil {
			return
		}
		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifOCFavourited,
			ReferenceID:   ocID,
			ReferenceType: "oc",
			ActorID:       userID,
			EmailActor:    actor.DisplayName,
			EmailAction:   "favourited your OC",
			EmailTitle:    row.Name,
			EmailLink:     fmt.Sprintf("/oc/%s", ocID),
		})
	}()

	return true, nil
}

func (s *service) CreateComment(ctx context.Context, ocID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error) {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return uuid.Nil, ErrEmptyBody
	}
	if err := s.contentFilter.Check(ctx, body); err != nil {
		return uuid.Nil, err
	}

	authorID, err := s.ocAuthor(ctx, ocID)
	if err != nil {
		return uuid.Nil, err
	}
	if err := s.assertNotBlocked(ctx, userID, authorID); err != nil {
		return uuid.Nil, err
	}

	id, err := s.mentionSvc.CreateComment(ctx, mention.CommentSpec{
		Kind:     mention.KindOCComment,
		EntityID: ocID,
		ParentID: req.ParentID,
		AuthorID: userID,
		Body:     body,
	})
	if err != nil {
		return uuid.Nil, err
	}

	go func() {
		bgCtx := context.Background()
		actor, err := s.userRepo.GetByID(bgCtx, userID)
		if err != nil {
			logger.Ctx(bgCtx).Warn().Err(err).Str("user_id", userID.String()).Msg("oc comment notification skipped, actor lookup failed")

			return
		}
		if actor == nil {
			return
		}
		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifOCCommented,
			ReferenceID:   ocID,
			ReferenceType: fmt.Sprintf("oc_comment:%s", id),
			ActorID:       userID,
			EmailActor:    actor.DisplayName,
			EmailAction:   "commented on your OC",
			EmailLink:     fmt.Sprintf("/oc/%s#comment-%s", ocID, id),
		})

		if req.ParentID != nil {
			parentAuthor, err := s.ocRepo.GetCommentAuthorID(bgCtx, *req.ParentID)
			if err != nil {
				logger.Ctx(bgCtx).Warn().Err(err).Str("comment_id", req.ParentID.String()).Msg("oc reply notification skipped, parent lookup failed")

				return
			}
			if parentAuthor != authorID {
				_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
					RecipientID:   parentAuthor,
					Type:          dto.NotifOCCommentReply,
					ReferenceID:   ocID,
					ReferenceType: fmt.Sprintf("oc_comment:%s", id),
					ActorID:       userID,
					EmailActor:    actor.DisplayName,
					EmailAction:   "replied to your comment",
					EmailLink:     fmt.Sprintf("/oc/%s#comment-%s", ocID, id),
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
	if err := s.contentFilter.Check(ctx, body); err != nil {
		return err
	}

	authorID, err := s.ocCommentAuthor(ctx, id)
	if err != nil {
		return err
	}

	asAdmin := s.authz.Can(ctx, userID, authz.PermEditAnyComment)

	if err := s.ocRepo.UpdateComment(ctx, spec.CommentUpdate{
		CommentID: id,
		UserID:    userID,
		Body:      body,
		AsAdmin:   asAdmin,
	}); err != nil {
		return err
	}

	if authorID == userID {
		return nil
	}

	s.writeAudit(ctx, audit.NewEntry{
		ActorID:    userID,
		Action:     audit.ActionOCCommentUpdateAdmin,
		TargetType: audit.TargetOCComment,
		TargetID:   id.String(),
		SubjectID:  authorID,
	})

	return nil
}

func (s *service) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	asAdmin := s.authz.Can(ctx, userID, authz.PermDeleteAnyComment)

	paths, err := s.ocRepo.DeleteCommentWithMedia(ctx, spec.CommentDeletion{
		CommentID: id,
		UserID:    userID,
		AsAdmin:   asAdmin,
	})
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(paths...)

	return nil
}

func (s *service) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	commentAuthorID, err := s.ocCommentAuthor(ctx, commentID)
	if err != nil {
		return err
	}
	if err := s.assertNotBlocked(ctx, userID, commentAuthorID); err != nil {
		return err
	}
	if err := s.ocRepo.LikeComment(ctx, spec.CommentLike{UserID: userID, CommentID: commentID}); err != nil {
		return err
	}

	go func() {
		if commentAuthorID == userID {
			return
		}
		bgCtx := context.Background()
		ocID, err := s.ocRepo.GetCommentEntityID(bgCtx, commentID)
		if err != nil {
			logger.Ctx(bgCtx).Warn().Err(err).Str("comment_id", commentID.String()).Msg("oc comment like notification skipped, oc lookup failed")

			return
		}
		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   commentAuthorID,
			Type:          dto.NotifOCCommentLiked,
			ReferenceID:   ocID,
			ReferenceType: fmt.Sprintf("oc_comment:%s", commentID),
			ActorID:       userID,
			EmailActor:    "Someone",
			EmailAction:   "liked your comment",
			EmailLink:     fmt.Sprintf("/oc/%s#comment-%s", ocID, commentID),
		})
	}()

	return nil
}

func (s *service) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return s.ocRepo.UnlikeComment(ctx, spec.CommentLike{UserID: userID, CommentID: commentID})
}

func (s *service) UploadCommentMedia(
	ctx context.Context,
	commentID uuid.UUID,
	userID uuid.UUID,
	contentType string,
	filename string,
	fileSize int64,
	reader io.Reader,
	isSpoiler bool,
) (*dto.PostMediaResponse, error) {
	authorID, err := s.ocCommentAuthor(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if authorID != userID {
		return nil, authz.ErrNotCommentAuthor
	}

	return s.uploader.SaveAndRecord(ctx, "ocs", contentType, filename, fileSize, reader, isSpoiler,
		func(mediaURL, mediaType, thumbURL, filename string, sortOrder int) (int64, error) {
			return s.ocRepo.AddCommentMedia(ctx, spec.NewMedia{
				TargetID:     commentID,
				MediaURL:     mediaURL,
				MediaType:    mediaType,
				ThumbnailURL: thumbURL,
				Filename:     filename,
				SortOrder:    sortOrder,
				IsSpoiler:    isSpoiler,
			})
		},
		s.ocRepo.UpdateCommentMediaURL,
		s.ocRepo.UpdateCommentMediaThumbnail,
	)
}

func ocCommentToResponse(c model.CommentRow, media []model.PostMediaRow) dto.OCCommentResponse {
	return dto.OCCommentResponse{
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
