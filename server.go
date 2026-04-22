package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"umineko_city_of_books/internal/admin"
	artsvc "umineko_city_of_books/internal/art"
	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	blocksvc "umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/chess"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	bannedgiphyrule "umineko_city_of_books/internal/contentfilter/rules/bannedgiphy"
	slursrule "umineko_city_of_books/internal/contentfilter/rules/slurs"
	"umineko_city_of_books/internal/controllers"
	"umineko_city_of_books/internal/credibility"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/email"
	fanficsvc "umineko_city_of_books/internal/fanfic"
	"umineko_city_of_books/internal/follow"
	"umineko_city_of_books/internal/gameroom"
	"umineko_city_of_books/internal/giphy"
	"umineko_city_of_books/internal/giphy/banlist"
	giphyfavourite "umineko_city_of_books/internal/giphy/favourite"
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
	secretsvc "umineko_city_of_books/internal/secret"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/ship"
	"umineko_city_of_books/internal/telemetry"
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
	settings        settings.Service
	auth            auth.Service
	profile         profile.Service
	theory          theory.Service
	notification    notification.Service
	admin           admin.Service
	authz           authz.Service
	chat            chat.Service
	report          report.Service
	post            postsvc.Service
	follow          follow.Service
	art             artsvc.Service
	ship            ship.Service
	mystery         mysterysvc.Service
	fanfic          fanficsvc.Service
	journal         journal.Service
	secret          secretsvc.Service
	block           blocksvc.Service
	email           email.Service
	session         *session.Manager
	upload          upload.Service
	hub             *ws.Hub
	mediaProc       *media.Processor
	giphy           giphy.Service
	giphyFavourites giphyfavourite.Service
	giphyBanlist    banlist.Service
	contentFilter   *contentfilter.Manager
	gameRoom        gameroom.Service
}

func initServer() *fiber.App {
	repos, settingsSvc := initDatabase()
	svc := initServices(repos, settingsSvc)
	app := initApp(svc, repos, settingsSvc)
	registerListeners(settingsSvc, app, svc, repos)
	return app
}

