package controllers

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	ctrlutils "umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/utils"
	"umineko_city_of_books/internal/ws"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllAnnouncementRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupListAnnouncements,
		s.setupGetAnnouncement,
		s.setupGetLatestAnnouncement,
		s.setupCreateAnnouncement,
		s.setupUpdateAnnouncement,
		s.setupDeleteAnnouncement,
		s.setupPinAnnouncement,
		s.setupCreateAnnouncementComment,
		s.setupUpdateAnnouncementComment,
		s.setupDeleteAnnouncementComment,
		s.setupLikeAnnouncementComment,
		s.setupUnlikeAnnouncementComment,
		s.setupUploadAnnouncementCommentMedia,
	}
}

func (s *Service) setupCreateAnnouncementComment(r fiber.Router) {
	r.Post("/announcements/:id/comments", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.createAnnouncementComment)
}

func (s *Service) setupUpdateAnnouncementComment(r fiber.Router) {
	r.Put("/announcement-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.updateAnnouncementComment)
}

func (s *Service) setupDeleteAnnouncementComment(r fiber.Router) {
	r.Delete("/announcement-comments/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.deleteAnnouncementComment)
}

func (s *Service) setupLikeAnnouncementComment(r fiber.Router) {
	r.Post("/announcement-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.likeAnnouncementComment)
}

func (s *Service) setupUnlikeAnnouncementComment(r fiber.Router) {
	r.Delete("/announcement-comments/:id/like", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.unlikeAnnouncementComment)
}

func (s *Service) setupUploadAnnouncementCommentMedia(r fiber.Router) {
	r.Post("/announcement-comments/:id/media", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.uploadAnnouncementCommentMedia)
}

func (s *Service) setupListAnnouncements(r fiber.Router) {
	r.Get("/announcements", s.listAnnouncements)
}

func (s *Service) setupGetAnnouncement(r fiber.Router) {
	r.Get("/announcements/:id", middleware.OptionalAuth(s.AuthSession, s.AuthzService), s.getAnnouncement)
}

func (s *Service) setupGetLatestAnnouncement(r fiber.Router) {
	r.Get("/announcements-latest", s.getLatestAnnouncement)
}

func (s *Service) setupCreateAnnouncement(r fiber.Router) {
	r.Post("/admin/announcements", s.requirePerm(authz.PermManageSettings), s.createAnnouncement)
}

func (s *Service) setupUpdateAnnouncement(r fiber.Router) {
	r.Put("/admin/announcements/:id", s.requirePerm(authz.PermManageSettings), s.updateAnnouncement)
}

func (s *Service) setupDeleteAnnouncement(r fiber.Router) {
	r.Delete("/admin/announcements/:id", s.requirePerm(authz.PermManageSettings), s.deleteAnnouncement)
}

func (s *Service) setupPinAnnouncement(r fiber.Router) {
	r.Post("/admin/announcements/:id/pin", s.requirePerm(authz.PermManageSettings), s.pinAnnouncement)
}

func (s *Service) listAnnouncements(ctx fiber.Ctx) error {
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	rows, total, err := s.AnnouncementRepo.List(ctx.Context(), limit, offset)
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to list announcements")
	}

	items := make([]fiber.Map, len(rows))
	for i, r := range rows {
		items[i] = fiber.Map{
			"id":         r.ID,
			"title":      r.Title,
			"body":       r.Body,
			"pinned":     r.Pinned,
			"created_at": r.CreatedAt,
			"updated_at": r.UpdatedAt,
			"author": dto.UserResponse{
				ID:          r.AuthorID,
				Username:    r.AuthorUsername,
				DisplayName: r.AuthorDisplayName,
				AvatarURL:   r.AuthorAvatarURL,
				Role:        role.Role(r.AuthorRole),
			},
		}
	}

	return ctx.JSON(fiber.Map{
		"announcements": items,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	})
}

func (s *Service) getAnnouncement(ctx fiber.Ctx) error {
	id, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}
	viewerID := ctrlutils.UserID(ctx)

	row, err := s.AnnouncementRepo.GetByID(ctx.Context(), id)
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to get announcement")
	}
	if row == nil {
		return ctrlutils.NotFound(ctx, "announcement not found")
	}

	blockedIDs, _ := s.BlockService.GetBlockedIDs(ctx.Context(), viewerID)
	commentRows, _, _ := s.AnnouncementRepo.GetComments(ctx.Context(), id, viewerID, 500, 0, blockedIDs)

	commentIDs := make([]uuid.UUID, len(commentRows))
	for i, c := range commentRows {
		commentIDs[i] = c.ID
	}
	mediaMap, _ := s.AnnouncementRepo.GetCommentMediaBatch(ctx.Context(), commentIDs)

	flatComments := make([]dto.AnnouncementCommentResponse, len(commentRows))
	for i, c := range commentRows {
		flatComments[i] = announcementCommentToDTO(c, mediaMap[c.ID])
	}
	comments := utils.BuildTree(flatComments,
		func(c dto.AnnouncementCommentResponse) uuid.UUID { return c.ID },
		func(c dto.AnnouncementCommentResponse) *uuid.UUID { return c.ParentID },
		func(c *dto.AnnouncementCommentResponse, replies []dto.AnnouncementCommentResponse) {
			c.Replies = replies
		},
	)

	return ctx.JSON(fiber.Map{
		"id":         row.ID,
		"title":      row.Title,
		"body":       row.Body,
		"pinned":     row.Pinned,
		"created_at": row.CreatedAt,
		"updated_at": row.UpdatedAt,
		"author": dto.UserResponse{
			ID:          row.AuthorID,
			Username:    row.AuthorUsername,
			DisplayName: row.AuthorDisplayName,
			AvatarURL:   row.AuthorAvatarURL,
			Role:        role.Role(row.AuthorRole),
		},
		"comments": comments,
	})
}

