package livekit

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testURL    = "ws://livekit.test:7880"
	testKey    = "devkey"
	testSecret = "this-is-a-sufficiently-long-test-secret"
)

func configuredService(t *testing.T, url, key, secret string) Service {
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingLiveKitURL).Return(url).Maybe()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingLiveKitAPIKey).Return(key).Maybe()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingLiveKitAPISecret).Return(secret).Maybe()

	return NewService(settingsSvc)
}

func TestEnabled(t *testing.T) {
	// given
	cases := []struct {
		name   string
		url    string
		key    string
		secret string
		want   bool
	}{
		{"all set", testURL, testKey, testSecret, true},
		{"missing url", "", testKey, testSecret, false},
		{"missing key", testURL, "", testSecret, false},
		{"missing secret", testURL, testKey, "", false},
		{"all empty", "", "", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			svc := configuredService(t, tc.url, tc.key, tc.secret)

			// then
			assert.Equal(t, tc.want, svc.Enabled())
		})
	}
}

func TestMintToken_Disabled(t *testing.T) {
	// given
	svc := configuredService(t, "", "", "")

	// when
	_, err := svc.MintToken("room", "user", "User", true, false)

	// then
	require.ErrorIs(t, err, ErrDisabled)
}

func TestMintToken_SignsGrant(t *testing.T) {
	// given
	svc := configuredService(t, testURL, testKey, testSecret)
	roomID := uuid.New().String()
	userID := uuid.New().String()

	// when
	token, err := svc.MintToken(roomID, userID, "Beatrice", true, false)
	require.NoError(t, err)

	// then
	verifier, err := auth.ParseAPIToken(token)
	require.NoError(t, err)

	_, grants, err := verifier.Verify(testSecret)
	require.NoError(t, err)
	assert.Equal(t, userID, verifier.Identity())
	require.NotNil(t, grants.Video)
	assert.True(t, grants.Video.RoomJoin)
	assert.Equal(t, roomID, grants.Video.Room)
	assert.True(t, grants.Video.GetCanPublishSource(livekit.TrackSource_MICROPHONE))
	assert.False(t, grants.Video.GetCanPublishSource(livekit.TrackSource_SCREEN_SHARE))
}

func TestParseWebhook_Disabled(t *testing.T) {
	// given
	svc := configuredService(t, "", "", "")

	// when
	_, err := svc.ParseWebhook("Bearer whatever", []byte("{}"))

	// then
	require.ErrorIs(t, err, ErrDisabled)
}

func TestParseWebhook_RejectsBadSignature(t *testing.T) {
	// given
	svc := configuredService(t, testURL, testKey, testSecret)

	// when
	_, err := svc.ParseWebhook("not-a-valid-token", []byte(`{"event":"room_finished"}`))

	// then
	require.Error(t, err)
}

func TestParseWebhook_VerifiesSignedEvent(t *testing.T) {
	// given
	svc := configuredService(t, testURL, testKey, testSecret)
	roomID := uuid.New().String()
	userID := uuid.New().String()
	body := []byte(fmt.Sprintf(`{"event":%q,"room":{"name":%q},"participant":{"identity":%q}}`, EventParticipantJoined, roomID, userID))

	sum := sha256.Sum256(body)
	sha := base64.StdEncoding.EncodeToString(sum[:])
	authToken, err := auth.NewAccessToken(testKey, testSecret).SetSha256(sha).ToJWT()
	require.NoError(t, err)

	// when
	event, err := svc.ParseWebhook(authToken, body)

	// then
	require.NoError(t, err)
	assert.Equal(t, EventParticipantJoined, event.Type)
	assert.Equal(t, roomID, event.RoomName)
	assert.Equal(t, userID, event.Identity)
}
