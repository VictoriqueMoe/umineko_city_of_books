package controllers

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"

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
	r.Post("/streams/:id/token", limiter.New(limiter.Config{
		Max:        30,
		Expiration: time.Minute,
	}), s.mintViewerToken)
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

	token, url, err := s.StreamService.MintViewerToken(ctx.Context(), streamID)
	if err != nil {
		return mapStreamError(ctx, err)
	}

	return ctx.JSON(dto.StreamViewerTokenResponse{Token: token, URL: url})
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
