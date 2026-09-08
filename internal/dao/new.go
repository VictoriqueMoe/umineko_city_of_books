package dao

import (
	"database/sql"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
)

func NewSession(db *sql.DB) SessionDAO { return &sessionDAO{db: db} }

func NewUser(db *sql.DB) UserDAO { return &userDAO{db: db} }

func NewTheory(db *sql.DB) TheoryDAO {
	return &theoryDAO{
		db:            db,
		theoryVotes:   newVoteDAO(db, "theory_votes", "theory_id", ""),
		responseVotes: newVoteDAO(db, "response_votes", "response_id", ""),
	}
}

func NewNotification(db *sql.DB) NotificationDAO { return &notificationDAO{db: db} }

func NewRole(db *sql.DB) RoleDAO { return &roleDAO{db: db} }

func NewSettings(db *sql.DB) SettingsDAO { return &settingsDAO{db: db} }

func NewAuditLog(db *sql.DB) AuditLogDAO { return &auditLogDAO{db: db} }

func NewStats(db *sql.DB) StatsDAO { return &statsDAO{db: db} }

func NewInvite(db *sql.DB) InviteDAO { return &inviteDAO{db: db} }

func NewPasswordReset(db *sql.DB) PasswordResetDAO {
	return &passwordResetDAO{db: db}
}

func NewEmailVerification(db *sql.DB) EmailVerificationDAO {
	return &emailVerificationDAO{db: db}
}

func NewChat(db *sql.DB) ChatDAO { return &chatDAO{db: db} }

func NewReport(db *sql.DB) ReportDAO { return &reportDAO{db: db} }

func NewPost(db *sql.DB) (PostDAO, CommentDAO[uuid.UUID]) {
	d := &postDAO{
		db:         db,
		ownedDAO:   newOwnedDAO(db, "posts", "post"),
		commentDAO: newCommentDAO[uuid.UUID](db, postCommentQuerier{}),
		likeDAO:    newLikeDAO(db, "post_likes", "post_id"),
		mediaDAO:   newMediaDAO(db, "post_media", "post_id"),
		viewDAO:    newViewDAO(db, "post_views", "post_id"),
	}

	return d, d.commentDAO
}

func NewFollow(db *sql.DB) FollowDAO { return &followDAO{db: db} }

func NewArt(db *sql.DB) (ArtDAO, CommentDAO[uuid.UUID]) {
	d := &artDAO{
		db:         db,
		ownedDAO:   newOwnedDAO(db, "art", "art"),
		commentDAO: newCommentDAO[uuid.UUID](db, artCommentQuerier{}),
		likeDAO:    newLikeDAO(db, "art_likes", "art_id"),
		viewDAO:    newViewDAO(db, "art_views", "art_id"),
	}

	return d, d.commentDAO
}

func NewUpload(db *sql.DB) UploadDAO { return &uploadDAO{db: db} }

func NewBlock(db *sql.DB) BlockDAO { return &blockDAO{db: db} }

func NewAnnouncement(db *sql.DB) (AnnouncementDAO, CommentDAO[uuid.UUID]) {
	d := &announcementDAO{
		db:         db,
		gen:        sqlcgen.New(db),
		commentDAO: newCommentDAO[uuid.UUID](db, announcementCommentQuerier{}),
	}

	return d, d.commentDAO
}

func NewMystery(db *sql.DB) (MysteryDAO, CommentDAO[uuid.UUID]) {
	d := &mysteryDAO{
		db:           db,
		ownedDAO:     newOwnedDAO(db, "mysteries", "mystery"),
		attemptVotes: newVoteDAO(db, "mystery_attempt_votes", "attempt_id", "vote attempt"),
		commentDAO:   newCommentDAO[uuid.UUID](db, mysteryCommentQuerier{}),
		mediaDAO:     newMediaDAO(db, "mystery_media", "mystery_id"),
	}

	return d, d.commentDAO
}

