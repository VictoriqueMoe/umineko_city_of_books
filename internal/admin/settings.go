package admin

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

func (s *service) GetSettings(ctx context.Context) (*dto.SettingsResponse, error) {
	all := s.settingsSvc.GetAll(ctx)

	result := make(map[string]string, len(all))
	for k, v := range all {
		if def, ok := config.SettingByKey(k); ok && def.Secret && v != "" {
			result[string(k)] = config.SecretMask
			continue
		}
		result[string(k)] = v
	}

	return &dto.SettingsResponse{Settings: result}, nil
}

func (s *service) UpdateSettings(ctx context.Context, actorID uuid.UUID, settings map[string]string) error {
	current := s.settingsSvc.GetAll(ctx)

	typed := make(map[config.SiteSettingKey]string, len(settings))
	for k, v := range settings {
		key := config.SiteSettingKey(k)

		def, ok := config.SettingByKey(key)
		if !ok {
			continue
		}

		if def.Secret && v == config.SecretMask {
			continue
		}

		typed[key] = v
	}

	changedKeys := make([]string, 0, len(typed))
	for k, v := range typed {
		if current[k] != v {
			changedKeys = append(changedKeys, string(k))
		}
	}
	slices.Sort(changedKeys)

	if err := s.settingsSvc.SetMultiple(ctx, typed, actorID); err != nil {
		return fmt.Errorf("update settings: %w", err)
	}

	if newRules, ok := typed[config.SettingRulesPage.Key]; ok && newRules != current[config.SettingRulesPage.Key] {
		s.hub.Broadcast(ws.Message{
			Type: "rules_page_changed",
			Data: map[string]any{},
		})
	}

	details := fmt.Sprintf("changed %d of %d submitted settings", len(changedKeys), len(settings))
	s.auditDetails(ctx, actorID, audit.ActionUpdateSettings, audit.TargetSettings, strings.Join(changedKeys, ","), details)

	return nil
}

func (s *service) SendTestEmail(ctx context.Context, actorID uuid.UUID) error {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return fmt.Errorf("get actor: %w", err)
	}
	if actor == nil {
		return ErrUserNotFound
	}
	if actor.Email == "" {
		return ErrNoEmailAddress
	}

	siteName := s.settingsSvc.Get(ctx, config.SettingSiteName)
	subject := "Test email from " + siteName
	body := fmt.Sprintf("<p>This is a test email from <bdi>%s</bdi> confirming your email settings are working.</p>", siteName)

	if err := s.emailSvc.SendTest(ctx, actor.Email, subject, body); err != nil {
		return fmt.Errorf("send test email: %w", err)
	}

	s.audit(ctx, actorID, audit.ActionSendTestEmail, audit.TargetSettings, "")
	return nil
}

func (s *service) GetAuditLog(ctx context.Context, action audit.Action, page bounds.Page) (*dto.AuditLogListResponse, error) {
	entries, total, err := s.auditRepo.List(ctx, spec.AuditLogListing{Action: action, Page: page})
	if err != nil {
		return nil, fmt.Errorf("get audit log: %w", err)
	}

	return &dto.AuditLogListResponse{
		Entries: toAuditLogEntries(entries),
		Total:   total,
		Limit:   page.Limit(),
		Offset:  page.Offset(),
	}, nil
}

func toAuditLogEntries(entries []audit.Entry) []dto.AuditLogEntryResponse {
	items := make([]dto.AuditLogEntryResponse, len(entries))
	for i, e := range entries {
		items[i] = dto.AuditLogEntryResponse{
			ID:              e.ID,
			ActorID:         e.ActorID,
			ActorName:       e.ActorName,
			Action:          string(e.Action),
			TargetType:      string(e.TargetType),
			TargetID:        e.TargetID,
			Details:         e.Details,
			CreatedAt:       e.CreatedAt,
			SubjectID:       e.SubjectID,
			SubjectName:     e.SubjectName,
			SubjectUsername: e.SubjectUsername,
		}
	}
	return items
}
