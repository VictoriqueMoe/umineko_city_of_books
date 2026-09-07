package art

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/homefeed"
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

	"github.com/google/uuid"
)

type (
	Service interface {
		CreateArt(ctx context.Context, userID uuid.UUID, req dto.CreateArtRequest, contentType string, fileSize int64, reader io.Reader) (uuid.UUID, error)
		GetArt(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, viewerHash string) (*dto.ArtDetailResponse, error)
		UpdateArt(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateArtRequest) error
		DeleteArt(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		ListArt(ctx context.Context, viewerID uuid.UUID, corner string, artType string, search string, tag string, sort string, page bounds.Page) (*dto.ArtListResponse, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, page bounds.Page) (*dto.ArtListResponse, error)
		LikeArt(ctx context.Context, userID uuid.UUID, artID uuid.UUID) error
		UnlikeArt(ctx context.Context, userID uuid.UUID, artID uuid.UUID) error
		GetCornerCounts(ctx context.Context) (map[string]int, error)
		GetPopularTags(ctx context.Context, corner string) ([]dto.TagCountResponse, error)

		CreateComment(ctx context.Context, artID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateCommentRequest) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, filename string, fileSize int64, reader io.Reader, isSpoiler bool) (*dto.PostMediaResponse, error)

		CreateGallery(ctx context.Context, userID uuid.UUID, req dto.CreateGalleryRequest) (uuid.UUID, error)
		UpdateGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateGalleryRequest) error
		SetGalleryCover(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, coverArtID *uuid.UUID) error
		DeleteGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		GetGallery(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, page bounds.Page) (*dto.GalleryResponse, []dto.ArtResponse, int, error)
		ListUserGalleries(ctx context.Context, userID uuid.UUID) ([]dto.GalleryResponse, error)
		ListAllGalleries(ctx context.Context, corner string) ([]dto.GalleryResponse, error)
		SetArtGallery(ctx context.Context, artID uuid.UUID, userID uuid.UUID, galleryID *uuid.UUID) error
	}

	service struct {
		artRepo       repository.ArtRepository
		postRepo      repository.PostRepository
		userRepo      repository.UserRepository
		auditRepo     repository.AuditLogRepository
		authz         authz.Service
		blockSvc      block.Service
		notifService  notification.Service
		mentionSvc    mention.Service
		uploadSvc     upload.Service
		mediaProc     *media.Processor
		uploader      *media.Uploader
		settingsSvc   settings.Service
		contentFilter *contentfilter.Manager
		ogCache       *og.Resolver
		echoCache     homefeed.Service
	}
)

func NewService(
	artRepo repository.ArtRepository,
	postRepo repository.PostRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditLogRepository,
	authzService authz.Service,
	blockSvc block.Service,
	notifService notification.Service,
	mentionSvc mention.Service,
	uploadSvc upload.Service,
	mediaProc *media.Processor,
	settingsSvc settings.Service,
	contentFilter *contentfilter.Manager,
	ogCache *og.Resolver,
	echoCache homefeed.Service,
) Service {
	return &service{
		artRepo:       artRepo,
		postRepo:      postRepo,
		userRepo:      userRepo,
		auditRepo:     auditRepo,
		authz:         authzService,
		blockSvc:      blockSvc,
		notifService:  notifService,
		mentionSvc:    mentionSvc,
		uploadSvc:     uploadSvc,
		mediaProc:     mediaProc,
		uploader:      media.NewUploader(uploadSvc, settingsSvc, mediaProc),
		settingsSvc:   settingsSvc,
		contentFilter: contentFilter,
		ogCache:       ogCache,
		echoCache:     echoCache,
	}
}

func (s *service) clearPageCache(ctx context.Context, kind og.Kind, id string) {
	if s.ogCache != nil {
		if err := s.ogCache.ClearMetaCache(ctx, kind, id); err != nil {
			logger.Ctx(ctx).Warn().Err(err).Str("id", id).Msg("clear og meta cache failed")
		}
	}

	if kind != og.KindArt || s.echoCache == nil {
		return
	}

	if err := s.echoCache.ClearEchoCache(ctx); err != nil {
		logger.Ctx(ctx).Warn().Err(err).Msg("clear echo cache failed")
	}
}

