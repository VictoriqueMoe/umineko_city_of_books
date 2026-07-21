package dao

import (
	"database/sql"

	"umineko_city_of_books/internal/repository"
)

func New(db *sql.DB) *repository.Repositories {
	repos := repository.NewRepositories(db)

	repos.Session = &sessionRepository{SessionRepository: &sessionDAO{db: db}}
	repos.User = &userRepository{UserRepository: &userDAO{db: db}}
	repos.Theory = &theoryRepository{TheoryRepository: &theoryDAO{db: db}}
	repos.Notification = &notificationRepository{NotificationRepository: &notificationDAO{db: db}}
	repos.Role = &roleRepository{RoleRepository: &roleDAO{db: db}}
	repos.Settings = &settingsRepository{SettingsRepository: &settingsDAO{db: db}}
	repos.AuditLog = &auditLogRepository{AuditLogRepository: &auditLogDAO{db: db}}
	repos.Stats = &statsRepository{StatsRepository: &statsDAO{db: db}}
	repos.Invite = &inviteRepository{InviteRepository: &inviteDAO{db: db}}
	repos.PasswordReset = &passwordResetRepository{PasswordResetRepository: &passwordResetDAO{db: db}}
	repos.EmailVerification = &emailVerificationRepository{EmailVerificationRepository: &emailVerificationDAO{db: db}}
	repos.Chat = &chatRepository{ChatRepository: &chatDAO{db: db}}
	repos.Report = &reportRepository{ReportRepository: &reportDAO{db: db}}
	repos.Post = &postRepository{PostRepository: &postDAO{db: db}}
	repos.Follow = &followRepository{FollowRepository: &followDAO{db: db}}
	repos.Art = &artRepository{ArtRepository: &artDAO{db: db}}
	repos.Upload = &uploadRepository{UploadRepository: &uploadDAO{db: db}}
	repos.Block = &blockRepository{BlockRepository: &blockDAO{db: db}}
	repos.Announcement = &announcementRepository{AnnouncementRepository: &announcementDAO{db: db}}
	repos.Mystery = &mysteryRepository{MysteryRepository: &mysteryDAO{db: db}}
	repos.Ship = &shipRepository{ShipRepository: &shipDAO{db: db}}
	repos.OC = &ocRepository{OCRepository: &ocDAO{db: db}}
	repos.Fanfic = &fanficRepository{FanficRepository: &fanficDAO{db: db}}
	repos.Journal = &journalRepository{JournalRepository: &journalDAO{db: db}}
	repos.VanityRole = &vanityRoleRepository{VanityRoleRepository: &vanityRoleDAO{db: db}}
	repos.GiphyFavourite = &giphyFavouriteRepository{GiphyFavouriteRepository: &giphyFavouriteDAO{db: db}}
	repos.BannedGiphy = &bannedGiphyRepository{BannedGiphyRepository: &bannedGiphyDAO{db: db}}
	repos.UserSecret = &userSecretRepository{UserSecretRepository: &userSecretDAO{db: db}}
	repos.Secret = &secretRepository{SecretRepository: &secretDAO{db: db}}
	repos.ChatRoomBan = &chatRoomBanRepository{ChatRoomBanRepository: &chatRoomBanDAO{db: db}}
	repos.ChatBannedWord = &chatBannedWordRepository{ChatBannedWordRepository: &chatBannedWordDAO{db: db}}
	repos.ChatWatchParty = &chatWatchPartyRepository{ChatWatchPartyRepository: &chatWatchPartyDAO{db: db}}
	repos.LiveStream = &liveStreamRepository{LiveStreamRepository: &liveStreamDAO{db: db}}
	repos.StreamCredentials = &streamCredentialsRepository{StreamCredentialsRepository: &streamCredentialsDAO{db: db}}
	repos.GameRoom = &gameRoomRepository{GameRoomRepository: &gameRoomDAO{db: db}}
	repos.HomeFeed = &homeFeedRepository{HomeFeedRepository: &homeFeedDAO{db: db}}
	repos.SidebarVisited = &sidebarLastVisitedRepository{SidebarLastVisitedRepository: &sidebarLastVisitedDAO{db: db}}
	repos.Search = &searchRepository{SearchRepository: &searchDAO{db: db}}
	repos.Sitemap = &sitemapRepository{SitemapRepository: &sitemapDAO{db: db}}
	repos.DeviceToken = &deviceTokenRepository{DeviceTokenRepository: &deviceTokenDAO{db: db}}
	repos.OverlayToken = &overlayTokenRepository{OverlayTokenRepository: &overlayTokenDAO{db: db}}

	return repos
}
