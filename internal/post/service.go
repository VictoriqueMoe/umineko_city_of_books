package post

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
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

type (
	Service interface {
		CreatePost(ctx context.Context, userID uuid.UUID, req dto.CreatePostRequest) (uuid.UUID, error)
		GetPost(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, viewerHash string) (*dto.PostDetailResponse, error)
		UpdatePost(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdatePostRequest) error
		DeletePost(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		ListFeed(ctx context.Context, tab string, viewerID uuid.UUID, corner string, search string, sort string, seed int, limit, offset int) (*dto.PostListResponse, error)
		ListUserPosts(ctx context.Context, targetUserID uuid.UUID, viewerID uuid.UUID, limit, offset int) (*dto.PostListResponse, error)
		UploadPostMedia(ctx context.Context, postID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error)
		DeletePostMedia(ctx context.Context, postID uuid.UUID, mediaID int64, userID uuid.UUID) error
		LikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
		UnlikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
		CreateComment(ctx context.Context, postID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateCommentRequest) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error)
		GetCornerCounts(ctx context.Context) (map[string]int, error)
		RefreshStaleEmbeds(ctx context.Context) int
	}

	service struct {
		postRepo     repository.PostRepository
		userRepo     repository.UserRepository
		authz        authz.Service
		notifService notification.Service
		uploadSvc    upload.Service
		mediaProc    *media.Processor
		settingsSvc  settings.Service
		hub          *ws.Hub
	}
)

func NewService(
	postRepo repository.PostRepository,
	userRepo repository.UserRepository,
	authzService authz.Service,
	notifService notification.Service,
	uploadSvc upload.Service,
	mediaProc *media.Processor,
	settingsSvc settings.Service,
	hub *ws.Hub,
) Service {
	return &service{
		postRepo:     postRepo,
		userRepo:     userRepo,
		authz:        authzService,
		notifService: notifService,
		uploadSvc:    uploadSvc,
		mediaProc:    mediaProc,
		settingsSvc:  settingsSvc,
		hub:          hub,
	}
}

func (s *service) CreatePost(ctx context.Context, userID uuid.UUID, req dto.CreatePostRequest) (uuid.UUID, error) {
	if strings.TrimSpace(req.Body) == "" {
		return uuid.Nil, ErrEmptyBody
	}

	limit := s.settingsSvc.GetInt(ctx, config.SettingMaxPostsPerDay)
	if limit > 0 {
		count, err := s.postRepo.CountUserPostsToday(ctx, userID)
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

	id := uuid.New()
	body := strings.TrimSpace(req.Body)
	if err := s.postRepo.Create(ctx, id, userID, corner, body); err != nil {
		return uuid.Nil, err
	}

	go social.ProcessEmbeds(s.postRepo, id.String(), "post", body)
	go social.ProcessMentions(s.userRepo, s.notifService, s.settingsSvc, userID, body, id, "post", fmt.Sprintf("/game-board/%s", id))

	return id, nil
}

func (s *service) GetPost(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, viewerHash string) (*dto.PostDetailResponse, error) {
	row, err := s.postRepo.GetByID(ctx, id, viewerID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}

	if viewerHash != "" {
		isNew, _ := s.postRepo.RecordView(ctx, id, viewerHash)
		if isNew {
			row.ViewCount++
		}
	}

	postMedia, _ := s.postRepo.GetMedia(ctx, id)
	postEmbeds, _ := s.postRepo.GetEmbeds(ctx, id.String(), "post")
	comments, _, _ := s.postRepo.GetComments(ctx, id, viewerID, 500, 0)

	var commentIDs []uuid.UUID
	var commentIDStrs []string
	for _, c := range comments {
		commentIDs = append(commentIDs, c.ID)
		commentIDStrs = append(commentIDStrs, c.ID.String())
	}
	commentMediaMap, _ := s.postRepo.GetCommentMediaBatch(ctx, commentIDs)
	commentEmbedMap, _ := s.postRepo.GetEmbedsBatch(ctx, commentIDStrs, "comment")

	flatComments := make([]dto.PostCommentResponse, len(comments))
	for i, c := range comments {
		flatComments[i] = commentToDTO(c, commentMediaMap[c.ID], commentEmbedMap[c.ID.String()])
	}
	dtoComments := utils.BuildTree(flatComments,
		func(c dto.PostCommentResponse) uuid.UUID { return c.ID },
		func(c dto.PostCommentResponse) *uuid.UUID { return c.ParentID },
		func(c *dto.PostCommentResponse, replies []dto.PostCommentResponse) { c.Replies = replies },
	)

	likeUsers, _ := s.postRepo.GetLikedBy(ctx, id)
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

	return &dto.PostDetailResponse{
		PostResponse: postRowToDTO(*row, postMedia, postEmbeds),
		Comments:     dtoComments,
		LikedBy:      likedBy,
	}, nil
}

func (s *service) UpdatePost(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdatePostRequest) error {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return ErrEmptyBody
	}
	if err := s.postRepo.UpdatePost(ctx, id, userID, body); err != nil {
		return err
	}
	go func() {
		_ = s.postRepo.DeleteEmbeds(context.Background(), id.String(), "post")
		social.ProcessEmbeds(s.postRepo, id.String(), "post", body)
	}()
	return nil
}

