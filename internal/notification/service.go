package notification

import (
	"context"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/email"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

type (
	Service interface {
		Notify(ctx context.Context, params dto.NotifyParams) error
		NotifyMany(ctx context.Context, params []dto.NotifyParams)
		List(ctx context.Context, userID uuid.UUID, limit, offset int) (*dto.NotificationListResponse, error)
		MarkRead(ctx context.Context, id int, userID uuid.UUID) error
		MarkAllRead(ctx context.Context, userID uuid.UUID) error
		UnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	}

	service struct {
		repo     repository.NotificationRepository
		userRepo repository.UserRepository
		hub      *ws.Hub
		emailSvc email.Service
	}
)

func NewService(repo repository.NotificationRepository, userRepo repository.UserRepository, hub *ws.Hub, emailSvc email.Service) Service {
	return &service{
		repo:     repo,
		userRepo: userRepo,
		hub:      hub,
		emailSvc: emailSvc,
	}
}

func (s *service) Notify(ctx context.Context, params dto.NotifyParams) error {
	if params.RecipientID == params.ActorID {
		return nil
	}

	emailDupe, _ := s.repo.HasRecentDuplicate(ctx, params.RecipientID, params.Type, params.ReferenceID, params.ActorID)

	id, err := s.repo.Create(ctx, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message)
	if err != nil {
		return err
	}

	logger.Log.Debug().Str("type", string(params.Type)).Str("recipient", params.RecipientID.String()).Msg("notification sent")
	s.pushNotification(ctx, int(id), params.RecipientID)

	if params.Type != dto.NotifChatMessage && params.EmailSubject != "" && !emailDupe {
		s.sendEmail(ctx, params)
	}

	return nil
}

func (s *service) NotifyMany(ctx context.Context, params []dto.NotifyParams) {
	for _, p := range params {
		if err := s.Notify(ctx, p); err != nil {
			logger.Log.Warn().Err(err).Str("type", string(p.Type)).Str("recipient", p.RecipientID.String()).Msg("notify failed")
		}
	}
}

func (s *service) sendEmail(ctx context.Context, params dto.NotifyParams) {
	recipient, err := s.userRepo.GetByID(ctx, params.RecipientID)
	if err != nil || recipient == nil || recipient.Email == "" {
		return
	}

	if params.Type != dto.NotifReport && !recipient.EmailNotifications {
		return
	}

	if err := s.emailSvc.Send(ctx, recipient.Email, params.EmailSubject, params.EmailBody); err != nil {
		logger.Log.Warn().Err(err).Str("to", recipient.Email).Msg("failed to send notification email")
	}
}

func (s *service) pushNotification(ctx context.Context, notifID int, recipientID uuid.UUID) {
	rows, _, err := s.repo.ListByUser(ctx, recipientID, 1, 0)
	if err != nil || len(rows) == 0 {
		return
	}

	var row repository.NotificationRow
	for _, r := range rows {
		if r.ID == notifID {
			row = r
			break
		}
	}
	if row.ID == 0 {
		return
	}

	s.hub.SendToUser(recipientID, ws.Message{
		Type: "notification",
		Data: rowToDTO(row),
	})
}

func (s *service) List(ctx context.Context, userID uuid.UUID, limit, offset int) (*dto.NotificationListResponse, error) {
	rows, total, err := s.repo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	notifications := make([]dto.NotificationResponse, len(rows))
	for i, row := range rows {
		notifications[i] = rowToDTO(row)
	}

	return &dto.NotificationListResponse{
		Notifications: notifications,
		Total:         total,
		Limit:         limit,
		Offset:        offset,
	}, nil
}

func (s *service) MarkRead(ctx context.Context, id int, userID uuid.UUID) error {
	return s.repo.MarkRead(ctx, id, userID)
}

func (s *service) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *service) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func rowToDTO(row repository.NotificationRow) dto.NotificationResponse {
	return dto.NotificationResponse{
		ID:            row.ID,
		Type:          row.Type,
		ReferenceID:   row.ReferenceID,
		ReferenceType: row.ReferenceType,
		Actor: dto.UserResponse{
			ID:          row.ActorID,
			Username:    row.ActorUsername,
			DisplayName: row.ActorDisplayName,
			AvatarURL:   row.ActorAvatarURL,
			Role:        role.Role(row.ActorRole),
		},
		Message:   row.Message,
		Read:      row.Read,
		CreatedAt: row.CreatedAt,
	}
}