func announcementCommentToDTO(c repository.AnnouncementCommentRow, media []repository.AnnouncementCommentMediaRow) dto.AnnouncementCommentResponse {
	mediaList := model.CommentMediaRowsToResponse(media)
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
		Media:     mediaList,
		LikeCount: c.LikeCount,
		UserLiked: c.UserLiked,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (s *Service) getLatestAnnouncement(ctx fiber.Ctx) error {
	row, err := s.AnnouncementRepo.GetLatest(ctx.Context())
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to get latest announcement")
	}
	if row == nil {
		return ctx.JSON(fiber.Map{"announcement": nil})
	}

	return ctx.JSON(fiber.Map{
		"announcement": fiber.Map{
			"id":         row.ID,
			"title":      row.Title,
			"body":       row.Body,
			"pinned":     row.Pinned,
			"created_at": row.CreatedAt,
			"updated_at": row.UpdatedAt,
			"author": dto.UserResponse{
				ID:          row.AuthorID,
				Username:    row.AuthorUsername,
				DisplayName: row.AuthorDisplayName,
				AvatarURL:   row.AuthorAvatarURL,
				Role:        role.Role(row.AuthorRole),
			},
		},
	})
}

func (s *Service) createAnnouncement(ctx fiber.Ctx) error {
	userID := ctrlutils.UserID(ctx)

	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctrlutils.BadRequest(ctx, "invalid request body")
	}
	if req.Title == "" || req.Body == "" {
		return ctrlutils.BadRequest(ctx, "title and body are required")
	}

	id := uuid.New()
	if err := s.AnnouncementRepo.Create(ctx.Context(), id, userID, req.Title, req.Body); err != nil {
		return ctrlutils.InternalError(ctx, "failed to create announcement")
	}

	s.Hub.Broadcast(ws.Message{
		Type: "new_announcement",
		Data: map[string]interface{}{
			"id":        id,
			"title":     req.Title,
			"author_id": userID,
		},
	})

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateAnnouncement(ctx fiber.Ctx) error {
	id, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}

	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctrlutils.BadRequest(ctx, "invalid request body")
	}
	if req.Title == "" || req.Body == "" {
		return ctrlutils.BadRequest(ctx, "title and body are required")
	}

	if err := s.AnnouncementRepo.Update(ctx.Context(), id, req.Title, req.Body); err != nil {
		return ctrlutils.InternalError(ctx, "failed to update announcement")
	}

	return ctrlutils.OK(ctx)
}

func (s *Service) deleteAnnouncement(ctx fiber.Ctx) error {
	id, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}

	if err := s.AnnouncementRepo.Delete(ctx.Context(), id); err != nil {
		return ctrlutils.InternalError(ctx, "failed to delete announcement")
	}

	return ctrlutils.OK(ctx)
}