func initDatabase() (*repository.Repositories, settings.Service) {
	if err := telemetry.Init(
		context.Background(),
		"umineko-city-of-books",
		config.Version,
		"",
	); err != nil {
		logger.Log.Warn().Err(err).Msg("otel init failed; traces disabled")
	}

	database, err := db.Open(config.Cfg.DBPath)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to open database")
	}

	if err := db.Migrate(database); err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to run migrations")
	}

	if err := db.SeedContent(database); err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to seed content")
	}

	repos := repository.New(database)

	settingsSvc := settings.NewService(repos.Settings)
	if err := settingsSvc.Refresh(context.Background()); err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to load settings")
	}

	logger.Init(settingsSvc.Get(context.Background(), config.SettingLogLevel))
	logger.ApplyDSN(settingsSvc.Get(context.Background(), config.SettingSentryDSN))

	if err := telemetry.Apply(settingsSvc.Get(context.Background(), config.SettingOTLPEndpoint)); err != nil {
		logger.Log.Warn().Err(err).Msg("otel apply failed")
	}

	hostname, _ := os.Hostname()
	if err := telemetry.InitProfiling(
		"umineko-city-of-books",
		hostname,
		settingsSvc.Get(context.Background(), config.SettingPyroscopeURL),
	); err != nil {
		logger.Log.Warn().Err(err).Msg("pyroscope init failed; profiling disabled")
	}

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
	mediaProc := media.NewProcessor(4)
	uploadSvc := upload.NewService(settingsSvc, mediaProc)
	authzSvc := authz.NewService(repos.Role, repos.User)
	giphyBanlist, err := banlist.NewService(context.Background(), repos.BannedGiphy)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to load giphy banlist")
	}
	giphySvc := giphy.NewService(giphyBanlist)
	if !giphySvc.Enabled() {
		logger.Log.Warn().Msg("GIPHY_API_KEY is not set: gif picker is disabled and direct-URL channel bans cannot resolve uploaders")
	}
	contentFilter := contentfilter.New(
		slursrule.New(),
		bannedgiphyrule.New(giphyBanlist, giphySvc),
	)
	userSvc := user.NewService(repos.User, repos.Role, authzSvc)
	hub := ws.NewHub()
	quoteClient := quotefinder.NewClient()
	credibilitySvc := credibility.NewService(repos.Theory)

	emailSvc := email.NewService(settingsSvc)
	blockSvc := blocksvc.NewService(repos.Block, repos.Follow, authzSvc)
	notifSvc := notification.NewService(repos.Notification, repos.User, hub, emailSvc)
	reportSvc := report.NewService(repos.Report, repos.Role, repos.User, notifSvc, settingsSvc)
	chatSvc := chat.NewService(repos.Chat, repos.User, repos.Role, repos.VanityRole, repos.ChatRoomBan, repos.ChatBannedWord, repos.AuditLog, authzSvc, notifSvc, blockSvc, uploadSvc, settingsSvc, mediaProc, hub, contentFilter)
	followSvc := follow.NewService(repos.Follow, repos.User, blockSvc, notifSvc, settingsSvc)
	postSvc := postsvc.NewService(repos.DB(), repos.Post, repos.User, repos.Role, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc, hub, contentFilter)
	artSvc := artsvc.NewService(repos.Art, repos.Post, repos.User, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc, contentFilter)
	shipSvc := ship.NewService(repos.Ship, repos.User, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc, quoteClient, contentFilter)
	mysterySvc := mysterysvc.NewService(repos.Mystery, repos.User, authzSvc, blockSvc, notifSvc, settingsSvc, uploadSvc, mediaProc, hub, contentFilter)
	fanficSvc := fanficsvc.NewService(repos.Fanfic, repos.User, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc, contentFilter)
	journalSvc := journal.NewService(repos.Journal, repos.User, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc, contentFilter)
	secretSvc := secretsvc.NewService(repos.Secret, repos.UserSecret, repos.User, authzSvc, blockSvc, notifSvc, settingsSvc, uploadSvc, mediaProc, hub, contentFilter)
	gameRoomSvc := gameroom.NewService(repos.GameRoom, repos.User, repos.Block, notifSvc, hub, contentFilter, []gameroom.GameHandler{chess.NewHandler()})

	return &services{
		settings:        settingsSvc,
		auth:            auth.NewService(userSvc, sessionMgr, settingsSvc, repos.Invite, repos.User, repos.AuditLog, contentFilter),
		profile:         profile.NewService(repos.User, repos.UserSecret, repos.Theory, authzSvc, uploadSvc, settingsSvc, contentFilter),
		theory:          theory.NewService(repos.Theory, repos.User, authzSvc, blockSvc, notifSvc, settingsSvc, credibilitySvc, quoteClient, contentFilter),
		notification:    notifSvc,
		admin:           admin.NewService(repos.User, repos.Role, repos.Stats, repos.AuditLog, repos.Invite, repos.VanityRole, giphyBanlist, authzSvc, settingsSvc, sessionMgr, uploadSvc, hub, chatSvc),
		authz:           authzSvc,
		chat:            chatSvc,
		report:          reportSvc,
		post:            postSvc,
		follow:          followSvc,
		art:             artSvc,
		ship:            shipSvc,
		mystery:         mysterySvc,
		fanfic:          fanficSvc,
		journal:         journalSvc,
		secret:          secretSvc,
		block:           blockSvc,
		email:           emailSvc,
		session:         sessionMgr,
		upload:          uploadSvc,
		hub:             hub,
		mediaProc:       mediaProc,
		giphy:           giphySvc,
		giphyFavourites: giphyfavourite.NewService(repos.GiphyFavourite),
		giphyBanlist:    giphyBanlist,
		contentFilter:   contentFilter,
		gameRoom:        gameRoomSvc,
	}
}

func registerListeners(settingsSvc settings.Service, app *fiber.App, svc *services, repos *repository.Repositories) {
	settingsSvc.Subscribe(logger.NewSettingsListener())
	settingsSvc.Subscribe(telemetry.NewSettingsListener())
	settingsSvc.Subscribe(telemetry.NewProfilingSettingsListener())
	settingsSvc.Subscribe(middleware.NewBodyLimitListener(app))
	settingsSvc.Subscribe(email.NewMailSettingListener(svc.email))

	if err := svc.chat.EnsureSystemRooms(context.Background()); err != nil {
		logger.Log.Error().Err(err).Msg("ensure system chat rooms at startup")
	}

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
	app.Use(middleware.Metrics())
	app.Get("/metrics", middleware.MetricsHandler())
	registerPprofRoutes(app, svc.session, svc.authz)

	lastSeenIP := middleware.NewLastSeenIP(repos.User, time.Hour)
	app.Use(middleware.RecordLastSeenIP(lastSeenIP))

	htmlBytes, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to read index.html from embedded files")
	}

	ctrlService := controllers.NewService(
		svc.auth, svc.profile, svc.theory, svc.notification, svc.admin,
		svc.authz, settingsSvc, svc.chat, svc.report, svc.post, svc.follow,
		svc.art, svc.block, repos.Announcement, svc.mystery, repos.User, svc.ship, svc.fanfic, svc.journal, svc.secret, svc.upload, svc.mediaProc, repos.VanityRole, repos.UserSecret, svc.session, svc.hub, svc.giphy, svc.giphyFavourites, svc.gameRoom, string(htmlBytes),
	)
	routes.PublicRoutes(ctrlService, app)

	baseURL := settingsSvc.Get(context.Background(), config.SettingBaseURL)
	sitemapHandler := controllers.NewSitemapHandler(repos.DB(), baseURL)
	sitemapHandler.Register(app)

	app.Get("/api/v1/ws", ws.Handler(svc.hub, svc.session, svc.chat, svc.gameRoom, func() string {
		return settingsSvc.Get(context.Background(), config.SettingBaseURL)
	}))
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

	logRoutes(app)

	return app
}

