package main

import (
	"cmp"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"time"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/notification/push"

	"umineko_city_of_books/internal/admin"
	announcementsvc "umineko_city_of_books/internal/announcement"
	artsvc "umineko_city_of_books/internal/art"
	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	blocksvc "umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/chatbot"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/controllers"
	"umineko_city_of_books/internal/email"
	fanficsvc "umineko_city_of_books/internal/fanfic"
	"umineko_city_of_books/internal/follow"
	"umineko_city_of_books/internal/gameroom"
	"umineko_city_of_books/internal/giphy"
	"umineko_city_of_books/internal/giphy/banlist"
	giphyfavourite "umineko_city_of_books/internal/giphy/favourite"
	"umineko_city_of_books/internal/health"
	"umineko_city_of_books/internal/homefeed"
	"umineko_city_of_books/internal/journal"
	"umineko_city_of_books/internal/linkpreview"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/middleware"
	mysterysvc "umineko_city_of_books/internal/mystery"
	"umineko_city_of_books/internal/notification"
	ocsvc "umineko_city_of_books/internal/oc"
	"umineko_city_of_books/internal/og"
	"umineko_city_of_books/internal/openai"
	"umineko_city_of_books/internal/overlay"
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
	"umineko_city_of_books/internal/siteinfo"
	"umineko_city_of_books/internal/sitemap"
	"umineko_city_of_books/internal/stream"
	"umineko_city_of_books/internal/theory"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/user"
	"umineko_city_of_books/internal/usersecret"
	"umineko_city_of_books/internal/vanityrole"
	"umineko_city_of_books/internal/ws"

	"github.com/gofiber/fiber/v3"
)

const (
	drainTimeout = 15 * time.Second
)

var (
	//go:embed static/*
	staticFiles embed.FS
)

type (
	services struct {
		settings        settings.Service
		cache           *cache.Manager
		auth            auth.Service
		profile         profile.Service
		theory          theory.Service
		notification    notification.Service
		admin           admin.Service
		authz           authz.Service
		chat            chat.Service
		openai          openai.Service
		chatbot         chatbot.Service
		chatbotAdmin    chatbot.AdminService
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
		siteInfo        siteinfo.Service
		vanityRole      vanityrole.Service
		userSecret      usersecret.Service
		search          searchsvc.Service
		user            user.Service
		push            push.Service
		stream          stream.Service
		overlay         overlay.Service
		health          health.Service
		sitemap         sitemap.Service
		linkPreview     linkpreview.Service
		ogResolver      *og.Resolver
		ogImage         *og.ImageService
		staticFS        fs.FS
		htmlContent     string
	}
)

func initServer() (*fiber.App, func()) {
	cacheMgr := cache.New()

	repos, settingsSvc := initDatabase(cacheMgr)

	svc := initServices(repos, settingsSvc, cacheMgr)
	app := initApp(svc, repos, settingsSvc)
	stopJobs := registerListeners(settingsSvc, app, svc, repos)

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), drainTimeout)
		defer cancel()

		if err := svc.chatbot.Shutdown(ctx); err != nil {
			logger.Log.Warn().Err(err).Msg("chatbot drain incomplete")
		}

		if err := svc.mediaProc.Shutdown(ctx); err != nil {
			logger.Log.Warn().Err(err).Msg("media processor drain incomplete")
		}

		stopJobs(ctx)

		if err := cacheMgr.Close(); err != nil {
			logger.Log.Warn().Err(err).Msg("valkey cache close error")
		}
	}

	return app, cleanup
}

func initApp(svc *services, repos *repository.Repositories, settingsSvc settings.Service) *fiber.App {
	siteName := settingsSvc.Get(context.Background(), config.SettingSiteName)

	app := fiber.New(fiber.Config{
		ProxyHeader: "CF-Connecting-IP",
		TrustProxy:  true,
		TrustProxyConfig: fiber.TrustProxyConfig{
			Loopback: true,
			Private:  true,
		},
		EnableIPValidation: true,
		AppName:            siteName,
	})

	middleware.Setup(app, settingsSvc, svc.session, svc.authz)
	app.Use(middleware.Metrics())
	app.Get("/metrics", middleware.RequireMetricsToken(settingsSvc), middleware.MetricsHandler())
	registerPprofRoutes(app, svc.session, svc.authz)

	lastSeenIP := middleware.NewLastSeenIP(repos.User, time.Hour)
	app.Use(middleware.RecordLastSeenIP(lastSeenIP))

	ctrlService := controllers.Service{
		AuthService:           svc.auth,
		ProfileService:        svc.profile,
		TheoryService:         svc.theory,
		NotificationService:   svc.notification,
		AdminService:          svc.admin,
		AuthzService:          svc.authz,
		SettingsService:       settingsSvc,
		ChatService:           svc.chat,
		ChatbotAdminService:   svc.chatbotAdmin,
		ChatbotService:        svc.chatbot,
		ReportService:         svc.report,
		PostService:           svc.post,
		FollowService:         svc.follow,
		ArtService:            svc.art,
		BlockService:          svc.block,
		AnnouncementService:   svc.announcement,
		MysteryService:        svc.mystery,
		UserService:           svc.user,
		ShipService:           svc.ship,
		OCService:             svc.oc,
		FanficService:         svc.fanfic,
		JournalService:        svc.journal,
		SecretService:         svc.secret,
		UploadService:         svc.upload,
		MediaProcessor:        svc.mediaProc,
		VanityRoleService:     svc.vanityRole,
		UserSecretService:     svc.userSecret,
		AuthSession:           svc.session,
		Hub:                   svc.hub,
		GiphyService:          svc.giphy,
		GiphyFavouriteService: svc.giphyFavourites,
		GameRoomService:       svc.gameRoom,
		HomeFeedService:       svc.homeFeed,
		SidebarService:        svc.sidebar,
		SearchService:         svc.search,
		PushService:           svc.push,
		StreamService:         svc.stream,
		HealthService:         svc.health,
		OverlayService:        svc.overlay,
		SitemapService:        svc.sitemap,
		LinkPreviewService:    svc.linkPreview,
		SiteInfoService:       svc.siteInfo,
		OGResolver:            svc.ogResolver,
		OGImageService:        svc.ogImage,
		StaticFS:              svc.staticFS,
		HTMLContent:           svc.htmlContent,
	}
	routes.PublicRoutes(ctrlService, app)

	logRoutes(app)

	return app
}

func logRoutes(app *fiber.App) {
	rs := app.GetRoutes(true)

	if logger.Log.Debug().Enabled() {
		slices.SortFunc(rs, func(a, b fiber.Route) int {
			if pathCmp := cmp.Compare(a.Path, b.Path); pathCmp != 0 {
				return pathCmp
			}
			return cmp.Compare(a.Method, b.Method)
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
