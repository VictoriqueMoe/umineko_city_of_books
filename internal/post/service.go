package post

import (
	"context"
	"fmt"
	"io"
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
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

type (
	Service interface {
		CreatePost(ctx context.Context, userID uuid.UUID, req dto.CreatePostRequest) (uuid.UUID, error)
		GetPost(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, viewerHash string) (*dto.PostDetailResponse, error)
		UpdatePost(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdatePostRequest) error
		DeletePost(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		ListFeed(ctx context.Context, tab string, viewerID uuid.UUID, corner string, search string, sort string, seed int, page bounds.Page, resolvedFilter string) (*dto.PostListResponse, error)
		ListUserPosts(ctx context.Context, targetUserID uuid.UUID, viewerID uuid.UUID, page bounds.Page) (*dto.PostListResponse, error)
		UploadPostMedia(ctx context.Context, postID uuid.UUID, userID uuid.UUID, contentType string, filename string, fileSize int64, reader io.Reader, isSpoiler bool) (*dto.PostMediaResponse, error)
		DeletePostMedia(ctx context.Context, postID uuid.UUID, mediaID int64, userID uuid.UUID) error
		LikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
		UnlikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
		CreateComment(ctx context.Context, postID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateCommentRequest) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, filename string, fileSize int64, reader io.Reader, isSpoiler bool) (*dto.PostMediaResponse, error)
		GetCornerCounts(ctx context.Context) (map[string]int, error)
		VotePoll(ctx context.Context, postID uuid.UUID, userID uuid.UUID, optionID int) (*dto.PollResponse, error)
		ResolveSuggestion(ctx context.Context, postID uuid.UUID, userID uuid.UUID, status string) error
		UnresolveSuggestion(ctx context.Context, postID uuid.UUID, userID uuid.UUID) error
		GetShareCount(ctx context.Context, contentID string, contentType string) (int, error)
		SetCommentObserver(obs CommentObserver)
	}

	service struct {
		postRepo      repository.PostRepository
		userRepo      repository.UserRepository
		roleRepo      repository.RoleRepository
		auditRepo     repository.AuditLogRepository
		authz         authz.Service
		blockSvc      block.Service
		notifService  notification.Service
		mentionSvc    mention.Service
		uploadSvc     upload.Service
		mediaProc     *media.Processor
		uploader      *media.Uploader
		settingsSvc   settings.Service
		hub           *ws.Hub
		contentFilter *contentfilter.Manager
		botObserver   CommentObserver
		ogCache       *og.Resolver
		echoCache     homefeed.Service
	}
)

var validPollDurations = map[int]bool{
	3600: true, 14400: true, 28800: true, 43200: true,
	86400: true, 259200: true, 604800: true, 1209600: true,
}

func NewService(
	postRepo repository.PostRepository,
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
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
	echoCache homefeed.Service,
) Service {
	return &service{
		postRepo:      postRepo,
		userRepo:      userRepo,
		roleRepo:      roleRepo,
		auditRepo:     auditRepo,
		authz:         authzService,
		blockSvc:      blockSvc,
		notifService:  notifService,
		mentionSvc:    mentionSvc,
		uploadSvc:     uploadSvc,
		mediaProc:     mediaProc,
		uploader:      media.NewUploader(uploadSvc, settingsSvc, mediaProc),
		settingsSvc:   settingsSvc,
		hub:           hub,
		contentFilter: contentFilter,
		ogCache:       ogCache,
		echoCache:     echoCache,
	}
}

func (s *service) writeAudit(ctx context.Context, entry audit.NewEntry) {
	if err := s.auditRepo.Create(ctx, entry); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("action", string(entry.Action)).Msg("failed to write audit log")
	}
}

var validSharedContentTypes = map[string]bool{
	"post": true, "art": true, "ship": true,
	"mystery": true, "theory": true, "fanfic": true,
}

