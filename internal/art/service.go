package art

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/social"
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
		ListArt(ctx context.Context, viewerID uuid.UUID, corner string, artType string, search string, tag string, sort string, limit, offset int) (*dto.ArtListResponse, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) (*dto.ArtListResponse, error)
		LikeArt(ctx context.Context, userID uuid.UUID, artID uuid.UUID) error
		UnlikeArt(ctx context.Context, userID uuid.UUID, artID uuid.UUID) error
		GetCornerCounts(ctx context.Context) (map[string]int, error)
		GetPopularTags(ctx context.Context, corner string) ([]dto.TagCountResponse, error)

		CreateComment(ctx context.Context, artID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateCommentRequest) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error)

		CreateGallery(ctx context.Context, userID uuid.UUID, req dto.CreateGalleryRequest) (uuid.UUID, error)
		UpdateGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateGalleryRequest) error
		SetGalleryCover(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, coverArtID *uuid.UUID) error
		DeleteGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		GetGallery(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, limit, offset int) (*dto.GalleryResponse, []dto.ArtResponse, int, error)
		ListUserGalleries(ctx context.Context, userID uuid.UUID) ([]dto.GalleryResponse, error)
		ListAllGalleries(ctx context.Context, corner string) ([]dto.GalleryResponse, error)
		SetArtGallery(ctx context.Context, artID uuid.UUID, userID uuid.UUID, galleryID *uuid.UUID) error
	}

	service struct {
		artRepo      repository.ArtRepository
		postRepo     repository.PostRepository
		userRepo     repository.UserRepository
		authz        authz.Service
		notifService notification.Service
		uploadSvc    upload.Service
		mediaProc    *media.Processor
		settingsSvc  settings.Service
	}
)

func NewService(
	artRepo repository.ArtRepository,
	postRepo repository.PostRepository,
	userRepo repository.UserRepository,
	authzService authz.Service,
	notifService notification.Service,
	uploadSvc upload.Service,
	mediaProc *media.Processor,
	settingsSvc settings.Service,
) Service {
	return &service{
		artRepo:      artRepo,
		postRepo:     postRepo,
		userRepo:     userRepo,
		authz:        authzService,
		notifService: notifService,
		uploadSvc:    uploadSvc,
		mediaProc:    mediaProc,
		settingsSvc:  settingsSvc,
	}
}

