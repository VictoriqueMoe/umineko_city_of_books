package notification

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

type EditNotifyParams struct {
	AuthorID      uuid.UUID
	EditorID      uuid.UUID
	ContentType   string
	ReferenceID   uuid.UUID
	ReferenceType string
	LinkPath      string
}

func SendEditNotification(
	ctx context.Context,
	userRepo repository.UserRepository,
	notifSvc Service,
	p EditNotifyParams,
) {
	if p.AuthorID == p.EditorID {
		return
	}

	actor, err := userRepo.GetByID(ctx, p.EditorID)
	if err != nil || actor == nil {
		return
	}

	message := fmt.Sprintf("your %s has been edited", p.ContentType)

	notifSvc.Notify(ctx, dto.NotifyParams{
		RecipientID:   p.AuthorID,
		Type:          dto.NotifContentEdited,
		ReferenceID:   p.ReferenceID,
		ReferenceType: p.ReferenceType,
		ActorID:       p.EditorID,
		Message:       message,
		EmailActor:    actor.DisplayName,
		EmailAction:   fmt.Sprintf("edited your %s", p.ContentType),
		EmailLink:     p.LinkPath,
	})
}
