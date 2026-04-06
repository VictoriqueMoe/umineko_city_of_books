package post

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"
	"umineko_city_of_books/internal/repository/model"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
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
		VotePoll(ctx context.Context, postID uuid.UUID, userID uuid.UUID, optionID int) (*dto.PollResponse, error)
	}

	service struct {
		postRepo     repository.PostRepository
		userRepo     repository.UserRepository
		roleRepo     repository.RoleRepository
		authz        authz.Service
		blockSvc     block.Service
		notifService notification.Service
		uploadSvc    upload.Service
		mediaProc    *media.Processor
		settingsSvc  settings.Service
		hub          *ws.Hub
	}

	updateURLFn func(ctx context.Context, id int64, url string) error
)

var validPollDurations = map[int]bool{
	3600: true, 14400: true, 28800: true, 43200: true,
	86400: true, 259200: true, 604800: true, 1209600: true,
}

func NewService(
	postRepo repository.PostRepository,
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	authzService authz.Service,
	blockSvc block.Service,
	notifService notification.Service,
	uploadSvc upload.Service,
	mediaProc *media.Processor,
	settingsSvc settings.Service,
	hub *ws.Hub,
) Service {
	return &service{
		postRepo:     postRepo,
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		authz:        authzService,
		blockSvc:     blockSvc,
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

	if req.Poll != nil {
		if err := validatePollInput(req.Poll); err != nil {
			return uuid.Nil, err
		}
	}

	id := uuid.New()
	body := strings.TrimSpace(req.Body)
	if err := s.postRepo.Create(ctx, id, userID, corner, body); err != nil {
		return uuid.Nil, err
	}

	if req.Poll != nil {
		labels := make([]string, len(req.Poll.Options))
		for i, o := range req.Poll.Options {
			labels[i] = strings.TrimSpace(o.Label)
		}
		expiresAt := time.Now().UTC().Add(time.Duration(req.Poll.DurationSeconds) * time.Second).Format(time.RFC3339)
		pollID := uuid.New()
		if err := s.postRepo.CreatePollWithOptions(ctx, pollID, id, req.Poll.DurationSeconds, expiresAt, labels); err != nil {
			return uuid.Nil, err
		}
	}

	go social.ProcessEmbeds(s.postRepo, id.String(), "post", body)

	if corner == "suggestions" {
		go s.notifySuggestionPosted(userID, id)
	} else {
		go social.ProcessMentions(s.userRepo, s.notifService, s.settingsSvc, userID, body, id, "post", fmt.Sprintf("/game-board/%s", id))
	}

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

	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)

	postMedia, _ := s.postRepo.GetMedia(ctx, id)
	postEmbeds, _ := s.postRepo.GetEmbeds(ctx, id.String(), "post")
	comments, _, _ := s.postRepo.GetComments(ctx, id, viewerID, 500, 0, blockedIDs)

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
		flatComments[i] = c.ToResponse(commentMediaMap[c.ID], commentEmbedMap[c.ID.String()])
	}
	dtoComments := utils.BuildTree(flatComments,
		func(c dto.PostCommentResponse) uuid.UUID { return c.ID },
		func(c dto.PostCommentResponse) *uuid.UUID { return c.ParentID },
		func(c *dto.PostCommentResponse, replies []dto.PostCommentResponse) { c.Replies = replies },
	)

	likeUsers, _ := s.postRepo.GetLikedBy(ctx, id, blockedIDs)
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

	viewerBlocked := false
	if viewerID != uuid.Nil {
		viewerBlocked, _ = s.blockSvc.IsBlockedEither(ctx, viewerID, row.UserID)
	}

	postResp := row.ToResponse(postMedia, postEmbeds)
	pollRow, pollOptions, votedOption, _ := s.postRepo.GetPollByPostID(ctx, id, viewerID)
	if pollRow != nil {
		postResp.Poll = pollRow.ToResponse(pollOptions, votedOption)
	}

	return &dto.PostDetailResponse{
		PostResponse:  postResp,
		Comments:      dtoComments,
		LikedBy:       likedBy,
		ViewerBlocked: viewerBlocked,
	}, nil
}

func (s *service) UpdatePost(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdatePostRequest) error {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return ErrEmptyBody
	}
	if s.authz.Can(ctx, userID, authz.PermEditAnyPost) {
		if err := s.postRepo.UpdatePostAsAdmin(ctx, id, body); err != nil {
			return err
		}
		go s.notifyContentEdited(ctx, id, "post", userID)
	} else if err := s.postRepo.UpdatePost(ctx, id, userID, body); err != nil {
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

	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)

	var rows []model.PostRow
	var total int
	var err error

	if tab == "following" && viewerID != uuid.Nil {
		rows, total, err = s.postRepo.ListByFollowing(ctx, viewerID, corner, sort, seed, limit, offset, blockedIDs)
	} else {
		rows, total, err = s.postRepo.ListAll(ctx, viewerID, corner, search, sort, seed, limit, offset, blockedIDs)
	}
	if err != nil {
		return nil, err
	}

	return s.buildPostList(ctx, rows, total, limit, offset, viewerID), nil
}

