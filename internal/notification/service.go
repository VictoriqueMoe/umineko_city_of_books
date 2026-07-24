package notification

import (
	"context"
	"strconv"
	"strings"
	"time"
	"umineko_city_of_books/internal/notification/push"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/email"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

const (
	notificationRetention = 90 * 24 * time.Hour
	pruneBatchSize        = 5000
)

type (
	Service interface {
		Notify(ctx context.Context, params dto.NotifyParams) error
		NotifyMany(ctx context.Context, params []dto.NotifyParams)
		List(ctx context.Context, userID uuid.UUID, page bounds.Page) (*dto.NotificationListResponse, error)
		MarkRead(ctx context.Context, id int, userID uuid.UUID) error
		MarkAllRead(ctx context.Context, userID uuid.UUID) error
		UnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
		PruneOld(ctx context.Context) (int, error)
	}

	OverlayDispatcher interface {
		DispatchNotification(recipientID uuid.UUID, resp dto.NotificationResponse)
	}

	service struct {
		repo        repository.NotificationRepository
		userRepo    repository.UserRepository
		blockRepo   repository.BlockRepository
		hub         *ws.Hub
		emailSvc    email.Service
		pushSvc     push.Service
		settingsSvc settings.Service
		overlay     OverlayDispatcher
	}
)

var notifText = map[dto.NotificationType]string{
	dto.NotifTheoryResponse:           "responded to your theory",
	dto.NotifResponseReply:            "replied to your response",
	dto.NotifTheoryUpvote:             "upvoted your theory",
	dto.NotifResponseUpvote:           "upvoted your response",
	dto.NotifChatMessage:              "sent you a message",
	dto.NotifChatRoomMessage:          "sent a message in a chat room",
	dto.NotifReport:                   "reported content",
	dto.NotifReportResolved:           "resolved your report",
	dto.NotifNewFollower:              "started following you",
	dto.NotifPostLiked:                "liked your post",
	dto.NotifPostCommented:            "commented on your post",
	dto.NotifPostCommentReply:         "replied to your comment",
	dto.NotifMention:                  "mentioned you",
	dto.NotifArtLiked:                 "liked your art",
	dto.NotifArtCommented:             "commented on your art",
	dto.NotifArtCommentReply:          "replied to your comment",
	dto.NotifCommentLiked:             "liked your comment",
	dto.NotifContentEdited:            "edited your content",
	dto.NotifMysteryAttempt:           "made an attempt on your mystery",
	dto.NotifMysteryReply:             "replied in a thread on your mystery",
	dto.NotifMysteryVote:              "voted on your attempt",
	dto.NotifMysterySolved:            "chose your attempt as the winner!",
	dto.NotifMysteryPaused:            "paused a mystery you are playing",
	dto.NotifMysteryUnpaused:          "resumed a mystery you are playing",
	dto.NotifMysteryGmAway:            "marked themselves as away on a mystery you are playing",
	dto.NotifMysteryGmBack:            "is back on a mystery you are playing",
	dto.NotifMysterySolvedAll:         "a mystery you were playing has been solved",
	dto.NotifMysteryCommentReply:      "replied to your comment on a mystery",
	dto.NotifMysteryPrivateClue:       "revealed a private red truth to you",
	dto.NotifFanficCommented:          "commented on your fanfic",
	dto.NotifFanficCommentReply:       "replied to your comment on a fanfic",
	dto.NotifFanficCommentLiked:       "liked your comment on a fanfic",
	dto.NotifFanficFavourited:         "favourited your fanfic",
	dto.NotifShipCommented:            "commented on your ship",
	dto.NotifShipCommentReply:         "replied to your comment",
	dto.NotifShipCommentLiked:         "liked your comment",
	dto.NotifOCCommented:              "commented on your OC",
	dto.NotifOCCommentReply:           "replied to your comment",
	dto.NotifOCCommentLiked:           "liked your comment",
	dto.NotifOCFavourited:             "favourited your OC",
	dto.NotifAnnouncementCommented:    "commented on your announcement",
	dto.NotifAnnouncementCommentReply: "replied to your comment",
	dto.NotifAnnouncementCommentLiked: "liked your comment",
	dto.NotifSuggestionPosted:         "posted a site suggestion",
	dto.NotifSuggestionResolved:       "marked your suggestion as done",
	dto.NotifContentShared:            "shared your content",
	dto.NotifJournalUpdate:            "posted a new update on a journal you follow",
	dto.NotifJournalCommented:         "commented on your journal",
	dto.NotifJournalCommentReply:      "replied to your comment on a journal",
	dto.NotifJournalCommentLiked:      "liked your comment",
	dto.NotifJournalFollowed:          "started following your journal",
	dto.NotifJournalArchived:          "your journal was archived after 7 days of inactivity",
	dto.NotifChatMention:              "mentioned you in a chat room",
	dto.NotifChatRoomInvite:           "added you to a chat room",
	dto.NotifChatReply:                "replied to your message",
	dto.NotifChatRoomBanned:           "banned you from a chat room",
	dto.NotifChatRoomKicked:           "kicked you from a chat room",
	dto.NotifChatRoomUnbanned:         "unbanned you from a chat room",
	dto.NotifSecretCommentReply:       "replied to your comment on a hunt",
	dto.NotifSecretCommented:          "commented on a hunt you're watching",
	dto.NotifSecretCommentLiked:       "liked your comment on a hunt",
	dto.NotifSecretSolvedByOther:      "solved a hunt before you could",
	dto.NotifGameInvite:               "invited you to a game",
	dto.NotifGameYourTurn:             "it's your move",
	dto.NotifGameFinished:             "your game has ended",
}

func NewService(repo repository.NotificationRepository, userRepo repository.UserRepository, blockRepo repository.BlockRepository, hub *ws.Hub, emailSvc email.Service, pushSvc push.Service, settingsSvc settings.Service, overlay OverlayDispatcher) Service {
	return &service{
		repo:        repo,
		userRepo:    userRepo,
		blockRepo:   blockRepo,
		hub:         hub,
		emailSvc:    emailSvc,
		pushSvc:     pushSvc,
		settingsSvc: settingsSvc,
		overlay:     overlay,
	}
}

func isChatRoomNotif(t dto.NotificationType) bool {
	switch t {
	case dto.NotifChatRoomMessage, dto.NotifChatMention, dto.NotifChatReply:
		return true
	default:
		return false
	}
}

func survivesBlock(t dto.NotificationType) bool {
	switch t {
	case dto.NotifReport, dto.NotifReportResolved, dto.NotifContentEdited, dto.NotifSuggestionResolved,
		dto.NotifChatRoomBanned, dto.NotifChatRoomKicked, dto.NotifChatRoomUnbanned,
		dto.NotifMysteryPaused, dto.NotifMysteryUnpaused, dto.NotifMysteryGmAway, dto.NotifMysteryGmBack,
		dto.NotifMysteryPrivateClue, dto.NotifGameYourTurn, dto.NotifGameFinished:
		return true
	default:
		return false
	}
}

func (s *service) blockedBetween(ctx context.Context, params dto.NotifyParams) bool {
	if params.ActorID == uuid.Nil || survivesBlock(params.Type) {
		return false
	}

	blocked, err := s.blockRepo.IsBlockedEither(ctx, params.RecipientID, params.ActorID)
	if err != nil {
		logger.Log.Warn().Err(err).Str("type", string(params.Type)).Msg("block check failed, delivering notification")
		return false
	}

	return blocked
}

func (s *service) Notify(ctx context.Context, params dto.NotifyParams) error {
	if params.RecipientID == params.ActorID {
		return nil
	}

	if s.blockedBetween(ctx, params) {
		return nil
	}

	willConsiderEmail := !isChatRoomNotif(params.Type) && params.EmailAction != ""
	var emailDupe bool
	if willConsiderEmail {
		emailDupe, _ = s.repo.HasRecentDuplicate(ctx, params.RecipientID, params.Type, params.ReferenceID, params.ActorID)
	}

	id, err := s.repo.Create(ctx, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message)
	if err != nil {
		return err
	}

	s.pushNotification(ctx, int(id), params.RecipientID)

	if willConsiderEmail && !emailDupe {
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

	subject, body := s.buildEmail(ctx, params)

	if err := s.emailSvc.Send(ctx, recipient.Email, subject, body); err != nil {
		logger.Log.Warn().Err(err).Str("to", recipient.Email).Msg("failed to send notification email")
	}
}

func (s *service) buildEmail(ctx context.Context, params dto.NotifyParams) (subject string, body string) {
	siteName := s.settingsSvc.Get(ctx, config.SettingSiteName)
	link := s.absoluteLink(ctx, params.EmailLink)

	switch params.Type {
	case dto.NotifReport:
		return reportEmail(params.EmailActor, params.EmailAction, params.EmailTitle, link, siteName)
	case dto.NotifReportResolved:
		return reportResolvedEmail(params.EmailActor, params.EmailAction, params.EmailTitle, link, siteName)
	default:
		return notifEmail(params.EmailActor, params.EmailAction, params.EmailTitle, link, siteName)
	}
}

func (s *service) absoluteLink(ctx context.Context, link string) string {
	if link == "" {
		return ""
	}

	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		return link
	}

	return s.settingsSvc.Get(ctx, config.SettingBaseURL) + link
}

func (s *service) pushNotification(ctx context.Context, notifID int, recipientID uuid.UUID) {
	row, err := s.repo.GetByID(ctx, notifID, recipientID)
	if err != nil || row == nil {
		return
	}

	resp := row.ToResponse()
	s.hub.SendToUser(recipientID, ws.Message{
		Type: "notification",
		Data: resp,
	})

	if s.overlay != nil {
		s.overlay.DispatchNotification(recipientID, resp)
	}

	if s.pushSvc != nil && !s.hub.IsOnline(recipientID) {
		siteName := s.settingsSvc.Get(ctx, config.SettingSiteName)
		go s.pushSvc.SendToUser(context.Background(), recipientID, pushPayload(resp, siteName))
	}
}

func pushPayload(resp dto.NotificationResponse, siteName string) push.Notification {
	title := resp.Actor.DisplayName
	if title == "" {
		title = resp.Actor.Username
	}
	if title == "" {
		title = siteName
	}

	body := resp.Message
	if body == "" || resp.Type == dto.NotifContentEdited {
		body = notifText[resp.Type]
	}
	if body == "" {
		body = "You have a new notification"
	}

	return push.Notification{
		Title: title,
		Body:  body,
		Data: map[string]string{
			"notification_id": strconv.Itoa(resp.ID),
			"type":            string(resp.Type),
			"reference_id":    resp.ReferenceID.String(),
			"reference_type":  resp.ReferenceType,
			"actor_username":  resp.Actor.Username,
		},
	}
}

func (s *service) List(ctx context.Context, userID uuid.UUID, page bounds.Page) (*dto.NotificationListResponse, error) {
	rows, total, err := s.repo.ListByUser(ctx, userID, page.Limit(), page.Offset())
	if err != nil {
		return nil, err
	}

	notifications := make([]dto.NotificationResponse, len(rows))
	for i, row := range rows {
		notifications[i] = row.ToResponse()
	}

	return &dto.NotificationListResponse{
		Notifications: notifications,
		Total:         total,
		Limit:         page.Limit(),
		Offset:        page.Offset(),
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

func (s *service) PruneOld(ctx context.Context) (int, error) {
	cutoff := time.Now().Add(-notificationRetention)

	total := 0
	for {
		deleted, err := s.repo.DeleteOlderThanBatch(ctx, cutoff, pruneBatchSize)
		if err != nil {
			return total, err
		}

		total += int(deleted)
		if deleted < int64(pruneBatchSize) {
			break
		}
	}

	return total, nil
}