func (s *service) writeAudit(ctx context.Context, entry audit.NewEntry) {
	if err := s.auditRepo.Create(ctx, entry); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("action", string(entry.Action)).Msg("failed to write audit log")
	}
}

func (s *service) CreateArt(ctx context.Context, userID uuid.UUID, req dto.CreateArtRequest, contentType string, fileSize int64, reader io.Reader) (uuid.UUID, error) {
	if strings.TrimSpace(req.Title) == "" {
		return uuid.Nil, ErrEmptyTitle
	}
	if err := s.contentFilter.Check(ctx, req.Title, req.Description); err != nil {
		return uuid.Nil, err
	}

	limit := s.settingsSvc.GetInt(ctx, config.SettingMaxArtPerDay)
	if limit > 0 {
		count, err := s.artRepo.CountUserArtToday(ctx, userID)
		if err != nil {
			return uuid.Nil, err
		}
		if count >= limit {
			return uuid.Nil, ErrRateLimited
		}
	}

	corner := req.Corner
	if corner == "" {
		corner = "general"
	}

	artType := req.ArtType
	if artType == "" {
		artType = "drawing"
	}

	maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
	mediaID := uuid.New()
	urlPath, err := s.uploadSvc.SaveImage(ctx, "art", mediaID, fileSize, maxSize, reader)
	if err != nil {
		return uuid.Nil, err
	}

	title := strings.TrimSpace(req.Title)
	description := strings.TrimSpace(req.Description)
	tags := req.Tags
	if len(tags) > 10 {
		tags = tags[:10]
	}

	newArt := spec.NewArtWithTags{
		NewArt: spec.NewArt{
			UserID:      userID,
			Corner:      corner,
			ArtType:     artType,
			Title:       title,
			Description: description,
			ImageURL:    urlPath,
			IsSpoiler:   req.IsSpoiler,
		},
		Tags: tags,
	}

	created, err := s.artRepo.CreateWithTags(ctx, newArt)
	if err != nil {
		return uuid.Nil, err
	}

	s.mentionSvc.NotifyAsync(ctx, mention.Reference{Kind: mention.KindArt, EntityID: created.ID}, userID, description)

	return created.ID, nil
}

func (s *service) generateThumbnailURL(imageURL string) string {
	baseURL := s.settingsSvc.Get(context.Background(), config.SettingBaseURL)
	day := time.Now().Unix() / 86400
	sourceURL := fmt.Sprintf("%s%s?v=%d", baseURL, imageURL, day)
	return "https://thumbnails.waifuvault.moe/api/v1/generateThumbnail/ext/fromURL?url=" + url.QueryEscape(sourceURL)
}

