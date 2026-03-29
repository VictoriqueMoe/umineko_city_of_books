package main

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/controllers"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/profile"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/routes"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/theory"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/user"
	"umineko_city_of_books/internal/ws"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

var (
	//go:embed static/*
	staticFiles embed.FS
)

func initServer() *fiber.App {
	if err := os.MkdirAll(filepath.Join(config.Cfg.UploadDir, "avatars"), 0755); err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to create uploads directory")
	}
	if err := os.MkdirAll(filepath.Join(config.Cfg.UploadDir, "banners"), 0755); err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to create banners directory")
	}

	database, err := db.Open(config.Cfg.DBPath)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to open database")
	}

	if err := db.Migrate(database); err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to run migrations")
	}

	repos := repository.New(database)

	sessionMgr := session.NewManager(repos.Session)
	uploadService := upload.NewService()
	authzService := authz.NewService(repos.Role)
	userService := user.NewService(repos.User, repos.Role, authzService)
	authService := auth.NewService(userService, sessionMgr)
	profileService := profile.NewService(repos.User, repos.Theory, authzService, uploadService)
	hub := ws.NewHub()
	notifService := notification.NewService(repos.Notification, repos.Theory, hub)
	theoryService := theory.NewService(repos.Theory, authzService, notifService)

	app := fiber.New(fiber.Config{
		BodyLimit: config.Cfg.MaxBodySize,
	})

	middleware.Setup(app, config.Cfg.BaseURL)

	htmlBytes, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to read index.html from embedded files")
	}

	service := controllers.NewService(authService, profileService, theoryService, notifService, sessionMgr, hub, string(htmlBytes))
	routes.PublicRoutes(service, app)

	app.Get("/api/v1/ws", ws.Handler(hub, sessionMgr))
	app.Get("/uploads/*", static.New(uploadService.GetUploadDir()))

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to create static sub-filesystem")
	}

	app.Get("/*", func(ctx fiber.Ctx) error {
		path := ctx.Path()
		if strings.Contains(path, ".") {
			return static.New("", static.Config{
				FS: staticFS,
			})(ctx)
		}
		return ctx.Type("html").Send(htmlBytes)
	})

	return app
}
