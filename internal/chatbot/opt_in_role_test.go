package chatbot

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOptInRoleValidator(t *testing.T) {
	const roleID = "characters"

	cases := []struct {
		name    string
		value   string
		row     *repository.VanityRoleRow
		rowErr  error
		perms   map[string][]string
		wantErr error
	}{
		{
			name:  "empty clears the setting",
			value: "",
		},
		{
			name:  "whitespace clears the setting",
			value: "   ",
		},
		{
			name:  "vanity role carrying use_chatbot",
			value: roleID,
			row:   &repository.VanityRoleRow{ID: roleID, Label: "Characters"},
			perms: map[string][]string{roleID: {string(authz.PermUseChatbot)}},
		},
		{
			name:    "role does not exist",
			value:   "ghost",
			row:     nil,
			wantErr: ErrOptInRoleNotFound,
		},
		{
			name:    "system vanity role",
			value:   roleID,
			row:     &repository.VanityRoleRow{ID: roleID, Label: "Bot", IsSystem: true},
			wantErr: ErrOptInRoleIsSystem,
		},
		{
			name:    "vanity role without use_chatbot",
			value:   roleID,
			row:     &repository.VanityRoleRow{ID: roleID, Label: "Characters"},
			perms:   map[string][]string{roleID: {string(authz.PermManageBannedWords)}},
			wantErr: ErrOptInRoleNoChatbot,
		},
		{
			name:    "vanity role with no permissions at all",
			value:   roleID,
			row:     &repository.VanityRoleRow{ID: roleID, Label: "Characters"},
			perms:   map[string][]string{},
			wantErr: ErrOptInRoleNoChatbot,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			vanityRepo := repository.NewMockVanityRoleRepository(t)
			permRepo := repository.NewMockPermissionRepository(t)
			vanityRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(tc.row, tc.rowErr).Maybe()
			permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).Return(tc.perms, nil).Maybe()
			validate := OptInRoleValidator(vanityRepo, permRepo)

			// when
			err := validate(context.Background(), tc.value)

			// then
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}

			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestOptInRoleValidator_RepositoryErrors(t *testing.T) {
	cases := []struct {
		name    string
		rowErr  error
		permErr error
		want    string
	}{
		{"vanity role lookup fails", errors.New("boom"), nil, "get vanity role"},
		{"permission lookup fails", nil, errors.New("boom"), "get vanity role permissions"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			vanityRepo := repository.NewMockVanityRoleRepository(t)
			permRepo := repository.NewMockPermissionRepository(t)
			vanityRepo.EXPECT().GetByID(mock.Anything, "characters").Return(&repository.VanityRoleRow{ID: "characters"}, tc.rowErr)
			permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).Return(nil, tc.permErr).Maybe()
			validate := OptInRoleValidator(vanityRepo, permRepo)

			// when
			err := validate(context.Background(), "characters")

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func newTestMigrator(t *testing.T, current string) (*OptInRoleMigrator, *repository.MockVanityRoleRepository) {
	vanityRepo := repository.NewMockVanityRoleRepository(t)
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingChatbotOptInRole).Return(current)

	return NewOptInRoleMigrator(vanityRepo, settingsSvc), vanityRepo
}

func TestOptInRoleMigrator_Plan(t *testing.T) {
	cases := []struct {
		name     string
		current  string
		key      config.SiteSettingKey
		value    string
		wantFrom string
		wantTo   string
		wantOK   bool
	}{
		{"another setting is ignored", "a", config.SettingChatbotEnabled.Key, "b", "", "", false},
		{"unchanged role", "a", config.SettingChatbotOptInRole.Key, "a", "", "", false},
		{"unchanged role with padding", "a", config.SettingChatbotOptInRole.Key, "  a  ", "", "", false},
		{"role set for the first time", "", config.SettingChatbotOptInRole.Key, "b", "", "", false},
		{"role cleared leaves holders alone", "a", config.SettingChatbotOptInRole.Key, "", "", "", false},
		{"role swapped", "a", config.SettingChatbotOptInRole.Key, "b", "a", "b", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			migrator, _ := newTestMigrator(t, tc.current)

			// when
			from, to, ok := migrator.plan(tc.key, tc.value)

			// then
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantFrom, from)
			assert.Equal(t, tc.wantTo, to)
		})
	}
}

