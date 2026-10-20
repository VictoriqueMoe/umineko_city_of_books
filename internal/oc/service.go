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
	"umineko_city_of_books/internal/comment"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/dto"
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
	}

	service struct {
		ocRepo        repository.OCRepository
		userRepo      repository.UserRepository
		auditRepo     repository.AuditLogRepository
		authz         authz.Service
		blockSvc      block.Service
		notifService  notification.Service
		mentionSvc    mention.Service
		commentSvc    comment.Service
		uploadSvc     upload.Service
		hub           *ws.Hub
		uploader      *media.Uploader
		settingsSvc   settings.Service
		contentFilter *contentfilter.Manager
		ogCache       *og.Resolver
		activity      sidebar.ActivityBumper
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
	commentSvc comment.Service,
	uploadSvc upload.Service,
	mediaProc *media.Processor,
	settingsSvc settings.Service,
	hub *ws.Hub,
	contentFilter *contentfilter.Manager,
	ogCache *og.Resolver,
	activity sidebar.ActivityBumper,
) Service {
	return &service{
		ocRepo:        ocRepo,
		userRepo:      userRepo,
		auditRepo:     auditRepo,
		authz:         authzService,
		blockSvc:      blockSvc,
		notifService:  notifService,
		mentionSvc:    mentionSvc,
		commentSvc:    commentSvc,
		uploadSvc:     uploadSvc,
		hub:           hub,
		uploader:      media.NewUploader(uploadSvc, settingsSvc, mediaProc),
		settingsSvc:   settingsSvc,
		contentFilter: contentFilter,
		ogCache:       ogCache,
		activity:      activity,
	}
}

func (s *service) writeAudit(ctx context.Context, entry audit.NewEntry) {
	if err := s.auditRepo.Create(ctx, entry); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("action", string(entry.Action)).Msg("failed to write audit log")
	}
}

func (s *service) ocAuthor(ctx context.Context, ocID uuid.UUID) (uuid.UUID, error) {
	authorID, err := s.ocRepo.GetAuthorID(ctx, ocID)
	if errors.Is(err, daoerr.ErrNotFound) {
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

	s.activity.BumpSidebarActivity(sidebar.KeyOCs)

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

	comments, err := s.commentSvc.Thread(ctx, spec.CommentTarget{Kind: spec.CommentKindOC, ID: id}, viewerID)
	if err != nil {
		return nil, fmt.Errorf("oc comments: %w", err)
	}

	viewerBlocked := false
	if viewerID != uuid.Nil {
		viewerBlocked, err = s.blockSvc.IsBlockedEither(ctx, viewerID, row.UserID)
		if err != nil {
			return nil, fmt.Errorf("viewer block check: %w", err)
		}
	}

	return &dto.OCDetailResponse{
		OCResponse:    row.ToResponse(gallery),
		Comments:      comments,
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