func (s *service) GetArt(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, viewerHash string) (*dto.ArtDetailResponse, error) {
	row, err := s.artRepo.GetByID(ctx, spec.ArtLookup{ID: id, ViewerID: viewerID})
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}

	if viewerHash != "" {
		isNew, _ := s.artRepo.RecordView(ctx, spec.ViewRecord{TargetID: id, ViewerHash: viewerHash})
		if isNew {
			row.ViewCount++
		}
	}

	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)

	tags, _ := s.artRepo.GetTags(ctx, id)
	comments, _, _ := s.artRepo.GetComments(ctx, spec.CommentQuery[uuid.UUID]{
		TargetID:       id,
		ViewerID:       viewerID,
		Limit:          500,
		ExcludeUserIDs: blockedIDs,
	})

	var commentIDs []uuid.UUID
	for _, c := range comments {
		commentIDs = append(commentIDs, c.ID)
	}
	commentMediaMap, _ := s.artRepo.GetCommentMediaBatch(ctx, commentIDs)

	flatComments := make([]dto.ArtCommentResponse, len(comments))
	for i, c := range comments {
		flatComments[i] = artCommentToResponse(c, commentMediaMap[c.ID])
	}
	dtoComments := utils.BuildTree(flatComments,
		func(c dto.ArtCommentResponse) uuid.UUID { return c.ID },
		func(c dto.ArtCommentResponse) *uuid.UUID { return c.ParentID },
		func(c *dto.ArtCommentResponse, replies []dto.ArtCommentResponse) { c.Replies = replies },
	)

	likeUsers, _ := s.artRepo.GetLikedBy(ctx, spec.LikedByQuery{TargetID: id, ExcludeUserIDs: blockedIDs})
	likedBy := make([]dto.UserResponse, len(likeUsers))
	for i, u := range likeUsers {
		likedBy[i] = dto.UserResponse{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			AvatarURL:   u.AvatarURL,
			Role:        role.Role(u.Role),
		}
	}

	artResp := row.ToResponse(tags)
	artResp.ThumbnailURL = s.generateThumbnailURL(row.ImageURL)

	viewerBlocked := false
	if viewerID != uuid.Nil {
		viewerBlocked, _ = s.blockSvc.IsBlockedEither(ctx, viewerID, row.UserID)
	}

	return &dto.ArtDetailResponse{
		ArtResponse:   artResp,
		Comments:      dtoComments,
		LikedBy:       likedBy,
		ViewerBlocked: viewerBlocked,
	}, nil
}

func (s *service) UpdateArt(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateArtRequest) error {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return ErrEmptyTitle
	}
	if err := s.contentFilter.Check(ctx, title, req.Description); err != nil {
		return err
	}
	description := strings.TrimSpace(req.Description)
	tags := req.Tags
	if len(tags) > 10 {
		tags = tags[:10]
	}

	asAdmin := s.authz.Can(ctx, userID, authz.PermEditAnyPost)

	update := spec.ArtUpdateWithTags{
		ArtUpdate: spec.ArtUpdate{
			ID:          id,
			UserID:      userID,
			Title:       title,
			Description: description,
			IsSpoiler:   req.IsSpoiler,
			AsAdmin:     asAdmin,
		},
		Tags: tags,
	}

	if err := s.artRepo.UpdateWithTags(ctx, update); err != nil {
		return err
	}

	if asAdmin {
		go s.notifyArtEdited(ctx, id, userID)
	}

	return nil
}

func (s *service) DeleteArt(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.artRepo.GetArtAuthorID(ctx, id)
	if err != nil {
		return err
	}

	action := audit.ActionArtDelete
	if authorID != userID {
		action = audit.ActionArtDeleteAdmin
	}

	asAdmin := s.authz.Can(ctx, userID, authz.PermDeleteAnyPost)

	paths, err := s.artRepo.DeleteWithImage(ctx, spec.ArtDelete{
		ID:      id,
		UserID:  userID,
		AsAdmin: asAdmin,
		Audit: audit.NewEntry{
			ActorID:    userID,
			Action:     action,
			TargetType: audit.TargetArt,
			TargetID:   id.String(),
			SubjectID:  authorID,
		},
	})
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(paths...)

	s.clearPageCache(ctx, og.KindArt, id.String())

	return nil
}

func (s *service) ListArt(ctx context.Context, viewerID uuid.UUID, corner string, artType string, search string, tag string, sort string, page bounds.Page) (*dto.ArtListResponse, error) {
	if corner == "" {
		corner = "general"
	}

	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)
	rows, total, err := s.artRepo.ListAll(ctx, spec.ArtFilter{
		ViewerID:       viewerID,
		Corner:         corner,
		ArtType:        artType,
		Search:         search,
		Tag:            tag,
		Sort:           sort,
		Limit:          page.Limit(),
		Offset:         page.Offset(),
		ExcludeUserIDs: blockedIDs,
	})
	if err != nil {
		return nil, err
	}

	return s.buildArtList(ctx, rows, total, page.Limit(), page.Offset()), nil
}