func (s *service) DeletePost(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyPost) {
		return s.postRepo.DeleteAsAdmin(ctx, id)
	}
	return s.postRepo.Delete(ctx, id, userID)
}

func (s *service) ListFeed(ctx context.Context, tab string, viewerID uuid.UUID, corner string, search string, sort string, seed int, limit, offset int) (*dto.PostListResponse, error) {
	if corner == "" {
		corner = "general"
	}

	var rows []repository.PostRow
	var total int
	var err error

	if tab == "following" && viewerID != uuid.Nil {
		rows, total, err = s.postRepo.ListByFollowing(ctx, viewerID, corner, sort, seed, limit, offset)
	} else {
		rows, total, err = s.postRepo.ListAll(ctx, viewerID, corner, search, sort, seed, limit, offset)
	}
	if err != nil {
		return nil, err
	}

	return s.buildPostList(ctx, rows, total, limit, offset), nil
}

func (s *service) ListUserPosts(ctx context.Context, targetUserID uuid.UUID, viewerID uuid.UUID, limit, offset int) (*dto.PostListResponse, error) {
	rows, total, err := s.postRepo.ListByUser(ctx, targetUserID, viewerID, limit, offset)
	if err != nil {
		return nil, err
	}
	return s.buildPostList(ctx, rows, total, limit, offset), nil
}