func (s *service) CreateArt(ctx context.Context, userID uuid.UUID, req dto.CreateArtRequest, contentType string, fileSize int64, reader io.Reader) (uuid.UUID, error) {
	if strings.TrimSpace(req.Title) == "" {
		return uuid.Nil, ErrEmptyTitle
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
	urlPath, err := s.uploadSvc.SaveImage(ctx, "art", mediaID, contentType, fileSize, maxSize, reader)
	if err != nil {
		return uuid.Nil, err
	}

	diskPath := s.uploadSvc.FullDiskPath(urlPath)
	done := make(chan string, 1)
	s.mediaProc.Enqueue(media.Job{
		Type:      media.JobImage,
		InputPath: diskPath,
		Callback: func(outputPath string) {
			done <- "/uploads/art/" + filepath.Base(outputPath)
		},
	})
	select {
	case newURL := <-done:
		urlPath = newURL
	case <-ctx.Done():
	}

	thumbnailURL := s.generateThumbnailURL(urlPath)

	id := uuid.New()
	title := strings.TrimSpace(req.Title)
	description := strings.TrimSpace(req.Description)
	if err := s.artRepo.Create(ctx, id, userID, corner, artType, title, description, urlPath, thumbnailURL); err != nil {
		return uuid.Nil, err
	}

	if len(req.Tags) > 0 {
		tags := req.Tags
		if len(tags) > 10 {
			tags = tags[:10]
		}
		_ = s.artRepo.AddTags(ctx, id, tags)
	}

	go social.ProcessMentions(s.userRepo, s.notifService, s.settingsSvc, userID, description, id, "art", fmt.Sprintf("/gallery/art/%s", id))

	return id, nil
}

func (s *service) generateThumbnailURL(imageURL string) string {
	baseURL := s.settingsSvc.Get(context.Background(), config.SettingBaseURL)
	fullURL := baseURL + imageURL
	return "https://thumbnails.waifuvault.moe/api/v1/generateThumbnail/ext/fromURL?url=" + fullURL
}

func (s *service) GetArt(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, viewerHash string) (*dto.ArtDetailResponse, error) {
	row, err := s.artRepo.GetByID(ctx, id, viewerID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}

	if viewerHash != "" {
		isNew, _ := s.artRepo.RecordView(ctx, id, viewerHash)
		if isNew {
			row.ViewCount++
		}
	}

	tags, _ := s.artRepo.GetTags(ctx, id)
	comments, _, _ := s.artRepo.GetComments(ctx, id, viewerID, 500, 0)

	var commentIDs []uuid.UUID
	var commentIDStrs []string
	for _, c := range comments {
		commentIDs = append(commentIDs, c.ID)
		commentIDStrs = append(commentIDStrs, c.ID.String())
	}
	commentMediaMap, _ := s.artRepo.GetCommentMediaBatch(ctx, commentIDs)
	commentEmbedMap, _ := s.postRepo.GetEmbedsBatch(ctx, commentIDStrs, "art_comment")

	flatComments := make([]dto.ArtCommentResponse, len(comments))
	for i, c := range comments {
		flatComments[i] = artCommentToDTO(c, commentMediaMap[c.ID], commentEmbedMap[c.ID.String()])
	}
	dtoComments := utils.BuildTree(flatComments,
		func(c dto.ArtCommentResponse) uuid.UUID { return c.ID },
		func(c dto.ArtCommentResponse) *uuid.UUID { return c.ParentID },
		func(c *dto.ArtCommentResponse, replies []dto.ArtCommentResponse) { c.Replies = replies },
	)

	likeUsers, _ := s.artRepo.GetLikedBy(ctx, id)
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

	return &dto.ArtDetailResponse{
		ArtResponse: row.ToResponse(tags),
		Comments:    dtoComments,
		LikedBy:     likedBy,
	}, nil
}

func (s *service) UpdateArt(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateArtRequest) error {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return ErrEmptyTitle
	}
	description := strings.TrimSpace(req.Description)

	if err := s.artRepo.Update(ctx, id, userID, title, description); err != nil {
		return err
	}

	_ = s.artRepo.DeleteTags(ctx, id)
	if len(req.Tags) > 0 {
		tags := req.Tags
		if len(tags) > 10 {
			tags = tags[:10]
		}
		_ = s.artRepo.AddTags(ctx, id, tags)
	}

	return nil
}

func (s *service) DeleteArt(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	imageURL, err := s.artRepo.GetImageURL(ctx, id)
	if err != nil {
		return err
	}

	if s.authz.Can(ctx, userID, authz.PermDeleteAnyPost) {
		if err := s.artRepo.DeleteAsAdmin(ctx, id); err != nil {
			return err
		}
	} else {
		if err := s.artRepo.Delete(ctx, id, userID); err != nil {
			return err
		}
	}

	_ = s.uploadSvc.Delete(imageURL)
	return nil
}

func (s *service) ListArt(ctx context.Context, viewerID uuid.UUID, corner string, artType string, search string, tag string, sort string, limit, offset int) (*dto.ArtListResponse, error) {
	if corner == "" {
		corner = "general"
	}

	rows, total, err := s.artRepo.ListAll(ctx, viewerID, corner, artType, search, tag, sort, limit, offset)
	if err != nil {
		return nil, err
	}

	return s.buildArtList(ctx, rows, total, limit, offset), nil
}

func (s *service) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) (*dto.ArtListResponse, error) {
	rows, total, err := s.artRepo.ListByUser(ctx, userID, viewerID, limit, offset)
	if err != nil {
		return nil, err
	}
	return s.buildArtList(ctx, rows, total, limit, offset), nil
}

func (s *service) buildArtList(ctx context.Context, rows []repository.ArtRow, total, limit, offset int) *dto.ArtListResponse {
	artIDs := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		artIDs[i] = r.ID
	}

	tagMap, _ := s.artRepo.GetTagsBatch(ctx, artIDs)

	arts := make([]dto.ArtResponse, len(rows))
	for i, r := range rows {
		arts[i] = r.ToResponse(tagMap[r.ID])
	}

	return &dto.ArtListResponse{
		Art:    arts,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
}

func (s *service) LikeArt(ctx context.Context, userID uuid.UUID, artID uuid.UUID) error {
	if err := s.artRepo.Like(ctx, userID, artID); err != nil {
		return err
	}

	go func() {
		authorID, err := s.artRepo.GetArtAuthorID(ctx, artID)
		if err != nil {
			return
		}
		actor, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || actor == nil {
			return
		}
		baseURL := s.settingsSvc.Get(ctx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/gallery/art/%s", baseURL, artID)
		subject, body := notification.NotifEmail(actor.DisplayName, "liked your art", "", linkURL)
		_ = s.notifService.Notify(ctx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifArtLiked,
			ReferenceID:   artID,
			ReferenceType: "art",
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     body,
		})
	}()

	return nil
}

