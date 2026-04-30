package main

import (
	"context"
	"os"
	"path/filepath"

	"umineko_city_of_books/internal/admin"
	announcementsvc "umineko_city_of_books/internal/announcement"
	artsvc "umineko_city_of_books/internal/art"
	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	blocksvc "umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	bannedgiphyrule "umineko_city_of_books/internal/contentfilter/rules/bannedgiphy"
	slursrule "umineko_city_of_books/internal/contentfilter/rules/slurs"
	"umineko_city_of_books/internal/credibility"
	"umineko_city_of_books/internal/email"
	fanficsvc "umineko_city_of_books/internal/fanfic"
	"umineko_city_of_books/internal/follow"
	"umineko_city_of_books/internal/game/checkers"
	"umineko_city_of_books/internal/game/chess"
	"umineko_city_of_books/internal/gameroom"
	"umineko_city_of_books/internal/giphy"
	"umineko_city_of_books/internal/giphy/banlist"
	giphyfavourite "umineko_city_of_books/internal/giphy/favourite"
	"umineko_city_of_books/internal/homefeed"
	"umineko_city_of_books/internal/journal"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	mysterysvc "umineko_city_of_books/internal/mystery"
	"umineko_city_of_books/internal/notification"
	ocsvc "umineko_city_of_books/internal/oc"
	postsvc "umineko_city_of_books/internal/post"
	"umineko_city_of_books/internal/profile"
	"umineko_city_of_books/internal/quotefinder"
	"umineko_city_of_books/internal/report"
	"umineko_city_of_books/internal/repository"
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
)

func initServices(repos *repository.Repositories, settingsSvc settings.Service) *services {
	uploadDir := settingsSvc.Get(context.Background(), config.SettingUploadDir)
	for _, sub := range []string{"avatars", "banners", "posts", "art"} {
		if err := os.MkdirAll(filepath.Join(uploadDir, sub), 0755); err != nil {
			logger.Log.Fatal().Err(err).Msgf("failed to create %s directory", sub)
		}
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
	ocSvc := ocsvc.NewService(repos.OC, repos.User, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc, hub, contentFilter)
	mysterySvc := mysterysvc.NewService(repos.Mystery, repos.User, authzSvc, blockSvc, notifSvc, settingsSvc, uploadSvc, mediaProc, hub, contentFilter)
	fanficSvc := fanficsvc.NewService(repos.Fanfic, repos.User, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc, contentFilter)
	journalSvc := journal.NewService(repos.Journal, repos.User, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc, contentFilter)
	secretSvc := secretsvc.NewService(repos.Secret, repos.UserSecret, repos.User, authzSvc, blockSvc, notifSvc, settingsSvc, uploadSvc, mediaProc, hub, contentFilter)
	gameRoomSvc := gameroom.NewService(repos.GameRoom, repos.User, repos.Block, notifSvc, hub, contentFilter, []gameroom.GameHandler{chess.NewHandler(), checkers.NewHandler()})
	announcementUploader := media.NewUploader(uploadSvc, settingsSvc, mediaProc)
	announcementSvc := announcementsvc.NewService(repos.Announcement, repos.User, blockSvc, notifSvc, settingsSvc, authzSvc, hub, announcementUploader)
	homeFeedSvc := homefeed.NewService(repos.HomeFeed, hub)
	sidebarSvc := sidebar.NewService(repos.SidebarVisited)
	vanityRoleSvc := vanityrole.NewService(repos.VanityRole)
	userSecretSvc := usersecret.NewService(repos.UserSecret)
	searchSvc := searchsvc.NewService(repos.Search)

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
		oc:              ocSvc,
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
		announcement:    announcementSvc,
		homeFeed:        homeFeedSvc,
		sidebar:         sidebarSvc,
		vanityRole:      vanityRoleSvc,
		userSecret:      userSecretSvc,
		search:          searchSvc,
		user:            userSvc,
	}
}
