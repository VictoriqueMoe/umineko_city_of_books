package notification

import (
	"context"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/ws"
)

type (
	Service interface {
		NotifyTheoryResponse(ctx context.Context, theoryID, actorID int) error
		NotifyResponseReply(ctx context.Context, parentResponseID, theoryID, actorID int) error
		NotifyTheoryUpvote(ctx context.Context, theoryID, actorID int) error
		NotifyResponseUpvote(ctx context.Context, responseID, theoryID, actorID int) error
		List(ctx context.Context, userID, limit, offset int) (*dto.NotificationListResponse, error)
		MarkRead(ctx context.Context, id, userID int) error
		MarkAllRead(ctx context.Context, userID int) error
		UnreadCount(ctx context.Context, userID int) (int, error)
	}

	service struct {
		repo       repository.NotificationRepository
		theoryRepo repository.TheoryRepository
		hub        *ws.Hub
	}
)

func NewService(repo repository.NotificationRepository, theoryRepo repository.TheoryRepository, hub *ws.Hub) Service {
	return &service{
		repo:       repo,
		theoryRepo: theoryRepo,
		hub:        hub,
	}
}

func (s *service) NotifyTheoryResponse(ctx context.Context, theoryID, actorID int) error {
	return s.notifyTheoryAuthor(ctx, dto.NotifTheoryResponse, theoryID, theoryID, actorID)
}

func (s *service) NotifyTheoryUpvote(ctx context.Context, theoryID, actorID int) error {
	return s.notifyTheoryAuthor(ctx, dto.NotifTheoryUpvote, theoryID, theoryID, actorID)
}

func (s *service) NotifyResponseReply(ctx context.Context, parentResponseID, theoryID, actorID int) error {
	return s.notifyResponseAuthor(ctx, dto.NotifResponseReply, parentResponseID, theoryID, actorID)
}

func (s *service) NotifyResponseUpvote(ctx context.Context, responseID, theoryID, actorID int) error {
	return s.notifyResponseAuthor(ctx, dto.NotifResponseUpvote, responseID, theoryID, actorID)
}

func (s *service) notifyTheoryAuthor(ctx context.Context, notifType string, referenceID, theoryID, actorID int) error {
	recipientID, err := s.theoryRepo.GetTheoryAuthorID(ctx, theoryID)
	if err != nil {
		return err
	}
	return s.send(ctx, recipientID, notifType, referenceID, theoryID, actorID)
}

func (s *service) notifyResponseAuthor(ctx context.Context, notifType string, referenceID, theoryID, actorID int) error {
	recipientID, _, err := s.theoryRepo.GetResponseInfo(ctx, referenceID)
	if err != nil {
		return err
	}
	return s.send(ctx, recipientID, notifType, referenceID, theoryID, actorID)
}

func (s *service) send(ctx context.Context, recipientID int, notifType string, referenceID, theoryID, actorID int) error {
	if recipientID == actorID {
		return nil
	}

	dupe, err := s.repo.HasRecentDuplicate(ctx, recipientID, notifType, referenceID, actorID)
	if err != nil {
		return err
	}
	if dupe {
		return nil
	}

	id, err := s.repo.Create(ctx, recipientID, notifType, referenceID, theoryID, actorID)
	if err != nil {
		return err
	}

	s.pushNotification(ctx, int(id), recipientID)
	return nil
}

func (s *service) pushNotification(ctx context.Context, notifID, recipientID int) {
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

func (s *service) List(ctx context.Context, userID, limit, offset int) (*dto.NotificationListResponse, error) {
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

func (s *service) MarkRead(ctx context.Context, id, userID int) error {
	return s.repo.MarkRead(ctx, id, userID)
}

func (s *service) MarkAllRead(ctx context.Context, userID int) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *service) UnreadCount(ctx context.Context, userID int) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func rowToDTO(row repository.NotificationRow) dto.NotificationResponse {
	return dto.NotificationResponse{
		ID:          row.ID,
		Type:        row.Type,
		ReferenceID: row.ReferenceID,
		TheoryID:    row.TheoryID,
		TheoryTitle: row.TheoryTitle,
		Actor: dto.UserResponse{
			ID:          row.ActorID,
			Username:    row.ActorUsername,
			DisplayName: row.ActorDisplayName,
			AvatarURL:   row.ActorAvatarURL,
		},
		Read:      row.Read,
		CreatedAt: row.CreatedAt,
	}
}