func (s *service) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, page bounds.Page) (*dto.ArtListResponse, error) {
	rows, total, err := s.artRepo.ListByUser(ctx, spec.ArtUserFilter{
		UserID:   userID,
		ViewerID: viewerID,
		Limit:    page.Limit(),
		Offset:   page.Offset(),
	})
	if err != nil {
		return nil, err
	}
	return s.buildArtList(ctx, rows, total, page.Limit(), page.Offset()), nil
}

func (s *service) buildArtList(ctx context.Context, rows []model.ArtRow, total, limit, offset int) *dto.ArtListResponse {
	artIDs := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		artIDs[i] = r.ID
	}

	tagMap, _ := s.artRepo.GetTagsBatch(ctx, artIDs)

	arts := make([]dto.ArtResponse, len(rows))
	for i, r := range rows {
		arts[i] = r.ToResponse(tagMap[r.ID])
		arts[i].ThumbnailURL = s.generateThumbnailURL(r.ImageURL)
	}

	return &dto.ArtListResponse{
		Art:    arts,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
}

func (s *service) LikeArt(ctx context.Context, userID uuid.UUID, artID uuid.UUID) error {
	authorID, err := s.artRepo.GetArtAuthorID(ctx, artID)
	if err != nil {
		return err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return block.ErrUserBlocked
	}

	if err := s.artRepo.Like(ctx, spec.Like{UserID: userID, TargetID: artID}); err != nil {
		return err
	}

	go func() {
		actor, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || actor == nil {
			return
		}
		_ = s.notifService.Notify(ctx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifArtLiked,
			ReferenceID:   artID,
			ReferenceType: "art",
			ActorID:       userID,
			EmailActor:    actor.DisplayName,
			EmailAction:   "liked your art",
			EmailLink:     fmt.Sprintf("/gallery/art/%s", artID),
		})
	}()

	return nil
}

func (s *service) UnlikeArt(ctx context.Context, userID uuid.UUID, artID uuid.UUID) error {
	return s.artRepo.Unlike(ctx, spec.Like{UserID: userID, TargetID: artID})
}

func (s *service) GetCornerCounts(ctx context.Context) (map[string]int, error) {
	return s.artRepo.GetCornerCounts(ctx)
}

func (s *service) GetPopularTags(ctx context.Context, corner string) ([]dto.TagCountResponse, error) {
	tags, err := s.artRepo.GetPopularTags(ctx, spec.PopularTagFilter{Corner: corner, Limit: 30})
	if err != nil {
		return nil, err
	}
	result := make([]dto.TagCountResponse, len(tags))
	for i, t := range tags {
		result[i] = dto.TagCountResponse{Tag: t.Tag, Count: t.Count}
	}
	return result, nil
}

