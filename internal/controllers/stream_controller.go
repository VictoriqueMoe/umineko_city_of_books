package controllers

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/google/uuid"

	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/stream"
)

func (s *Service) getAllStreamRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupListLiveStreamsRoute,
		s.setupMyStreamRoute,
		s.setupStartStreamRoute,
		s.setupStopStreamRoute,
		s.setupStreamViewerTokenRoute,
		s.setupJoinStreamChatRoute,
		s.setupUploadStreamThumbnailRoute,
		s.setupStreamCredentialsRoute,
		s.setupResetStreamCredentialsRoute,
		s.setupGetStreamRoute,
	}
}

func (s *Service) setupListLiveStreamsRoute(r fiber.Router) {
	r.Get("/streams/live", s.listLiveStreams)
}

func (s *Service) setupMyStreamRoute(r fiber.Router) {
	r.Get("/streams/mine", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.myStream)
}

func (s *Service) setupStartStreamRoute(r fiber.Router) {
	r.Post("/streams", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.startStream)
}

func (s *Service) setupStopStreamRoute(r fiber.Router) {
	r.Delete("/streams/:id", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.stopStream)
}

func (s *Service) setupStreamViewerTokenRoute(r fiber.Router) {
	r.Post("/streams/:id/token", middleware.OptionalAuth(s.AuthSession, s.AuthzService), limiter.New(limiter.Config{
		Max:        30,
		Expiration: time.Minute,
	}), s.mintViewerToken)
}

func (s *Service) setupJoinStreamChatRoute(r fiber.Router) {
	r.Post("/streams/:id/join-chat", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.joinStreamChat)
}

func (s *Service) setupUploadStreamThumbnailRoute(r fiber.Router) {
	r.Post("/streams/:id/thumbnail", middleware.RequireAuth(s.AuthSession, s.AuthzService), limiter.New(limiter.Config{
		Max:        6,
		Expiration: time.Minute,
	}), s.uploadStreamThumbnail)
}

func (s *Service) setupStreamCredentialsRoute(r fiber.Router) {
	r.Get("/streams/credentials", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.streamCredentials)
}

func (s *Service) setupResetStreamCredentialsRoute(r fiber.Router) {
	r.Post("/streams/credentials/reset", middleware.RequireAuth(s.AuthSession, s.AuthzService), limiter.New(limiter.Config{
		Max:        5,
		Expiration: time.Minute,
	}), s.resetStreamCredentials)
}

func (s *Service) setupGetStreamRoute(r fiber.Router) {
	r.Get("/streams/:id", s.getStream)
}

func (s *Service) startStream(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.StartStreamRequest](ctx)
	if !ok {
		return nil
	}

	resp, err := s.StreamService.StartStream(ctx.Context(), userID, req.Title)
	if err != nil {
		return mapStreamError(ctx, err)
	}

	return ctx.JSON(resp)
}

func (s *Service) stopStream(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	streamID, ok := utils.ParseIDParam(ctx, "id")
	if !ok {
		return nil
	}

	if err := s.StreamService.StopStream(ctx.Context(), userID, streamID); err != nil {
		return mapStreamError(ctx, err)
	}

	return utils.OK(ctx)
}

func (s *Service) myStream(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	resp, err := s.StreamService.MyStream(ctx.Context(), userID)
	if err != nil {
		return mapStreamError(ctx, err)
	}

	return ctx.JSON(resp)
}

func (s *Service) listLiveStreams(ctx fiber.Ctx) error {
	streams, err := s.StreamService.ListLive(ctx.Context())
	if err != nil {
		return mapStreamError(ctx, err)
	}

	return ctx.JSON(dto.LiveStreamListResponse{
		Streams: streams,
		Enabled: s.StreamService.Enabled(),
	})
}

