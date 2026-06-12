package chat

import (
	"context"
	"io"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/hyperbeam"
	"umineko_city_of_books/internal/livekit"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

type (
	Service interface {
		EnsureSystemRooms(ctx context.Context) error
		SyncSystemRoomMembership(ctx context.Context, userID uuid.UUID, newRole role.Role) error

		CreateStreamRoom(ctx context.Context, streamID, streamerID uuid.UUID, title string) error
		JoinStreamChat(ctx context.Context, streamID, userID uuid.UUID) error
		DeleteStreamRoom(ctx context.Context, streamID uuid.UUID) error

		ResolveDMRoom(ctx context.Context, senderID, recipientID uuid.UUID) (*dto.ResolveDMResponse, error)
		SendDMMessage(ctx context.Context, senderID, recipientID uuid.UUID, body string, files []FileUpload) (*dto.SendDMResponse, error)
		CreateGroupRoom(ctx context.Context, creatorID uuid.UUID, req dto.CreateGroupRoomRequest) (*dto.ChatRoomResponse, error)
		ListRooms(ctx context.Context, userID uuid.UUID) (*dto.ChatRoomListResponse, error)
		ListUserGroupRooms(ctx context.Context, userID uuid.UUID, search string, isRPOnly bool, tag, role string, includeArchived bool, limit, offset int) (*dto.ChatRoomListResponse, error)
		ListPublicRooms(ctx context.Context, search string, isRPOnly bool, tag string, viewerID uuid.UUID, includeArchived bool, limit, offset int) (*dto.ChatRoomListResponse, error)
		ArchiveStale(ctx context.Context) (int, error)
		GetMessages(ctx context.Context, userID, roomID uuid.UUID, limit, offset int) (*dto.ChatMessageListResponse, error)
		GetMessagesBefore(ctx context.Context, userID, roomID uuid.UUID, before string, limit int) (*dto.ChatMessageListResponse, error)

		SendMessage(ctx context.Context, senderID, roomID uuid.UUID, req dto.SendMessageRequest, files []FileUpload) (*dto.ChatMessageResponse, error)
		GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
		DeleteChat(ctx context.Context, roomID, userID uuid.UUID) error
		JoinRoom(ctx context.Context, roomID, userID uuid.UUID, ghost bool) (*dto.ChatRoomResponse, error)
		SetRoomMuted(ctx context.Context, roomID, userID uuid.UUID, muted bool) error
		IsRoomMuted(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
		LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error
		KickMember(ctx context.Context, hostID, roomID, targetID uuid.UUID) error
		InviteMembers(ctx context.Context, hostID, roomID uuid.UUID, userIDs []uuid.UUID) (*dto.InviteMembersResponse, error)
		SetMemberTimeout(ctx context.Context, roomID, actorID, targetID uuid.UUID, req dto.SetMemberTimeoutRequest) (*dto.ChatRoomMemberResponse, error)
		ClearMemberTimeout(ctx context.Context, roomID, actorID, targetID uuid.UUID) (*dto.ChatRoomMemberResponse, error)
		GetMembers(ctx context.Context, viewerID, roomID uuid.UUID) ([]dto.ChatRoomMemberResponse, error)
		GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
		MarkRead(ctx context.Context, roomID, userID uuid.UUID) error

		SetRoomNickname(ctx context.Context, roomID, userID uuid.UUID, nickname string) (*dto.ChatRoomMemberResponse, error)
		SetRoomAvatar(ctx context.Context, roomID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.ChatRoomMemberResponse, error)
		ClearRoomAvatar(ctx context.Context, roomID, userID uuid.UUID) (*dto.ChatRoomMemberResponse, error)
		SetMemberNicknameAsMod(ctx context.Context, roomID, actorID, targetID uuid.UUID, nickname string) (*dto.ChatRoomMemberResponse, error)
		UnlockMemberNickname(ctx context.Context, roomID, actorID, targetID uuid.UUID) (*dto.ChatRoomMemberResponse, error)
		PinMessage(ctx context.Context, messageID, userID uuid.UUID) error
		UnpinMessage(ctx context.Context, messageID, userID uuid.UUID) error
		ListPinnedMessages(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.ChatMessageListResponse, error)
		AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
		RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
		DeleteMessage(ctx context.Context, messageID, actorID uuid.UUID) error
		EditMessage(ctx context.Context, messageID, actorID uuid.UUID, body string) (*dto.ChatMessageResponse, error)

		BanMember(ctx context.Context, actorID, roomID, targetID uuid.UUID, reason string) error
		UnbanMember(ctx context.Context, actorID, roomID, targetID uuid.UUID) error
		ListRoomBans(ctx context.Context, actorID, roomID uuid.UUID) ([]dto.ChatRoomBanResponse, error)

		ListRoomBannedWords(ctx context.Context, actorID, roomID uuid.UUID) ([]dto.BannedWordRuleResponse, error)
		CreateRoomBannedWord(ctx context.Context, actorID, roomID uuid.UUID, req dto.CreateBannedWordRequest) (*dto.BannedWordRuleResponse, error)
		UpdateRoomBannedWord(ctx context.Context, actorID, roomID, ruleID uuid.UUID, req dto.UpdateBannedWordRequest) (*dto.BannedWordRuleResponse, error)
		DeleteRoomBannedWord(ctx context.Context, actorID, roomID, ruleID uuid.UUID) error

		ListGlobalBannedWords(ctx context.Context, actorID uuid.UUID) ([]dto.BannedWordRuleResponse, error)
		CreateGlobalBannedWord(ctx context.Context, actorID uuid.UUID, req dto.CreateBannedWordRequest) (*dto.BannedWordRuleResponse, error)
		UpdateGlobalBannedWord(ctx context.Context, actorID, ruleID uuid.UUID, req dto.UpdateBannedWordRequest) (*dto.BannedWordRuleResponse, error)
		DeleteGlobalBannedWord(ctx context.Context, actorID, ruleID uuid.UUID) error

		WatchPartyEnabled() bool
		StartWatchParty(ctx context.Context, roomID, actorID uuid.UUID, startURL, region, title, sessionType string) (*dto.StartWatchPartyResponse, error)
		JoinWatchParty(ctx context.Context, roomID, sessionID, actorID uuid.UUID) (*dto.JoinWatchPartyResponse, error)
		LeaveWatchParty(ctx context.Context, roomID, sessionID, actorID uuid.UUID) error
		GrantWatchPartyControl(ctx context.Context, roomID, sessionID, callerID, targetID uuid.UUID) error
		KickWatchPartyParticipant(ctx context.Context, roomID, sessionID, callerID, targetID uuid.UUID) error
		EndWatchParty(ctx context.Context, roomID, sessionID, actorID uuid.UUID, reason string) error
		HandleClientDisconnect(ctx context.Context, userID uuid.UUID, roomIDs []uuid.UUID)
		ListWatchParties(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.WatchPartyListResponse, error)
		IdentifyWatchPartyParticipant(ctx context.Context, roomID, sessionID, userID uuid.UUID, identifier string) error
		SendWatchPartyMessage(ctx context.Context, roomID, sessionID, senderID uuid.UUID, body string) (*dto.WatchPartyMessage, error)
		GetWatchPartyMessages(ctx context.Context, roomID, sessionID, viewerID uuid.UUID) (*dto.WatchPartyMessagesResponse, error)
		MintSessionVoiceToken(ctx context.Context, roomID, sessionID, userID uuid.UUID) (token, url string, err error)
		ForceMuteSessionVoice(ctx context.Context, roomID, sessionID, actorID, targetID uuid.UUID, muted bool) error
		StartWatchPartyReconcileLoop(ctx context.Context)

		VoiceEnabled() bool
		MintVoiceToken(ctx context.Context, roomID, userID uuid.UUID) (token, url string, err error)
		ForceMuteVoice(ctx context.Context, roomID, actorID, targetID uuid.UUID, muted bool) error
		HandleVoiceWebhook(ctx context.Context, authHeader string, body []byte) error
		VoiceParticipants(roomID uuid.UUID) []uuid.UUID
		VoiceCount(roomID uuid.UUID) int
		ReconcilePresence(ctx context.Context) (int, error)
	}

	service struct {
		*core
		*systemService
		*streamChatService
		*dmService
		*reactionsService
		*membersService
		*roomsService
		*messagesService
		*moderationService
		*watchPartyService
		*voiceService
	}
)

func NewService(
	chatRepo repository.ChatRepository,
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	vanityRoleRepo repository.VanityRoleRepository,
	banRepo repository.ChatRoomBanRepository,
	bannedWordRepo repository.ChatBannedWordRepository,
	watchPartyRepo repository.ChatWatchPartyRepository,
	auditRepo repository.AuditLogRepository,
	authzSvc authz.Service,
	notifSvc notification.Service,
	blockSvc block.Service,
	uploadSvc upload.Service,
	settingsSvc settings.Service,
	mediaProc *media.Processor,
	hub *ws.Hub,
	hyperbeamSvc hyperbeam.Service,
	livekitSvc livekit.Service,
	contentFilter *contentfilter.Manager,
) Service {
	c := &core{
		chatRepo:        chatRepo,
		userRepo:        userRepo,
		roleRepo:        roleRepo,
		vanityRoleRepo:  vanityRoleRepo,
		banRepo:         banRepo,
		bannedWordRepo:  bannedWordRepo,
		watchPartyRepo:  watchPartyRepo,
		auditRepo:       auditRepo,
		authzSvc:        authzSvc,
		notifSvc:        notifSvc,
		blockSvc:        blockSvc,
		settingsSvc:     settingsSvc,
		uploadSvc:       uploadSvc,
		uploader:        media.NewUploader(uploadSvc, settingsSvc, mediaProc),
		hub:             hub,
		hyperbeamSvc:    hyperbeamSvc,
		livekitSvc:      livekitSvc,
		contentFilter:   contentFilter,
		bannedWordsRule: contentfilter.NewChatBannedWordsRule(bannedWordRepo),
		voiceMuted:      make(map[string]map[uuid.UUID]struct{}),
	}

	svs := &service{
		core:              c,
		systemService:     &systemService{core: c},
		streamChatService: &streamChatService{core: c},
		reactionsService:  &reactionsService{core: c},
		roomsService:      &roomsService{core: c},
		moderationService: &moderationService{core: c},
		watchPartyService: &watchPartyService{core: c},
		voiceService:      newVoiceService(c),
	}
	svs.dmService = &dmService{core: c, parent: svs}
	svs.membersService = &membersService{core: c, parent: svs}
	svs.messagesService = &messagesService{core: c, parent: svs}

	svs.StartWatchPartyReconcileLoop(context.Background())
	return svs
}