func TestOptInRoleMigrator_PlanRemembersTheLatestRole(t *testing.T) {
	// given
	migrator, _ := newTestMigrator(t, "a")

	// when
	_, _, first := migrator.plan(config.SettingChatbotOptInRole.Key, "b")
	_, _, second := migrator.plan(config.SettingChatbotOptInRole.Key, "b")

	// then
	assert.True(t, first)
	assert.False(t, second)
}

func moveSpec(userID uuid.UUID, from, to string) repository.VanityRoleMove {
	return repository.VanityRoleMove{UserID: userID, FromRoleID: from, ToRoleID: to}
}

func TestOptInRoleMigrator_MigrateMovesHolders(t *testing.T) {
	// given
	migrator, vanityRepo := newTestMigrator(t, "a")
	holders := []repository.VanityRoleUserRow{{UserID: uuid.New()}, {UserID: uuid.New()}}
	vanityRepo.EXPECT().GetUsersForRole(mock.Anything, "a", "", optInRolePageSize, 0).Return(holders, len(holders), nil)
	for _, holder := range holders {
		vanityRepo.EXPECT().MoveUserRole(mock.Anything, moveSpec(holder.UserID, "a", "b")).Return(nil)
	}

	// when
	migrator.Migrate(context.Background(), "a", "b")

	// then
	vanityRepo.AssertExpectations(t)
}

func TestOptInRoleMigrator_MigratePagesThroughHolders(t *testing.T) {
	// given
	migrator, vanityRepo := newTestMigrator(t, "a")
	first := make([]repository.VanityRoleUserRow, optInRolePageSize)
	for i := range first {
		first[i] = repository.VanityRoleUserRow{UserID: uuid.New()}
	}
	second := []repository.VanityRoleUserRow{{UserID: uuid.New()}}
	total := len(first) + len(second)
	vanityRepo.EXPECT().GetUsersForRole(mock.Anything, "a", "", optInRolePageSize, 0).Return(first, total, nil)
	vanityRepo.EXPECT().GetUsersForRole(mock.Anything, "a", "", optInRolePageSize, optInRolePageSize).Return(second, total, nil)
	vanityRepo.EXPECT().MoveUserRole(mock.Anything, mock.Anything).Return(nil).Times(total)

	// when
	migrator.Migrate(context.Background(), "a", "b")

	// then
	vanityRepo.AssertExpectations(t)
}

func TestOptInRoleMigrator_MigrateContinuesAfterFailures(t *testing.T) {
	// given
	migrator, vanityRepo := newTestMigrator(t, "a")
	moveFails := uuid.New()
	succeeds := uuid.New()
	alsoSucceeds := uuid.New()
	holders := []repository.VanityRoleUserRow{{UserID: moveFails}, {UserID: succeeds}, {UserID: alsoSucceeds}}
	vanityRepo.EXPECT().GetUsersForRole(mock.Anything, "a", "", optInRolePageSize, 0).Return(holders, len(holders), nil)
	vanityRepo.EXPECT().MoveUserRole(mock.Anything, moveSpec(moveFails, "a", "b")).Return(errors.New("boom"))
	vanityRepo.EXPECT().MoveUserRole(mock.Anything, moveSpec(succeeds, "a", "b")).Return(nil)
	vanityRepo.EXPECT().MoveUserRole(mock.Anything, moveSpec(alsoSucceeds, "a", "b")).Return(nil)

	// when
	migrator.Migrate(context.Background(), "a", "b")

	// then
	vanityRepo.AssertExpectations(t)
}

func TestOptInRoleMigrator_MigrateStopsWhenHoldersCannotBeListed(t *testing.T) {
	// given
	migrator, vanityRepo := newTestMigrator(t, "a")
	vanityRepo.EXPECT().GetUsersForRole(mock.Anything, "a", "", optInRolePageSize, 0).Return(nil, 0, errors.New("boom"))

	// when
	migrator.Migrate(context.Background(), "a", "b")

	// then
	vanityRepo.AssertNotCalled(t, "MoveUserRole", mock.Anything, mock.Anything)
}
