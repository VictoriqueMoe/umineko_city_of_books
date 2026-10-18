package store

import (
	"database/sql"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dao/dynamicsql"
	"umineko_city_of_books/internal/repository"
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

	comments := dao.NewComments(db)
	repos.Comment = repository.NewCommentRepo(db, comments, repos.AuditLog)
	repos.Post = repository.NewPostRepo(db, dao.NewPost(db), comments, repos.AuditLog, c)

	repos.Follow = repository.NewFollowRepo(dao.NewFollow(db))

	repos.Art = repository.NewArtRepo(db, dao.NewArt(db), comments, repos.Post, repos.AuditLog, c)

	repos.Upload = repository.NewUploadRepo(dynamicsql.NewUpload(db))
	repos.Block = repository.NewBlockRepo(dao.NewBlock(db), c)

	repos.Announcement = repository.NewAnnouncementRepo(db, dao.NewAnnouncement(db), comments)
	repos.Mystery = repository.NewMysteryRepo(db, dao.NewMystery(db), comments, repos.AuditLog, c)
	repos.Ship = repository.NewShipRepo(db, dao.NewShip(db), comments)
	repos.OC = repository.NewOCRepo(db, dao.NewOC(db), comments)
	repos.Fanfic = repository.NewFanficRepo(db, dao.NewFanfic(db), comments, repos.AuditLog)
	repos.Journal = repository.NewJournalRepo(db, dao.NewJournal(db), comments)

	repos.VanityRole = repository.NewVanityRoleRepo(db, dao.NewVanityRole(db), c)
	repos.Permission = repository.NewPermissionRepo(dao.NewPermission(db), c)
	repos.GiphyFavourite = repository.NewGiphyFavouriteRepo(dao.NewGiphyFavourite(db))
	repos.BannedGiphy = repository.NewBannedGiphyRepo(dao.NewBannedGiphy(db))
	repos.StoredFile = repository.NewStoredFileRepo(dao.NewStoredFile(db))
	repos.UserSecret = repository.NewUserSecretRepo(dao.NewUserSecret(db), c)
	repos.Secret = repository.NewSecretRepo(dao.NewSecret(db))

	repos.ChatRoomBan = repository.NewChatRoomBanRepo(db, dao.NewChatRoomBan(db), repos.AuditLog)
	repos.ChatBannedWord = repository.NewChatBannedWordRepo(db, dao.NewChatBannedWord(db), repos.AuditLog, c)
	repos.ChatWatchParty = repository.NewChatWatchPartyRepo(db, dao.NewChatWatchParty(db), repos.Chat, repos.AuditLog)
	repos.LiveStream = repository.NewLiveStreamRepo(db, dao.NewLiveStream(db))
	repos.StreamCredentials = repository.NewStreamCredentialsRepo(dao.NewStreamCredentials(db))
	repos.GameRoom = repository.NewGameRoomRepo(db, dao.NewGameRoom(db), c)
	repos.HomeFeed = repository.NewHomeFeedRepo(dao.NewHomeFeed(db))
	repos.SidebarVisited = repository.NewSidebarLastVisitedRepo(dao.NewSidebarVisited(db))
	repos.Search = repository.NewSearchRepo(dynamicsql.NewSearch(db))
	repos.Sitemap = repository.NewSitemapRepo(dao.NewSitemap(db))
	repos.DeviceToken = repository.NewDeviceTokenRepo(dao.NewDeviceToken(db))
	repos.OverlayToken = repository.NewOverlayTokenRepo(dao.NewOverlayToken(db))
	basePrompts := repository.NewChatbotBasePromptRepo(dao.NewChatbotBasePrompt(db), c)
	repos.ChatbotBasePrompt = basePrompts
	repos.Chatbot = repository.NewChatbotRepo(db, dao.NewChatbot(db), repos.User, repos.VanityRole, basePrompts, c)

	return repos
}
