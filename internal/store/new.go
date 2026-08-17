package store

import (
	"database/sql"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/repository"
)

func New(db *sql.DB, c *cache.Manager) *repository.Repositories {
	repos := repository.NewRepositories(db)

	repos.Session = repository.NewSessionRepo(dao.NewSession(db))
	repos.Notification = repository.NewNotificationRepo(dao.NewNotification(db))
	repos.Role = repository.NewRoleRepo(dao.NewRole(db), c)
	repos.Settings = repository.NewSettingsRepo(db, dao.NewSettings(db))
	repos.AuditLog = repository.NewAuditLogRepo(dao.NewAuditLog(db))
	repos.Theory = repository.NewTheoryRepo(db, dao.NewTheory(db), repos.AuditLog)
	repos.Stats = repository.NewStatsRepo(dao.NewStats(db))
	repos.Invite = repository.NewInviteRepo(dao.NewInvite(db))
	repos.PasswordReset = repository.NewPasswordResetRepo(db, dao.NewPasswordReset(db))
	repos.EmailVerification = repository.NewEmailVerificationRepo(db, dao.NewEmailVerification(db))
	repos.User = repository.NewUserRepo(db, dao.NewUser(db), c, repos.Role, repos.AuditLog, repos.EmailVerification, repos.Invite, repos.Session, repos.PasswordReset)
	repos.Chat = repository.NewChatRepo(db, dao.NewChat(db), repos.AuditLog)
	repos.Report = repository.NewReportRepo(dao.NewReport(db))
	repos.Post = repository.NewPostRepo(db, dao.NewPost(db), repos.AuditLog)
	repos.Follow = repository.NewFollowRepo(dao.NewFollow(db))
	repos.Art = repository.NewArtRepo(db, dao.NewArt(db), repos.Post, repos.AuditLog)
	repos.Upload = repository.NewUploadRepo(dao.NewUpload(db))
	repos.Block = repository.NewBlockRepo(dao.NewBlock(db))
	repos.Announcement = repository.NewAnnouncementRepo(db, dao.NewAnnouncement(db), repos.AuditLog)
	repos.Mystery = repository.NewMysteryRepo(db, dao.NewMystery(db), repos.AuditLog, c)
	repos.Ship = repository.NewShipRepo(db, dao.NewShip(db), repos.AuditLog)
	repos.OC = repository.NewOCRepo(db, dao.NewOC(db), repos.AuditLog)
	repos.Fanfic = repository.NewFanficRepo(db, dao.NewFanfic(db), repos.AuditLog)
	repos.Journal = repository.NewJournalRepo(db, dao.NewJournal(db), repos.AuditLog)
	repos.VanityRole = repository.NewVanityRoleRepo(db, dao.NewVanityRole(db), c)
	repos.Permission = repository.NewPermissionRepo(dao.NewPermission(db), c)
	repos.GiphyFavourite = repository.NewGiphyFavouriteRepo(dao.NewGiphyFavourite(db))
	repos.BannedGiphy = repository.NewBannedGiphyRepo(dao.NewBannedGiphy(db))
	repos.UserSecret = repository.NewUserSecretRepo(dao.NewUserSecret(db), c)
	repos.Secret = repository.NewSecretRepo(db, dao.NewSecret(db), repos.AuditLog)
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

	return repos
}