func (s *service) CreatePost(ctx context.Context, userID uuid.UUID, req dto.CreatePostRequest) (uuid.UUID, error) {
	isShare := req.SharedContentID != ""
	if isShare {
		if !validSharedContentTypes[req.SharedContentType] {
			return uuid.Nil, ErrInvalidShareType
		}
	}

	if strings.TrimSpace(req.Body) == "" && !isShare {
		return uuid.Nil, ErrEmptyBody
	}

	pollLabels := pollOptionLabels(req.Poll)
	if err := s.contentFilter.Check(ctx, append([]string{req.Body}, pollLabels...)...); err != nil {
		return uuid.Nil, err
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

	body := strings.TrimSpace(req.Body)

	newPost := spec.NewPost{
		UserID: userID,
		Corner: corner,
		Body:   body,
	}
	if isShare {
		newPost.SharedContent = &model.SharedContentRef{ID: req.SharedContentID, Type: req.SharedContentType}
	}
	if req.Poll != nil {
		newPost.Poll = newPollSpec(req.Poll)
	}

	created, err := s.postRepo.CreateWithDetails(ctx, newPost)
	if err != nil {
		return uuid.Nil, err
	}

	id := created.ID

	if isShare {
		shared := model.SharedContentRef{ID: req.SharedContentID, Type: req.SharedContentType}

		go s.postRepo.IncrementShareCount(context.Background(), shared)
		go s.notifyContentShared(userID, id, req.SharedContentID, req.SharedContentType)
	}

	if corner == "suggestions" {
		go s.notifySuggestionPosted(userID, id)
	} else {
		s.mentionSvc.NotifyAsync(ctx, mention.Reference{Kind: mention.KindPost, EntityID: id}, userID, body)
		go s.observeForBot(id, nil, userID, body, nil)
	}

	return id, nil
}

func (s *service) GetPost(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, viewerHash string) (*dto.PostDetailResponse, error) {
	row, err := s.postRepo.GetByID(ctx, spec.PostLookup{ID: id, ViewerID: viewerID})
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}

	if viewerHash != "" {
		isNew, _ := s.postRepo.RecordView(ctx, spec.ViewRecord{TargetID: id, ViewerHash: viewerHash})
		if isNew {
			row.ViewCount++
		}
	}

	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)

	postMedia, _ := s.postRepo.GetMedia(ctx, id)
	comments, _, _ := s.postRepo.GetComments(ctx, spec.CommentQuery[uuid.UUID]{
		TargetID:       id,
		ViewerID:       viewerID,
		Limit:          500,
		Offset:         0,
		ExcludeUserIDs: blockedIDs,
	})

	var commentIDs []uuid.UUID
	for _, c := range comments {
		commentIDs = append(commentIDs, c.ID)
	}
	commentMediaMap, _ := s.postRepo.GetCommentMediaBatch(ctx, commentIDs)

	flatComments := make([]dto.PostCommentResponse, len(comments))
	for i, c := range comments {
		flatComments[i] = postCommentToResponse(c, commentMediaMap[c.ID])
	}
	dtoComments := utils.BuildTree(flatComments,
		func(c dto.PostCommentResponse) uuid.UUID { return c.ID },
		func(c dto.PostCommentResponse) *uuid.UUID { return c.ParentID },
		func(c *dto.PostCommentResponse, replies []dto.PostCommentResponse) { c.Replies = replies },
	)

	likeUsers, _ := s.postRepo.GetLikedBy(ctx, spec.LikedByQuery{TargetID: id, ExcludeUserIDs: blockedIDs})
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

	postResp := row.ToResponse(postMedia)
	pollRow, pollOptions, votedOption, _ := s.postRepo.GetPollByPostID(ctx, spec.PostPollQuery{PostID: id, ViewerID: viewerID})
	if pollRow != nil {
		postResp.Poll = pollRow.ToResponse(pollOptions, votedOption)
	}

	if row.SharedContentID != nil && row.SharedContentType != nil {
		refs := []model.SharedContentRef{{ID: *row.SharedContentID, Type: *row.SharedContentType}}
		previews := s.postRepo.GetSharedContentPreviews(refs)
		key := *row.SharedContentType + ":" + *row.SharedContentID
		postResp.SharedContent = previews[key]
	}

	shareCount, _ := s.postRepo.GetShareCount(ctx, model.SharedContentRef{ID: id.String(), Type: "post"})
	postResp.ShareCount = shareCount

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

	if err := s.contentFilter.Check(ctx, body); err != nil {
		return err
	}

	asAdmin := s.authz.Can(ctx, userID, authz.PermEditAnyPost)

	update := spec.PostUpdate{
		ID:      id,
		UserID:  userID,
		Body:    body,
		AsAdmin: asAdmin,
	}
	if err := s.postRepo.UpdateWithDetails(ctx, update); err != nil {
		return err
	}

	if asAdmin {
		go s.notifyContentEdited(ctx, id, "post", userID)
	}

	return nil
}

