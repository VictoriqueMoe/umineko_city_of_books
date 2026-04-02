package social

import (
	"context"
	"regexp"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
)

var MentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_]+)`)

func ProcessEmbeds(postRepo repository.PostRepository, ownerID string, ownerType string, body string) {
	urls := media.ExtractURLs(body)
	for i, rawURL := range urls {
		if i >= 5 {
			break
		}
		embed := media.ParseEmbed(rawURL)
		if embed == nil {
			continue
		}
		_ = postRepo.AddEmbed(
			context.Background(), ownerID, ownerType,
			embed.URL, embed.Type, embed.Title, embed.Desc, embed.Image, embed.SiteName, embed.VideoID, i,
		)
	}
}

func ProcessMentions(
	userRepo repository.UserRepository,
	notifSvc notification.Service,
	settingsSvc settings.Service,
	actorID uuid.UUID,
	body string,
	referenceID uuid.UUID,
	referenceType string,
	linkURL string,
) {
	matches := MentionRegex.FindAllStringSubmatch(body, 20)
	seen := make(map[string]bool)

	for _, m := range matches {
		username := m[1]
		if seen[username] {
			continue
		}
		seen[username] = true

		mentioned, err := userRepo.GetByUsername(context.Background(), username)
		if err != nil || mentioned == nil || mentioned.ID == actorID {
			continue
		}

		actor, err := userRepo.GetByID(context.Background(), actorID)
		if err != nil || actor == nil {
			continue
		}

		baseURL := settingsSvc.Get(context.Background(), config.SettingBaseURL)
		subject, emailBody := notification.NotifEmail(actor.DisplayName, "mentioned you", "", baseURL+linkURL)
		_ = notifSvc.Notify(context.Background(), dto.NotifyParams{
			RecipientID:   mentioned.ID,
			Type:          dto.NotifMention,
			ReferenceID:   referenceID,
			ReferenceType: referenceType,
			ActorID:       actorID,
			EmailSubject:  subject,
			EmailBody:     emailBody,
		})
	}
}
