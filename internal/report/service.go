package report

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
)

type (
	Service interface {
		Create(ctx context.Context, reporterID uuid.UUID, req CreateReportRequest) error
		List(ctx context.Context, status string, limit, offset int) (*ReportListResponse, error)
		Resolve(ctx context.Context, id int, resolvedBy uuid.UUID, comment string) error
	}

	service struct {
		reportRepo  repository.ReportRepository
		roleRepo    repository.RoleRepository
		userRepo    repository.UserRepository
		notifSvc    notification.Service
		settingsSvc settings.Service
	}

	CreateReportRequest struct {
		TargetType string `json:"target_type"`
		TargetID   string `json:"target_id"`
		ContextID  string `json:"context_id,omitempty"`
		Reason     string `json:"reason"`
	}

	ReportResponse struct {
		ID             int    `json:"id"`
		ReporterName   string `json:"reporter_name"`
		ReporterAvatar string `json:"reporter_avatar"`
		TargetType     string `json:"target_type"`
		TargetID       string `json:"target_id"`
		ContextID      string `json:"context_id,omitempty"`
		Reason         string `json:"reason"`
		Status         string `json:"status"`
		ResolvedBy     string `json:"resolved_by,omitempty"`
		CreatedAt      string `json:"created_at"`
	}

	ReportListResponse struct {
		Reports []ReportResponse `json:"reports"`
		Total   int              `json:"total"`
		Limit   int              `json:"limit"`
		Offset  int              `json:"offset"`
	}
)

func NewService(reportRepo repository.ReportRepository, roleRepo repository.RoleRepository, userRepo repository.UserRepository, notifSvc notification.Service, settingsSvc settings.Service) Service {
	return &service{
		reportRepo:  reportRepo,
		roleRepo:    roleRepo,
		userRepo:    userRepo,
		notifSvc:    notifSvc,
		settingsSvc: settingsSvc,
	}
}

func (s *service) Create(ctx context.Context, reporterID uuid.UUID, req CreateReportRequest) error {
	if req.TargetType == "" || req.TargetID == "" || req.Reason == "" {
		return ErrMissingFields
	}

	_, err := s.reportRepo.Create(ctx, reporterID, req.TargetType, req.TargetID, req.ContextID, req.Reason)
	if err != nil {
		return fmt.Errorf("create report: %w", err)
	}

	targetUUID, err := uuid.Parse(req.TargetID)
	if err != nil {
		targetUUID = uuid.Nil
	}

	go func() {
		modIDs, err := s.roleRepo.GetUsersByRoles(ctx, []role.Role{
			authz.RoleSuperAdmin,
			authz.RoleAdmin,
			authz.RoleModerator,
		})
		if err != nil {
			return
		}

		reporterName := "Someone"
		if u, err := s.userRepo.GetByID(ctx, reporterID); err == nil && u != nil {
			reporterName = u.DisplayName
		}

		baseURL := s.settingsSvc.Get(ctx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/admin/reports", baseURL)
		subject, body := notification.ReportEmail(reporterName, req.TargetType, req.Reason, linkURL)

		params := make([]dto.NotifyParams, len(modIDs))
		for i, modID := range modIDs {
			params[i] = dto.NotifyParams{
				RecipientID:   modID,
				Type:          dto.NotifReport,
				ReferenceID:   targetUUID,
				ReferenceType: req.TargetType,
				ActorID:       reporterID,
				EmailSubject:  subject,
				EmailBody:     body,
			}
		}
		s.notifSvc.NotifyMany(ctx, params)
	}()

	return nil
}

func (s *service) List(ctx context.Context, status string, limit, offset int) (*ReportListResponse, error) {
	rows, total, err := s.reportRepo.List(ctx, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}

	reports := make([]ReportResponse, len(rows))
	for i, row := range rows {
		reports[i] = ReportResponse{
			ID:             row.ID,
			ReporterName:   row.ReporterName,
			ReporterAvatar: row.ReporterAvatar,
			TargetType:     row.TargetType,
			TargetID:       row.TargetID,
			ContextID:      row.ContextID,
			Reason:         row.Reason,
			Status:         row.Status,
			ResolvedBy:     row.ResolvedByName,
			CreatedAt:      row.CreatedAt,
		}
	}

	return &ReportListResponse{
		Reports: reports,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}

func (s *service) Resolve(ctx context.Context, id int, resolvedBy uuid.UUID, comment string) error {
	row, err := s.reportRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("resolve report: %w", err)
	}

	if err := s.reportRepo.Resolve(ctx, id, resolvedBy, comment); err != nil {
		return err
	}

	go func() {
		resolverName := "A moderator"
		if u, err := s.userRepo.GetByID(ctx, resolvedBy); err == nil && u != nil {
			resolverName = u.DisplayName
		}

		baseURL := s.settingsSvc.Get(ctx, config.SettingBaseURL)
		linkURL := baseURL
		subject, body := notification.ReportResolvedEmail(resolverName, row.TargetType, comment, linkURL)

		targetUUID, err := uuid.Parse(row.TargetID)
		if err != nil {
			targetUUID = uuid.Nil
		}

		message := fmt.Sprintf("resolved your report on a %s", row.TargetType)
		if comment != "" {
			message = fmt.Sprintf("%s: %s", message, comment)
		}

		s.notifSvc.Notify(ctx, dto.NotifyParams{
			RecipientID:   row.ReporterID,
			Type:          dto.NotifReportResolved,
			ReferenceID:   targetUUID,
			ReferenceType: row.TargetType,
			ActorID:       resolvedBy,
			Message:       message,
			EmailSubject:  subject,
			EmailBody:     body,
		})
	}()

	return nil
}