func (s *service) DeletePost(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.postRepo.GetPostAuthorID(ctx, id)
	if err != nil {
		return err
	}

	postMedia, err := s.postRepo.GetMedia(ctx, id)
	if err != nil {
		return err
	}

	action := audit.ActionPostDelete
	if authorID != userID {
		action = audit.ActionPostDeleteAdmin
	}

	deletion := spec.PostDelete{
		ID:      id,
		UserID:  userID,
		AsAdmin: s.authz.Can(ctx, userID, authz.PermDeleteAnyPost),
		Audit: audit.NewEntry{
			ActorID:    userID,
			Action:     action,
			TargetType: audit.TargetPost,
			TargetID:   id.String(),
			Details:    fmt.Sprintf("media=%d", len(postMedia)),
			SubjectID:  authorID,
		},
	}

	shared, paths, err := s.postRepo.DeleteWithSharedContent(ctx, deletion)
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(paths...)

	if shared != nil {
		go s.postRepo.DecrementShareCount(context.Background(), *shared)
	}

	s.clearPageCache(ctx, id.String())

	return nil
}

func (s *service) clearPageCache(ctx context.Context, id string) {
	if s.ogCache != nil {
		if err := s.ogCache.ClearMetaCache(ctx, og.KindPost, id); err != nil {
			logger.Ctx(ctx).Warn().Err(err).Str("post_id", id).Msg("clear og meta cache failed")
		}
	}

	if s.echoCache == nil {
		return
	}

	if err := s.echoCache.ClearEchoCache(ctx); err != nil {
		logger.Ctx(ctx).Warn().Err(err).Msg("clear echo cache failed")
	}
}

func (s *service) ListFeed(ctx context.Context, tab string, viewerID uuid.UUID, corner string, search string, sort string, seed int, page bounds.Page, resolvedFilter string) (*dto.PostListResponse, error) {
	if corner == "" {
		corner = "general"
	}

	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)

	var rows []model.PostRow
	var total int
	var err error

	if tab == "following" && viewerID != uuid.Nil {
		rows, total, err = s.postRepo.ListByFollowing(ctx, spec.PostFollowingFeedQuery{
			UserID:         viewerID,
			Corner:         corner,
			Sort:           sort,
			Seed:           seed,
			Limit:          page.Limit(),
			Offset:         page.Offset(),
			ExcludeUserIDs: blockedIDs,
		})
	} else {
		rows, total, err = s.postRepo.ListAll(ctx, spec.PostFeedQuery{
			ViewerID:       viewerID,
			Corner:         corner,
			Search:         search,
			Sort:           sort,
			Seed:           seed,
			Limit:          page.Limit(),
			Offset:         page.Offset(),
			ExcludeUserIDs: blockedIDs,
			ResolvedFilter: resolvedFilter,
		})
	}
	if err != nil {
		return nil, err
	}

	return s.buildPostList(ctx, rows, total, page, viewerID), nil
}

func (s *service) ListUserPosts(ctx context.Context, targetUserID uuid.UUID, viewerID uuid.UUID, page bounds.Page) (*dto.PostListResponse, error) {
	rows, total, err := s.postRepo.ListByUser(ctx, spec.PostUserPage{
		UserID:   targetUserID,
		ViewerID: viewerID,
		Limit:    page.Limit(),
		Offset:   page.Offset(),
	})
	if err != nil {
		return nil, err
	}
	return s.buildPostList(ctx, rows, total, page, viewerID), nil
}

