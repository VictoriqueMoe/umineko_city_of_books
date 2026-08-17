package user

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type (
	Service interface {
		NewAccountSpec(ctx context.Context, username, email, password, displayName string) (repository.NewAccount, error)
		GetByID(ctx context.Context, id uuid.UUID) (*dto.UserResponse, error)
		ListStaff(ctx context.Context) ([]*dto.UserResponse, error)
		ValidateCredentials(ctx context.Context, username, password string) (*dto.UserResponse, error)
		CheckUsernameAvailable(ctx context.Context, username string) error

		UpdateIP(ctx context.Context, id uuid.UUID, ip string) error
		UpdateGameBoardSort(ctx context.Context, id uuid.UUID, sort string) error
		UpdateAppearance(ctx context.Context, id uuid.UUID, theme, font string, wideLayout bool) error
		IsChatbotOptedIn(ctx context.Context, id uuid.UUID) (bool, error)
		SetChatbotOptIn(ctx context.Context, id uuid.UUID, optIn bool) error

		GetDetectiveRawScore(ctx context.Context, id uuid.UUID) (int, error)
		GetGMRawScore(ctx context.Context, id uuid.UUID) (int, error)
		UpdateMysteryScoreAdjustment(ctx context.Context, actorID uuid.UUID, id uuid.UUID, adjustment int) error
		UpdateGMScoreAdjustment(ctx context.Context, actorID uuid.UUID, id uuid.UUID, adjustment int) error
	}

	service struct {
		repo       repository.UserRepository
		roleRepo   repository.RoleRepository
		vanityRepo repository.VanityRoleRepository
		auditRepo  repository.AuditLogRepository
		authz      authz.Service
		settings   settings.Service
	}
)

func NewService(repo repository.UserRepository, roleRepo repository.RoleRepository, vanityRepo repository.VanityRoleRepository, auditRepo repository.AuditLogRepository, authzService authz.Service, settingsSvc settings.Service) Service {
	return &service{repo: repo, roleRepo: roleRepo, vanityRepo: vanityRepo, auditRepo: auditRepo, authz: authzService, settings: settingsSvc}
}

func (s *service) audit(ctx context.Context, entry repository.NewAuditEntry) {
	if err := s.auditRepo.Create(ctx, entry); err != nil {
		logger.Log.Error().Err(err).Str("action", string(entry.Action)).Msg("failed to write audit log")
	}
}

func (s *service) NewAccountSpec(ctx context.Context, username, email, password, displayName string) (repository.NewAccount, error) {
	count, err := s.repo.Count(ctx)
	if err != nil {
		return repository.NewAccount{}, fmt.Errorf("count users: %w", err)
	}

	displayName = ClampDisplayName(displayName)
	if displayName == "" {
		displayName = username
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return repository.NewAccount{}, fmt.Errorf("hash password: %w", err)
	}

	spec := repository.NewAccount{
		User: repository.NewUser{
			Username:     username,
			Email:        email,
			PasswordHash: string(hash),
			DisplayName:  displayName,
			HomePage:     "landing",
			DMsEnabled:   true,
		},
	}

	if count == 0 {
		spec.Role = authz.RoleSuperAdmin
	}

	return spec, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*dto.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user.ToResponse(), nil
}

func (s *service) ListStaff(ctx context.Context) ([]*dto.UserResponse, error) {
	ids, err := s.roleRepo.GetUsersByRoles(ctx, []role.Role{authz.RoleSuperAdmin, authz.RoleAdmin})
	if err != nil {
		return nil, fmt.Errorf("get staff ids: %w", err)
	}
	if len(ids) == 0 {
		return []*dto.UserResponse{}, nil
	}

	users, err := s.repo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get staff users: %w", err)
	}

	staff := make([]*dto.UserResponse, 0, len(users))
	for i := range users {
		if users[i].BannedAt != nil {
			continue
		}
		staff = append(staff, users[i].ToResponse())
	}

	slices.SortFunc(staff, func(a, b *dto.UserResponse) int {
		if a.Role != b.Role {
			if a.Role == authz.RoleSuperAdmin {
				return -1
			}
			return 1
		}
		return strings.Compare(strings.ToLower(a.DisplayName), strings.ToLower(b.DisplayName))
	})
	return staff, nil
}

func (s *service) ValidateCredentials(ctx context.Context, username, password string) (*dto.UserResponse, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("validate credentials: %w", err)
	}
	if user == nil || user.IsBot {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user.ToResponse(), nil
}

func (s *service) CheckUsernameAvailable(ctx context.Context, username string) error {
	exists, err := s.repo.ExistsByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("check username: %w", err)
	}
	if exists {
		return ErrUsernameTaken
	}
	return nil
}

func (s *service) UpdateIP(ctx context.Context, id uuid.UUID, ip string) error {
	return s.repo.UpdateIP(ctx, id, ip)
}