func (s *service) UnlikeArt(ctx context.Context, userID uuid.UUID, artID uuid.UUID) error {
	return s.artRepo.Unlike(ctx, userID, artID)
}

func (s *service) GetCornerCounts(ctx context.Context) (map[string]int, error) {
	return s.artRepo.GetCornerCounts(ctx)
}

func (s *service) GetPopularTags(ctx context.Context, corner string) ([]dto.TagCountResponse, error) {
	tags, err := s.artRepo.GetPopularTags(ctx, corner, 30)
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

	id := uuid.New()
	body := strings.TrimSpace(req.Body)
	if err := s.artRepo.CreateComment(ctx, id, artID, req.ParentID, userID, body); err != nil {
		return uuid.Nil, err
	}

	go social.ProcessEmbeds(s.postRepo, id.String(), "art_comment", body)
	go social.ProcessMentions(s.userRepo, s.notifService, s.settingsSvc, userID, body, artID, "art", fmt.Sprintf("/gallery/art/%s#comment-%s", artID, id))

	go func() {
		authorID, err := s.artRepo.GetArtAuthorID(ctx, artID)
		if err != nil {
			return
		}
		actor, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || actor == nil {
			return
		}
		baseURL := s.settingsSvc.Get(ctx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/gallery/art/%s#comment-%s", baseURL, artID, id)
		subject, emailBody := notification.NotifEmail(actor.DisplayName, "commented on your art", "", linkURL)
		_ = s.notifService.Notify(ctx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifArtCommented,
			ReferenceID:   artID,
			ReferenceType: "art",
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     emailBody,
		})
	}()

	return id, nil
}

func (s *service) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateCommentRequest) error {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return fmt.Errorf("comment body cannot be empty")
	}
	if err := s.artRepo.UpdateComment(ctx, id, userID, body); err != nil {
		return err
	}
	go func() {
		_ = s.postRepo.DeleteEmbeds(context.Background(), id.String(), "art_comment")
		social.ProcessEmbeds(s.postRepo, id.String(), "art_comment", body)
	}()
	return nil
}

func (s *service) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyComment) {
		return s.artRepo.DeleteCommentAsAdmin(ctx, id)
	}
	return s.artRepo.DeleteComment(ctx, id, userID)
}

func (s *service) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return s.artRepo.LikeComment(ctx, userID, commentID)
}

func (s *service) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return s.artRepo.UnlikeComment(ctx, userID, commentID)
}

func (s *service) UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error) {
	authorID, err := s.artRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID {
		return nil, fmt.Errorf("not the comment author")
	}

	isVideo := strings.HasPrefix(contentType, "video/")
	mediaID := uuid.New()

	var urlPath string
	if isVideo {
		maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxVideoSize))
		urlPath, err = s.uploadSvc.SaveVideo(ctx, "art", mediaID, contentType, fileSize, maxSize, reader)
	} else {
		maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
		urlPath, err = s.uploadSvc.SaveImage(ctx, "art", mediaID, contentType, fileSize, maxSize, reader)
	}
	if err != nil {
		return nil, err
	}

	mediaType := "image"
	if isVideo {
		mediaType = "video"
	}

	rowID, err := s.artRepo.AddCommentMedia(ctx, commentID, urlPath, mediaType, "", 0)
	if err != nil {
		return nil, err
	}

	diskPath := s.uploadSvc.FullDiskPath(urlPath)
	if isVideo {
		s.mediaProc.Enqueue(media.Job{
			Type:      media.JobVideo,
			InputPath: diskPath,
			Callback: func(outputPath string) {
				newURL := "/uploads/art/" + filepath.Base(outputPath)
				if err := s.artRepo.UpdateCommentMediaURL(context.Background(), rowID, newURL); err != nil {
					logger.Log.Error().Err(err).Msg("failed to update art comment video url")
				}
				thumbName, err := media.GenerateThumbnail(outputPath, filepath.Dir(outputPath), filepath.Base(outputPath))
				if err != nil {
					logger.Log.Error().Err(err).Msg("failed to generate art comment video thumbnail")
					return
				}
				thumbURL := "/uploads/art/" + thumbName
				if err := s.artRepo.UpdateCommentMediaThumbnail(context.Background(), rowID, thumbURL); err != nil {
					logger.Log.Error().Err(err).Msg("failed to update art comment video thumbnail")
				}
			},
		})
	} else {
		done := make(chan string, 1)
		s.mediaProc.Enqueue(media.Job{
			Type:      media.JobImage,
			InputPath: diskPath,
			Callback: func(outputPath string) {
				newURL := "/uploads/art/" + filepath.Base(outputPath)
				if err := s.artRepo.UpdateCommentMediaURL(context.Background(), rowID, newURL); err != nil {
					logger.Log.Error().Err(err).Msg("failed to update art comment image url")
				}
				done <- newURL
			},
		})
		select {
		case newURL := <-done:
			urlPath = newURL
		case <-ctx.Done():
		}
	}

	return &dto.PostMediaResponse{
		ID:        int(rowID),
		MediaURL:  urlPath,
		MediaType: mediaType,
	}, nil
}