func (s *service) CreateComment(ctx context.Context, artID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error) {
	if strings.TrimSpace(req.Body) == "" {
		return uuid.Nil, fmt.Errorf("comment body cannot be empty")
	}
	if err := s.contentFilter.Check(ctx, req.Body); err != nil {
		return uuid.Nil, err
	}

	authorID, err := s.artRepo.GetArtAuthorID(ctx, artID)
	if err != nil {
		return uuid.Nil, err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return uuid.Nil, block.ErrUserBlocked
	}

	body := strings.TrimSpace(req.Body)

	id, err := s.mentionSvc.CreateComment(ctx, mention.CommentSpec{
		Kind:     mention.KindArtComment,
		EntityID: artID,
		ParentID: req.ParentID,
		AuthorID: userID,
		Body:     body,
	})
	if err != nil {
		return uuid.Nil, err
	}

	go func() {
		actor, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || actor == nil {
			return
		}
		linkURL := fmt.Sprintf("/gallery/art/%s#comment-%s", artID, id)

		if req.ParentID == nil {
			_ = s.notifService.Notify(ctx, dto.NotifyParams{
				RecipientID:   authorID,
				Type:          dto.NotifArtCommented,
				ReferenceID:   artID,
				ReferenceType: fmt.Sprintf("art_comment:%s", id),
				ActorID:       userID,
				EmailActor:    actor.DisplayName,
				EmailAction:   "commented on your art",
				EmailLink:     linkURL,
			})
			return
		}

		parentAuthorID, err := s.artRepo.GetCommentAuthorID(ctx, *req.ParentID)
		if err == nil && parentAuthorID != userID {
			_ = s.notifService.Notify(ctx, dto.NotifyParams{
				RecipientID:   parentAuthorID,
				Type:          dto.NotifArtCommentReply,
				ReferenceID:   artID,
				ReferenceType: fmt.Sprintf("art_comment:%s", id),
				ActorID:       userID,
				EmailActor:    actor.DisplayName,
				EmailAction:   "replied to your comment",
				EmailLink:     linkURL,
			})
		}

		if authorID != userID && authorID != parentAuthorID {
			_ = s.notifService.Notify(ctx, dto.NotifyParams{
				RecipientID:   authorID,
				Type:          dto.NotifArtCommented,
				ReferenceID:   artID,
				ReferenceType: fmt.Sprintf("art_comment:%s", id),
				ActorID:       userID,
				EmailActor:    actor.DisplayName,
				EmailAction:   "commented on your art",
				EmailLink:     linkURL,
			})
		}
	}()

	return id, nil
}

func (s *service) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateCommentRequest) error {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return fmt.Errorf("comment body cannot be empty")
	}
	if err := s.contentFilter.Check(ctx, body); err != nil {
		return err
	}

	authorID, err := s.artRepo.GetCommentAuthorID(ctx, id)
	if err != nil {
		return err
	}

	asAdmin := s.authz.Can(ctx, userID, authz.PermEditAnyComment)

	update := spec.CommentUpdate{
		CommentID: id,
		UserID:    userID,
		Body:      body,
		AsAdmin:   asAdmin,
	}

	if err := s.artRepo.UpdateCommentWithDetails(ctx, update); err != nil {
		return err
	}

	if authorID != userID {
		s.writeAudit(ctx, audit.NewEntry{
			ActorID:    userID,
			Action:     audit.ActionArtCommentUpdateAdmin,
			TargetType: audit.TargetArtComment,
			TargetID:   id.String(),
			SubjectID:  authorID,
		})
	}

	if asAdmin {
		go s.notifyArtCommentEdited(ctx, id, userID)
	}

	return nil
}

func (s *service) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.artRepo.GetCommentAuthorID(ctx, id)
	if err != nil {
		return err
	}

	action := audit.ActionArtCommentDelete
	if authorID != userID {
		action = audit.ActionArtCommentDeleteAdmin
	}

	paths, err := s.artRepo.DeleteCommentWithAudit(ctx, spec.ArtCommentDeletion{
		CommentDeletion: spec.CommentDeletion{
			CommentID: id,
			UserID:    userID,
			AsAdmin:   s.authz.Can(ctx, userID, authz.PermDeleteAnyComment),
		},
		Audit: audit.NewEntry{
			ActorID:    userID,
			Action:     action,
			TargetType: audit.TargetArtComment,
			TargetID:   id.String(),
			SubjectID:  authorID,
		},
	})
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(paths...)

	return nil
}

func (s *service) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	commentAuthorID, err := s.artRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, commentAuthorID); blocked {
		return block.ErrUserBlocked
	}

	if err := s.artRepo.LikeComment(ctx, spec.CommentLike{UserID: userID, CommentID: commentID}); err != nil {
		return err
	}

	go func() {
		artID, err := s.artRepo.GetCommentEntityID(ctx, commentID)
		if err != nil {
			return
		}
		actor, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || actor == nil {
			return
		}
		_ = s.notifService.Notify(ctx, dto.NotifyParams{
			RecipientID:   commentAuthorID,
			Type:          dto.NotifCommentLiked,
			ReferenceID:   artID,
			ReferenceType: fmt.Sprintf("art_comment:%s", commentID),
			ActorID:       userID,
			EmailActor:    actor.DisplayName,
			EmailAction:   "liked your comment",
			EmailLink:     fmt.Sprintf("/gallery/art/%s#comment-%s", artID, commentID),
		})
	}()

	return nil
}

