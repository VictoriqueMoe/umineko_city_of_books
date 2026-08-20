package controllers

import (
	"context"
	"strings"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

func (s *Service) getAllUploadRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupUploads,
	}
}

func (s *Service) setupUploads(r fiber.Router) {
	r.Get("/uploads/*", static.New(s.UploadService.GetUploadDir(), static.Config{
		Browse:    false,
		ByteRange: true,
	}))
}

func (s *Service) getAllHLSRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupHLS,
	}
}

func (s *Service) setupHLS(r fiber.Router) {
	r.Get("/hls/*", static.New(s.SettingsService.Get(context.Background(), config.SettingStreamHLSOutputDir), static.Config{
		Browse:    false,
		ByteRange: true,
	}))
}

func (s *Service) getAllRobotsRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupRobots,
	}
}

func (s *Service) setupRobots(r fiber.Router) {
	r.Get("/robots.txt", func(ctx fiber.Ctx) error {
		ctx.Type("txt")

		if s.SettingsService.GetBool(ctx.Context(), config.SettingPrivateMode) {
			return ctx.SendString("User-agent: *\nDisallow: /\n")
		}

		base := strings.TrimSuffix(s.SettingsService.Get(ctx.Context(), config.SettingBaseURL), "/")

		return ctx.SendString("User-agent: *\nAllow: /\n\nSitemap: " + base + "/sitemap.xml\n")
	})
}

func (s *Service) getAllSPARoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupSPA,
	}
}

func (s *Service) setupSPA(r fiber.Router) {
	embeddedStaticHandler := static.New("", static.Config{
		FS: s.StaticFS,
	})

	r.Get("/*", func(ctx fiber.Ctx) error {
		path := ctx.Path()
		if strings.Contains(path, ".") {
			return embeddedStaticHandler(ctx)
		}

		if middleware.PrivateGated(ctx) {
			path = "/"
		}

		html := s.OGResolver.Resolve(ctx.Context(), path, ctx.Query("party"))
		return ctx.Type("html").SendString(html)
	})
}
