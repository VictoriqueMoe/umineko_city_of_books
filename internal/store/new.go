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
	repos.User = repository.NewUserRepo(dao.NewUser(db), c)
	repos.Theory = repository.NewTheoryRepo(dao.NewTheory(db))
	repos.Notification = repository.NewNotificationRepo(dao.NewNotification(db))
	repos.Role = repository.NewRoleRepo(dao.NewRole(db), c)
	repos.Settings = repository.NewSettingsRepo(dao.NewSettings(db))
	repos.AuditLog = repository.NewAuditLogRepo(dao.NewAuditLog(db))
	repos.Stats = repository.NewStatsRepo(dao.NewStats(db))
	repos.Invite = repository.NewInviteRepo(dao.NewInvite(db))
	repos.PasswordReset = repository.NewPasswordResetRepo(dao.NewPasswordReset(db))
	repos.EmailVerification = repository.NewEmailVerificationRepo(dao.NewEmailVerification(db))
	repos.Chat = repository.NewChatRepo(dao.NewChat(db))
	repos.Report = repository.NewReportRepo(dao.NewReport(db))
	repos.Post = repository.NewPostRepo(dao.NewPost(db))
	repos.Follow = repository.NewFollowRepo(dao.NewFollow(db))
	repos.Art = repository.NewArtRepo(dao.NewArt(db))
	repos.Upload = repository.NewUploadRepo(dao.NewUpload(db))
	repos.Block = repository.NewBlockRepo(dao.NewBlock(db))
	repos.Announcement = repository.NewAnnouncementRepo(dao.NewAnnouncement(db))
	repos.Mystery = repository.NewMysteryRepo(dao.NewMystery(db), c)
	repos.Ship = repository.NewShipRepo(dao.NewShip(db))
	repos.OC = repository.NewOCRepo(dao.NewOC(db))
	repos.Fanfic = repository.NewFanficRepo(dao.NewFanfic(db))
	repos.Journal = repository.NewJournalRepo(dao.NewJournal(db))
	repos.VanityRole = repository.NewVanityRoleRepo(dao.NewVanityRole(db), c)
	repos.GiphyFavourite = repository.NewGiphyFavouriteRepo(dao.NewGiphyFavourite(db))
	repos.BannedGiphy = repository.NewBannedGiphyRepo(dao.NewBannedGiphy(db))
	repos.UserSecret = repository.NewUserSecretRepo(dao.NewUserSecret(db), c)
	repos.Secret = repository.NewSecretRepo(dao.NewSecret(db))
	repos.ChatRoomBan = repository.NewChatRoomBanRepo(dao.NewChatRoomBan(db))
	repos.ChatBannedWord = repository.NewChatBannedWordRepo(dao.NewChatBannedWord(db))
	repos.ChatWatchParty = repository.NewChatWatchPartyRepo(dao.NewChatWatchParty(db))
	repos.LiveStream = repository.NewLiveStreamRepo(dao.NewLiveStream(db))
	repos.StreamCredentials = repository.NewStreamCredentialsRepo(dao.NewStreamCredentials(db))
	repos.GameRoom = repository.NewGameRoomRepo(dao.NewGameRoom(db), c)
	repos.HomeFeed = repository.NewHomeFeedRepo(dao.NewHomeFeed(db))
	repos.SidebarVisited = repository.NewSidebarLastVisitedRepo(dao.NewSidebarVisited(db))
	repos.Search = repository.NewSearchRepo(dao.NewSearch(db))
	repos.Sitemap = repository.NewSitemapRepo(dao.NewSitemap(db))
	repos.DeviceToken = repository.NewDeviceTokenRepo(dao.NewDeviceToken(db))
	repos.OverlayToken = repository.NewOverlayTokenRepo(dao.NewOverlayToken(db))

	return repos
}
