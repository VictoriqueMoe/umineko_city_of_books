package social

import (
	"context"
	"regexp"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
)

var MentionRegex = regexp.MustCompile(`\B@([a-zA-Z0-9_]+)`)

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
		_ = postRepo.AddEmbed(context.Background(), repository.NewEmbed{
			OwnerID:     ownerID,
			OwnerType:   ownerType,
			URL:         embed.URL,
			EmbedType:   embed.Type,
			Title:       embed.Title,
			Description: embed.Desc,
			Image:       embed.Image,
			SiteName:    embed.SiteName,
			VideoID:     embed.VideoID,
			SortOrder:   i,
		})
	}
}

func ProcessMentions(
	userRepo repository.UserRepository,
	blockSvc block.Service,
	notifSvc notification.Service,
	settingsSvc settings.Service,
	actorID uuid.UUID,
	body string,
	referenceID uuid.UUID,
	referenceType string,
	linkURL string,
) {
	matches := MentionRegex.FindAllStringSubmatch(body, 20)
	if len(matches) == 0 {
		return
	}

	actor, err := userRepo.GetByID(context.Background(), actorID)
	if err != nil || actor == nil || actor.IsBot {
		return
	}

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

		if blocked, _ := blockSvc.IsBlockedEither(context.Background(), actorID, mentioned.ID); blocked {
			continue
		}

		_ = notifSvc.Notify(context.Background(), dto.NotifyParams{
			RecipientID:   mentioned.ID,
			Type:          dto.NotifMention,
			ReferenceID:   referenceID,
			ReferenceType: referenceType,
			ActorID:       actorID,
			EmailActor:    actor.DisplayName,
			EmailAction:   "mentioned you",
			EmailLink:     linkURL,
		})
	}
}