func (s *service) ListUserPosts(ctx context.Context, targetUserID uuid.UUID, viewerID uuid.UUID, limit, offset int) (*dto.PostListResponse, error) {
	rows, total, err := s.postRepo.ListByUser(ctx, targetUserID, viewerID, limit, offset)
	if err != nil {
		return nil, err
	}
	return s.buildPostList(ctx, rows, total, limit, offset, viewerID), nil
}

func (s *service) buildPostList(ctx context.Context, rows []model.PostRow, total, limit, offset int, viewerID uuid.UUID) *dto.PostListResponse {
	postIDs := make([]uuid.UUID, len(rows))
	postIDStrs := make([]string, len(rows))
	for i, r := range rows {
		postIDs[i] = r.ID
		postIDStrs[i] = r.ID.String()
	}

	mediaMap, _ := s.postRepo.GetMediaBatch(ctx, postIDs)
	embedMap, _ := s.postRepo.GetEmbedsBatch(ctx, postIDStrs, "post")

	pollMap, pollOptionMap, pollVoteMap, _ := s.postRepo.GetPollsByPostIDs(ctx, postIDs, viewerID)

	posts := make([]dto.PostResponse, len(rows))
	for i, r := range rows {
		posts[i] = r.ToResponse(mediaMap[r.ID], embedMap[r.ID.String()])
		if p, ok := pollMap[r.ID]; ok {
			posts[i].Poll = p.ToResponse(pollOptionMap[r.ID], pollVoteMap[r.ID])
		}
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
	authorID, err := s.postRepo.GetPostAuthorID(ctx, postID)
	if err != nil {
		return err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return block.ErrUserBlocked
	}

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

	authorID, err := s.postRepo.GetPostAuthorID(ctx, postID)
	if err != nil {
		return uuid.Nil, err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return uuid.Nil, block.ErrUserBlocked
	}

	id := uuid.New()
	body := strings.TrimSpace(req.Body)
	if err := s.postRepo.CreateComment(ctx, id, postID, req.ParentID, userID, body); err != nil {
		return uuid.Nil, err
	}

	go social.ProcessEmbeds(s.postRepo, id.String(), "comment", body)
	go social.ProcessMentions(s.userRepo, s.notifService, s.settingsSvc, userID, body, postID, fmt.Sprintf("post_comment:%s", id), fmt.Sprintf("/game-board/%s#comment-%s", postID, id))

	go func() {
		postAuthorID, err := s.postRepo.GetPostAuthorID(ctx, postID)
		if err != nil {
			return
		}
		actor, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || actor == nil {
			return
		}
		baseURL := s.settingsSvc.Get(ctx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/game-board/%s#comment-%s", baseURL, postID, id)

		if req.ParentID == nil {
			subject, body := notification.NotifEmail(actor.DisplayName, "commented on your post", "", linkURL)
			_ = s.notifService.Notify(ctx, dto.NotifyParams{
				RecipientID:   postAuthorID,
				Type:          dto.NotifPostCommented,
				ReferenceID:   postID,
				ReferenceType: fmt.Sprintf("post_comment:%s", id),
				ActorID:       userID,
				EmailSubject:  subject,
				EmailBody:     body,
			})
			return
		}

		parentAuthorID, err := s.postRepo.GetCommentAuthorID(ctx, *req.ParentID)
		if err == nil && parentAuthorID != userID {
			replySubject, replyBody := notification.NotifEmail(actor.DisplayName, "replied to your comment", "", linkURL)
			_ = s.notifService.Notify(ctx, dto.NotifyParams{
				RecipientID:   parentAuthorID,
				Type:          dto.NotifPostCommentReply,
				ReferenceID:   postID,
				ReferenceType: fmt.Sprintf("post_comment:%s", id),
				ActorID:       userID,
				EmailSubject:  replySubject,
				EmailBody:     replyBody,
			})
		}

		if postAuthorID != userID && postAuthorID != parentAuthorID {
			subject, body := notification.NotifEmail(actor.DisplayName, "commented on your post", "", linkURL)
			_ = s.notifService.Notify(ctx, dto.NotifyParams{
				RecipientID:   postAuthorID,
				Type:          dto.NotifPostCommented,
				ReferenceID:   postID,
				ReferenceType: fmt.Sprintf("post_comment:%s", id),
				ActorID:       userID,
				EmailSubject:  subject,
				EmailBody:     body,
			})
		}
	}()

	return id, nil
}

func (s *service) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateCommentRequest) error {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return ErrEmptyBody
	}
	if s.authz.Can(ctx, userID, authz.PermEditAnyComment) {
		if err := s.postRepo.UpdateCommentAsAdmin(ctx, id, body); err != nil {
			return err
		}
		go s.notifyCommentEdited(ctx, id, "post", userID)
	} else if err := s.postRepo.UpdateComment(ctx, id, userID, body); err != nil {
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
	commentAuthorID, err := s.postRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, commentAuthorID); blocked {
		return block.ErrUserBlocked
	}

	if err := s.postRepo.LikeComment(ctx, userID, commentID); err != nil {
		return err
	}

	go func() {
		authorID, err := s.postRepo.GetCommentAuthorID(ctx, commentID)
		if err != nil {
			return
		}
		postID, err := s.postRepo.GetCommentPostID(ctx, commentID)
		if err != nil {
			return
		}
		actor, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || actor == nil {
			return
		}
		baseURL := s.settingsSvc.Get(ctx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/game-board/%s#comment-%s", baseURL, postID, commentID)
		subject, body := notification.NotifEmail(actor.DisplayName, "liked your comment", "", linkURL)
		_ = s.notifService.Notify(ctx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifCommentLiked,
			ReferenceID:   postID,
			ReferenceType: fmt.Sprintf("post_comment:%s", commentID),
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     body,
		})
	}()

	return nil
}

