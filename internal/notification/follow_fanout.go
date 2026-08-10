package notification

import (
	"context"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

type (
	FollowerNotifyParams struct {
		ActorID       uuid.UUID
		Type          dto.NotificationType
		ReferenceID   uuid.UUID
		ReferenceType string
		Action        string
		Title         string
	}
)

const (
	maxFollowerNotifTitle = 120
)

func clampTitle(s string) string {
	runes := []rune(s)
	if len(runes) <= maxFollowerNotifTitle {
		return s
	}

	return string(runes[:maxFollowerNotifTitle])
}

func SendFollowerNotification(
	ctx context.Context,
	followRepo repository.FollowRepository,
	notifSvc Service,
	p FollowerNotifyParams,
) {
	followerIDs, err := followRepo.GetFollowerIDsToNotify(ctx, p.ActorID)
	if err != nil {
		logger.Log.Warn().Err(err).Str("type", string(p.Type)).Str("actor", p.ActorID.String()).Msg("follower fan-out lookup failed")
		return
	}

	message := p.Action
	if p.Title != "" {
		message = p.Action + ": " + clampTitle(p.Title)
	}

	params := make([]dto.NotifyParams, 0, len(followerIDs))
	for _, followerID := range followerIDs {
		if followerID == p.ActorID {
			continue
		}

		params = append(params, dto.NotifyParams{
			RecipientID:   followerID,
			Type:          p.Type,
			ReferenceID:   p.ReferenceID,
			ReferenceType: p.ReferenceType,
			ActorID:       p.ActorID,
			Message:       message,
		})
	}

	notifSvc.NotifyMany(ctx, params)
}
