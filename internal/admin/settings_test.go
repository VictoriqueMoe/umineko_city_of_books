package admin

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/email"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetUserAuditLog_ScopesToTarget(t *testing.T) {
	// given
	svc, m := newTestService(t)
	target := uuid.New()
	m.auditRepo.EXPECT().ListForUser(mock.Anything, spec.AuditLogUserListing{UserID: target, Page: bounds.NewPage(20, 0)}).Return([]audit.Entry{
		{ID: 1, Action: audit.ActionBanUser, TargetType: audit.TargetUser, TargetID: target.String()},
	}, 1, nil)

	// when
	result, err := svc.GetUserAuditLog(context.Background(), target, bounds.NewPage(20, 0))

	// then
	require.NoError(t, err)
	require.Len(t, result.Entries, 1)
	assert.Equal(t, "ban_user", result.Entries[0].Action)
	assert.Equal(t, 1, result.Total)
}

func TestGetSettings_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.settingsSvc.EXPECT().GetAll(mock.Anything).Return(map[config.SiteSettingKey]string{
		"site_name": "umineko",
		"foo":       "bar",
	})

	// when
	got, err := svc.GetSettings(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, "umineko", got.Settings["site_name"])
	assert.Equal(t, "bar", got.Settings["foo"])
}

func TestUpdateSettings_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.settingsSvc.EXPECT().GetAll(mock.Anything).Return(map[config.SiteSettingKey]string{"site_name": "old"})
	m.settingsSvc.EXPECT().SetMultiple(mock.Anything, mock.Anything, actor).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry audit.NewEntry) bool {
		return entry.ActorID == actor && entry.Action == audit.ActionUpdateSettings && entry.TargetType == audit.TargetSettings && entry.TargetID == "site_name"
	})).Return(nil)

	// when
	err := svc.UpdateSettings(context.Background(), actor, map[string]string{"site_name": "umineko"})

	// then
	require.NoError(t, err)
}

func TestUpdateSettings_Error(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.settingsSvc.EXPECT().GetAll(mock.Anything).Return(map[config.SiteSettingKey]string{})
	m.settingsSvc.EXPECT().SetMultiple(mock.Anything, mock.Anything, actor).Return(errors.New("boom"))

	// when
	err := svc.UpdateSettings(context.Background(), actor, map[string]string{"site_name": "umineko"})

	// then
	require.Error(t, err)
}

func TestGetSettings_MasksSecrets(t *testing.T) {
	tests := []struct {
		name   string
		key    config.SiteSettingKey
		stored string
		want   string
	}{
		{name: "smtp password is masked", key: config.SettingSMTPPassword.Key, stored: "hunter2", want: config.SecretMask},
		{name: "livekit api secret is masked", key: config.SettingLiveKitAPISecret.Key, stored: "lk-secret", want: config.SecretMask},
		{name: "cloudflare api token is masked", key: config.SettingCloudflareAPIToken.Key, stored: "cf-token", want: config.SecretMask},
		{name: "turnstile secret key is masked", key: config.SettingTurnstileSecretKey.Key, stored: "ts-secret", want: config.SecretMask},
		{name: "valkey url is masked", key: config.SettingValkeyURL.Key, stored: "redis://user:pw@valkey:6379", want: config.SecretMask},
		{name: "unset secret stays empty", key: config.SettingSMTPPassword.Key, stored: "", want: ""},
		{name: "site name is returned verbatim", key: config.SettingSiteName.Key, stored: "City of Books", want: "City of Books"},
		{name: "smtp username is returned verbatim", key: config.SettingSMTPUsername.Key, stored: "postmaster", want: "postmaster"},
		{name: "livekit api key is returned verbatim", key: config.SettingLiveKitAPIKey.Key, stored: "lk-key", want: "lk-key"},
		{name: "turnstile site key is returned verbatim", key: config.SettingTurnstileSiteKey.Key, stored: "ts-site", want: "ts-site"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.settingsSvc.EXPECT().GetAll(mock.Anything).Return(map[config.SiteSettingKey]string{tc.key: tc.stored})

			// when
			got, err := svc.GetSettings(context.Background())

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.Settings[string(tc.key)])
		})
	}
}