func (s *service) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return s.postRepo.UnlikeComment(ctx, userID, commentID)
}

func (s *service) GetCornerCounts(ctx context.Context) (map[string]int, error) {
	return s.postRepo.GetCornerCounts(ctx)
}

func (s *service) notifySuggestionPosted(actorID uuid.UUID, postID uuid.UUID) {
	ctx := context.Background()
	adminIDs, err := s.roleRepo.GetUsersByRoles(ctx, []role.Role{authz.RoleSuperAdmin, authz.RoleAdmin})
	if err != nil {
		return
	}
	baseURL := s.settingsSvc.Get(ctx, config.SettingBaseURL)
	linkURL := fmt.Sprintf("%s/suggestions/%s", baseURL, postID)
	subject, body := notification.NotifEmail("Someone", "posted a site suggestion", "", linkURL)
	for _, adminID := range adminIDs {
		if adminID == actorID {
			continue
		}
		_ = s.notifService.Notify(ctx, dto.NotifyParams{
			RecipientID:   adminID,
			Type:          dto.NotifSuggestionPosted,
			ReferenceID:   postID,
			ReferenceType: "post",
			ActorID:       actorID,
			EmailSubject:  subject,
			EmailBody:     body,
		})
	}
}

func (s *service) notifyContentEdited(ctx context.Context, postID uuid.UUID, contentType string, editorID uuid.UUID) {
	authorID, err := s.postRepo.GetPostAuthorID(ctx, postID)
	if err != nil {
		return
	}
	notification.SendEditNotification(ctx, s.userRepo, s.settingsSvc, s.notifService, notification.EditNotifyParams{
		AuthorID:      authorID,
		EditorID:      editorID,
		ContentType:   contentType,
		ReferenceID:   postID,
		ReferenceType: "post",
		LinkPath:      fmt.Sprintf("/game-board/%s", postID),
	})
}

func (s *service) notifyCommentEdited(ctx context.Context, commentID uuid.UUID, commentType string, editorID uuid.UUID) {
	authorID, err := s.postRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return
	}
	postID, err := s.postRepo.GetCommentPostID(ctx, commentID)
	if err != nil {
		return
	}
	notification.SendEditNotification(ctx, s.userRepo, s.settingsSvc, s.notifService, notification.EditNotifyParams{
		AuthorID:      authorID,
		EditorID:      editorID,
		ContentType:   "comment",
		ReferenceID:   postID,
		ReferenceType: fmt.Sprintf("post_comment:%s", commentID),
		LinkPath:      fmt.Sprintf("/game-board/%s#comment-%s", postID, commentID),
	})
}

func (s *service) VotePoll(ctx context.Context, postID uuid.UUID, userID uuid.UUID, optionID int) (*dto.PollResponse, error) {
	pollRow, options, votedOption, err := s.postRepo.GetPollByPostID(ctx, postID, userID)
	if err != nil {
		return nil, err
	}
	if pollRow == nil {
		return nil, ErrNotFound
	}
	if votedOption != nil {
		return nil, ErrAlreadyVoted
	}
	if time.Now().UTC().After(model.ParseTime(pollRow.ExpiresAt)) {
		return nil, ErrPollExpired
	}
	validOption := false
	for _, o := range options {
		if o.ID == optionID {
			validOption = true
			break
		}
	}
	if !validOption {
		return nil, ErrInvalidOption
	}

	pollID, _ := uuid.Parse(pollRow.ID)
	if err := s.postRepo.VotePoll(ctx, pollID, userID, optionID); err != nil {
		if strings.Contains(err.Error(), "already voted") {
			return nil, ErrAlreadyVoted
		}
		return nil, err
	}

	pollRow, options, votedOption, err = s.postRepo.GetPollByPostID(ctx, postID, userID)
	if err != nil {
		return nil, err
	}
	return pollRow.ToResponse(options, votedOption), nil
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

func validatePollInput(poll *dto.CreatePollInput) error {
	if len(poll.Options) < 2 || len(poll.Options) > 10 {
		return ErrInvalidPoll
	}
	for _, o := range poll.Options {
		if strings.TrimSpace(o.Label) == "" {
			return ErrInvalidPoll
		}
	}
	if !validPollDurations[poll.DurationSeconds] {
		return ErrInvalidDuration
	}
	return nil
}
