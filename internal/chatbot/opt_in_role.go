package chatbot

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
)

const (
	optInRolePageSize    = 100
	optInMigrationWindow = 10 * time.Minute
)

type (
	OptInRoleMigrator struct {
		vanityRepo repository.VanityRoleRepository
		mu         sync.Mutex
		current    string
	}
)

func OptInRoleValidator(vanityRepo repository.VanityRoleRepository, permRepo repository.PermissionRepository) settings.Validator {
	return func(ctx context.Context, value string) error {
		return validateOptInRole(ctx, vanityRepo, permRepo, value)
	}
}

func validateOptInRole(ctx context.Context, vanityRepo repository.VanityRoleRepository, permRepo repository.PermissionRepository, value string) error {
	roleID := strings.TrimSpace(value)
	if roleID == "" {
		return nil
	}

	row, err := vanityRepo.GetByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("get vanity role: %w", err)
	}
	if row == nil {
		return ErrOptInRoleNotFound
	}
	if row.IsSystem {
		return ErrOptInRoleIsSystem
	}

	table, err := permRepo.GetVanityRolePermissions(ctx)
	if err != nil {
		return fmt.Errorf("get vanity role permissions: %w", err)
	}

	if !slices.Contains(table[roleID], string(authz.PermUseChatbot)) {
		return ErrOptInRoleNoChatbot
	}

	return nil
}

func NewOptInRoleMigrator(vanityRepo repository.VanityRoleRepository, settingsSvc settings.Service) *OptInRoleMigrator {
	current := strings.TrimSpace(settingsSvc.Get(context.Background(), config.SettingChatbotOptInRole))

	return &OptInRoleMigrator{vanityRepo: vanityRepo, current: current}
}

func (m *OptInRoleMigrator) OnSettingChanged(key config.SiteSettingKey, value string) {
	from, to, ok := m.plan(key, value)
	if !ok {
		return
	}

	go m.Migrate(context.Background(), from, to)
}

func (m *OptInRoleMigrator) plan(key config.SiteSettingKey, value string) (string, string, bool) {
	if key != config.SettingChatbotOptInRole.Key {
		return "", "", false
	}

	next := strings.TrimSpace(value)

	m.mu.Lock()
	previous := m.current
	m.current = next
	m.mu.Unlock()

	if previous == next {
		return "", "", false
	}

	if previous == "" || next == "" {
		logger.Log.Info().Str("from", previous).Str("to", next).Msg("chatbot opt-in role changed, nothing to migrate")

		return "", "", false
	}

	return previous, next, true
}

func (m *OptInRoleMigrator) Migrate(parent context.Context, from, to string) {
	ctx, cancel := context.WithTimeout(parent, optInMigrationWindow)
	defer cancel()

	holders, err := m.holders(ctx, from)
	if err != nil {
		logger.Log.Error().Err(err).Str("from", from).Str("to", to).Msg("chatbot opt-in role migration could not list holders")

		return
	}

	moved := 0
	failed := 0

	for _, userID := range holders {
		if err := m.vanityRepo.AssignToUser(ctx, userID, to); err != nil {
			failed++
			logger.Log.Error().Err(err).Str("user_id", userID.String()).Str("to", to).Msg("chatbot opt-in role migration could not grant the new role")

			continue
		}

		if err := m.vanityRepo.UnassignFromUser(ctx, userID, from); err != nil {
			failed++
			logger.Log.Error().Err(err).Str("user_id", userID.String()).Str("from", from).Msg("chatbot opt-in role migration could not revoke the old role")

			continue
		}

		moved++
	}

	logger.Log.Info().Str("from", from).Str("to", to).Int("holders", len(holders)).Int("moved", moved).Int("failed", failed).Msg("chatbot opt-in role migration finished")
}

func (m *OptInRoleMigrator) holders(ctx context.Context, roleID string) ([]uuid.UUID, error) {
	var ids []uuid.UUID

	for offset := 0; ; offset += optInRolePageSize {
		rows, total, err := m.vanityRepo.GetUsersForRole(ctx, roleID, "", optInRolePageSize, offset)
		if err != nil {
			return nil, fmt.Errorf("get users for role: %w", err)
		}

		for _, row := range rows {
			ids = append(ids, row.UserID)
		}

		if len(rows) < optInRolePageSize || len(ids) >= total {
			return ids, nil
		}
	}
}