func artCommentToDTO(c repository.ArtCommentRow, mediaRows []repository.PostMediaRow, embedRows []repository.EmbedRow) dto.ArtCommentResponse {
	mediaList := make([]dto.PostMediaResponse, len(mediaRows))
	for i, m := range mediaRows {
		mediaList[i] = dto.PostMediaResponse{
			ID:           m.ID,
			MediaURL:     m.MediaURL,
			MediaType:    m.MediaType,
			ThumbnailURL: m.ThumbnailURL,
			SortOrder:    m.SortOrder,
		}
	}

	var embeds []dto.EmbedResponse
	if len(embedRows) > 0 {
		embeds = make([]dto.EmbedResponse, len(embedRows))
		for i, e := range embedRows {
			embeds[i] = dto.EmbedResponse{
				URL:      e.URL,
				Type:     e.EmbedType,
				Title:    e.Title,
				Desc:     e.Desc,
				Image:    e.Image,
				SiteName: e.SiteName,
				VideoID:  e.VideoID,
			}
		}
	}

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
		Media:     mediaList,
		Embeds:    embeds,
		LikeCount: c.LikeCount,
		UserLiked: c.UserLiked,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (s *service) CreateGallery(ctx context.Context, userID uuid.UUID, req dto.CreateGalleryRequest) (uuid.UUID, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return uuid.Nil, ErrEmptyTitle
	}
	id := uuid.New()
	if err := s.artRepo.CreateGallery(ctx, id, userID, name, strings.TrimSpace(req.Description)); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (s *service) UpdateGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateGalleryRequest) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ErrEmptyTitle
	}
	return s.artRepo.UpdateGallery(ctx, id, userID, name, strings.TrimSpace(req.Description))
}

func (s *service) SetGalleryCover(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, coverArtID *uuid.UUID) error {
	return s.artRepo.SetGalleryCover(ctx, galleryID, userID, coverArtID)
}

func (s *service) DeleteGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	rows, _, err := s.artRepo.ListArtInGallery(ctx, id, uuid.Nil, 10000, 0)
	if err != nil {
		return err
	}

	if err := s.artRepo.DeleteGallery(ctx, id, userID); err != nil {
		return err
	}

	for _, a := range rows {
		_ = s.uploadSvc.Delete(a.ImageURL)
	}

	return nil
}

func (s *service) GetGallery(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, limit, offset int) (*dto.GalleryResponse, []dto.ArtResponse, int, error) {
	g, err := s.artRepo.GetGalleryByID(ctx, id)
	if err != nil {
		return nil, nil, 0, err
	}
	if g == nil {
		return nil, nil, 0, ErrNotFound
	}

	rows, total, err := s.artRepo.ListArtInGallery(ctx, id, viewerID, limit, offset)
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
	}

	gallery := g.ToResponse()
	return &gallery, arts, total, nil
}

func (s *service) galleriesWithPreviews(ctx context.Context, rows []repository.GalleryRow) []dto.GalleryResponse {
	result := make([]dto.GalleryResponse, len(rows))
	for i, g := range rows {
		result[i] = g.ToResponse()
		if g.CoverArtID == nil && g.ArtCount > 0 {
			imgs, _ := s.artRepo.GetGalleryPreviewImages(ctx, g.ID, 3)
			previews := make([]dto.PreviewImageDTO, len(imgs))
			for j, img := range imgs {
				previews[j] = dto.PreviewImageDTO{
					Thumbnail: img.ThumbnailURL,
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
	return s.artRepo.SetGallery(ctx, artID, userID, galleryID)
}
