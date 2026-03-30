package main

import (
	"context"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"umineko_city_of_books/internal/admin"
	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/controllers"
	"umineko_city_of_books/internal/credibility"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/profile"
	"umineko_city_of_books/internal/quotefinder"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/routes"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"
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

type services struct {
	settings     settings.Service
	auth         auth.Service
	profile      profile.Service
	theory       theory.Service
	notification notification.Service
	admin        admin.Service
	authz        authz.Service
	chat         chat.Service
	session      *session.Manager
	upload       upload.Service
	hub          *ws.Hub
}

func initServer() *fiber.App {
	repos, settingsSvc := initDatabase()
	svc := initServices(repos, settingsSvc)
	app := initApp(svc, settingsSvc)
	registerListeners(settingsSvc, app)
	return app
}

func initDatabase() (*repository.Repositories, settings.Service) {
	database, err := db.Open(config.Cfg.DBPath)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to open database")
	}

	if err := db.Migrate(database); err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to run migrations")
	}

	repos := repository.New(database)

	settingsSvc := settings.NewService(repos.Settings)
	if err := settingsSvc.Refresh(context.Background()); err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to load settings")
	}

	logger.Init(settingsSvc.Get(context.Background(), config.SettingLogLevel))

	return repos, settingsSvc
}

func initServices(repos *repository.Repositories, settingsSvc settings.Service) *services {
	uploadDir := settingsSvc.Get(context.Background(), config.SettingUploadDir)
	if err := os.MkdirAll(filepath.Join(uploadDir, "avatars"), 0755); err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to create uploads directory")
	}
	if err := os.MkdirAll(filepath.Join(uploadDir, "banners"), 0755); err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to create banners directory")
	}

	sessionMgr := session.NewManager(repos.Session, settingsSvc)
	uploadSvc := upload.NewService(settingsSvc)
	authzSvc := authz.NewService(repos.Role, repos.User)
	userSvc := user.NewService(repos.User, repos.Role, authzSvc)
	hub := ws.NewHub()
	quoteClient := quotefinder.NewClient()
	credibilitySvc := credibility.NewService(repos.Theory)

	chatSvc := chat.NewService(repos.Chat, repos.User, repos.Notification, hub)

	return &services{
		settings:     settingsSvc,
		auth:         auth.NewService(userSvc, sessionMgr, settingsSvc, repos.Invite),
		profile:      profile.NewService(repos.User, repos.Theory, authzSvc, uploadSvc, settingsSvc),
		theory:       theory.NewService(repos.Theory, authzSvc, notification.NewService(repos.Notification, repos.Theory, hub), settingsSvc, credibilitySvc, quoteClient),
		notification: notification.NewService(repos.Notification, repos.Theory, hub),
		admin:        admin.NewService(repos.User, repos.Role, repos.Stats, repos.AuditLog, repos.Invite, authzSvc, settingsSvc),
		authz:        authzSvc,
		chat:         chatSvc,
		session:      sessionMgr,
		upload:       uploadSvc,
		hub:          hub,
	}
}

func registerListeners(settingsSvc settings.Service, app *fiber.App) {
	settingsSvc.Subscribe(logger.NewSettingsListener())
	settingsSvc.Subscribe(middleware.NewBodyLimitListener(app))
}

func initApp(svc *services, settingsSvc settings.Service) *fiber.App {
	app := fiber.New()

	middleware.Setup(app, settingsSvc, svc.session, svc.authz)

	htmlBytes, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to read index.html from embedded files")
	}

	ctrlService := controllers.NewService(
		svc.auth, svc.profile, svc.theory, svc.notification, svc.admin,
		svc.authz, settingsSvc, svc.chat, svc.session, svc.hub, string(htmlBytes),
	)
	routes.PublicRoutes(ctrlService, app)

	app.Get("/api/v1/ws", ws.Handler(svc.hub, svc.session, svc.chat))
	app.Get("/uploads/*", static.New(svc.upload.GetUploadDir()))

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
