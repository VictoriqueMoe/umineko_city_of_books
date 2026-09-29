package chat

import (
	"context"
	"database/sql"
	"slices"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/hyperbeam"
	"umineko_city_of_books/internal/livekit"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

const (
	voiceTestKey    = "devkey"
	voiceTestSecret = "this-is-a-sufficiently-long-test-secret"
	voiceTestURL    = "ws://livekit.test:7880"
)

type (
	testMocks struct {
		chatRepo       *repository.MockChatRepository
		userRepo       *repository.MockUserRepository
		roleRepo       *repository.MockRoleRepository
		vanityRoleRepo *repository.MockVanityRoleRepository
		banRepo        *repository.MockChatRoomBanRepository
		bannedWordRepo *repository.MockChatBannedWordRepository
		watchPartyRepo *repository.MockChatWatchPartyRepository
		auditRepo      *repository.MockAuditLogRepository
		authzSvc       *authz.MockService
		notifSvc       *notification.MockService
		blockSvc       *block.MockService
		uploadSvc      *upload.MockService
		settingsSvc    *settings.MockService
		hyperbeamSvc   *hyperbeam.MockService
		hub            *ws.Hub
	}

	rejectRoomEditRule struct{}

	captureMessageObserver struct {
		events []BotMessageEvent
	}
)

func newTestService(t *testing.T) (*service, *testMocks) {
	chatRepo := repository.NewMockChatRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	roleRepo := repository.NewMockRoleRepository(t)
	vanityRoleRepo := repository.NewMockVanityRoleRepository(t)
	banRepo := repository.NewMockChatRoomBanRepository(t)
	bannedWordRepo := repository.NewMockChatBannedWordRepository(t)
	watchPartyRepo := repository.NewMockChatWatchPartyRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	authzSvc := authz.NewMockService(t)
	notifSvc := notification.NewMockService(t)
	blockSvc := block.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	mediaProc := &media.Processor{}
	hub := ws.NewHub()
	hyperbeamSvc := hyperbeam.NewMockService(t)
	livekitSvc := livekit.NewService(settingsSvc)
	svc := NewService(chatRepo, userRepo, roleRepo, vanityRoleRepo, banRepo, bannedWordRepo, watchPartyRepo, auditRepo, authzSvc, notifSvc, blockSvc, uploadSvc, settingsSvc, mediaProc, hub, hyperbeamSvc, livekitSvc, contentfilter.New(), nil).(*service)

	chatRepo.EXPECT().HasGhostMembers(mock.Anything, mock.Anything).Return(false, nil).Maybe()
	chatRepo.EXPECT().IsGhostMember(mock.Anything, mock.Anything).Return(false, nil).Maybe()
	chatRepo.EXPECT().HasActiveMemberTimeout(mock.Anything, mock.Anything).Return(false, nil).Maybe()
	banRepo.EXPECT().IsBanned(mock.Anything, mock.Anything).Return(false, nil).Maybe()
	banRepo.EXPECT().BannedRoomIDsForUser(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()
	chatRepo.EXPECT().GetRoomByID(mock.Anything, mock.MatchedBy(func(s spec.ChatRoomViewer) bool { return s.ViewerID == uuid.Nil })).Return(nil, nil).Maybe()
	chatRepo.EXPECT().GetMemberNickname(mock.Anything, mock.Anything).Return("", nil).Maybe()
	userRepo.EXPECT().IsLocked(mock.Anything, mock.Anything).Return(false, nil).Maybe()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingHyperbeamRegion).Return("EU").Maybe()

	return svc, &testMocks{
		chatRepo:       chatRepo,
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		vanityRoleRepo: vanityRoleRepo,
		banRepo:        banRepo,
		bannedWordRepo: bannedWordRepo,
		watchPartyRepo: watchPartyRepo,
		auditRepo:      auditRepo,
		authzSvc:       authzSvc,
		notifSvc:       notifSvc,
		blockSvc:       blockSvc,
		uploadSvc:      uploadSvc,
		settingsSvc:    settingsSvc,
		hyperbeamSvc:   hyperbeamSvc,
		hub:            hub,
	}
}

func expectVoiceConfigured(m *testMocks, enabled bool) {
	m.settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingVoiceEnabled).Return(enabled).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingLiveKitURL).Return(voiceTestURL).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingLiveKitAPIKey).Return(voiceTestKey).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingLiveKitAPISecret).Return(voiceTestSecret).Maybe()
}

func expectRoomKind(m *testMocks, roomID uuid.UUID, roomType dto.RoomType) {
	m.chatRepo.EXPECT().GetRoomSendContext(mock.Anything, roomID).
		Return(&model.ChatRoomSendContext{ID: roomID, Type: roomType, LastMessageAt: ongoingThread()}, nil)
}

func ongoingThread() sql.NullString {
	return sql.NullString{Valid: true, String: "2026-01-01T00:00:00Z"}
}

func sampleUser(id uuid.UUID) *model.User {
	return &model.User{
		ID:          id,
		Username:    "user",
		DisplayName: "User",
		AvatarURL:   "avatar.png",
		DmsEnabled:  true,
	}
}

func botUser(id uuid.UUID) *model.User {
	user := sampleUser(id)
	user.DisplayName = "Beatrice"
	user.IsBot = true

	return user
}

func (rejectRoomEditRule) Name() contentfilter.RuleName { return "test_reject" }

func (rejectRoomEditRule) Check(_ context.Context, _ []string) (*contentfilter.Rejection, error) {
	return &contentfilter.Rejection{Rule: "test_reject", Reason: "nope", Detail: "slur"}, nil
}

func editableRoom(roomID uuid.UUID) *model.ChatRoomRow {
	return &model.ChatRoomRow{
		ID:          roomID,
		Name:        "Old name",
		Description: "old description",
		Type:        dto.RoomTypeGroup,
		IsPublic:    true,
		Tags:        []string{"tag"},
	}
}

func editRequest(name string) dto.UpdateGroupRoomRequest {
	return dto.UpdateGroupRoomRequest{
		Name:        name,
		Description: "old description",
		Tags:        []string{"tag"},
		IsPublic:    true,
	}
}

func (c *captureMessageObserver) ObserveMessage(ev BotMessageEvent) {
	c.events = append(c.events, ev)
}

func unsetDefaults(m *mock.Mock, methods ...string) {
	var kept []*mock.Call
	for _, c := range m.ExpectedCalls {
		if slices.Contains(methods, c.Method) {
			continue
		}
		kept = append(kept, c)
	}
	m.ExpectedCalls = kept
}