func (s *Service) pinAnnouncement(ctx fiber.Ctx) error {
	id, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}

	var req struct {
		Pinned bool `json:"pinned"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctrlutils.BadRequest(ctx, "invalid request body")
	}

	if err := s.AnnouncementRepo.SetPinned(ctx.Context(), id, req.Pinned); err != nil {
		return ctrlutils.InternalError(ctx, "failed to pin announcement")
	}

	return ctrlutils.OK(ctx)
}

func (s *Service) createAnnouncementComment(ctx fiber.Ctx) error {
	announcementID, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := ctrlutils.UserID(ctx)

	req, ok := ctrlutils.BindJSON[dto.CreateCommentRequest](ctx)
	if !ok {
		return nil
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return ctrlutils.BadRequest(ctx, "body is required")
	}

	ann, err := s.AnnouncementRepo.GetByID(ctx.Context(), announcementID)
	if err != nil || ann == nil {
		return ctrlutils.NotFound(ctx, "announcement not found")
	}
	if blocked, _ := s.BlockService.IsBlockedEither(ctx.Context(), userID, ann.AuthorID); blocked {
		return ctrlutils.Forbidden(ctx, "user is blocked")
	}

	id := uuid.New()
	if err := s.AnnouncementRepo.CreateComment(ctx.Context(), id, announcementID, req.ParentID, userID, body); err != nil {
		logger.Log.Error().Err(err).
			Str("announcement_id", announcementID.String()).
			Str("user_id", userID.String()).
			Msg("failed to create announcement comment")
		return ctrlutils.InternalError(ctx, "failed to create comment")
	}

	go func() {
		bgCtx := context.Background()
		actor, err := s.UserRepo.GetByID(bgCtx, userID)
		if err != nil || actor == nil {
			return
		}
		baseURL := s.SettingsService.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/announcements/%s#comment-%s", baseURL, announcementID, id)

		subject, emailBody := notification.NotifEmail(actor.DisplayName, "commented on your announcement", ann.Title, linkURL)
		_ = s.NotificationService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   ann.AuthorID,
			Type:          dto.NotifAnnouncementCommented,
			ReferenceID:   announcementID,
			ReferenceType: fmt.Sprintf("announcement_comment:%s", id),
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     emailBody,
		})

		if req.ParentID != nil {
			parentAuthor, err := s.AnnouncementRepo.GetCommentAuthorID(bgCtx, *req.ParentID)
			if err == nil && parentAuthor != ann.AuthorID {
				replySubject, replyBody := notification.NotifEmail(actor.DisplayName, "replied to your comment", ann.Title, linkURL)
				_ = s.NotificationService.Notify(bgCtx, dto.NotifyParams{
					RecipientID:   parentAuthor,
					Type:          dto.NotifAnnouncementCommentReply,
					ReferenceID:   announcementID,
					ReferenceType: fmt.Sprintf("announcement_comment:%s", id),
					ActorID:       userID,
					EmailSubject:  replySubject,
					EmailBody:     replyBody,
				})
			}
		}
	}()

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *Service) updateAnnouncementComment(ctx fiber.Ctx) error {
	id, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := ctrlutils.UserID(ctx)

	req, ok := ctrlutils.BindJSON[dto.UpdateCommentRequest](ctx)
	if !ok {
		return nil
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return ctrlutils.BadRequest(ctx, "body is required")
	}

	if s.AuthzService.Can(ctx.Context(), userID, authz.PermEditAnyComment) {
		if err := s.AnnouncementRepo.UpdateCommentAsAdmin(ctx.Context(), id, body); err != nil {
			return ctrlutils.InternalError(ctx, "failed to update comment")
		}
	} else if err := s.AnnouncementRepo.UpdateComment(ctx.Context(), id, userID, body); err != nil {
		return ctrlutils.Forbidden(ctx, "cannot update this comment")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) deleteAnnouncementComment(ctx fiber.Ctx) error {
	id, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := ctrlutils.UserID(ctx)

	if s.AuthzService.Can(ctx.Context(), userID, authz.PermDeleteAnyComment) {
		if err := s.AnnouncementRepo.DeleteCommentAsAdmin(ctx.Context(), id); err != nil {
			return ctrlutils.InternalError(ctx, "failed to delete comment")
		}
	} else if err := s.AnnouncementRepo.DeleteComment(ctx.Context(), id, userID); err != nil {
		return ctrlutils.Forbidden(ctx, "cannot delete this comment")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) likeAnnouncementComment(ctx fiber.Ctx) error {
	commentID, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := ctrlutils.UserID(ctx)

	commentAuthorID, err := s.AnnouncementRepo.GetCommentAuthorID(ctx.Context(), commentID)
	if err != nil {
		return ctrlutils.NotFound(ctx, "comment not found")
	}
	if blocked, _ := s.BlockService.IsBlockedEither(ctx.Context(), userID, commentAuthorID); blocked {
		return ctrlutils.Forbidden(ctx, "user is blocked")
	}

	if err := s.AnnouncementRepo.LikeComment(ctx.Context(), userID, commentID); err != nil {
		if errors.Is(err, block.ErrUserBlocked) {
			return ctrlutils.Forbidden(ctx, "user is blocked")
		}
		return ctrlutils.InternalError(ctx, "failed to like comment")
	}

	go func() {
		bgCtx := context.Background()
		announcementID, err := s.AnnouncementRepo.GetCommentAnnouncementID(bgCtx, commentID)
		if err != nil {
			return
		}
		actor, err := s.UserRepo.GetByID(bgCtx, userID)
		if err != nil || actor == nil {
			return
		}
		baseURL := s.SettingsService.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/announcements/%s#comment-%s", baseURL, announcementID, commentID)
		subject, emailBody := notification.NotifEmail(actor.DisplayName, "liked your comment", "", linkURL)
		_ = s.NotificationService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   commentAuthorID,
			Type:          dto.NotifAnnouncementCommentLiked,
			ReferenceID:   announcementID,
			ReferenceType: fmt.Sprintf("announcement_comment:%s", commentID),
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     emailBody,
		})
	}()

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) unlikeAnnouncementComment(ctx fiber.Ctx) error {
	commentID, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := ctrlutils.UserID(ctx)

	if err := s.AnnouncementRepo.UnlikeComment(ctx.Context(), userID, commentID); err != nil {
		return ctrlutils.InternalError(ctx, "failed to unlike comment")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) uploadAnnouncementCommentMedia(ctx fiber.Ctx) error {
	commentID, ok := ctrlutils.ParseID(ctx)
	if !ok {
		return nil
	}
	userID := ctrlutils.UserID(ctx)

	authorID, err := s.AnnouncementRepo.GetCommentAuthorID(ctx.Context(), commentID)
	if err != nil {
		return ctrlutils.NotFound(ctx, "comment not found")
	}
	if authorID != userID {
		return ctrlutils.Forbidden(ctx, "not the comment author")
	}

	file, err := ctx.FormFile("media")
	if err != nil {
		return ctrlutils.BadRequest(ctx, "no media file provided")
	}
	reader, err := file.Open()
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to read file")
	}
	defer reader.Close()

	contentType := file.Header.Get("Content-Type")
	isVideo := strings.HasPrefix(contentType, "video/")
	mediaID := uuid.New()

	var urlPath string
	if isVideo {
		maxSize := int64(s.SettingsService.GetInt(ctx.Context(), config.SettingMaxVideoSize))
		urlPath, err = s.UploadService.SaveVideo(ctx.Context(), "announcements", mediaID, contentType, file.Size, maxSize, reader)
	} else {
		maxSize := int64(s.SettingsService.GetInt(ctx.Context(), config.SettingMaxImageSize))
		urlPath, err = s.UploadService.SaveImage(ctx.Context(), "announcements", mediaID, contentType, file.Size, maxSize, reader)
	}
	if err != nil {
		return ctrlutils.BadRequest(ctx, err.Error())
	}

	mediaType := "image"
	if isVideo {
		mediaType = "video"
	}

	rowID, err := s.AnnouncementRepo.AddCommentMedia(ctx.Context(), commentID, urlPath, mediaType, "", 0)
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to save media record")
	}

	diskPath := s.UploadService.FullDiskPath(urlPath)
	if isVideo {
		s.MediaProcessor.Enqueue(media.Job{
			Type:      media.JobVideo,
			InputPath: diskPath,
			Callback: func(outputPath string) {
				newURL := "/uploads/announcements/" + filepath.Base(outputPath)
				if err := s.AnnouncementRepo.UpdateCommentMediaURL(context.Background(), rowID, newURL); err != nil {
					logger.Log.Error().Err(err).Msg("failed to update announcement comment video url")
				}
				thumbName, err := media.GenerateThumbnail(outputPath, filepath.Dir(outputPath), filepath.Base(outputPath))
				if err != nil {
					return
				}
				thumbURL := "/uploads/announcements/" + thumbName
				_ = s.AnnouncementRepo.UpdateCommentMediaThumbnail(context.Background(), rowID, thumbURL)
			},
		})
	} else {
		done := make(chan string, 1)
		s.MediaProcessor.Enqueue(media.Job{
			Type:      media.JobImage,
			InputPath: diskPath,
			Callback: func(outputPath string) {
				newURL := "/uploads/announcements/" + filepath.Base(outputPath)
				if err := s.AnnouncementRepo.UpdateCommentMediaURL(context.Background(), rowID, newURL); err != nil {
					logger.Log.Error().Err(err).Msg("failed to update announcement comment image url")
				}
				done <- newURL
			},
		})
		select {
		case newURL := <-done:
			urlPath = newURL
		case <-ctx.Context().Done():
		}
	}

	return ctx.Status(fiber.StatusCreated).JSON(dto.PostMediaResponse{
		ID:        int(rowID),
		MediaURL:  urlPath,
		MediaType: mediaType,
	})
}