func (s *service) buildPostList(ctx context.Context, rows []repository.PostRow, total, limit, offset int) *dto.PostListResponse {
	postIDs := make([]uuid.UUID, len(rows))
	postIDStrs := make([]string, len(rows))
	for i, r := range rows {
		postIDs[i] = r.ID
		postIDStrs[i] = r.ID.String()
	}

	mediaMap, _ := s.postRepo.GetMediaBatch(ctx, postIDs)
	embedMap, _ := s.postRepo.GetEmbedsBatch(ctx, postIDStrs, "post")

	posts := make([]dto.PostResponse, len(rows))
	for i, r := range rows {
		posts[i] = postRowToDTO(r, mediaMap[r.ID], embedMap[r.ID.String()])
	}

	return &dto.PostListResponse{
		Posts:  posts,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
}

func (s *service) UploadPostMedia(ctx context.Context, postID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error) {
	authorID, err := s.postRepo.GetPostAuthorID(ctx, postID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID {
		return nil, fmt.Errorf("not the post author")
	}

	return s.saveMedia(ctx, contentType, fileSize, reader,
		func(mediaURL, mediaType, thumbURL string, sortOrder int) (int64, error) {
			return s.postRepo.AddMedia(ctx, postID, mediaURL, mediaType, thumbURL, sortOrder)
		},
		s.postRepo.UpdateMediaURL,
		s.postRepo.UpdateMediaThumbnail,
	)
}

func (s *service) DeletePostMedia(ctx context.Context, postID uuid.UUID, mediaID int64, userID uuid.UUID) error {
	authorID, err := s.postRepo.GetPostAuthorID(ctx, postID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID {
		return fmt.Errorf("not the post author")
	}

	mediaURL, err := s.postRepo.DeleteMedia(ctx, mediaID, postID)
	if err != nil {
		return err
	}

	_ = s.uploadSvc.Delete(mediaURL)
	return nil
}

func (s *service) UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error) {
	authorID, err := s.postRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID {
		return nil, fmt.Errorf("not the comment author")
	}

	return s.saveMedia(ctx, contentType, fileSize, reader,
		func(mediaURL, mediaType, thumbURL string, sortOrder int) (int64, error) {
			return s.postRepo.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbURL, sortOrder)
		},
		s.postRepo.UpdateCommentMediaURL,
		s.postRepo.UpdateCommentMediaThumbnail,
	)
}

type updateURLFn func(ctx context.Context, id int64, url string) error

func (s *service) saveMedia(ctx context.Context, contentType string, fileSize int64, reader io.Reader, addFn func(string, string, string, int) (int64, error), updateURL updateURLFn, updateThumb updateURLFn) (*dto.PostMediaResponse, error) {
	isVideo := strings.HasPrefix(contentType, "video/")
	mediaID := uuid.New()

	var urlPath string
	var err error
	if isVideo {
		maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxVideoSize))
		logger.Log.Debug().Str("content_type", contentType).Int64("file_size", fileSize).Int64("max_size", maxSize).Msg("uploading video")
		urlPath, err = s.uploadSvc.SaveVideo(ctx, "posts", mediaID, contentType, fileSize, maxSize, reader)
	} else {
		maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
		logger.Log.Debug().Str("content_type", contentType).Int64("file_size", fileSize).Int64("max_size", maxSize).Msg("uploading image")
		urlPath, err = s.uploadSvc.SaveImage(ctx, "posts", mediaID, contentType, fileSize, maxSize, reader)
	}
	if err != nil {
		return nil, err
	}

	mediaType := "image"
	if isVideo {
		mediaType = "video"
	}

	rowID, err := addFn(urlPath, mediaType, "", 0)
	if err != nil {
		return nil, err
	}

	diskPath := s.uploadSvc.FullDiskPath(urlPath)
	if isVideo {
		s.mediaProc.Enqueue(media.Job{
			Type:      media.JobVideo,
			InputPath: diskPath,
			Callback: func(outputPath string) {
				newURL := "/uploads/posts/" + filepath.Base(outputPath)
				if err := updateURL(context.Background(), rowID, newURL); err != nil {
					logger.Log.Error().Err(err).Msg("failed to update video media url")
				}
				thumbName, err := media.GenerateThumbnail(outputPath, filepath.Dir(outputPath), filepath.Base(outputPath))
				if err != nil {
					logger.Log.Error().Err(err).Msg("failed to generate video thumbnail")
					return
				}
				thumbURL := "/uploads/posts/" + thumbName
				if err := updateThumb(context.Background(), rowID, thumbURL); err != nil {
					logger.Log.Error().Err(err).Msg("failed to update video thumbnail url")
				}
			},
		})
	} else {
		done := make(chan string, 1)
		s.mediaProc.Enqueue(media.Job{
			Type:      media.JobImage,
			InputPath: diskPath,
			Callback: func(outputPath string) {
				newURL := "/uploads/posts/" + filepath.Base(outputPath)
				if err := updateURL(context.Background(), rowID, newURL); err != nil {
					logger.Log.Error().Err(err).Msg("failed to update image media url")
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

func (s *service) broadcastLikeUpdate(postID uuid.UUID, delta int) {
	s.hub.Broadcast(ws.Message{
		Type: "post_like",
		Data: map[string]interface{}{
			"post_id": postID,
			"delta":   delta,
		},
	})
}

func (s *service) LikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error {
	if err := s.postRepo.Like(ctx, userID, postID); err != nil {
		return err
	}

	s.broadcastLikeUpdate(postID, 1)

	go func() {
		authorID, err := s.postRepo.GetPostAuthorID(ctx, postID)
		if err != nil {
			return
		}
		actor, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || actor == nil {
			return
		}
		baseURL := s.settingsSvc.Get(ctx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/game-board/%s", baseURL, postID)
		subject, body := notification.NotifEmail(actor.DisplayName, "liked your post", "", linkURL)
		_ = s.notifService.Notify(ctx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifPostLiked,
			ReferenceID:   postID,
			ReferenceType: "post",
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     body,
		})
	}()

	return nil
}

func (s *service) UnlikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error {
	if err := s.postRepo.Unlike(ctx, userID, postID); err != nil {
		return err
	}
	s.broadcastLikeUpdate(postID, -1)
	return nil
}

func (s *service) CreateComment(ctx context.Context, postID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error) {
	if strings.TrimSpace(req.Body) == "" {
		return uuid.Nil, ErrEmptyBody
	}

	id := uuid.New()
	body := strings.TrimSpace(req.Body)
	if err := s.postRepo.CreateComment(ctx, id, postID, req.ParentID, userID, body); err != nil {
		return uuid.Nil, err
	}

	go social.ProcessEmbeds(s.postRepo, id.String(), "comment", body)
	go social.ProcessMentions(s.userRepo, s.notifService, s.settingsSvc, userID, body, postID, "post", fmt.Sprintf("/game-board/%s#comment-%s", postID, id))

	go func() {
		authorID, err := s.postRepo.GetPostAuthorID(ctx, postID)
		if err != nil {
			return
		}
		actor, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || actor == nil {
			return
		}
		baseURL := s.settingsSvc.Get(ctx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/game-board/%s#comment-%s", baseURL, postID, id)
		subject, body := notification.NotifEmail(actor.DisplayName, "commented on your post", "", linkURL)
		_ = s.notifService.Notify(ctx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifPostCommented,
			ReferenceID:   postID,
			ReferenceType: "post",
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     body,
		})
	}()

	return id, nil
}

func (s *service) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateCommentRequest) error {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return ErrEmptyBody
	}
	if err := s.postRepo.UpdateComment(ctx, id, userID, body); err != nil {
		return err
	}
	go func() {
		_ = s.postRepo.DeleteEmbeds(context.Background(), id.String(), "comment")
		social.ProcessEmbeds(s.postRepo, id.String(), "comment", body)
	}()
	return nil
}

func (s *service) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyComment) {
		return s.postRepo.DeleteCommentAsAdmin(ctx, id)
	}
	return s.postRepo.DeleteComment(ctx, id, userID)
}

func (s *service) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return s.postRepo.LikeComment(ctx, userID, commentID)
}

