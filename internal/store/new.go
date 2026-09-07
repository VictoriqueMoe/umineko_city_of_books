package store

import (
	"database/sql"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

func New(db *sql.DB, c *cache.Manager) *repository.Repositories {
	repos := repository.NewRepositories(db)

	repos.Session = repository.NewSessionRepo(dao.NewSession(db))
	repos.Notification = repository.NewNotificationRepo(dao.NewNotification(db))
	repos.Role = repository.NewRoleRepo(dao.NewRole(db), c)
	repos.Settings = repository.NewSettingsRepo(db, dao.NewSettings(db), c)
	repos.AuditLog = repository.NewAuditLogRepo(dao.NewAuditLog(db))
	repos.Theory = repository.NewTheoryRepo(db, dao.NewTheory(db), repos.AuditLog)
	repos.Stats = repository.NewStatsRepo(dao.NewStats(db))
	repos.Invite = repository.NewInviteRepo(dao.NewInvite(db))
	repos.PasswordReset = repository.NewPasswordResetRepo(db, dao.NewPasswordReset(db))
	repos.EmailVerification = repository.NewEmailVerificationRepo(db, dao.NewEmailVerification(db))
	repos.User = repository.NewUserRepo(db, dao.NewUser(db), c, repos.Role, repos.AuditLog, repos.EmailVerification, repos.Invite, repos.Session, repos.PasswordReset)
	repos.Chat = repository.NewChatRepo(db, dao.NewChat(db), repos.AuditLog)
	repos.Report = repository.NewReportRepo(dao.NewReport(db))
	postDAO, postComments := dao.NewPost(db)
	repos.Post = repository.NewPostRepo(db, postDAO, repos.AuditLog)

	repos.Follow = repository.NewFollowRepo(dao.NewFollow(db))

	artDAO, artComments := dao.NewArt(db)
	repos.Art = repository.NewArtRepo(db, artDAO, repos.Post, repos.AuditLog)

	repos.Upload = repository.NewUploadRepo(dao.NewUpload(db))
	repos.Block = repository.NewBlockRepo(dao.NewBlock(db))

	announcementDAO, announcementComments := dao.NewAnnouncement(db)
	repos.Announcement = repository.NewAnnouncementRepo(db, announcementDAO, repos.AuditLog)

	mysteryDAO, mysteryComments := dao.NewMystery(db)
	repos.Mystery = repository.NewMysteryRepo(db, mysteryDAO, repos.AuditLog, c)

	shipDAO, shipComments := dao.NewShip(db)
	repos.Ship = repository.NewShipRepo(db, shipDAO, repos.AuditLog)

	ocDAO, ocComments := dao.NewOC(db)
	repos.OC = repository.NewOCRepo(db, ocDAO, repos.AuditLog)

	fanficDAO, fanficComments := dao.NewFanfic(db)
	repos.Fanfic = repository.NewFanficRepo(db, fanficDAO, repos.AuditLog)

	journalRepo, journalComments := repository.NewJournalRepo(db, dao.NewJournal(db), repos.AuditLog)
	repos.Journal = journalRepo

	repos.VanityRole = repository.NewVanityRoleRepo(db, dao.NewVanityRole(db), c)
	repos.Permission = repository.NewPermissionRepo(dao.NewPermission(db), c)
	repos.GiphyFavourite = repository.NewGiphyFavouriteRepo(dao.NewGiphyFavourite(db))
	repos.BannedGiphy = repository.NewBannedGiphyRepo(dao.NewBannedGiphy(db))
	repos.UserSecret = repository.NewUserSecretRepo(dao.NewUserSecret(db), c)
	secretDAO, secretComments := dao.NewSecret(db)
	repos.Secret = repository.NewSecretRepo(db, secretDAO, repos.AuditLog)

	repos.ChatRoomBan = repository.NewChatRoomBanRepo(db, dao.NewChatRoomBan(db), repos.AuditLog)
	repos.ChatBannedWord = repository.NewChatBannedWordRepo(db, dao.NewChatBannedWord(db), repos.AuditLog)
	repos.ChatWatchParty = repository.NewChatWatchPartyRepo(db, dao.NewChatWatchParty(db), repos.Chat, repos.AuditLog)
	repos.LiveStream = repository.NewLiveStreamRepo(db, dao.NewLiveStream(db))
	repos.StreamCredentials = repository.NewStreamCredentialsRepo(dao.NewStreamCredentials(db))
	repos.GameRoom = repository.NewGameRoomRepo(db, dao.NewGameRoom(db), c)
	repos.HomeFeed = repository.NewHomeFeedRepo(dao.NewHomeFeed(db))
	repos.SidebarVisited = repository.NewSidebarLastVisitedRepo(dao.NewSidebarVisited(db))
	repos.Search = repository.NewSearchRepo(dao.NewSearch(db))
	repos.Sitemap = repository.NewSitemapRepo(dao.NewSitemap(db))
	repos.DeviceToken = repository.NewDeviceTokenRepo(dao.NewDeviceToken(db))
	repos.OverlayToken = repository.NewOverlayTokenRepo(dao.NewOverlayToken(db))
	basePrompts := repository.NewChatbotBasePromptRepo(dao.NewChatbotBasePrompt(db), c)
	repos.ChatbotBasePrompt = basePrompts
	repos.Chatbot = repository.NewChatbotRepo(db, dao.NewChatbot(db), repos.User, repos.VanityRole, basePrompts, c)

	repos.Comments = dao.CommentDAOs{
		ByID: map[string]dao.CommentDAO[uuid.UUID]{
			string(mention.KindPostComment):         postComments,
			string(mention.KindArtComment):          artComments,
			string(mention.KindAnnouncementComment): announcementComments,
			string(mention.KindMysteryComment):      mysteryComments,
			string(mention.KindShipComment):         shipComments,
			string(mention.KindOCComment):           ocComments,
			string(mention.KindFanficComment):       fanficComments,
		},
		BySlug: map[string]dao.CommentDAO[string]{
			string(mention.KindSecretComment): secretComments,
		},
		Journal: journalComments,
	}

	return repos
}