func (s *service) UpdateGameBoardSort(ctx context.Context, id uuid.UUID, sort string) error {
	return s.repo.UpdateGameBoardSort(ctx, id, sort)
}

func (s *service) UpdateAppearance(ctx context.Context, id uuid.UUID, theme, font string, wideLayout bool) error {
	return s.repo.UpdateAppearance(ctx, id, theme, font, wideLayout)
}

func (s *service) IsChatbotOptedIn(ctx context.Context, id uuid.UUID) (bool, error) {
	roleID := s.chatbotOptInRole(ctx)
	if roleID == "" {
		return false, nil
	}

	rows, err := s.vanityRepo.GetRolesForUser(ctx, id)
	if err != nil {
		return false, fmt.Errorf("get vanity roles: %w", err)
	}

	for _, row := range rows {
		if row.ID == roleID {
			return true, nil
		}
	}

	return false, nil
}

func (s *service) SetChatbotOptIn(ctx context.Context, id uuid.UUID, optIn bool) error {
	roleID := s.chatbotOptInRole(ctx)
	if roleID == "" {
		return ErrChatbotOptInUnavailable
	}

	if !optIn {
		if err := s.vanityRepo.UnassignFromUser(ctx, id, roleID); err != nil {
			return fmt.Errorf("revoke chatbot opt-in role: %w", err)
		}

		s.audit(ctx, repository.NewAuditEntry{
			ActorID:    id,
			Action:     repository.AuditActionUnassignVanityRole,
			TargetType: repository.AuditTargetVanityRole,
			TargetID:   roleID,
			SubjectID:  id,
		})

		return nil
	}

	if !s.settings.GetBool(ctx, config.SettingChatbotEnabled) {
		return ErrChatbotOptInUnavailable
	}
	if !s.settings.GetBool(ctx, config.SettingChatbotRequirePermission) {
		return ErrChatbotOptInUnavailable
	}

	if err := s.assertOptInRoleGrantable(ctx, roleID); err != nil {
		return err
	}

	if err := s.vanityRepo.AssignToUser(ctx, id, roleID); err != nil {
		return fmt.Errorf("grant chatbot opt-in role: %w", err)
	}

	s.audit(ctx, repository.NewAuditEntry{
		ActorID:    id,
		Action:     repository.AuditActionAssignVanityRole,
		TargetType: repository.AuditTargetVanityRole,
		TargetID:   roleID,
		SubjectID:  id,
	})

	return nil
}

func (s *service) assertOptInRoleGrantable(ctx context.Context, roleID string) error {
	row, err := s.vanityRepo.GetByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("get chatbot opt-in role: %w", err)
	}

	if row == nil || row.IsSystem {
		return ErrChatbotOptInUnavailable
	}

	return nil
}

func (s *service) chatbotOptInRole(ctx context.Context) string {
	return strings.TrimSpace(s.settings.Get(ctx, config.SettingChatbotOptInRole))
}

func (s *service) GetDetectiveRawScore(ctx context.Context, id uuid.UUID) (int, error) {
	return s.repo.GetDetectiveRawScore(ctx, id)
}

func (s *service) GetGMRawScore(ctx context.Context, id uuid.UUID) (int, error) {
	return s.repo.GetGMRawScore(ctx, id)
}

func (s *service) UpdateMysteryScoreAdjustment(ctx context.Context, actorID uuid.UUID, id uuid.UUID, adjustment int) error {
	previous, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if previous == nil {
		return ErrUserNotFound
	}

	if err := s.repo.UpdateMysteryScoreAdjustment(ctx, id, adjustment); err != nil {
		return err
	}

	s.audit(ctx, repository.NewAuditEntry{
		ActorID:    actorID,
		Action:     repository.AuditActionMysteryScoreAdjust,
		TargetType: repository.AuditTargetUser,
		TargetID:   id.String(),
		Details:    fmt.Sprintf("%d -> %d", previous.MysteryScoreAdjustment, adjustment),
		SubjectID:  id,
	})

	return nil
}

func (s *service) UpdateGMScoreAdjustment(ctx context.Context, actorID uuid.UUID, id uuid.UUID, adjustment int) error {
	previous, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if previous == nil {
		return ErrUserNotFound
	}

	if err := s.repo.UpdateGMScoreAdjustment(ctx, id, adjustment); err != nil {
		return err
	}

	s.audit(ctx, repository.NewAuditEntry{
		ActorID:    actorID,
		Action:     repository.AuditActionGMScoreAdjust,
		TargetType: repository.AuditTargetUser,
		TargetID:   id.String(),
		Details:    fmt.Sprintf("%d -> %d", previous.GMScoreAdjustment, adjustment),
		SubjectID:  id,
	})

	return nil
}
