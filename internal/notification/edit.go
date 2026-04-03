package notification

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"

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
	settingsSvc settings.Service,
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
	baseURL := settingsSvc.Get(ctx, config.SettingBaseURL)
	linkURL := baseURL + p.LinkPath
	subject, body := NotifEmail(actor.DisplayName, fmt.Sprintf("edited your %s", p.ContentType), "", linkURL)

	notifSvc.Notify(ctx, dto.NotifyParams{
		RecipientID:   p.AuthorID,
		Type:          dto.NotifContentEdited,
		ReferenceID:   p.ReferenceID,
		ReferenceType: p.ReferenceType,
		ActorID:       p.EditorID,
		Message:       message,
		EmailSubject:  subject,
		EmailBody:     body,
	})
}
