package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"umineko_city_of_books/internal/admin"
	announcementsvc "umineko_city_of_books/internal/announcement"
	artsvc "umineko_city_of_books/internal/art"
	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	blocksvc "umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/controllers"
	"umineko_city_of_books/internal/email"
	fanficsvc "umineko_city_of_books/internal/fanfic"
	"umineko_city_of_books/internal/follow"
	"umineko_city_of_books/internal/gameroom"
	"umineko_city_of_books/internal/giphy"
	"umineko_city_of_books/internal/giphy/banlist"
	giphyfavourite "umineko_city_of_books/internal/giphy/favourite"
	"umineko_city_of_books/internal/homefeed"
	"umineko_city_of_books/internal/journal"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/middleware"
	mysterysvc "umineko_city_of_books/internal/mystery"
	"umineko_city_of_books/internal/notification"
	ocsvc "umineko_city_of_books/internal/oc"
	"umineko_city_of_books/internal/og"
	postsvc "umineko_city_of_books/internal/post"
	"umineko_city_of_books/internal/profile"
	"umineko_city_of_books/internal/report"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/routes"
	searchsvc "umineko_city_of_books/internal/search"
	secretsvc "umineko_city_of_books/internal/secret"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/ship"
	"umineko_city_of_books/internal/sidebar"
	"umineko_city_of_books/internal/theory"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/user"
	"umineko_city_of_books/internal/usersecret"
	"umineko_city_of_books/internal/vanityrole"
	"umineko_city_of_books/internal/ws"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/valyala/fasthttp"
)

var (
	//go:embed static/*
	staticFiles embed.FS
)

type (
	services struct {
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
		oc              ocsvc.Service
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
		announcement    announcementsvc.Service
		homeFeed        homefeed.Service
		sidebar         sidebar.Service
		vanityRole      vanityrole.Service
		userSecret      usersecret.Service
		search          searchsvc.Service
		user            user.Service
	}
)

func initServer() *fiber.App {
	repos, settingsSvc := initDatabase()
	svc := initServices(repos, settingsSvc)
	app := initApp(svc, repos, settingsSvc)
	registerListeners(settingsSvc, app, svc, repos)
	return app
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
		svc.art, svc.block, svc.announcement, svc.mystery, repos.User, svc.user, svc.ship, svc.oc, svc.fanfic, svc.journal, svc.secret, svc.upload, svc.mediaProc, svc.vanityRole, svc.userSecret, svc.session, svc.hub, svc.giphy, svc.giphyFavourites, svc.gameRoom, svc.homeFeed, svc.sidebar, svc.search, string(htmlBytes),
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

	ogResolver := og.NewResolver(repos.Theory, repos.User, repos.Post, repos.Art, repos.Mystery, repos.Ship, repos.OC, repos.Fanfic, repos.Announcement, repos.Journal, repos.Chat, string(htmlBytes), baseURL)

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