func (s *service) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return s.artRepo.UnlikeComment(ctx, spec.CommentLike{UserID: userID, CommentID: commentID})
}

func (s *service) UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, filename string, fileSize int64, reader io.Reader, isSpoiler bool) (*dto.PostMediaResponse, error) {
	authorID, err := s.artRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID {
		return nil, fmt.Errorf("not the comment author")
	}

	return s.uploader.SaveAndRecord(ctx, "art", contentType, filename, fileSize, reader, isSpoiler,
		func(mediaURL, mediaType, thumbURL, filename string, sortOrder int) (int64, error) {
			return s.artRepo.AddCommentMedia(ctx, spec.NewMedia{
				TargetID:     commentID,
				MediaURL:     mediaURL,
				MediaType:    mediaType,
				ThumbnailURL: thumbURL,
				Filename:     filename,
				SortOrder:    sortOrder,
				IsSpoiler:    isSpoiler,
			})
		},
		s.artRepo.UpdateCommentMediaURL,
		s.artRepo.UpdateCommentMediaThumbnail,
	)
}

func (s *service) CreateGallery(ctx context.Context, userID uuid.UUID, req dto.CreateGalleryRequest) (uuid.UUID, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return uuid.Nil, ErrEmptyTitle
	}
	if err := s.contentFilter.Check(ctx, name, req.Description); err != nil {
		return uuid.Nil, err
	}
	created, err := s.artRepo.CreateGallery(ctx, spec.NewGallery{
		UserID:      userID,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
	})
	if err != nil {
		return uuid.Nil, err
	}

	return created.ID, nil
}

func (s *service) UpdateGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateGalleryRequest) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ErrEmptyTitle
	}
	if err := s.contentFilter.Check(ctx, name, req.Description); err != nil {
		return err
	}
	return s.artRepo.UpdateGallery(ctx, spec.GalleryUpdate{
		ID:          id,
		UserID:      userID,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
	})
}

func (s *service) SetGalleryCover(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, coverArtID *uuid.UUID) error {
	return s.artRepo.SetGalleryCover(ctx, spec.GalleryCoverUpdate{
		GalleryID:  galleryID,
		UserID:     userID,
		CoverArtID: coverArtID,
	})
}

func (s *service) DeleteGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	gallery, err := s.artRepo.GetGalleryByID(ctx, id)
	if err != nil {
		return err
	}
	if gallery == nil {
		return ErrNotFound
	}

	paths, err := s.artRepo.DeleteGallery(ctx, spec.GalleryRef{GalleryID: id, UserID: userID})
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(paths...)

	s.writeAudit(ctx, audit.NewEntry{
		ActorID:    userID,
		Action:     audit.ActionGalleryDelete,
		TargetType: audit.TargetGallery,
		TargetID:   id.String(),
		Details:    fmt.Sprintf("name=%q art=%d files=%d", gallery.Name, gallery.ArtCount, len(paths)),
		SubjectID:  gallery.UserID,
	})

	s.clearPageCache(ctx, og.KindGallery, id.String())

	return nil
}

