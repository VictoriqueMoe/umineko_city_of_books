package push

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T, pushEnabled bool, credsFile string) (*service, *repository.MockDeviceTokenRepository, *settings.MockService) {
	repo := repository.NewMockDeviceTokenRepository(t)
	settingsSvc := settings.NewMockService(t)

	settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingPushEnabled).Return(pushEnabled).Maybe()

	svc := NewService(settingsSvc, repo, credsFile).(*service)

	return svc, repo, settingsSvc
}

func TestNewService_PushDisabled_NotEnabled(t *testing.T) {
	// given
	svc, _, _ := newTestService(t, false, "creds.json")

	// when
	enabled := svc.Enabled()

	// then
	assert.False(t, enabled)
}

func TestNewService_PushEnabledWithoutCreds_NotEnabled(t *testing.T) {
	// given
	svc, _, _ := newTestService(t, true, "")

	// when
	enabled := svc.Enabled()

	// then
	assert.False(t, enabled)
}

func TestRegisterToken_DelegatesToRepo(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
	}{
		{name: "success", repoErr: nil},
		{name: "repo error propagates", repoErr: errors.New("db down")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, repo, _ := newTestService(t, false, "")
			userID := uuid.New()
			repo.EXPECT().Upsert(mock.Anything, userID, "token-123", "android").Return(tc.repoErr)

			// when
			err := svc.RegisterToken(context.Background(), userID, "token-123", "android")

			// then
			assert.Equal(t, tc.repoErr, err)
		})
	}
}

func TestUnregisterToken_DelegatesToRepo(t *testing.T) {
	// given
	svc, repo, _ := newTestService(t, false, "")
	repo.EXPECT().Delete(mock.Anything, "token-123").Return(nil)

	// when
	err := svc.UnregisterToken(context.Background(), "token-123")

	// then
	require.NoError(t, err)
}

func TestSendToUser_ClientNil_NoRepoCalls(t *testing.T) {
	// given
	svc, _, _ := newTestService(t, false, "")

	// when
	svc.SendToUser(context.Background(), uuid.New(), Notification{Title: "hi", Body: "there"})

	// then
	// repo mock has no expectations, so any call would fail the test
}

func TestSettingListener_RebuildsOnlyForPushEnabledKey(t *testing.T) {
	tests := []struct {
		name      string
		key       config.SiteSettingKey
		wantBuild bool
	}{
		{name: "push key rebuilds", key: config.SettingPushEnabled.Key, wantBuild: true},
		{name: "other key ignored", key: config.SettingMaintenanceMode.Key, wantBuild: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo := repository.NewMockDeviceTokenRepository(t)
			settingsSvc := settings.NewMockService(t)

			wantCalls := 1
			if tc.wantBuild {
				wantCalls = 2
			}
			settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingPushEnabled).Return(false).Times(wantCalls)

			svc := NewService(settingsSvc, repo, "")
			listener := NewSettingListener(svc)

			// when
			listener.OnSettingChanged(tc.key, "true")

			// then
			settingsSvc.AssertExpectations(t)
		})
	}
}
