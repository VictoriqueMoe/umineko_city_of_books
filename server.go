package main

import (
	"context"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"umineko_city_of_books/internal/admin"
	artsvc "umineko_city_of_books/internal/art"
	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	blocksvc "umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/controllers"
	"umineko_city_of_books/internal/credibility"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/email"
	fanficsvc "umineko_city_of_books/internal/fanfic"
	"umineko_city_of_books/internal/follow"
	"umineko_city_of_books/internal/journal"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/middleware"
	mysterysvc "umineko_city_of_books/internal/mystery"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/og"
	postsvc "umineko_city_of_books/internal/post"
	"umineko_city_of_books/internal/profile"
	"umineko_city_of_books/internal/quotefinder"
	"umineko_city_of_books/internal/report"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/routes"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/ship"
	"umineko_city_of_books/internal/theory"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/user"
	"umineko_city_of_books/internal/ws"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/valyala/fasthttp"
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
	report       report.Service
	post         postsvc.Service
	follow       follow.Service
	art          artsvc.Service
	ship         ship.Service
	mystery      mysterysvc.Service
	fanfic       fanficsvc.Service
	journal      journal.Service
	block        blocksvc.Service
	email        email.Service
	session      *session.Manager
	upload       upload.Service
	hub          *ws.Hub
	mediaProc    *media.Processor
}

func initServer() *fiber.App {
	repos, settingsSvc := initDatabase()
	svc := initServices(repos, settingsSvc)
	app := initApp(svc, repos, settingsSvc)
	registerListeners(settingsSvc, app, svc, repos)
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
	if err := os.MkdirAll(filepath.Join(uploadDir, "posts"), 0755); err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to create posts directory")
	}
	if err := os.MkdirAll(filepath.Join(uploadDir, "art"), 0755); err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to create art directory")
	}

	sessionMgr := session.NewManager(repos.Session, settingsSvc)
	uploadSvc := upload.NewService(settingsSvc)
	authzSvc := authz.NewService(repos.Role, repos.User)
	userSvc := user.NewService(repos.User, repos.Role, authzSvc)
	hub := ws.NewHub()
	quoteClient := quotefinder.NewClient()
	credibilitySvc := credibility.NewService(repos.Theory)

	emailSvc := email.NewService(settingsSvc)
	blockSvc := blocksvc.NewService(repos.Block, repos.Follow, authzSvc)
	notifSvc := notification.NewService(repos.Notification, repos.User, hub, emailSvc)
	reportSvc := report.NewService(repos.Report, repos.Role, repos.User, notifSvc, settingsSvc)
	mediaProc := media.NewProcessor(4)
	chatSvc := chat.NewService(repos.Chat, repos.User, notifSvc, blockSvc, uploadSvc, settingsSvc, mediaProc, hub)
	followSvc := follow.NewService(repos.Follow, repos.User, blockSvc, notifSvc, settingsSvc)
	postSvc := postsvc.NewService(repos.DB(), repos.Post, repos.User, repos.Role, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc, hub)
	artSvc := artsvc.NewService(repos.Art, repos.Post, repos.User, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc)
	shipSvc := ship.NewService(repos.Ship, repos.User, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc, quoteClient)
	mysterySvc := mysterysvc.NewService(repos.Mystery, repos.User, authzSvc, blockSvc, notifSvc, settingsSvc, uploadSvc, mediaProc, hub)
	fanficSvc := fanficsvc.NewService(repos.Fanfic, repos.User, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc)
	journalSvc := journal.NewService(repos.Journal, repos.User, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc)

	return &services{
		settings:     settingsSvc,
		auth:         auth.NewService(userSvc, sessionMgr, settingsSvc, repos.Invite, repos.User),
		profile:      profile.NewService(repos.User, repos.Theory, authzSvc, uploadSvc, settingsSvc),
		theory:       theory.NewService(repos.Theory, repos.User, authzSvc, blockSvc, notifSvc, settingsSvc, credibilitySvc, quoteClient),
		notification: notifSvc,
		admin:        admin.NewService(repos.User, repos.Role, repos.Stats, repos.AuditLog, repos.Invite, repos.VanityRole, authzSvc, settingsSvc, sessionMgr, uploadSvc, hub),
		authz:        authzSvc,
		chat:         chatSvc,
		report:       reportSvc,
		post:         postSvc,
		follow:       followSvc,
		art:          artSvc,
		ship:         shipSvc,
		mystery:      mysterySvc,
		fanfic:       fanficSvc,
		journal:      journalSvc,
		block:        blockSvc,
		email:        emailSvc,
		session:      sessionMgr,
		upload:       uploadSvc,
		hub:          hub,
		mediaProc:    mediaProc,
	}
}

