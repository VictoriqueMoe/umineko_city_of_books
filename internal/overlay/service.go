package overlay

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/ws"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

var ErrNotConnected = errors.New("no overlay connection")

type (
	Service interface {
		Handler() fiber.Handler
		Token(ctx context.Context, userID uuid.UUID) (string, error)
		ResetToken(ctx context.Context, userID uuid.UUID) (string, error)
		Validate(ctx context.Context, token string) (uuid.UUID, error)
		DispatchNotification(recipientID uuid.UUID, resp dto.NotificationResponse)
		TestFire(userID uuid.UUID) error
		ConnectURL(ctx context.Context, userID uuid.UUID) (string, error)
		BuildSEF(ctx context.Context, userID uuid.UUID) (string, error)
		IsConnected(userID uuid.UUID) bool
	}

	service struct {
		repo        repository.OverlayTokenRepository
		hub         *ws.Hub
		settingsSvc settings.Service
	}

	overlayPayload struct {
		Actor       string `json:"actor"`
		DisplayName string `json:"display_name"`
		Avatar      string `json:"avatar,omitempty"`
		Text        string `json:"text"`
		At          string `json:"at"`
	}

	eventMapping struct {
		event  string
		action string
	}
)

var overlayEvents = map[dto.NotificationType]eventMapping{
	dto.NotifPostLiked:      {"post_liked", "liked your post"},
	dto.NotifNewFollower:    {"new_follower", "started following you"},
	dto.NotifPostCommented:  {"post_commented", "commented on your post"},
	dto.NotifTheoryUpvote:   {"theory_upvote", "upvoted your theory"},
	dto.NotifTheoryResponse: {"theory_response", "responded to your theory"},
	dto.NotifCommentLiked:   {"comment_liked", "liked your comment"},
	dto.NotifMention:        {"mention", "mentioned you"},
	dto.NotifContentShared:  {"content_shared", "shared your content"},
	dto.NotifArtLiked:       {"art_liked", "liked your art"},
}

func NewService(repo repository.OverlayTokenRepository, hub *ws.Hub, settingsSvc settings.Service) Service {
	return &service{
		repo:        repo,
		hub:         hub,
		settingsSvc: settingsSvc,
	}
}

func (s *service) Token(ctx context.Context, userID uuid.UUID) (string, error) {
	existing, err := s.repo.GetByUser(ctx, userID)
	if err != nil {
		return "", err
	}
	if existing != "" {
		return existing, nil
	}
	return s.ResetToken(ctx, userID)
}

func (s *service) ResetToken(ctx context.Context, userID uuid.UUID) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}
	if err := s.repo.Upsert(ctx, userID, token); err != nil {
		return "", err
	}
	return token, nil
}

func (s *service) Validate(ctx context.Context, token string) (uuid.UUID, error) {
	if token == "" {
		return uuid.Nil, nil
	}
	return s.repo.GetUserByToken(ctx, token)
}

func (s *service) DispatchNotification(recipientID uuid.UUID, resp dto.NotificationResponse) {
	mapping, ok := overlayEvents[resp.Type]
	if !ok {
		return
	}
	if !s.hub.IsOnline(recipientID) {
		return
	}
	s.hub.SendToUser(recipientID, ws.Message{
		Type: mapping.event,
		Data: overlayPayload{
			Actor:       resp.Actor.Username,
			DisplayName: resp.Actor.DisplayName,
			Avatar:      resp.Actor.AvatarURL,
			Text:        mapping.action,
			At:          resp.CreatedAt,
		},
	})
}

func (s *service) TestFire(userID uuid.UUID) error {
	if !s.hub.IsOnline(userID) {
		return ErrNotConnected
	}
	s.hub.SendToUser(userID, ws.Message{
		Type: "test",
		Data: overlayPayload{
			Actor:       "beatrice",
			DisplayName: "Beatrice",
			Text:        "sent a test overlay",
			At:          time.Now().UTC().Format(time.RFC3339),
		},
	})
	return nil
}

func (s *service) ConnectURL(ctx context.Context, userID uuid.UUID) (string, error) {
	token, err := s.Token(ctx, userID)
	if err != nil {
		return "", err
	}
	return s.urlForToken(ctx, token), nil
}

func (s *service) BuildSEF(ctx context.Context, userID uuid.UUID) (string, error) {
	connectURL, err := s.ConnectURL(ctx, userID)
	if err != nil {
		return "", err
	}
	siteName := s.settingsSvc.Get(ctx, config.SettingSiteName)
	return renderSEF(connectURL, siteName)
}

func (s *service) IsConnected(userID uuid.UUID) bool {
	return s.hub.IsOnline(userID)
}

func (s *service) urlForToken(ctx context.Context, token string) string {
	base := strings.TrimSuffix(s.settingsSvc.Get(ctx, config.SettingBaseURL), "/")
	if strings.HasPrefix(base, "https://") {
		base = "wss://" + strings.TrimPrefix(base, "https://")
	} else if strings.HasPrefix(base, "http://") {
		base = "ws://" + strings.TrimPrefix(base, "http://")
	}
	return base + "/api/v1/overlay?token=" + url.QueryEscape(token)
}

func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
