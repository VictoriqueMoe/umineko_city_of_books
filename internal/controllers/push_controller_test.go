package controllers

import (
	"context"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/notification/push"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newPushHarness(t *testing.T) (*testutil.Harness, *repository.MockDeviceTokenRepository) {
	h := testutil.NewHarness(t)
	repo := repository.NewMockDeviceTokenRepository(t)

	h.SettingsService.EXPECT().GetBool(mock.Anything, config.SettingPushEnabled).Return(false).Maybe()

	s := &Service{
		PushService:  push.NewService(h.SettingsService, repo, ""),
		AuthSession:  h.SessionManager,
		AuthzService: h.AuthzService,
	}
	for _, setup := range s.getAllPushRoutes() {
		setup(h.App)
	}

	return h, repo
}

func TestUnregisterDeviceToken_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newPushHarness, "DELETE", "/push/device", dto.DeviceTokenRequest{Token: "tok-1"})
}

func TestUnregisterDeviceToken_ScopesDeleteToSessionUser(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "owner unregisters their own device"},
		{name: "another user submits the victim's token"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, repo := newPushHarness(t)
			caller := uuid.New()
			h.ExpectValidSession("valid-cookie", caller)

			var gotUserID uuid.UUID
			repo.EXPECT().Delete(mock.Anything, mock.Anything, "victim-token").
				RunAndReturn(func(_ context.Context, userID uuid.UUID, _ string) error {
					gotUserID = userID
					return nil
				})

			// when
			status, _ := h.NewRequest("DELETE", "/push/device").
				WithCookie("valid-cookie").
				WithJSONBody(dto.DeviceTokenRequest{Token: "victim-token"}).
				Do()

			// then
			require.Equal(t, http.StatusOK, status)
			assert.Equal(t, caller, gotUserID)
		})
	}
}

func TestRegisterDeviceToken_BindsSessionUserNotRequestBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "plain registration", body: `{"token":"tok-handover","platform":"android"}`},
		{name: "registration smuggling a user_id", body: `{"token":"tok-handover","platform":"android","user_id":"00000000-0000-0000-0000-00000000dead"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, repo := newPushHarness(t)
			caller := uuid.New()
			h.ExpectValidSession("valid-cookie", caller)

			var gotUserID uuid.UUID
			repo.EXPECT().Upsert(mock.Anything, mock.Anything, "tok-handover", "android").
				RunAndReturn(func(_ context.Context, userID uuid.UUID, _ string, _ string) error {
					gotUserID = userID
					return nil
				})

			// when
			status, _ := h.NewRequest("POST", "/push/device").
				WithCookie("valid-cookie").
				WithRawBody(tc.body, "application/json").
				Do()

			// then
			require.Equal(t, http.StatusOK, status)
			assert.Equal(t, caller, gotUserID)
		})
	}
}