func NewShip(db *sql.DB) (ShipDAO, CommentDAO[uuid.UUID]) {
	d := &shipDAO{
		db:         db,
		ownedDAO:   newOwnedDAO(db, "ships", "ship"),
		voteDAO:    newVoteDAO(db, "ship_votes", "ship_id", "vote ship"),
		commentDAO: newCommentDAO[uuid.UUID](db, shipCommentQuerier{}),
	}

	return d, d.commentDAO
}

func NewOC(db *sql.DB) (OCDAO, CommentDAO[uuid.UUID]) {
	d := &ocDAO{
		db:         db,
		ownedDAO:   newOwnedDAO(db, "ocs", "oc"),
		voteDAO:    newVoteDAO(db, "oc_votes", "oc_id", "vote oc"),
		commentDAO: newCommentDAO[uuid.UUID](db, oCCommentQuerier{}),
	}

	return d, d.commentDAO
}

func NewFanfic(db *sql.DB) (FanficDAO, CommentDAO[uuid.UUID]) {
	d := &fanficDAO{
		db:         db,
		ownedDAO:   newOwnedDAO(db, "fanfics", "fanfic"),
		commentDAO: newCommentDAO[uuid.UUID](db, fanficCommentQuerier{}),
		viewDAO:    newViewDAO(db, "fanfic_views", "fanfic_id"),
	}

	return d, d.commentDAO
}

func NewJournal(db *sql.DB) JournalDAO {
	return &journalDAO{
		db:         db,
		ownedDAO:   newOwnedDAO(db, "journals", "journal"),
		commentDAO: newCommentDAO[uuid.UUID](db, journalCommentQuerier{}),
		mediaDAO:   newMediaDAO(db, "journal_entry_media", "entry_id"),
	}
}

func NewVanityRole(db *sql.DB) VanityRoleDAO { return &vanityRoleDAO{db: db} }

func NewPermission(db *sql.DB) PermissionDAO { return &permissionDAO{db: db} }

func NewGiphyFavourite(db *sql.DB) GiphyFavouriteDAO {
	return &giphyFavouriteDAO{db: db}
}

func NewBannedGiphy(db *sql.DB) BannedGiphyDAO { return &bannedGiphyDAO{db: db} }

func NewUserSecret(db *sql.DB) UserSecretDAO { return &userSecretDAO{db: db} }

func NewSecret(db *sql.DB) (SecretDAO, CommentDAO[string]) {
	d := &secretDAO{
		db:         db,
		commentDAO: newCommentDAO[string](db, secretCommentQuerier{}),
	}

	return d, d.commentDAO
}

func NewChatRoomBan(db *sql.DB) ChatRoomBanDAO { return &chatRoomBanDAO{db: db} }

func NewChatBannedWord(db *sql.DB) ChatBannedWordDAO {
	return &chatBannedWordDAO{db: db}
}

func NewChatWatchParty(db *sql.DB) ChatWatchPartyDAO {
	return &chatWatchPartyDAO{db: db}
}

func NewLiveStream(db *sql.DB) LiveStreamDAO { return &liveStreamDAO{db: db} }

func NewStreamCredentials(db *sql.DB) StreamCredentialsDAO {
	return &streamCredentialsDAO{db: db}
}

func NewGameRoom(db *sql.DB) GameRoomDAO { return &gameRoomDAO{db: db} }

func NewHomeFeed(db *sql.DB) HomeFeedDAO { return &homeFeedDAO{db: db} }

func NewSidebarVisited(db *sql.DB) SidebarLastVisitedDAO {
	return &sidebarLastVisitedDAO{db: db}
}

func NewSearch(db *sql.DB) SearchDAO { return &searchDAO{db: db} }

func NewSitemap(db *sql.DB) SitemapDAO { return &sitemapDAO{db: db} }

func NewDeviceToken(db *sql.DB) DeviceTokenDAO { return &deviceTokenDAO{db: db} }

func NewOverlayToken(db *sql.DB) OverlayTokenDAO { return &overlayTokenDAO{db: db} }

func NewChatbot(db *sql.DB) ChatbotDAO { return &chatbotDAO{db: db} }

func NewChatbotBasePrompt(db *sql.DB) ChatbotBasePromptDAO {
	return &chatbotBasePromptDAO{db: db}
}