func (s *service) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return s.postRepo.UnlikeComment(ctx, userID, commentID)
}

func postRowToDTO(r repository.PostRow, mediaRows []repository.PostMediaRow, embedRows []repository.EmbedRow) dto.PostResponse {
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

	return dto.PostResponse{
		ID: r.ID,
		Author: dto.UserResponse{
			ID:          r.UserID,
			Username:    r.AuthorUsername,
			DisplayName: r.AuthorDisplayName,
			AvatarURL:   r.AuthorAvatarURL,
			Role:        role.Role(r.AuthorRole),
		},
		Body:         r.Body,
		Media:        mediaList,
		Embeds:       embedRowsToDTO(embedRows),
		LikeCount:    r.LikeCount,
		CommentCount: r.CommentCount,
		ViewCount:    r.ViewCount,
		UserLiked:    r.UserLiked,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

func commentToDTO(c repository.PostCommentRow, mediaRows []repository.PostMediaRow, embedRows []repository.EmbedRow) dto.PostCommentResponse {
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

	return dto.PostCommentResponse{
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
		Embeds:    embedRowsToDTO(embedRows),
		LikeCount: c.LikeCount,
		UserLiked: c.UserLiked,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func embedRowsToDTO(rows []repository.EmbedRow) []dto.EmbedResponse {
	if len(rows) == 0 {
		return nil
	}
	result := make([]dto.EmbedResponse, len(rows))
	for i, e := range rows {
		result[i] = dto.EmbedResponse{
			URL:      e.URL,
			Type:     e.EmbedType,
			Title:    e.Title,
			Desc:     e.Desc,
			Image:    e.Image,
			SiteName: e.SiteName,
			VideoID:  e.VideoID,
		}
	}
	return result
}

func (s *service) GetCornerCounts(ctx context.Context) (map[string]int, error) {
	return s.postRepo.GetCornerCounts(ctx)
}

func (s *service) RefreshStaleEmbeds(ctx context.Context) int {
	stale, err := s.postRepo.GetStaleEmbeds(ctx, "-1 day", 50)
	if err != nil {
		return 0
	}
	refreshed := 0
	for _, e := range stale {
		embed := media.ParseEmbed(e.URL)
		if embed == nil {
			continue
		}
		if embed.Type == "link" {
			_ = s.postRepo.UpdateEmbed(ctx, e.ID, embed.Title, embed.Desc, embed.Image, embed.SiteName)
			refreshed++
		}
	}
	return refreshed
}
