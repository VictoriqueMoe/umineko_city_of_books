package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validSettings() map[SiteSettingKey]string {
	all := make(map[SiteSettingKey]string, len(AllSiteSettings))
	for _, def := range AllSiteSettings {
		all[def.Key] = def.Default
	}

	all[SettingMaxBodySize.Key] = "104857600"
	all[SettingMaxVideoSize.Key] = "52428800"
	all[SettingMaxGeneralSize.Key] = "52428800"

	return all
}

func TestValidateSettings_HyperbeamRegion(t *testing.T) {
	cases := []struct {
		name    string
		region  string
		wantErr bool
	}{
		{"north america", "NA", false},
		{"europe", "EU", false},
		{"asia", "AS", false},
		{"blank falls back to the provider default", "", false},
		{"surrounding whitespace is tolerated", "  EU  ", false},
		{"a lowercase region is refused", "eu", true},
		{"an invented region is refused", "atlantis", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			all := validSettings()
			all[SettingHyperbeamRegion.Key] = tc.region

			// when
			err := ValidateSettings(all)

			// then
			if !tc.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), "NA, EU or AS")
		})
	}
}

func TestValidateSettings_ChatbotOptInRolePairing(t *testing.T) {
	cases := []struct {
		name       string
		enabled    string
		restricted string
		role       string
		wantErr    bool
	}{
		{"restriction off and no role", "true", "false", "", false},
		{"restriction off with a role", "true", "false", "characters", false},
		{"restriction on with a role", "true", "true", "characters", false},
		{"restriction on and no role", "true", "true", "", true},
		{"restriction on and whitespace role", "true", "true", "   ", true},
		{"restriction on and no role cannot lock the admin out while chatbots are off", "false", "true", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			all := validSettings()
			all[SettingChatbotEnabled.Key] = tc.enabled
			all[SettingChatbotAPIKey.Key] = "sk-test"
			all[SettingChatbotModel.Key] = "gpt-5.6-luna"
			all[SettingChatbotRequirePermission.Key] = tc.restricted
			all[SettingChatbotOptInRole.Key] = tc.role

			// when
			err := ValidateSettings(all)

			// then
			if !tc.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), "opt-in role")
		})
	}
}

func TestChatbotOptInRoleSettingIsRegistered(t *testing.T) {
	// given
	key := SettingChatbotOptInRole.Key

	// when
	def, ok := SettingByKey(key)

	// then
	require.True(t, ok)
	assert.Equal(t, SiteSettingKey("chatbot_opt_in_role"), def.Key)
	assert.Equal(t, TypeString, def.Type)
	assert.Empty(t, def.Default)
	assert.False(t, def.Secret)
}

func TestValidateSettings_StorageBackend(t *testing.T) {
	completeS3 := map[*SiteSettingDef]string{
		SettingS3Region:          "auto",
		SettingS3Bucket:          "uploads",
		SettingS3AccessKeyID:     "key",
		SettingS3SecretAccessKey: "secret",
	}

	cases := []struct {
		name    string
		backend string
		s3      map[*SiteSettingDef]string
		wantErr string
	}{
		{name: "local needs nothing else", backend: "local"},
		{name: "s3 with every required field", backend: "s3", s3: completeS3},
		{name: "s3 without credentials is refused", backend: "s3", s3: map[*SiteSettingDef]string{SettingS3Region: "auto", SettingS3Bucket: "uploads"}, wantErr: "s3 storage requires"},
		{name: "unknown backend is refused", backend: "ftp", wantErr: "storage backend must be"},
		{name: "blank backend is refused", backend: "", wantErr: "storage backend must be"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			all := validSettings()
			all[SettingStorageBackend.Key] = tc.backend
			for def, value := range tc.s3 {
				all[def.Key] = value
			}

			// when
			err := ValidateSettings(all)

			// then
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestStorageSettingDefinitions(t *testing.T) {
	// then
	assert.Equal(t, string(StorageBackendLocal), SettingStorageBackend.Default)
	assert.True(t, SettingS3SecretAccessKey.Secret)
	assert.False(t, SettingS3AccessKeyID.Secret)
	assert.Equal(t, TypeBool, SettingS3ForcePathStyle.Type)
}
