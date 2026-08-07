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