func (s *service) buildPostList(ctx context.Context, rows []model.PostRow, total int, page bounds.Page, viewerID uuid.UUID) *dto.PostListResponse {
	postIDs := make([]uuid.UUID, len(rows))
	postIDStrs := make([]string, len(rows))
	for i, r := range rows {
		postIDs[i] = r.ID
		postIDStrs[i] = r.ID.String()
	}

	mediaMap, _ := s.postRepo.GetMediaBatch(ctx, postIDs)
	pollMap, pollOptionMap, pollVoteMap, _ := s.postRepo.GetPollsByPostIDs(ctx, spec.PostPollBatchQuery{PostIDs: postIDs, ViewerID: viewerID})

	var sharedRefs []model.SharedContentRef
	for _, r := range rows {
		if r.SharedContentID != nil && r.SharedContentType != nil {
			sharedRefs = append(sharedRefs, model.SharedContentRef{
				ID:   *r.SharedContentID,
				Type: *r.SharedContentType,
			})
		}
	}
	sharedPreviews := s.postRepo.GetSharedContentPreviews(sharedRefs)

	postShareCounts, _ := s.postRepo.GetShareCountsBatch(ctx, spec.SharedContentBatchRef{ContentIDs: postIDStrs, ContentType: "post"})

	posts := make([]dto.PostResponse, len(rows))
	for i, r := range rows {
		posts[i] = r.ToResponse(mediaMap[r.ID])
		if p, ok := pollMap[r.ID]; ok {
			posts[i].Poll = p.ToResponse(pollOptionMap[r.ID], pollVoteMap[r.ID])
		}
		if r.SharedContentID != nil && r.SharedContentType != nil {
			key := *r.SharedContentType + ":" + *r.SharedContentID
			posts[i].SharedContent = sharedPreviews[key]
		}
		if count, ok := postShareCounts[r.ID.String()]; ok {
			posts[i].ShareCount = count
		}
	}

	return &dto.PostListResponse{
		Posts:  posts,
		Total:  total,
		Limit:  page.Limit(),
		Offset: page.Offset(),
	}
}