func logRoutes(app *fiber.App) {
	rs := app.GetRoutes(true)

	if logger.Log.Debug().Enabled() {
		sort.Slice(rs, func(i, j int) bool {
			if rs[i].Path == rs[j].Path {
				return rs[i].Method < rs[j].Method
			}
			return rs[i].Path < rs[j].Path
		})

		methodWidth := len("METHOD")
		pathWidth := len("PATH")
		for _, r := range rs {
			if len(r.Method) > methodWidth {
				methodWidth = len(r.Method)
			}
			if len(r.Path) > pathWidth {
				pathWidth = len(r.Path)
			}
		}

		border := "+" + strings.Repeat("-", methodWidth+2) + "+" + strings.Repeat("-", pathWidth+2) + "+"
		var b strings.Builder
		b.WriteString("\n")
		b.WriteString(border + "\n")
		b.WriteString(fmt.Sprintf("| %-*s | %-*s |\n", methodWidth, "METHOD", pathWidth, "PATH"))
		b.WriteString(border + "\n")
		for _, r := range rs {
			b.WriteString(fmt.Sprintf("| %-*s | %-*s |\n", methodWidth, r.Method, pathWidth, r.Path))
		}
		b.WriteString(border)

		logger.Log.Debug().Msgf("registered routes:%s", b.String())
	}

	logger.Log.Info().Msgf("%d routes mounted", len(rs))
}

func registerPprofRoutes(app *fiber.App, sessionMgr *session.Manager, authzSvc authz.Service) {
	gate := middleware.RequirePermission(sessionMgr, authzSvc, authz.PermManageSettings)

	pprofAdapter := func(h http.HandlerFunc) fiber.Handler {
		return func(ctx fiber.Ctx) error {
			handler := http.HandlerFunc(h)
			req, err := http.NewRequest(ctx.Method(), ctx.OriginalURL(), nil)
			if err != nil {
				return err
			}
			rw := &pprofResponseWriter{ctx: ctx, header: http.Header{}}
			handler.ServeHTTP(rw, req)
			return nil
		}
	}

	app.Get("/debug/pprof/", gate, pprofAdapter(pprof.Index))
	app.Get("/debug/pprof/cmdline", gate, pprofAdapter(pprof.Cmdline))
	app.Get("/debug/pprof/profile", gate, pprofAdapter(pprof.Profile))
	app.Get("/debug/pprof/symbol", gate, pprofAdapter(pprof.Symbol))
	app.Get("/debug/pprof/trace", gate, pprofAdapter(pprof.Trace))
	app.Get("/debug/pprof/allocs", gate, pprofAdapter(pprof.Handler("allocs").ServeHTTP))
	app.Get("/debug/pprof/block", gate, pprofAdapter(pprof.Handler("block").ServeHTTP))
	app.Get("/debug/pprof/goroutine", gate, pprofAdapter(pprof.Handler("goroutine").ServeHTTP))
	app.Get("/debug/pprof/heap", gate, pprofAdapter(pprof.Handler("heap").ServeHTTP))
	app.Get("/debug/pprof/mutex", gate, pprofAdapter(pprof.Handler("mutex").ServeHTTP))
	app.Get("/debug/pprof/threadcreate", gate, pprofAdapter(pprof.Handler("threadcreate").ServeHTTP))
}

type pprofResponseWriter struct {
	ctx    fiber.Ctx
	header http.Header
	wrote  bool
}

func (w *pprofResponseWriter) Header() http.Header {
	return w.header
}

func (w *pprofResponseWriter) WriteHeader(status int) {
	if w.wrote {
		return
	}
	w.wrote = true
	for k, v := range w.header {
		if len(v) > 0 {
			w.ctx.Set(k, v[0])
		}
	}
	w.ctx.Status(status)
}

func (w *pprofResponseWriter) Write(p []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	_, err := w.ctx.Write(p)
	return len(p), err
}