func (s *service) GetGallery(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, page bounds.Page) (*dto.GalleryResponse, []dto.ArtResponse, int, error) {
	g, err := s.artRepo.GetGalleryByID(ctx, id)
	if err != nil {
		return nil, nil, 0, err
	}
	if g == nil {
		return nil, nil, 0, ErrNotFound
	}

	rows, total, err := s.artRepo.ListArtInGallery(ctx, spec.GalleryArtFilter{
		GalleryID: id,
		ViewerID:  viewerID,
		Limit:     page.Limit(),
		Offset:    page.Offset(),
	})
	if err != nil {
		return nil, nil, 0, err
	}

	artIDs := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		artIDs[i] = r.ID
	}
	tagMap, _ := s.artRepo.GetTagsBatch(ctx, artIDs)

	arts := make([]dto.ArtResponse, len(rows))
	for i, r := range rows {
		arts[i] = r.ToResponse(tagMap[r.ID])
		arts[i].ThumbnailURL = s.generateThumbnailURL(r.ImageURL)
	}

	gallery := g.ToResponse()
	if g.CoverImageURL != "" {
		gallery.CoverThumbnailURL = s.generateThumbnailURL(g.CoverImageURL)
	}
	return &gallery, arts, total, nil
}

func (s *service) galleriesWithPreviews(ctx context.Context, rows []model.GalleryRow) []dto.GalleryResponse {
	result := make([]dto.GalleryResponse, len(rows))
	for i, g := range rows {
		result[i] = g.ToResponse()
		if g.CoverImageURL != "" {
			result[i].CoverThumbnailURL = s.generateThumbnailURL(g.CoverImageURL)
		}
		if g.CoverArtID == nil && g.ArtCount > 0 {
			imgs, _ := s.artRepo.GetGalleryPreviewImages(ctx, spec.GalleryPreviewFilter{GalleryID: g.ID, Limit: 3})
			previews := make([]dto.PreviewImageDTO, len(imgs))
			for j, img := range imgs {
				previews[j] = dto.PreviewImageDTO{
					Thumbnail: s.generateThumbnailURL(img.ImageURL),
					Full:      img.ImageURL,
				}
			}
			result[i].PreviewImages = previews
		}
	}
	return result
}

func (s *service) ListUserGalleries(ctx context.Context, userID uuid.UUID) ([]dto.GalleryResponse, error) {
	rows, err := s.artRepo.ListGalleriesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.galleriesWithPreviews(ctx, rows), nil
}

func (s *service) ListAllGalleries(ctx context.Context, corner string) ([]dto.GalleryResponse, error) {
	rows, err := s.artRepo.ListAllGalleries(ctx, corner)
	if err != nil {
		return nil, err
	}
	return s.galleriesWithPreviews(ctx, rows), nil
}

func (s *service) SetArtGallery(ctx context.Context, artID uuid.UUID, userID uuid.UUID, galleryID *uuid.UUID) error {
	return s.artRepo.SetGallery(ctx, spec.ArtGalleryAssignment{
		ArtID:     artID,
		UserID:    userID,
		GalleryID: galleryID,
	})
}

func (s *service) notifyArtEdited(ctx context.Context, artID uuid.UUID, editorID uuid.UUID) {
	authorID, err := s.artRepo.GetArtAuthorID(ctx, artID)
	if err != nil {
		return
	}
	notification.SendEditNotification(ctx, s.userRepo, s.notifService, notification.EditNotifyParams{
		AuthorID:      authorID,
		EditorID:      editorID,
		ContentType:   "art",
		ReferenceID:   artID,
		ReferenceType: "art",
		LinkPath:      fmt.Sprintf("/gallery/art/%s", artID),
	})
}

func (s *service) notifyArtCommentEdited(ctx context.Context, commentID uuid.UUID, editorID uuid.UUID) {
	authorID, err := s.artRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return
	}
	artID, err := s.artRepo.GetCommentEntityID(ctx, commentID)
	if err != nil {
		return
	}
	notification.SendEditNotification(ctx, s.userRepo, s.notifService, notification.EditNotifyParams{
		AuthorID:      authorID,
		EditorID:      editorID,
		ContentType:   "comment",
		ReferenceID:   artID,
		ReferenceType: fmt.Sprintf("art_comment:%s", commentID),
		LinkPath:      fmt.Sprintf("/gallery/art/%s#comment-%s", artID, commentID),
	})
}

func artCommentToResponse(c model.CommentRow, media []model.PostMediaRow) dto.ArtCommentResponse {
	return dto.ArtCommentResponse{
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
