package repository

import "database/sql"

type (
	Repositories struct {
		db                *sql.DB
		Session           SessionRepository
		User              UserRepository
		Theory            TheoryRepository
		Notification      NotificationRepository
		Role              RoleRepository
		Settings          SettingsRepository
		AuditLog          AuditLogRepository
		Stats             StatsRepository
		Invite            InviteRepository
		PasswordReset     PasswordResetRepository
		EmailVerification EmailVerificationRepository
		Chat              ChatRepository
		Report            ReportRepository
		Post              PostRepository
		Follow            FollowRepository
		Art               ArtRepository
		Upload            UploadRepository
		Block             BlockRepository
		Announcement      AnnouncementRepository
		Mystery           MysteryRepository
		Ship              ShipRepository
		OC                OCRepository
		Fanfic            FanficRepository
		Journal           JournalRepository
		VanityRole        VanityRoleRepository
		GiphyFavourite    GiphyFavouriteRepository
		BannedGiphy       BannedGiphyRepository
		UserSecret        UserSecretRepository
		Secret            SecretRepository
		ChatRoomBan       ChatRoomBanRepository
		ChatBannedWord    ChatBannedWordRepository
		ChatWatchParty    ChatWatchPartyRepository
		LiveStream        LiveStreamRepository
		StreamCredentials StreamCredentialsRepository
		GameRoom          GameRoomRepository
		HomeFeed          HomeFeedRepository
		SidebarVisited    SidebarLastVisitedRepository
		Search            SearchRepository
		Sitemap           SitemapRepository
		DeviceToken       DeviceTokenRepository
		OverlayToken      OverlayTokenRepository
	}
)

func (r *Repositories) DB() *sql.DB {
	return r.db
}

func NewRepositories(db *sql.DB) *Repositories {
	return &Repositories{db: db}
}