func registerListeners(settingsSvc settings.Service, app *fiber.App, svc *services, repos *repository.Repositories) {
	settingsSvc.Subscribe(logger.NewSettingsListener())
	settingsSvc.Subscribe(middleware.NewBodyLimitListener(app))
	settingsSvc.Subscribe(email.NewMailSettingListener(svc.email))

	logger.Log.Info().Str("interval", "1h").Msg("registered job: refresh stale embeds")
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			n := svc.post.RefreshStaleEmbeds(context.Background())
			if n > 0 {
				logger.Log.Info().Int("count", n).Msg("refreshed stale embeds")
			}
		}
	}()

	logger.Log.Info().Str("interval", "24h").Msg("registered job: clean orphaned uploads")
	go func() {
		uploadDir := svc.upload.GetUploadDir()
		n := upload.CleanOrphanedFiles(repos.Upload, uploadDir)
		if n > 0 {
			logger.Log.Info().Int("count", n).Msg("cleaned orphaned upload files")
		}
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			n := upload.CleanOrphanedFiles(repos.Upload, uploadDir)
			if n > 0 {
				logger.Log.Info().Int("count", n).Msg("cleaned orphaned upload files")
			}
		}
	}()

	logger.Log.Info().Str("interval", "1h").Msg("registered job: archive stale journals")
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			n, err := svc.journal.ArchiveStale(context.Background())
			if err != nil {
				logger.Log.Error().Err(err).Msg("archive stale journals failed")
				continue
			}
			if n > 0 {
				logger.Log.Info().Int("count", n).Msg("archived stale journals")
			}
		}
	}()
}

func initApp(svc *services, repos *repository.Repositories, settingsSvc settings.Service) *fiber.App {
	app := fiber.New(fiber.Config{
		ProxyHeader: "CF-Connecting-IP",
		TrustProxy:  true,
		TrustProxyConfig: fiber.TrustProxyConfig{
			Loopback: true,
			Private:  true,
		},
	})

	middleware.Setup(app, settingsSvc, svc.session, svc.authz)

	htmlBytes, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to read index.html from embedded files")
	}

	ctrlService := controllers.NewService(
		svc.auth, svc.profile, svc.theory, svc.notification, svc.admin,
		svc.authz, settingsSvc, svc.chat, svc.report, svc.post, svc.follow,
		svc.art, svc.block, repos.Announcement, svc.mystery, repos.User, svc.ship, svc.fanfic, svc.journal, svc.upload, svc.mediaProc, repos.VanityRole, svc.session, svc.hub, string(htmlBytes),
	)
	routes.PublicRoutes(ctrlService, app)

	baseURL := settingsSvc.Get(context.Background(), config.SettingBaseURL)
	sitemapHandler := controllers.NewSitemapHandler(repos.DB(), baseURL)
	sitemapHandler.Register(app)

	app.Get("/api/v1/ws", ws.Handler(svc.hub, svc.session, svc.chat))
	app.Get("/uploads/*", func(ctx fiber.Ctx) error {
		filePath := filepath.Join(svc.upload.GetUploadDir(), ctx.Params("*"))
		fasthttp.ServeFile(ctx.RequestCtx(), filePath)
		return nil
	})

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to create static sub-filesystem")
	}

	ogResolver := og.NewResolver(repos.Theory, repos.User, repos.Post, repos.Art, repos.Mystery, repos.Ship, repos.Fanfic, repos.Announcement, repos.Journal, repos.Chat, string(htmlBytes), baseURL)

	app.Get("/*", func(ctx fiber.Ctx) error {
		path := ctx.Path()
		if strings.Contains(path, ".") {
			return static.New("", static.Config{
				FS: staticFS,
			})(ctx)
		}
		html := ogResolver.Resolve(ctx.Context(), path)
		return ctx.Type("html").SendString(html)
	})

	return app
}
