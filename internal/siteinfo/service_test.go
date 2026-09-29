package siteinfo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/gameroom"
	"umineko_city_of_books/internal/mystery"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/usersecret"
	"umineko_city_of_books/internal/vanityrole"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	errBoom = errors.New("boom")
)

func newTestService(
	t *testing.T,
	stringValues map[config.SiteSettingKey]string,
	intValues map[config.SiteSettingKey]int,
	failAt string,
) Service {
	t.Helper()

	errAt := func(step string) error {
		if step == failAt {
			return errBoom
		}

		return nil
	}

	settingsSvc := settings.NewMockService(t)
	mysterySvc := mystery.NewMockService(t)
	gameRoomSvc := gameroom.NewMockService(t)
	vanityRoleSvc := vanityrole.NewMockService(t)
	userSecretSvc := usersecret.NewMockService(t)
	authSvc := auth.NewMockService(t)

	settingsSvc.EXPECT().
		Get(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, def *config.SiteSettingDef) string {
			return stringValues[def.Key]
		}).
		Maybe()
	settingsSvc.EXPECT().
		GetInt(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, def *config.SiteSettingDef) int {
			return intValues[def.Key]
		}).
		Maybe()
	settingsSvc.EXPECT().GetBool(mock.Anything, mock.Anything).Return(false).Maybe()

	mysterySvc.EXPECT().GetTopDetectiveIDs(mock.Anything).Return(nil, errAt("top detectives")).Maybe()
	mysterySvc.EXPECT().GetTopGMIDs(mock.Anything).Return(nil, errAt("top game masters")).Maybe()
	gameRoomSvc.EXPECT().GetTopWinnerIDs(mock.Anything, mock.Anything).Return(nil, errAt("top game winners")).Maybe()
	vanityRoleSvc.EXPECT().List(mock.Anything).Return(nil, errAt("vanity roles")).Maybe()
	vanityRoleSvc.EXPECT().GetAllAssignments(mock.Anything).Return(nil, errAt("vanity assignments")).Maybe()
	userSecretSvc.EXPECT().GetUserIDsWithSecret(mock.Anything, mock.Anything).Return(nil, errAt("secret holders")).Maybe()
	userSecretSvc.EXPECT().IsSolvedByAnyone(mock.Anything, mock.Anything).Return(false, errAt("secret solved")).Maybe()
	authSvc.EXPECT().EmailEnabled(mock.Anything).Return(false).Maybe()

	return NewService(settingsSvc, mysterySvc, gameRoomSvc, vanityRoleSvc, userSecretSvc, authSvc)
}

func TestGet_ChatbotMemorySettingsAreExposed(t *testing.T) {
	tests := []struct {
		name              string
		contextMessages   int
		maxReplyChain     int
		wantContextMsgs   int
		wantMaxReplyChain int
	}{
		{name: "defaults", contextMessages: 20, maxReplyChain: 25, wantContextMsgs: 20, wantMaxReplyChain: 25},
		{name: "admin raised both", contextMessages: 50, maxReplyChain: 80, wantContextMsgs: 50, wantMaxReplyChain: 80},
		{name: "not swapped", contextMessages: 7, maxReplyChain: 9, wantContextMsgs: 7, wantMaxReplyChain: 9},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc := newTestService(t, nil, map[config.SiteSettingKey]int{
				config.SettingChatbotContextMessages.Key: tc.contextMessages,
				config.SettingChatbotMaxReplyChain.Key:   tc.maxReplyChain,
			}, "")

			// when
			got, err := svc.Get(t.Context())

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantContextMsgs, got.ChatbotContextMsgs)
			assert.Equal(t, tc.wantMaxReplyChain, got.ChatbotMaxReplyChain)
		})
	}
}

func TestGet_SerialisesBothChatbotMemoryKeys(t *testing.T) {
	// given
	svc := newTestService(t, nil, map[config.SiteSettingKey]int{
		config.SettingChatbotContextMessages.Key: 20,
		config.SettingChatbotMaxReplyChain.Key:   25,
	}, "")

	// when
	info, err := svc.Get(t.Context())
	require.NoError(t, err)

	raw, err := json.Marshal(info)
	require.NoError(t, err)

	payload := make(map[string]any)
	require.NoError(t, json.Unmarshal(raw, &payload))

	// then
	assert.Equal(t, float64(20), payload["chatbot_context_messages"])
	assert.Equal(t, float64(25), payload["chatbot_max_reply_chain"])
}

func TestGet_AFailedBadgeReadIsSurfacedInsteadOfDroppingTheBadges(t *testing.T) {
	steps := []string{"top detectives", "top game masters", "top game winners", "vanity roles", "vanity assignments", "secret holders", "secret solved"}

	for _, failAt := range steps {
		t.Run("the "+failAt+" read failing", func(t *testing.T) {
			// given
			svc := newTestService(t, nil, nil, failAt)

			// when
			_, err := svc.Get(t.Context())

			// then
			require.ErrorIs(t, err, errBoom)
		})
	}
}

func TestGet_LeaksNoSecretSetting(t *testing.T) {
	// given
	sentinels := make(map[config.SiteSettingKey]string)
	for _, def := range config.AllSiteSettings {
		if def.Secret {
			sentinels[def.Key] = fmt.Sprintf("LEAKED-SENTINEL-%s", def.Key)
		}
	}
	require.Contains(t, sentinels, config.SettingChatbotAPIKey.Key)
	require.Contains(t, sentinels, config.SettingChatbotAdminKey.Key)

	svc := newTestService(t, sentinels, map[config.SiteSettingKey]int{
		config.SettingChatbotContextMessages.Key: 20,
		config.SettingChatbotMaxReplyChain.Key:   25,
	}, "")

	// when
	info, err := svc.Get(t.Context())
	require.NoError(t, err)

	raw, err := json.Marshal(info)
	require.NoError(t, err)

	// then
	body := string(raw)
	for key, sentinel := range sentinels {
		assert.NotContains(t, body, sentinel, "secret setting %s reached the public site-info payload", key)
		assert.NotContains(t, body, string(key), "secret setting key %s reached the public site-info payload", key)
	}
}