func (s *Service) getStream(ctx fiber.Ctx) error {
	streamID, ok := utils.ParseIDParam(ctx, "id")
	if !ok {
		return nil
	}

	streamView, err := s.StreamService.Get(ctx.Context(), streamID)
	if err != nil {
		return mapStreamError(ctx, err)
	}

	return ctx.JSON(streamView)
}

func (s *Service) mintViewerToken(ctx fiber.Ctx) error {
	streamID, ok := utils.ParseIDParam(ctx, "id")
	if !ok {
		return nil
	}

	var viewer *dto.StreamViewer
	if userID := utils.UserID(ctx); userID != uuid.Nil {
		if u, err := s.UserService.GetByID(ctx.Context(), userID); err == nil && u != nil {
			viewer = &dto.StreamViewer{
				UserID:      u.ID,
				DisplayName: u.DisplayName,
				Username:    u.Username,
				AvatarURL:   u.AvatarURL,
			}
		}
	}

	token, url, err := s.StreamService.MintViewerToken(ctx.Context(), streamID, viewer)
	if err != nil {
		return mapStreamError(ctx, err)
	}

	return ctx.JSON(dto.StreamViewerTokenResponse{Token: token, URL: url})
}

func (s *Service) uploadStreamThumbnail(ctx fiber.Ctx) error {
	streamID, ok := utils.ParseIDParam(ctx, "id")
	if !ok {
		return nil
	}

	file, err := ctx.FormFile("thumbnail")
	if err != nil {
		return utils.BadRequest(ctx, "thumbnail file is required")
	}

	src, err := file.Open()
	if err != nil {
		return utils.BadRequest(ctx, "failed to read file")
	}
	defer src.Close()

	if err := s.StreamService.SaveThumbnail(ctx.Context(), streamID, file.Size, src); err != nil {
		return mapStreamError(ctx, err)
	}

	return utils.OK(ctx)
}

func (s *Service) joinStreamChat(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	streamID, ok := utils.ParseIDParam(ctx, "id")
	if !ok {
		return nil
	}

	if err := s.StreamService.JoinChat(ctx.Context(), streamID, userID); err != nil {
		return mapStreamError(ctx, err)
	}

	return utils.OK(ctx)
}

func (s *Service) streamCredentials(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	resp, err := s.StreamService.Credentials(ctx.Context(), userID, s.streamerDisplayName(ctx, userID))
	if err != nil {
		return mapStreamError(ctx, err)
	}

	return ctx.JSON(resp)
}

func (s *Service) resetStreamCredentials(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)

	resp, err := s.StreamService.ResetCredentials(ctx.Context(), userID, s.streamerDisplayName(ctx, userID))
	if err != nil {
		return mapStreamError(ctx, err)
	}

	return ctx.JSON(resp)
}

func (s *Service) streamerDisplayName(ctx fiber.Ctx, userID uuid.UUID) string {
	u, err := s.UserService.GetByID(ctx.Context(), userID)
	if err != nil || u == nil {
		return ""
	}

	if u.DisplayName != "" {
		return u.DisplayName
	}

	return u.Username
}

func mapStreamError(ctx fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, stream.ErrDisabled):
		{
			return utils.ServiceUnavailable(ctx, "streaming is not configured")
		}
	case errors.Is(err, stream.ErrAtCapacity):
		{
			return utils.Conflict(ctx, "the maximum number of concurrent streams has been reached")
		}
	case errors.Is(err, stream.ErrAlreadyLive):
		{
			return utils.Conflict(ctx, "you already have an active stream")
		}
	case errors.Is(err, stream.ErrTitleRequired):
		{
			return utils.BadRequest(ctx, "a stream title is required")
		}
	case errors.Is(err, stream.ErrStreamNotFound):
		{
			return utils.NotFound(ctx, "stream not found")
		}
	case errors.Is(err, stream.ErrNotOwner):
		{
			return utils.Forbidden(ctx, "you do not own this stream")
		}
	}
	return utils.InternalError(ctx, "stream request failed: "+err.Error(), err)
}