func TestUpdateSettings_SecretsAndAudit(t *testing.T) {
	tests := []struct {
		name         string
		stored       map[config.SiteSettingKey]string
		submitted    map[string]string
		wantStored   map[config.SiteSettingKey]string
		wantTargetID string
	}{
		{
			name:         "masked secret is left untouched",
			stored:       map[config.SiteSettingKey]string{"smtp_password": "hunter2", "site_name": "old"},
			submitted:    map[string]string{"smtp_password": config.SecretMask, "site_name": "new"},
			wantStored:   map[config.SiteSettingKey]string{"site_name": "new"},
			wantTargetID: "site_name",
		},
		{
			name:         "real secret value replaces the stored one",
			stored:       map[config.SiteSettingKey]string{"smtp_password": "hunter2"},
			submitted:    map[string]string{"smtp_password": "correct-horse"},
			wantStored:   map[config.SiteSettingKey]string{"smtp_password": "correct-horse"},
			wantTargetID: "smtp_password",
		},
		{
			name:         "mask on a non secret setting is stored verbatim",
			stored:       map[config.SiteSettingKey]string{"site_name": "old"},
			submitted:    map[string]string{"site_name": config.SecretMask},
			wantStored:   map[config.SiteSettingKey]string{"site_name": config.SecretMask},
			wantTargetID: "site_name",
		},
		{
			name:         "audit names every changed key in order",
			stored:       map[config.SiteSettingKey]string{"site_name": "old", "smtp_host": "mail", "base_url": "http://a"},
			submitted:    map[string]string{"site_name": "new", "smtp_host": "mail", "base_url": "http://b"},
			wantStored:   map[config.SiteSettingKey]string{"site_name": "new", "smtp_host": "mail", "base_url": "http://b"},
			wantTargetID: "base_url,site_name",
		},
		{
			name:         "unchanged submission names no keys",
			stored:       map[config.SiteSettingKey]string{"site_name": "old"},
			submitted:    map[string]string{"site_name": "old"},
			wantStored:   map[config.SiteSettingKey]string{"site_name": "old"},
			wantTargetID: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			actor := uuid.New()
			m.settingsSvc.EXPECT().GetAll(mock.Anything).Return(tc.stored)
			m.settingsSvc.EXPECT().SetMultiple(mock.Anything, tc.wantStored, actor).Return(nil)
			m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry audit.NewEntry) bool {
				return entry.ActorID == actor && entry.Action == audit.ActionUpdateSettings && entry.TargetType == audit.TargetSettings && entry.TargetID == tc.wantTargetID
			})).Return(nil)

			// when
			err := svc.UpdateSettings(context.Background(), actor, tc.submitted)

			// then
			require.NoError(t, err)
		})
	}
}

func TestUpdateSettings_AuditDetailsHideValues(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	var gotDetails string
	m.settingsSvc.EXPECT().GetAll(mock.Anything).Return(map[config.SiteSettingKey]string{"smtp_password": "hunter2"})
	m.settingsSvc.EXPECT().SetMultiple(mock.Anything, mock.Anything, actor).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry audit.NewEntry) bool {
		return entry.ActorID == actor && entry.Action == audit.ActionUpdateSettings && entry.TargetType == audit.TargetSettings && entry.TargetID == "smtp_password"
	})).
		Run(func(_ context.Context, entry audit.NewEntry, _ ...*sql.Tx) { gotDetails = entry.Details }).
		Return(nil)

	// when
	err := svc.UpdateSettings(context.Background(), actor, map[string]string{"smtp_password": "correct-horse"})

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, gotDetails)
	assert.NotContains(t, gotDetails, "hunter2")
	assert.NotContains(t, gotDetails, "correct-horse")
}

func TestSendTestEmail_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, actor).Return(&model.User{ID: actor, Email: "admin@example.com"}, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("City of Books")
	m.emailSvc.EXPECT().SendTest(mock.Anything, "admin@example.com", mock.Anything, mock.Anything).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{ActorID: actor, Action: audit.ActionSendTestEmail, TargetType: audit.TargetSettings, TargetID: "", Details: ""}).Return(nil)

	// when
	err := svc.SendTestEmail(context.Background(), actor)

	// then
	require.NoError(t, err)
}

func TestSendTestEmail_NoEmailAddress(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, actor).Return(&model.User{ID: actor, Email: ""}, nil)

	// when
	err := svc.SendTestEmail(context.Background(), actor)

	// then
	require.ErrorIs(t, err, ErrNoEmailAddress)
}

func TestSendTestEmail_SendError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, actor).Return(&model.User{ID: actor, Email: "admin@example.com"}, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("City of Books")
	m.emailSvc.EXPECT().SendTest(mock.Anything, "admin@example.com", mock.Anything, mock.Anything).Return(email.ErrNotConfigured)

	// when
	err := svc.SendTestEmail(context.Background(), actor)

	// then
	require.Error(t, err)
}

func TestGetAuditLog_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.auditRepo.EXPECT().List(mock.Anything, spec.AuditLogListing{Action: audit.ActionBanUser, Page: bounds.NewPage(20, 0)}).Return([]audit.Entry{
		{ID: 1, ActorID: actor, ActorName: "victorique", Action: audit.ActionBanUser, TargetType: audit.TargetUser, TargetID: "t", Details: "d", CreatedAt: "now"},
	}, 1, nil)

	// when
	got, err := svc.GetAuditLog(context.Background(), audit.ActionBanUser, bounds.NewPage(20, 0))

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	require.Len(t, got.Entries, 1)
	assert.Equal(t, "ban_user", got.Entries[0].Action)
	assert.Equal(t, "victorique", got.Entries[0].ActorName)
}

func TestGetAuditLog_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.auditRepo.EXPECT().List(mock.Anything, spec.AuditLogListing{Action: audit.Action(""), Page: bounds.NewPage(20, 0)}).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.GetAuditLog(context.Background(), "", bounds.NewPage(20, 0))

	// then
	require.Error(t, err)
}
