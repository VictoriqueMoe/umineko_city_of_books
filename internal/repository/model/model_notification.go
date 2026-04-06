package model

import (
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	NotificationRow struct {
		ID               int
		UserID           uuid.UUID
		Type             string
		ReferenceID      uuid.UUID
		ReferenceType    string
		ActorID          uuid.UUID
		Message          string
		Read             bool
		CreatedAt        string
		ActorUsername    string
		ActorDisplayName string
		ActorAvatarURL   string
		ActorRole        string
	}
)

func (n *NotificationRow) ToResponse() dto.NotificationResponse {
	return dto.NotificationResponse{
		ID:            n.ID,
		Type:          n.Type,
		ReferenceID:   n.ReferenceID,
		ReferenceType: n.ReferenceType,
		Actor: dto.UserResponse{
			ID:          n.ActorID,
			Username:    n.ActorUsername,
			DisplayName: n.ActorDisplayName,
			AvatarURL:   n.ActorAvatarURL,
			Role:        role.Role(n.ActorRole),
		},
		Message:   n.Message,
		Read:      n.Read,
		CreatedAt: n.CreatedAt,
	}
}
