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
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

type (
	Service interface {
		CreatePost(ctx context.Context, userID uuid.UUID, req dto.CreatePostRequest) (uuid.UUID, error)
		GetPost(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.PostDetailResponse, error)
		DeletePost(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		ListFeed(ctx context.Context, tab string, viewerID uuid.UUID, search string, sort string, limit, offset int) (*dto.PostListResponse, error)
		ListUserPosts(ctx context.Context, targetUserID uuid.UUID, viewerID uuid.UUID, limit, offset int) (*dto.PostListResponse, error)
		UploadPostMedia(ctx context.Context, postID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error)
		LikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
		UnlikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
		CreateComment(ctx context.Context, postID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error)
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error)
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

	id := uuid.New()
	if err := s.postRepo.Create(ctx, id, userID, strings.TrimSpace(req.Body)); err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (s *service) GetPost(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.PostDetailResponse, error) {
	row, err := s.postRepo.GetByID(ctx, id, viewerID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}

	_ = s.postRepo.IncrementViewCount(ctx, id)
	row.ViewCount++

	postMedia, _ := s.postRepo.GetMedia(ctx, id)
	comments, _, _ := s.postRepo.GetComments(ctx, id, 100, 0)

	var commentIDs []uuid.UUID
	for _, c := range comments {
		commentIDs = append(commentIDs, c.ID)
	}
	commentMediaMap, _ := s.postRepo.GetCommentMediaBatch(ctx, commentIDs)

	dtoComments := make([]dto.PostCommentResponse, len(comments))
	for i, c := range comments {
		dtoComments[i] = commentToDTO(c, commentMediaMap[c.ID])
	}

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
		PostResponse: postRowToDTO(*row, postMedia),
		Comments:     dtoComments,
		LikedBy:      likedBy,
	}, nil
}

func (s *service) DeletePost(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyPost) {
		return s.postRepo.DeleteAsAdmin(ctx, id)
	}
	return s.postRepo.Delete(ctx, id, userID)
}

func (s *service) ListFeed(ctx context.Context, tab string, viewerID uuid.UUID, search string, sort string, limit, offset int) (*dto.PostListResponse, error) {
	var rows []repository.PostRow
	var total int
	var err error

	if tab == "following" && viewerID != uuid.Nil {
		rows, total, err = s.postRepo.ListByFollowing(ctx, viewerID, sort, limit, offset)
	} else {
		rows, total, err = s.postRepo.ListAll(ctx, viewerID, search, sort, limit, offset)
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
	for i, r := range rows {
		postIDs[i] = r.ID
	}

	mediaMap, _ := s.postRepo.GetMediaBatch(ctx, postIDs)

	posts := make([]dto.PostResponse, len(rows))
	for i, r := range rows {
		posts[i] = postRowToDTO(r, mediaMap[r.ID])
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

	return s.saveMedia(ctx, contentType, fileSize, reader, func(mediaURL, mediaType, thumbURL string, sortOrder int) (int64, error) {
		return s.postRepo.AddMedia(ctx, postID, mediaURL, mediaType, thumbURL, sortOrder)
	})
}

func (s *service) UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error) {
	return s.saveMedia(ctx, contentType, fileSize, reader, func(mediaURL, mediaType, thumbURL string, sortOrder int) (int64, error) {
		return s.postRepo.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbURL, sortOrder)
	})
}

func (s *service) saveMedia(ctx context.Context, contentType string, fileSize int64, reader io.Reader, addFn func(string, string, string, int) (int64, error)) (*dto.PostMediaResponse, error) {
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
				if err := s.postRepo.UpdateMediaURL(context.Background(), rowID, newURL); err != nil {
					logger.Log.Error().Err(err).Msg("failed to update video media url")
				}
				baseURL := s.settingsSvc.Get(context.Background(), config.SettingBaseURL)
				thumbName, err := media.GenerateThumbnail(baseURL+newURL, filepath.Dir(outputPath), filepath.Base(outputPath))
				if err != nil {
					logger.Log.Error().Err(err).Msg("failed to generate video thumbnail")
					return
				}
				thumbURL := "/uploads/posts/" + thumbName
				if err := s.postRepo.UpdateMediaThumbnail(context.Background(), rowID, thumbURL); err != nil {
					logger.Log.Error().Err(err).Msg("failed to update video thumbnail url")
				}
			},
		})
	} else {
		s.mediaProc.Enqueue(media.Job{
			Type:      media.JobImage,
			InputPath: diskPath,
			Callback: func(outputPath string) {
				newURL := "/uploads/posts/" + filepath.Base(outputPath)
				if err := s.postRepo.UpdateMediaURL(context.Background(), rowID, newURL); err != nil {
					logger.Log.Error().Err(err).Msg("failed to update image media url")
				}
			},
		})
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
	if err := s.postRepo.CreateComment(ctx, id, postID, userID, strings.TrimSpace(req.Body)); err != nil {
		return uuid.Nil, err
	}

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

func (s *service) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyComment) {
		return s.postRepo.DeleteCommentAsAdmin(ctx, id)
	}
	return s.postRepo.DeleteComment(ctx, id, userID)
}

func postRowToDTO(r repository.PostRow, mediaRows []repository.PostMediaRow) dto.PostResponse {
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
		LikeCount:    r.LikeCount,
		CommentCount: r.CommentCount,
		ViewCount:    r.ViewCount,
		UserLiked:    r.UserLiked,
		CreatedAt:    r.CreatedAt,
	}
}

func commentToDTO(c repository.PostCommentRow, mediaRows []repository.PostMediaRow) dto.PostCommentResponse {
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
		ID: c.ID,
		Author: dto.UserResponse{
			ID:          c.UserID,
			Username:    c.AuthorUsername,
			DisplayName: c.AuthorDisplayName,
			AvatarURL:   c.AuthorAvatarURL,
			Role:        role.Role(c.AuthorRole),
		},
		Body:      c.Body,
		Media:     mediaList,
		CreatedAt: c.CreatedAt,
	}
}
