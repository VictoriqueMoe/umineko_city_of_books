package dao

import (
	"database/sql"

	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

func NewSession(db *sql.DB) repository.SessionRepository { return &sessionDAO{db: db} }

func NewUser(db *sql.DB) repository.UserRepository { return &userDAO{db: db} }

func NewTheory(db *sql.DB) repository.TheoryRepository { return &theoryDAO{db: db} }

func NewNotification(db *sql.DB) repository.NotificationRepository { return &notificationDAO{db: db} }

func NewRole(db *sql.DB) repository.RoleRepository { return &roleDAO{db: db} }

func NewSettings(db *sql.DB) repository.SettingsRepository { return &settingsDAO{db: db} }

func NewAuditLog(db *sql.DB) repository.AuditLogRepository { return &auditLogDAO{db: db} }

func NewStats(db *sql.DB) repository.StatsRepository { return &statsDAO{db: db} }

func NewInvite(db *sql.DB) repository.InviteRepository { return &inviteDAO{db: db} }

func NewPasswordReset(db *sql.DB) repository.PasswordResetRepository {
	return &passwordResetDAO{db: db}
}

func NewEmailVerification(db *sql.DB) repository.EmailVerificationRepository {
	return &emailVerificationDAO{db: db}
}

func NewChat(db *sql.DB) repository.ChatRepository { return &chatDAO{db: db} }

func NewReport(db *sql.DB) repository.ReportRepository { return &reportDAO{db: db} }

func NewPost(db *sql.DB) repository.PostRepository {
	return &postDAO{
		db:         db,
		commentDAO: newCommentDAO[uuid.UUID](db, "post_comments", "post_id", "post_comment_likes", "post_comment_media"),
		likeDAO:    newLikeDAO(db, "post_likes", "post_id"),
		mediaDAO:   newMediaDAO(db, "post_media", "post_id"),
		viewDAO:    newViewDAO(db, "post_views", "post_id", "posts"),
	}
}

func NewFollow(db *sql.DB) repository.FollowRepository { return &followDAO{db: db} }

func NewArt(db *sql.DB) repository.ArtRepository {
	return &artDAO{
		db:         db,
		commentDAO: newCommentDAO[uuid.UUID](db, "art_comments", "art_id", "art_comment_likes", "art_comment_media"),
		likeDAO:    newLikeDAO(db, "art_likes", "art_id"),
		viewDAO:    newViewDAO(db, "art_views", "art_id", "art"),
	}
}

func NewUpload(db *sql.DB) repository.UploadRepository { return &uploadDAO{db: db} }

func NewBlock(db *sql.DB) repository.BlockRepository { return &blockDAO{db: db} }

func NewAnnouncement(db *sql.DB) repository.AnnouncementRepository {
	return &announcementDAO{
		db:         db,
		commentDAO: newCommentDAO[uuid.UUID](db, "announcement_comments", "announcement_id", "announcement_comment_likes", "announcement_comment_media"),
	}
}

func NewMystery(db *sql.DB) repository.MysteryRepository {
	return &mysteryDAO{
		db:         db,
		commentDAO: newCommentDAO[uuid.UUID](db, "mystery_comments", "mystery_id", "mystery_comment_likes", "mystery_comment_media"),
		mediaDAO:   newMediaDAO(db, "mystery_media", "mystery_id"),
	}
}

func NewShip(db *sql.DB) repository.ShipRepository {
	return &shipDAO{
		db:         db,
		commentDAO: newCommentDAO[uuid.UUID](db, "ship_comments", "ship_id", "ship_comment_likes", "ship_comment_media"),
	}
}

func NewOC(db *sql.DB) repository.OCRepository {
	return &ocDAO{
		db:         db,
		commentDAO: newCommentDAO[uuid.UUID](db, "oc_comments", "oc_id", "oc_comment_likes", "oc_comment_media"),
	}
}

func NewFanfic(db *sql.DB) repository.FanficRepository {
	return &fanficDAO{
		db:         db,
		commentDAO: newCommentDAO[uuid.UUID](db, "fanfic_comments", "fanfic_id", "fanfic_comment_likes", "fanfic_comment_media"),
		viewDAO:    newViewDAO(db, "fanfic_views", "fanfic_id", "fanfics"),
	}
}

func NewJournal(db *sql.DB) repository.JournalRepository {
	return &journalDAO{
		db:         db,
		commentDAO: newCommentDAO[uuid.UUID](db, "journal_comments", "journal_id", "journal_comment_likes", "journal_comment_media"),
		mediaDAO:   newMediaDAO(db, "journal_entry_media", "entry_id"),
	}
}

func NewVanityRole(db *sql.DB) repository.VanityRoleRepository { return &vanityRoleDAO{db: db} }

func NewGiphyFavourite(db *sql.DB) repository.GiphyFavouriteRepository {
	return &giphyFavouriteDAO{db: db}
}

func NewBannedGiphy(db *sql.DB) repository.BannedGiphyRepository { return &bannedGiphyDAO{db: db} }

func NewUserSecret(db *sql.DB) repository.UserSecretRepository { return &userSecretDAO{db: db} }

func NewSecret(db *sql.DB) repository.SecretRepository {
	return &secretDAO{
		db:         db,
		commentDAO: newCommentDAO[string](db, "secret_comments", "secret_id", "secret_comment_likes", "secret_comment_media"),
	}
}

func NewChatRoomBan(db *sql.DB) repository.ChatRoomBanRepository { return &chatRoomBanDAO{db: db} }

func NewChatBannedWord(db *sql.DB) repository.ChatBannedWordRepository {
	return &chatBannedWordDAO{db: db}
}

func NewChatWatchParty(db *sql.DB) repository.ChatWatchPartyRepository {
	return &chatWatchPartyDAO{db: db}
}

func NewLiveStream(db *sql.DB) repository.LiveStreamRepository { return &liveStreamDAO{db: db} }

func NewStreamCredentials(db *sql.DB) repository.StreamCredentialsRepository {
	return &streamCredentialsDAO{db: db}
}

func NewGameRoom(db *sql.DB) repository.GameRoomRepository { return &gameRoomDAO{db: db} }

func NewHomeFeed(db *sql.DB) repository.HomeFeedRepository { return &homeFeedDAO{db: db} }

func NewSidebarVisited(db *sql.DB) repository.SidebarLastVisitedRepository {
	return &sidebarLastVisitedDAO{db: db}
}

func NewSearch(db *sql.DB) repository.SearchRepository { return &searchDAO{db: db} }

func NewSitemap(db *sql.DB) repository.SitemapRepository { return &sitemapDAO{db: db} }

func NewDeviceToken(db *sql.DB) repository.DeviceTokenRepository { return &deviceTokenDAO{db: db} }

func NewOverlayToken(db *sql.DB) repository.OverlayTokenRepository { return &overlayTokenDAO{db: db} }
