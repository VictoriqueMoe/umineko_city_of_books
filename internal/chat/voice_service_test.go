package chat

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/livekit"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/livekit/protocol/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	voiceTestKey    = "devkey"
	voiceTestSecret = "this-is-a-sufficiently-long-test-secret"
	voiceTestURL    = "ws://livekit.test:7880"
)

func expectVoiceConfigured(m *testMocks, enabled bool) {
	m.settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingVoiceEnabled).Return(enabled).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingLiveKitURL).Return(voiceTestURL).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingLiveKitAPIKey).Return(voiceTestKey).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingLiveKitAPISecret).Return(voiceTestSecret).Maybe()
}

func TestVoicePresenceAccounting(t *testing.T) {
	// given
	vs := &voiceService{presence: make(map[uuid.UUID]map[uuid.UUID]any)}
	roomID := uuid.New()
	alice := uuid.New()
	bob := uuid.New()

	// when
	vs.addParticipant(roomID, alice)
	vs.addParticipant(roomID, bob)
	vs.addParticipant(roomID, alice)

	// then
	assert.Equal(t, 2, vs.VoiceCount(roomID))
	assert.Len(t, vs.VoiceParticipants(roomID), 2)

	// when
	vs.removeParticipant(roomID, alice)

	// then
	assert.Equal(t, 1, vs.VoiceCount(roomID))
	assert.Equal(t, []uuid.UUID{bob}, vs.VoiceParticipants(roomID))

	// when
	vs.removeParticipant(roomID, bob)

	// then
	assert.Equal(t, 0, vs.VoiceCount(roomID))
	assert.Empty(t, vs.VoiceParticipants(roomID))
}

func TestVoiceClearRoom(t *testing.T) {
	// given
	vs := &voiceService{presence: make(map[uuid.UUID]map[uuid.UUID]any)}
	roomID := uuid.New()
	vs.addParticipant(roomID, uuid.New())
	vs.addParticipant(roomID, uuid.New())

	// when
	vs.clearRoom(roomID)

	// then
	assert.Equal(t, 0, vs.VoiceCount(roomID))
}

func TestMintVoiceToken_Disabled(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectVoiceConfigured(m, false)
	roomID := uuid.New()
	userID := uuid.New()

	// when
	_, _, err := svc.MintVoiceToken(context.Background(), roomID, userID)

	// then
	require.ErrorIs(t, err, ErrVoiceDisabled)
}

func TestMintVoiceToken_NotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectVoiceConfigured(m, true)
	roomID := uuid.New()
	userID := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{ID: roomID, Type: "group"}, nil)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(false, nil)

	// when
	_, _, err := svc.MintVoiceToken(context.Background(), roomID, userID)

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestMintVoiceToken_HappyPath(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectVoiceConfigured(m, true)
	roomID := uuid.New()
	userID := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{ID: roomID, Type: "group"}, nil)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)

	// when
	token, url, err := svc.MintVoiceToken(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
	assert.Equal(t, voiceTestURL, url)

	verifier, err := auth.ParseAPIToken(token)
	require.NoError(t, err)
	_, grants, err := verifier.Verify(voiceTestSecret)
	require.NoError(t, err)
	assert.Equal(t, userID.String(), verifier.Identity())
	assert.Equal(t, roomID.String(), grants.Video.Room)
}

func TestMintVoiceToken_DMBlocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectVoiceConfigured(m, true)
	roomID := uuid.New()
	userID := uuid.New()
	otherID := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{ID: roomID, Type: "dm"}, nil)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID, otherID}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, otherID).Return(true, nil)

	// when
	_, _, err := svc.MintVoiceToken(context.Background(), roomID, userID)

	// then
	require.ErrorIs(t, err, ErrUserBlocked)
}

func TestReconcilePresence_Disabled(t *testing.T) {
	// given
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingVoiceEnabled).Return(false).Maybe()
	lk := livekit.NewMockService(t)
	vs := newVoiceService(&core{settingsSvc: settingsSvc}, lk)

	// when
	n, err := vs.ReconcilePresence(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestReconcilePresence_RebuildsFromLiveKit(t *testing.T) {
	// given
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingVoiceEnabled).Return(true).Maybe()
	lk := livekit.NewMockService(t)
	lk.EXPECT().Enabled().Return(true).Maybe()
	chatRepo := repository.NewMockChatRepository(t)
	vs := newVoiceService(&core{chatRepo: chatRepo, hub: ws.NewHub(), settingsSvc: settingsSvc}, lk)

	liveRoom := uuid.New()
	liveUser := uuid.New()
	staleRoom := uuid.New()
	vs.addParticipant(staleRoom, uuid.New())

	lk.EXPECT().ActiveRooms(mock.Anything).Return(map[string][]string{
		liveRoom.String(): {liveUser.String()},
	}, nil)
	chatRepo.EXPECT().GetRoomMembers(mock.Anything, mock.Anything).Return([]uuid.UUID{liveUser}, nil).Maybe()

	// when
	n, err := vs.ReconcilePresence(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, 1, vs.VoiceCount(liveRoom))
	assert.Equal(t, []uuid.UUID{liveUser}, vs.VoiceParticipants(liveRoom))
	assert.Equal(t, 0, vs.VoiceCount(staleRoom))
}

func TestHandleVoiceWebhook_UpdatesPresence(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expectVoiceConfigured(m, true)
	roomID := uuid.New()
	userID := uuid.New()

	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil).Maybe()
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.Anything, roomID, userID, mock.Anything).Return(nil)
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, mock.Anything).Return(nil, nil)

	body := []byte(fmt.Sprintf(`{"event":"participant_joined","room":{"name":%q},"participant":{"identity":%q}}`, roomID, userID))
	sum := sha256.Sum256(body)
	authToken, err := auth.NewAccessToken(voiceTestKey, voiceTestSecret).SetSha256(base64.StdEncoding.EncodeToString(sum[:])).ToJWT()
	require.NoError(t, err)

	// when
	err = svc.HandleVoiceWebhook(context.Background(), authToken, body)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, svc.VoiceCount(roomID))
	assert.Equal(t, []uuid.UUID{userID}, svc.VoiceParticipants(roomID))
}