func (s *service) UploadPostMedia(ctx context.Context, postID uuid.UUID, userID uuid.UUID, contentType string, filename string, fileSize int64, reader io.Reader, isSpoiler bool) (*dto.PostMediaResponse, error) {
	authorID, err := s.postRepo.GetPostAuthorID(ctx, postID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID {
		return nil, fmt.Errorf("not the post author")
	}

	return s.uploader.SaveAndRecord(ctx, "posts", contentType, filename, fileSize, reader, isSpoiler,
		func(mediaURL, mediaType, thumbURL, filename string, sortOrder int) (int64, error) {
			return s.postRepo.AddMedia(ctx, spec.NewMedia{
				TargetID:     postID,
				MediaURL:     mediaURL,
				MediaType:    mediaType,
				ThumbnailURL: thumbURL,
				Filename:     filename,
				SortOrder:    sortOrder,
				IsSpoiler:    isSpoiler,
			})
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

	mediaURL, err := s.postRepo.DeleteMedia(ctx, spec.MediaDeletion{ID: mediaID, TargetID: postID})
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(mediaURL)
	return nil
}

func (s *service) UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, filename string, fileSize int64, reader io.Reader, isSpoiler bool) (*dto.PostMediaResponse, error) {
	authorID, err := s.postRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID {
		return nil, fmt.Errorf("not the comment author")
	}

	return s.uploader.SaveAndRecord(ctx, "posts", contentType, filename, fileSize, reader, isSpoiler,
		func(mediaURL, mediaType, thumbURL, filename string, sortOrder int) (int64, error) {
			return s.postRepo.AddCommentMedia(ctx, spec.NewMedia{
				TargetID:     commentID,
				MediaURL:     mediaURL,
				MediaType:    mediaType,
				ThumbnailURL: thumbURL,
				Filename:     filename,
				SortOrder:    sortOrder,
				IsSpoiler:    isSpoiler,
			})
		},
		s.postRepo.UpdateCommentMediaURL,
		s.postRepo.UpdateCommentMediaThumbnail,
	)
}

func (s *service) broadcastCommentAdded(postID, commentID uuid.UUID) {
	s.hub.Broadcast(ws.Message{
		Type: "post_comment",
		Data: map[string]any{
			"post_id":    postID,
			"comment_id": commentID,
		},
	})
}

func (s *service) broadcastLikeUpdate(postID uuid.UUID, delta int) {
	s.hub.Broadcast(ws.Message{
		Type: "post_like",
		Data: map[string]any{
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

	if err := s.postRepo.Like(ctx, spec.Like{UserID: userID, TargetID: postID}); err != nil {
		return err
	}

	s.broadcastLikeUpdate(postID, 1)

	go func() {
		actor, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || actor == nil {
			return
		}
		_ = s.notifService.Notify(ctx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifPostLiked,
			ReferenceID:   postID,
			ReferenceType: "post",
			ActorID:       userID,
			EmailActor:    actor.DisplayName,
			EmailAction:   "liked your post",
			EmailLink:     fmt.Sprintf("/game-board/%s", postID),
		})
	}()

	return nil
}

func (s *service) UnlikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error {
	if err := s.postRepo.Unlike(ctx, spec.Like{UserID: userID, TargetID: postID}); err != nil {
		return err
	}
	s.broadcastLikeUpdate(postID, -1)
	return nil
}

func (s *service) CreateComment(ctx context.Context, postID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error) {
	if strings.TrimSpace(req.Body) == "" {
		return uuid.Nil, ErrEmptyBody
	}
	if err := s.contentFilter.Check(ctx, req.Body); err != nil {
		return uuid.Nil, err
	}

	authorID, err := s.postRepo.GetPostAuthorID(ctx, postID)
	if err != nil {
		return uuid.Nil, err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return uuid.Nil, block.ErrUserBlocked
	}

	body := strings.TrimSpace(req.Body)

	id, err := s.mentionSvc.CreateComment(ctx, mention.CommentSpec{
		Kind:     mention.KindPostComment,
		EntityID: postID,
		ParentID: req.ParentID,
		AuthorID: userID,
		Body:     body,
	})
	if err != nil {
		return uuid.Nil, err
	}

	s.broadcastCommentAdded(postID, id)

	go s.observeForBot(postID, new(id), userID, body, req.ParentID)

	go func() {
		ctx := context.Background()

		actor, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || actor == nil {
			return
		}
		if req.ParentID == nil {
			_ = s.notifService.Notify(ctx, dto.NotifyParams{
				RecipientID:   authorID,
				Type:          dto.NotifPostCommented,
				ReferenceID:   postID,
				ReferenceType: fmt.Sprintf("post_comment:%s", id),
				ActorID:       userID,
				EmailActor:    actor.DisplayName,
				EmailAction:   "commented on your post",
				EmailLink:     fmt.Sprintf("/game-board/%s#comment-%s", postID, id),
			})
			return
		}

		parentAuthorID, err := s.postRepo.GetCommentAuthorID(ctx, *req.ParentID)
		if err == nil && parentAuthorID != userID {
			_ = s.notifService.Notify(ctx, dto.NotifyParams{
				RecipientID:   parentAuthorID,
				Type:          dto.NotifPostCommentReply,
				ReferenceID:   postID,
				ReferenceType: fmt.Sprintf("post_comment:%s", id),
				ActorID:       userID,
				EmailActor:    actor.DisplayName,
				EmailAction:   "replied to your comment",
				EmailLink:     fmt.Sprintf("/game-board/%s#comment-%s", postID, id),
			})
		}

		if authorID != userID && authorID != parentAuthorID {
			_ = s.notifService.Notify(ctx, dto.NotifyParams{
				RecipientID:   authorID,
				Type:          dto.NotifPostCommented,
				ReferenceID:   postID,
				ReferenceType: fmt.Sprintf("post_comment:%s", id),
				ActorID:       userID,
				EmailActor:    actor.DisplayName,
				EmailAction:   "commented on your post",
				EmailLink:     fmt.Sprintf("/game-board/%s#comment-%s", postID, id),
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

	if err := s.contentFilter.Check(ctx, body); err != nil {
		return err
	}

	authorID, err := s.postRepo.GetCommentAuthorID(ctx, id)
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
	if err := s.postRepo.UpdateCommentWithDetails(ctx, update); err != nil {
		return err
	}

	if authorID != userID {
		s.writeAudit(ctx, audit.NewEntry{
			ActorID:    userID,
			Action:     audit.ActionPostCommentUpdateAdmin,
			TargetType: audit.TargetPostComment,
			TargetID:   id.String(),
			SubjectID:  authorID,
		})
	}

	if asAdmin {
		go s.notifyCommentEdited(ctx, id, "post", userID)
	}

	return nil
}

func (s *service) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.postRepo.GetCommentAuthorID(ctx, id)
	if err != nil {
		return err
	}

	action := audit.ActionPostCommentDelete
	if authorID != userID {
		action = audit.ActionPostCommentDeleteAdmin
	}

	paths, err := s.postRepo.DeleteCommentWithAudit(ctx, spec.PostCommentDelete{
		CommentDeletion: spec.CommentDeletion{
			CommentID: id,
			UserID:    userID,
			AsAdmin:   s.authz.Can(ctx, userID, authz.PermDeleteAnyComment),
		},
		Audit: audit.NewEntry{
			ActorID:    userID,
			Action:     action,
			TargetType: audit.TargetPostComment,
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
	commentAuthorID, err := s.postRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, commentAuthorID); blocked {
		return block.ErrUserBlocked
	}

	if err := s.postRepo.LikeComment(ctx, spec.CommentLike{UserID: userID, CommentID: commentID}); err != nil {
		return err
	}

	go func() {
		postID, err := s.postRepo.GetCommentEntityID(ctx, commentID)
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
			ReferenceID:   postID,
			ReferenceType: fmt.Sprintf("post_comment:%s", commentID),
			ActorID:       userID,
			EmailActor:    actor.DisplayName,
			EmailAction:   "liked your comment",
			EmailLink:     fmt.Sprintf("/game-board/%s#comment-%s", postID, commentID),
		})
	}()

	return nil
}

func (s *service) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return s.postRepo.UnlikeComment(ctx, spec.CommentLike{UserID: userID, CommentID: commentID})
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
			EmailActor:    "Someone",
			EmailAction:   "posted a site suggestion",
			EmailLink:     fmt.Sprintf("/suggestions/%s", postID),
		})
	}
}

func (s *service) notifyContentEdited(ctx context.Context, postID uuid.UUID, contentType string, editorID uuid.UUID) {
	authorID, err := s.postRepo.GetPostAuthorID(ctx, postID)
	if err != nil {
		return
	}
	notification.SendEditNotification(ctx, s.userRepo, s.notifService, notification.EditNotifyParams{
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
	postID, err := s.postRepo.GetCommentEntityID(ctx, commentID)
	if err != nil {
		return
	}
	notification.SendEditNotification(ctx, s.userRepo, s.notifService, notification.EditNotifyParams{
		AuthorID:      authorID,
		EditorID:      editorID,
		ContentType:   "comment",
		ReferenceID:   postID,
		ReferenceType: fmt.Sprintf("post_comment:%s", commentID),
		LinkPath:      fmt.Sprintf("/game-board/%s#comment-%s", postID, commentID),
	})
}

func (s *service) VotePoll(ctx context.Context, postID uuid.UUID, userID uuid.UUID, optionID int) (*dto.PollResponse, error) {
	pollRow, options, votedOption, err := s.postRepo.GetPollByPostID(ctx, spec.PostPollQuery{PostID: postID, ViewerID: userID})
	if err != nil {
		return nil, err
	}
	if pollRow == nil {
		return nil, ErrNotFound
	}

	authorID, err := s.postRepo.GetPostAuthorID(ctx, postID)
	if err != nil {
		return nil, err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return nil, block.ErrUserBlocked
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
	if err := s.postRepo.VotePoll(ctx, spec.PostPollVote{PollID: pollID, UserID: userID, OptionID: optionID}); err != nil {
		if strings.Contains(err.Error(), "already voted") {
			return nil, ErrAlreadyVoted
		}
		return nil, err
	}

	pollRow, options, votedOption, err = s.postRepo.GetPollByPostID(ctx, spec.PostPollQuery{PostID: postID, ViewerID: userID})
	if err != nil {
		return nil, err
	}
	return pollRow.ToResponse(options, votedOption), nil
}

func newPollSpec(poll *dto.CreatePollInput) *spec.NewPoll {
	labels := make([]string, len(poll.Options))
	for i, o := range poll.Options {
		labels[i] = strings.TrimSpace(o.Label)
	}

	return &spec.NewPoll{
		DurationSeconds: poll.DurationSeconds,
		ExpiresAt:       time.Now().UTC().Add(time.Duration(poll.DurationSeconds) * time.Second).Format(time.RFC3339),
		Options:         labels,
	}
}

func pollOptionLabels(poll *dto.CreatePollInput) []string {
	if poll == nil {
		return nil
	}
	out := make([]string, 0, len(poll.Options))
	for _, o := range poll.Options {
		out = append(out, o.Label)
	}
	return out
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

func (s *service) ResolveSuggestion(ctx context.Context, postID uuid.UUID, userID uuid.UUID, status string) error {
	if !s.authz.Can(ctx, userID, authz.PermResolveSuggestion) {
		return fmt.Errorf("not authorised")
	}
	if status != "done" && status != "archived" {
		status = "done"
	}
	if err := s.postRepo.ResolveSuggestion(ctx, spec.SuggestionResolution{PostID: postID, ResolvedBy: userID, Status: status}); err != nil {
		return err
	}

	if status == "done" {
		go func() {
			bgCtx := context.Background()
			authorID, err := s.postRepo.GetPostAuthorID(bgCtx, postID)
			if err != nil || authorID == userID {
				return
			}
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   authorID,
				Type:          dto.NotifSuggestionResolved,
				ReferenceID:   postID,
				ReferenceType: "post",
				ActorID:       userID,
				EmailActor:    "An admin",
				EmailAction:   "marked your suggestion as done",
				EmailLink:     fmt.Sprintf("/suggestions/%s", postID),
			})
		}()
	}

	return nil
}

func (s *service) UnresolveSuggestion(ctx context.Context, postID uuid.UUID, userID uuid.UUID) error {
	if !s.authz.Can(ctx, userID, authz.PermResolveSuggestion) {
		return fmt.Errorf("not authorised")
	}
	return s.postRepo.UnresolveSuggestion(ctx, postID)
}

func (s *service) GetShareCount(ctx context.Context, contentID string, contentType string) (int, error) {
	return s.postRepo.GetShareCount(ctx, model.SharedContentRef{ID: contentID, Type: contentType})
}

func (s *service) notifyContentShared(sharerID uuid.UUID, postID uuid.UUID, contentID string, contentType string) {
	bgCtx := context.Background()

	authorID, err := s.postRepo.GetSharedContentAuthor(bgCtx, model.SharedContentRef{ID: contentID, Type: contentType})
	if err != nil {
		logger.Ctx(bgCtx).
			Err(err).
			Str("content_id", contentID).
			Str("content_type", contentType).
			Msg("notify content shared: lookup author failed")
		return
	}
	if authorID == sharerID {
		return
	}

	_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
		RecipientID:   authorID,
		Type:          dto.NotifContentShared,
		ReferenceID:   postID,
		ReferenceType: "post",
		ActorID:       sharerID,
		EmailActor:    "Someone",
		EmailAction:   "shared your content",
		EmailLink:     fmt.Sprintf("/game-board/%s", postID),
	})
}

func postCommentToResponse(c model.CommentRow, media []model.PostMediaRow) dto.PostCommentResponse {
	return dto.PostCommentResponse{
		ID:       c.ID,
		ParentID: c.ParentID,
		Author: dto.UserResponse{
			ID:          c.UserID,
			Username:    c.AuthorUsername,
			DisplayName: c.AuthorDisplayName,
			AvatarURL:   c.AuthorAvatarURL,
			Role:        role.Role(c.AuthorRole),
			Banned:      c.AuthorBanned,
		},
		Body:      c.Body,
		Media:     model.MediaRowsToResponse(media),
		LikeCount: c.LikeCount,
		UserLiked: c.UserLiked,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
