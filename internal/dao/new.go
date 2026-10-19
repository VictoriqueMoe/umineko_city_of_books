package dao

import (
	"database/sql"

	"umineko_city_of_books/internal/dao/dynamicsql"
	"umineko_city_of_books/internal/model/spec"
)

func NewSession(db *sql.DB) SessionDAO { return &sessionDAO{db: db} }

func NewUser(db *sql.DB) UserDAO { return &userDAO{db: db} }

func NewTheory(db *sql.DB) TheoryDAO {
	return &theoryDAO{
		db:            db,
		theoryVotes:   newVoteDAO(db, theoryVoteQuerier{}, ""),
		responseVotes: newVoteDAO(db, responseVoteQuerier{}, ""),
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

func NewChat(db *sql.DB) ChatDAO {
	return &chatDAO{ChatRooms: dynamicsql.NewChatRooms(db), db: db}
}

func NewReport(db *sql.DB) ReportDAO { return &reportDAO{db: db} }

func NewComments(db *sql.DB) CommentDAO {
	return &commentDAO{
		sqlcSource: newSQLCSource(db),
		tables: map[spec.CommentKind]commentQuerier{
			spec.CommentKindPost:         postCommentQuerier{},
			spec.CommentKindArt:          artCommentQuerier{},
			spec.CommentKindAnnouncement: announcementCommentQuerier{},
			spec.CommentKindMystery:      mysteryCommentQuerier{},
			spec.CommentKindShip:         shipCommentQuerier{},
			spec.CommentKindOC:           oCCommentQuerier{},
			spec.CommentKindFanfic:       fanficCommentQuerier{},
			spec.CommentKindJournal:      journalRootCommentQuerier{},
			spec.CommentKindJournalEntry: journalEntryCommentQuerier{},
			spec.CommentKindSecret:       secretCommentTable{},
		},
	}
}

func NewPost(db *sql.DB) PostDAO {
	return &postDAO{
		db:       db,
		ownedDAO: newOwnedDAO(db, postOwnedQuerier{}, "post"),
		likeDAO:  newLikeDAO(db, postLikeQuerier{}),
		mediaDAO: newMediaDAO(db, postMediaQuerier{}),
		viewDAO:  newViewDAO(db, postViewQuerier{}),
	}
}

func NewFollow(db *sql.DB) FollowDAO { return &followDAO{db: db} }

func NewArt(db *sql.DB) ArtDAO {
	return &artDAO{
		db:       db,
		ownedDAO: newOwnedDAO(db, artOwnedQuerier{}, "art"),
		likeDAO:  newLikeDAO(db, artLikeQuerier{}),
		viewDAO:  newViewDAO(db, artViewQuerier{}),
	}
}

func NewBlock(db *sql.DB) BlockDAO { return &blockDAO{db: db} }

func NewAnnouncement(db *sql.DB) AnnouncementDAO {
	return &announcementDAO{db: db}
}

func NewMystery(db *sql.DB) MysteryDAO {
	return &mysteryDAO{
		db:           db,
		ownedDAO:     newOwnedDAO(db, mysteryOwnedQuerier{}, "mystery"),
		attemptVotes: newVoteDAO(db, mysteryAttemptVoteQuerier{}, "vote attempt"),
		mediaDAO:     newMediaDAO(db, mysteryMediaQuerier{}),
	}
}

func NewShip(db *sql.DB) ShipDAO {
	return &shipDAO{
		db:       db,
		ownedDAO: newOwnedDAO(db, shipOwnedQuerier{}, "ship"),
		voteDAO:  newVoteDAO(db, shipVoteQuerier{}, "vote ship"),
	}
}

func NewOC(db *sql.DB) OCDAO {
	return &ocDAO{
		db:       db,
		ownedDAO: newOwnedDAO(db, oCOwnedQuerier{}, "oc"),
		voteDAO:  newVoteDAO(db, oCVoteQuerier{}, "vote oc"),
	}
}

func NewFanfic(db *sql.DB) FanficDAO {
	return &fanficDAO{
		db:       db,
		ownedDAO: newOwnedDAO(db, fanficOwnedQuerier{}, "fanfic"),
		viewDAO:  newViewDAO(db, fanficViewQuerier{}),
	}
}

func NewJournal(db *sql.DB) JournalDAO {
	return &journalDAO{
		db:       db,
		ownedDAO: newOwnedDAO(db, journalOwnedQuerier{}, "journal"),
		mediaDAO: newMediaDAO(db, journalEntryMediaQuerier{}),
	}
}

func NewVanityRole(db *sql.DB) VanityRoleDAO { return &vanityRoleDAO{db: db} }

func NewPermission(db *sql.DB) PermissionDAO { return &permissionDAO{db: db} }

func NewGiphyFavourite(db *sql.DB) GiphyFavouriteDAO {
	return &giphyFavouriteDAO{db: db}
}

func NewBannedGiphy(db *sql.DB) BannedGiphyDAO { return &bannedGiphyDAO{db: db} }

func NewStoredFile(db *sql.DB) StoredFileDAO { return &storedFileDAO{db: db} }

func NewUserSecret(db *sql.DB) UserSecretDAO { return &userSecretDAO{db: db} }

func NewSecret(db *sql.DB) SecretDAO {
	return &secretDAO{db: db}
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

func NewSitemap(db *sql.DB) SitemapDAO { return &sitemapDAO{db: db} }

func NewDeviceToken(db *sql.DB) DeviceTokenDAO { return &deviceTokenDAO{db: db} }

func NewOverlayToken(db *sql.DB) OverlayTokenDAO { return &overlayTokenDAO{db: db} }

func NewChatbot(db *sql.DB) ChatbotDAO { return &chatbotDAO{db: db} }

func NewChatbotBasePrompt(db *sql.DB) ChatbotBasePromptDAO {
	return &chatbotBasePromptDAO{db: db}
}
