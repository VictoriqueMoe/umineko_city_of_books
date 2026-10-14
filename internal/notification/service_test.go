package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/email"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	notifyManyFixture struct {
		svc       *service
		notifRepo *repository.MockNotificationRepository
		userRepo  *repository.MockUserRepository
		blockRepo *repository.MockBlockRepository
		emailSvc  *email.MockService
		overlay   *MockOverlayDispatcher
	}
)

func newTestService(t *testing.T) (
	*service,
	*repository.MockNotificationRepository,
	*repository.MockUserRepository,
	*email.MockService,
	*ws.Hub,
) {
	notifRepo := repository.NewMockNotificationRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	blockRepo := repository.NewMockBlockRepository(t)
	blockRepo.EXPECT().IsBlockedEither(mock.Anything, mock.Anything).Return(false, nil).Maybe()
	emailSvc := email.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	hub := ws.NewHub()

	settingsSvc.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("Test Site").Maybe()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://test.example").Maybe()

	svc := NewService(notifRepo, userRepo, blockRepo, hub, emailSvc, nil, settingsSvc, nil).(*service)
	return svc, notifRepo, userRepo, emailSvc, hub
}

func TestNotify_SelfNotifyReturnsNilNoCalls(t *testing.T) {
	// given
	svc, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	params := dto.NotifyParams{
		RecipientID: userID,
		ActorID:     userID,
		Type:        dto.NotifPostLiked,
	}

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
}

func TestNotify_CreateErrorPropagates(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	params := dto.NotifyParams{
		RecipientID:   uuid.New(),
		ActorID:       uuid.New(),
		Type:          dto.NotifPostLiked,
		ReferenceID:   uuid.New(),
		ReferenceType: "post",
		Message:       "liked your post",
	}
	notifRepo.EXPECT().
		Create(mock.Anything, spec.NewNotification{
			UserID:        params.RecipientID,
			Type:          params.Type,
			ReferenceID:   params.ReferenceID,
			ReferenceType: params.ReferenceType,
			ActorID:       params.ActorID,
			Message:       params.Message,
		}).
		Return(nil, errors.New("db down"))

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.Error(t, err)
	assert.EqualError(t, err, "db down")
}

func TestNotify_HasRecentDuplicateErrorIgnoredCreateProceeds(t *testing.T) {
	// given
	svc, notifRepo, userRepo, _, _ := newTestService(t)
	params := dto.NotifyParams{
		RecipientID:   uuid.New(),
		ActorID:       uuid.New(),
		Type:          dto.NotifPostLiked,
		ReferenceID:   uuid.New(),
		ReferenceType: "post",
		EmailActor:    "Beatrice",
		EmailAction:   "liked your post",
	}
	notifRepo.EXPECT().
		HasRecentDuplicate(mock.Anything, spec.NotificationDuplicateCheck{
			UserID:      params.RecipientID,
			Type:        params.Type,
			ReferenceID: params.ReferenceID,
			ActorID:     params.ActorID,
		}).
		Return(false, errors.New("lookup failed"))
	notifRepo.EXPECT().
		Create(mock.Anything, spec.NewNotification{
			UserID:        params.RecipientID,
			Type:          params.Type,
			ReferenceID:   params.ReferenceID,
			ReferenceType: params.ReferenceType,
			ActorID:       params.ActorID,
			Message:       params.Message,
		}).
		Return(&model.NotificationRow{ID: 42}, nil)
	userRepo.EXPECT().
		GetByID(mock.Anything, params.RecipientID).
		Return(nil, nil)

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
}

func TestNotify_DirectMessageSendsEmailWhenActionSet(t *testing.T) {
	// given
	svc, notifRepo, userRepo, emailSvc, _ := newTestService(t)
	params := dto.NotifyParams{
		RecipientID: uuid.New(),
		ActorID:     uuid.New(),
		Type:        dto.NotifChatMessage,
		ReferenceID: uuid.New(),
		EmailActor:  "Alice",
		EmailAction: "messaged you",
	}
	notifRepo.EXPECT().
		HasRecentDuplicate(mock.Anything, spec.NotificationDuplicateCheck{
			UserID:      params.RecipientID,
			Type:        params.Type,
			ReferenceID: params.ReferenceID,
			ActorID:     params.ActorID,
		}).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, spec.NewNotification{
			UserID:        params.RecipientID,
			Type:          params.Type,
			ReferenceID:   params.ReferenceID,
			ReferenceType: params.ReferenceType,
			ActorID:       params.ActorID,
			Message:       params.Message,
		}).
		Return(&model.NotificationRow{ID: 1}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, params.RecipientID).Return(&model.User{
		Email:              "recipient@example.com",
		EmailNotifications: true,
	}, nil)
	emailSvc.EXPECT().Send(mock.Anything, "recipient@example.com", "Alice messaged you", mock.Anything).Return(nil)

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
}

func TestNotify_ChatTypesSkipEmail(t *testing.T) {
	cases := []struct {
		name      string
		notifType dto.NotificationType
	}{
		{"room message", dto.NotifChatRoomMessage},
		{"mention", dto.NotifChatMention},
		{"reply", dto.NotifChatReply},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, notifRepo, _, _, _ := newTestService(t)
			params := dto.NotifyParams{
				RecipientID: uuid.New(),
				ActorID:     uuid.New(),
				Type:        tc.notifType,
				ReferenceID: uuid.New(),
				EmailActor:  "Alice",
				EmailAction: "messaged you",
			}
			notifRepo.EXPECT().
				Create(mock.Anything, spec.NewNotification{
					UserID:        params.RecipientID,
					Type:          params.Type,
					ReferenceID:   params.ReferenceID,
					ReferenceType: params.ReferenceType,
					ActorID:       params.ActorID,
					Message:       params.Message,
				}).
				Return(&model.NotificationRow{ID: 1}, nil)

			// when
			err := svc.Notify(context.Background(), params)

			// then
			require.NoError(t, err)
		})
	}
}

func TestNotify_ChatRoomInviteStillSendsEmail(t *testing.T) {
	// given
	svc, notifRepo, userRepo, emailSvc, _ := newTestService(t)
	params := dto.NotifyParams{
		RecipientID: uuid.New(),
		ActorID:     uuid.New(),
		Type:        dto.NotifChatRoomInvite,
		ReferenceID: uuid.New(),
		EmailActor:  "Beatrice",
		EmailAction: "liked your post",
	}
	notifRepo.EXPECT().
		HasRecentDuplicate(mock.Anything, spec.NotificationDuplicateCheck{
			UserID:      params.RecipientID,
			Type:        params.Type,
			ReferenceID: params.ReferenceID,
			ActorID:     params.ActorID,
		}).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, spec.NewNotification{
			UserID:        params.RecipientID,
			Type:          params.Type,
			ReferenceID:   params.ReferenceID,
			ReferenceType: params.ReferenceType,
			ActorID:       params.ActorID,
			Message:       params.Message,
		}).
		Return(&model.NotificationRow{ID: 1}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, params.RecipientID).Return(&model.User{
		Email:              "recipient@example.com",
		EmailNotifications: true,
	}, nil)
	emailSvc.EXPECT().Send(mock.Anything, "recipient@example.com", "Beatrice liked your post", mock.Anything).Return(nil)

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
}

func TestNotify_NoEmailActionSkipsEmail(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	params := dto.NotifyParams{
		RecipientID: uuid.New(),
		ActorID:     uuid.New(),
		Type:        dto.NotifPostLiked,
		ReferenceID: uuid.New(),
	}
	notifRepo.EXPECT().
		Create(mock.Anything, spec.NewNotification{
			UserID:        params.RecipientID,
			Type:          params.Type,
			ReferenceID:   params.ReferenceID,
			ReferenceType: params.ReferenceType,
			ActorID:       params.ActorID,
			Message:       params.Message,
		}).
		Return(&model.NotificationRow{ID: 1}, nil)

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
}

func TestNotify_EmailDupeSkipsEmail(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	params := dto.NotifyParams{
		RecipientID: uuid.New(),
		ActorID:     uuid.New(),
		Type:        dto.NotifPostLiked,
		ReferenceID: uuid.New(),
		EmailActor:  "Beatrice",
		EmailAction: "liked your post",
	}
	notifRepo.EXPECT().
		HasRecentDuplicate(mock.Anything, spec.NotificationDuplicateCheck{
			UserID:      params.RecipientID,
			Type:        params.Type,
			ReferenceID: params.ReferenceID,
			ActorID:     params.ActorID,
		}).
		Return(true, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, spec.NewNotification{
			UserID:        params.RecipientID,
			Type:          params.Type,
			ReferenceID:   params.ReferenceID,
			ReferenceType: params.ReferenceType,
			ActorID:       params.ActorID,
			Message:       params.Message,
		}).
		Return(&model.NotificationRow{ID: 1}, nil)

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
}

func TestNotify_EmailSentWhenEligible(t *testing.T) {
	// given
	svc, notifRepo, userRepo, emailSvc, _ := newTestService(t)
	params := dto.NotifyParams{
		RecipientID: uuid.New(),
		ActorID:     uuid.New(),
		Type:        dto.NotifPostLiked,
		ReferenceID: uuid.New(),
		EmailActor:  "Beatrice",
		EmailAction: "liked your post",
	}
	notifRepo.EXPECT().
		HasRecentDuplicate(mock.Anything, spec.NotificationDuplicateCheck{
			UserID:      params.RecipientID,
			Type:        params.Type,
			ReferenceID: params.ReferenceID,
			ActorID:     params.ActorID,
		}).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, spec.NewNotification{
			UserID:        params.RecipientID,
			Type:          params.Type,
			ReferenceID:   params.ReferenceID,
			ReferenceType: params.ReferenceType,
			ActorID:       params.ActorID,
			Message:       params.Message,
		}).
		Return(&model.NotificationRow{ID: 1}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, params.RecipientID).Return(&model.User{
		Email:              "recipient@example.com",
		EmailNotifications: true,
	}, nil)
	emailSvc.EXPECT().Send(mock.Anything, "recipient@example.com", "Beatrice liked your post", mock.Anything).Return(nil)

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
}

func TestNotify_EmailSendErrorDoesNotBubble(t *testing.T) {
	// given
	svc, notifRepo, userRepo, emailSvc, _ := newTestService(t)
	params := dto.NotifyParams{
		RecipientID: uuid.New(),
		ActorID:     uuid.New(),
		Type:        dto.NotifPostLiked,
		ReferenceID: uuid.New(),
		EmailActor:  "Beatrice",
		EmailAction: "liked your post",
	}
	notifRepo.EXPECT().
		HasRecentDuplicate(mock.Anything, spec.NotificationDuplicateCheck{
			UserID:      params.RecipientID,
			Type:        params.Type,
			ReferenceID: params.ReferenceID,
			ActorID:     params.ActorID,
		}).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, spec.NewNotification{
			UserID:        params.RecipientID,
			Type:          params.Type,
			ReferenceID:   params.ReferenceID,
			ReferenceType: params.ReferenceType,
			ActorID:       params.ActorID,
			Message:       params.Message,
		}).
		Return(&model.NotificationRow{ID: 1}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, params.RecipientID).Return(&model.User{
		Email:              "recipient@example.com",
		EmailNotifications: true,
	}, nil)
	emailSvc.EXPECT().Send(mock.Anything, "recipient@example.com", "Beatrice liked your post", mock.Anything).Return(errors.New("smtp down"))

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
}

func TestNotify_EmailSkippedWhenUserLookupErrors(t *testing.T) {
	// given
	svc, notifRepo, userRepo, _, _ := newTestService(t)
	params := dto.NotifyParams{
		RecipientID: uuid.New(),
		ActorID:     uuid.New(),
		Type:        dto.NotifPostLiked,
		ReferenceID: uuid.New(),
		EmailActor:  "Beatrice",
		EmailAction: "liked your post",
	}
	notifRepo.EXPECT().
		HasRecentDuplicate(mock.Anything, spec.NotificationDuplicateCheck{
			UserID:      params.RecipientID,
			Type:        params.Type,
			ReferenceID: params.ReferenceID,
			ActorID:     params.ActorID,
		}).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, spec.NewNotification{
			UserID:        params.RecipientID,
			Type:          params.Type,
			ReferenceID:   params.ReferenceID,
			ReferenceType: params.ReferenceType,
			ActorID:       params.ActorID,
			Message:       params.Message,
		}).
		Return(&model.NotificationRow{ID: 1}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, params.RecipientID).Return(nil, errors.New("boom"))

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
}

func TestNotify_EmailSkippedWhenUserNil(t *testing.T) {
	// given
	svc, notifRepo, userRepo, _, _ := newTestService(t)
	params := dto.NotifyParams{
		RecipientID: uuid.New(),
		ActorID:     uuid.New(),
		Type:        dto.NotifPostLiked,
		ReferenceID: uuid.New(),
		EmailActor:  "Beatrice",
		EmailAction: "liked your post",
	}
	notifRepo.EXPECT().
		HasRecentDuplicate(mock.Anything, spec.NotificationDuplicateCheck{
			UserID:      params.RecipientID,
			Type:        params.Type,
			ReferenceID: params.ReferenceID,
			ActorID:     params.ActorID,
		}).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, spec.NewNotification{
			UserID:        params.RecipientID,
			Type:          params.Type,
			ReferenceID:   params.ReferenceID,
			ReferenceType: params.ReferenceType,
			ActorID:       params.ActorID,
			Message:       params.Message,
		}).
		Return(&model.NotificationRow{ID: 1}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, params.RecipientID).Return(nil, nil)

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
}

func TestNotify_EmailSkippedWhenEmailEmpty(t *testing.T) {
	// given
	svc, notifRepo, userRepo, _, _ := newTestService(t)
	params := dto.NotifyParams{
		RecipientID: uuid.New(),
		ActorID:     uuid.New(),
		Type:        dto.NotifPostLiked,
		ReferenceID: uuid.New(),
		EmailActor:  "Beatrice",
		EmailAction: "liked your post",
	}
	notifRepo.EXPECT().
		HasRecentDuplicate(mock.Anything, spec.NotificationDuplicateCheck{
			UserID:      params.RecipientID,
			Type:        params.Type,
			ReferenceID: params.ReferenceID,
			ActorID:     params.ActorID,
		}).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, spec.NewNotification{
			UserID:        params.RecipientID,
			Type:          params.Type,
			ReferenceID:   params.ReferenceID,
			ReferenceType: params.ReferenceType,
			ActorID:       params.ActorID,
			Message:       params.Message,
		}).
		Return(&model.NotificationRow{ID: 1}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, params.RecipientID).Return(&model.User{Email: ""}, nil)

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
}

func TestNotify_EmailSkippedWhenNotificationsDisabledAndNotReport(t *testing.T) {
	// given
	svc, notifRepo, userRepo, _, _ := newTestService(t)
	params := dto.NotifyParams{
		RecipientID: uuid.New(),
		ActorID:     uuid.New(),
		Type:        dto.NotifPostLiked,
		ReferenceID: uuid.New(),
		EmailActor:  "Beatrice",
		EmailAction: "liked your post",
	}
	notifRepo.EXPECT().
		HasRecentDuplicate(mock.Anything, spec.NotificationDuplicateCheck{
			UserID:      params.RecipientID,
			Type:        params.Type,
			ReferenceID: params.ReferenceID,
			ActorID:     params.ActorID,
		}).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, spec.NewNotification{
			UserID:        params.RecipientID,
			Type:          params.Type,
			ReferenceID:   params.ReferenceID,
			ReferenceType: params.ReferenceType,
			ActorID:       params.ActorID,
			Message:       params.Message,
		}).
		Return(&model.NotificationRow{ID: 1}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, params.RecipientID).Return(&model.User{
		Email:              "r@example.com",
		EmailNotifications: false,
	}, nil)

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
}

func TestNotify_ReportTypeSendsEmailEvenWithNotificationsDisabled(t *testing.T) {
	// given
	svc, notifRepo, userRepo, emailSvc, _ := newTestService(t)
	params := dto.NotifyParams{
		RecipientID: uuid.New(),
		ActorID:     uuid.New(),
		Type:        dto.NotifReport,
		ReferenceID: uuid.New(),
		EmailActor:  "Reporter",
		EmailAction: "post",
		EmailTitle:  "spam",
	}
	notifRepo.EXPECT().
		HasRecentDuplicate(mock.Anything, spec.NotificationDuplicateCheck{
			UserID:      params.RecipientID,
			Type:        params.Type,
			ReferenceID: params.ReferenceID,
			ActorID:     params.ActorID,
		}).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, spec.NewNotification{
			UserID:        params.RecipientID,
			Type:          params.Type,
			ReferenceID:   params.ReferenceID,
			ReferenceType: params.ReferenceType,
			ActorID:       params.ActorID,
			Message:       params.Message,
		}).
		Return(&model.NotificationRow{ID: 1}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, params.RecipientID).Return(&model.User{
		Email:              "admin@example.com",
		EmailNotifications: false,
	}, nil)
	emailSvc.EXPECT().Send(mock.Anything, "admin@example.com", "New report from Reporter", mock.Anything).Return(nil)

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
}

func TestNotify_PushNotificationFindsRowSendsToHub(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	params := dto.NotifyParams{
		RecipientID: uuid.New(),
		ActorID:     uuid.New(),
		Type:        dto.NotifChatMessage,
		ReferenceID: uuid.New(),
	}
	notifRepo.EXPECT().
		Create(mock.Anything, spec.NewNotification{
			UserID:        params.RecipientID,
			Type:          params.Type,
			ReferenceID:   params.ReferenceID,
			ReferenceType: params.ReferenceType,
			ActorID:       params.ActorID,
			Message:       params.Message,
		}).
		Return(&model.NotificationRow{ID: 123, UserID: params.RecipientID, Type: params.Type}, nil)

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
}

func TestNotify_DispatchesToOverlay(t *testing.T) {
	// given
	notifRepo := repository.NewMockNotificationRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	emailSvc := email.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("Test Site").Maybe()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://test.example").Maybe()
	overlay := NewMockOverlayDispatcher(t)
	blockRepo := repository.NewMockBlockRepository(t)
	blockRepo.EXPECT().IsBlockedEither(mock.Anything, mock.Anything).Return(false, nil).Maybe()
	svc := NewService(notifRepo, userRepo, blockRepo, ws.NewHub(), emailSvc, nil, settingsSvc, overlay)
	recipient := uuid.New()
	params := dto.NotifyParams{
		RecipientID: recipient,
		ActorID:     uuid.New(),
		Type:        dto.NotifPostLiked,
		ReferenceID: uuid.New(),
	}
	notifRepo.EXPECT().
		Create(mock.Anything, spec.NewNotification{
			UserID:        params.RecipientID,
			Type:          params.Type,
			ReferenceID:   params.ReferenceID,
			ReferenceType: params.ReferenceType,
			ActorID:       params.ActorID,
			Message:       params.Message,
		}).
		Return(&model.NotificationRow{ID: 55, UserID: params.RecipientID, Type: params.Type}, nil)
	overlay.EXPECT().DispatchNotification(recipient, mock.MatchedBy(func(resp dto.NotificationResponse) bool {
		return resp.Type == dto.NotifPostLiked
	})).Return().Once()

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
	overlay.AssertExpectations(t)
}

func newNotifyManyFixture(t *testing.T) notifyManyFixture {
	notifRepo := repository.NewMockNotificationRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	blockRepo := repository.NewMockBlockRepository(t)
	emailSvc := email.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	overlay := NewMockOverlayDispatcher(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("Test Site").Maybe()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://test.example").Maybe()

	svc := NewService(notifRepo, userRepo, blockRepo, ws.NewHub(), emailSvc, nil, settingsSvc, overlay).(*service)

	return notifyManyFixture{
		svc:       svc,
		notifRepo: notifRepo,
		userRepo:  userRepo,
		blockRepo: blockRepo,
		emailSvc:  emailSvc,
		overlay:   overlay,
	}
}

func createdRows(batch []spec.NewNotification) []model.NotificationRow {
	rows := make([]model.NotificationRow, 0, len(batch))
	for i, n := range batch {
		rows = append(rows, model.NotificationRow{
			ID:            i + 1,
			UserID:        n.UserID,
			Type:          n.Type,
			ReferenceID:   n.ReferenceID,
			ReferenceType: n.ReferenceType,
			ActorID:       n.ActorID,
			Message:       n.Message,
		})
	}

	return rows
}

func TestNotifyMany_InsertsEveryRowInOneCreateManyAndPushesEachRow(t *testing.T) {
	// given
	f := newNotifyManyFixture(t)
	actor := uuid.New()
	roomID := uuid.New()
	first := uuid.New()
	second := uuid.New()
	third := uuid.New()
	params := []dto.NotifyParams{
		{RecipientID: first, ActorID: actor, Type: dto.NotifChatRoomMessage, ReferenceID: roomID, ReferenceType: "chat_message:1", Message: "sent a message in Room"},
		{RecipientID: second, ActorID: actor, Type: dto.NotifChatMention, ReferenceID: roomID, ReferenceType: "chat_message:1"},
		{RecipientID: third, ActorID: actor, Type: dto.NotifChatReply, ReferenceID: roomID, ReferenceType: "chat_message:1"},
	}
	want := []spec.NewNotification{
		{UserID: first, Type: dto.NotifChatRoomMessage, ReferenceID: roomID, ReferenceType: "chat_message:1", ActorID: actor, Message: "sent a message in Room"},
		{UserID: second, Type: dto.NotifChatMention, ReferenceID: roomID, ReferenceType: "chat_message:1", ActorID: actor},
		{UserID: third, Type: dto.NotifChatReply, ReferenceID: roomID, ReferenceType: "chat_message:1", ActorID: actor},
	}
	f.blockRepo.EXPECT().GetBlockedIDs(mock.Anything, actor).Return([]uuid.UUID{}, nil).Once()
	f.notifRepo.EXPECT().CreateMany(mock.Anything, want).Return(createdRows(want), nil).Once()
	for _, n := range want {
		f.overlay.EXPECT().DispatchNotification(n.UserID, mock.MatchedBy(func(resp dto.NotificationResponse) bool {
			return resp.Type == n.Type && resp.ReferenceID == roomID
		})).Return().Once()
	}

	// when
	f.svc.NotifyMany(context.Background(), params)

	// then
	f.notifRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestNotifyMany_SkipsSelfNotifications(t *testing.T) {
	// given
	f := newNotifyManyFixture(t)
	actor := uuid.New()
	other := uuid.New()
	ref := uuid.New()
	params := []dto.NotifyParams{
		{RecipientID: actor, ActorID: actor, Type: dto.NotifChatRoomMessage, ReferenceID: ref},
		{RecipientID: other, ActorID: actor, Type: dto.NotifChatRoomMessage, ReferenceID: ref},
	}
	want := []spec.NewNotification{
		{UserID: other, Type: dto.NotifChatRoomMessage, ReferenceID: ref, ActorID: actor},
	}
	f.blockRepo.EXPECT().GetBlockedIDs(mock.Anything, actor).Return([]uuid.UUID{}, nil).Once()
	f.notifRepo.EXPECT().CreateMany(mock.Anything, want).Return(createdRows(want), nil).Once()
	f.overlay.EXPECT().DispatchNotification(other, mock.Anything).Return().Once()

	// when
	f.svc.NotifyMany(context.Background(), params)

	// then
}

func TestNotifyMany_EmailEligibleParamsGoThroughNotify(t *testing.T) {
	// given
	f := newNotifyManyFixture(t)
	actor := uuid.New()
	emailRecipient := uuid.New()
	chatRecipient := uuid.New()
	emailRef := uuid.New()
	roomID := uuid.New()
	emailParams := dto.NotifyParams{
		RecipientID: emailRecipient,
		ActorID:     actor,
		Type:        dto.NotifMysterySolvedAll,
		ReferenceID: emailRef,
		EmailActor:  "Battler",
		EmailAction: "solved a mystery",
	}
	chatParams := dto.NotifyParams{
		RecipientID: chatRecipient,
		ActorID:     actor,
		Type:        dto.NotifChatMention,
		ReferenceID: roomID,
		EmailAction: "mentioned you",
	}
	f.blockRepo.EXPECT().IsBlockedEither(mock.Anything, spec.BlockPairSpec{UserA: emailRecipient, UserB: actor}).Return(false, nil).Once()
	f.notifRepo.EXPECT().
		HasRecentDuplicate(mock.Anything, spec.NotificationDuplicateCheck{
			UserID:      emailRecipient,
			Type:        dto.NotifMysterySolvedAll,
			ReferenceID: emailRef,
			ActorID:     actor,
		}).
		Return(false, nil).Once()
	f.notifRepo.EXPECT().
		Create(mock.Anything, spec.NewNotification{
			UserID:      emailRecipient,
			Type:        dto.NotifMysterySolvedAll,
			ReferenceID: emailRef,
			ActorID:     actor,
		}).
		Return(&model.NotificationRow{ID: 9, UserID: emailRecipient, Type: dto.NotifMysterySolvedAll}, nil).Once()
	f.userRepo.EXPECT().GetByID(mock.Anything, emailRecipient).Return(&model.User{
		Email:              "battler@example.com",
		EmailNotifications: true,
	}, nil).Once()
	f.emailSvc.EXPECT().Send(mock.Anything, "battler@example.com", mock.Anything, mock.Anything).Return(nil).Once()
	wantBatch := []spec.NewNotification{
		{UserID: chatRecipient, Type: dto.NotifChatMention, ReferenceID: roomID, ActorID: actor},
	}
	f.blockRepo.EXPECT().GetBlockedIDs(mock.Anything, actor).Return([]uuid.UUID{}, nil).Once()
	f.notifRepo.EXPECT().CreateMany(mock.Anything, wantBatch).Return(createdRows(wantBatch), nil).Once()
	f.overlay.EXPECT().DispatchNotification(emailRecipient, mock.Anything).Return().Once()
	f.overlay.EXPECT().DispatchNotification(chatRecipient, mock.Anything).Return().Once()

	// when
	f.svc.NotifyMany(context.Background(), []dto.NotifyParams{emailParams, chatParams})

	// then
}

func TestNotifyMany_CreateManyErrorPushesNothing(t *testing.T) {
	// given
	f := newNotifyManyFixture(t)
	actor := uuid.New()
	recipient := uuid.New()
	ref := uuid.New()
	params := []dto.NotifyParams{
		{RecipientID: recipient, ActorID: actor, Type: dto.NotifChatRoomMessage, ReferenceID: ref},
	}
	f.blockRepo.EXPECT().GetBlockedIDs(mock.Anything, actor).Return([]uuid.UUID{}, nil).Once()
	f.notifRepo.EXPECT().
		CreateMany(mock.Anything, []spec.NewNotification{
			{UserID: recipient, Type: dto.NotifChatRoomMessage, ReferenceID: ref, ActorID: actor},
		}).
		Return(nil, errors.New("db down")).Once()

	// when
	f.svc.NotifyMany(context.Background(), params)

	// then
	f.overlay.AssertNotCalled(t, "DispatchNotification", mock.Anything, mock.Anything)
}

func TestNotifyMany_EmptyListIsNoop(t *testing.T) {
	// given
	f := newNotifyManyFixture(t)

	// when
	f.svc.NotifyMany(context.Background(), nil)

	// then
	f.notifRepo.AssertNotCalled(t, "CreateMany", mock.Anything, mock.Anything)
	f.blockRepo.AssertNotCalled(t, "GetBlockedIDs", mock.Anything, mock.Anything)
}

func TestList_OK(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	userID := uuid.New()
	rows := []model.NotificationRow{
		{ID: 1, UserID: userID, Type: dto.NotifPostLiked, ActorUsername: "alice"},
		{ID: 2, UserID: userID, Type: dto.NotifMention, ActorUsername: "bob"},
	}
	notifRepo.EXPECT().ListByUser(mock.Anything, spec.NotificationListing{UserID: userID, Limit: 10, Offset: 0}).Return(rows, 2, nil)

	// when
	got, err := svc.List(context.Background(), userID, bounds.NewPage(10, 0))

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 2, got.Total)
	assert.Equal(t, 10, got.Limit)
	assert.Equal(t, 0, got.Offset)
	require.Len(t, got.Notifications, 2)
	assert.Equal(t, 1, got.Notifications[0].ID)
	assert.Equal(t, "alice", got.Notifications[0].Actor.Username)
}

func TestList_EmptyRows(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	userID := uuid.New()
	notifRepo.EXPECT().ListByUser(mock.Anything, spec.NotificationListing{UserID: userID, Limit: 5, Offset: 10}).Return(nil, 0, nil)

	// when
	got, err := svc.List(context.Background(), userID, bounds.NewPage(5, 10))

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 0, got.Total)
	assert.Equal(t, 5, got.Limit)
	assert.Equal(t, 10, got.Offset)
	assert.Empty(t, got.Notifications)
}

func TestList_RepoError(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	userID := uuid.New()
	notifRepo.EXPECT().ListByUser(mock.Anything, spec.NotificationListing{UserID: userID, Limit: 10, Offset: 0}).Return(nil, 0, errors.New("db down"))

	// when
	got, err := svc.List(context.Background(), userID, bounds.NewPage(10, 0))

	// then
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestMarkRead_Delegates(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	userID := uuid.New()
	notifRepo.EXPECT().MarkRead(mock.Anything, spec.NotificationLookup{ID: 42, UserID: userID}).Return(nil)

	// when
	err := svc.MarkRead(context.Background(), 42, userID)

	// then
	require.NoError(t, err)
}

func TestMarkRead_RepoError(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	userID := uuid.New()
	notifRepo.EXPECT().MarkRead(mock.Anything, spec.NotificationLookup{ID: 1, UserID: userID}).Return(errors.New("boom"))

	// when
	err := svc.MarkRead(context.Background(), 1, userID)

	// then
	require.Error(t, err)
}

func TestMarkAllRead_Delegates(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	userID := uuid.New()
	notifRepo.EXPECT().MarkAllRead(mock.Anything, userID).Return(nil)

	// when
	err := svc.MarkAllRead(context.Background(), userID)

	// then
	require.NoError(t, err)
}

func TestMarkAllRead_RepoError(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	userID := uuid.New()
	notifRepo.EXPECT().MarkAllRead(mock.Anything, userID).Return(errors.New("boom"))

	// when
	err := svc.MarkAllRead(context.Background(), userID)

	// then
	require.Error(t, err)
}

func TestPruneOld_SingleBatch(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	notifRepo.EXPECT().
		DeleteOlderThanBatch(mock.Anything, mock.MatchedBy(func(s spec.NotificationPruneBatch) bool {
			return time.Since(s.Cutoff) > 89*24*time.Hour && s.Limit == pruneBatchSize
		})).
		Return(int64(3), nil).
		Once()

	// when
	total, err := svc.PruneOld(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
}

func TestPruneOld_MultipleBatches(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	notifRepo.EXPECT().DeleteOlderThanBatch(mock.Anything, mock.MatchedBy(func(s spec.NotificationPruneBatch) bool {
		return s.Limit == pruneBatchSize
	})).Return(int64(pruneBatchSize), nil).Twice()
	notifRepo.EXPECT().DeleteOlderThanBatch(mock.Anything, mock.MatchedBy(func(s spec.NotificationPruneBatch) bool {
		return s.Limit == pruneBatchSize
	})).Return(int64(42), nil).Once()

	// when
	total, err := svc.PruneOld(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 2*pruneBatchSize+42, total)
}

func TestPruneOld_ReturnsErrorFromRepo(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	notifRepo.EXPECT().DeleteOlderThanBatch(mock.Anything, mock.MatchedBy(func(s spec.NotificationPruneBatch) bool {
		return s.Limit == pruneBatchSize
	})).Return(int64(0), errors.New("boom")).Once()

	// when
	total, err := svc.PruneOld(context.Background())

	// then
	require.Error(t, err)
	assert.Equal(t, 0, total)
}

func TestUnreadCount_Delegates(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	userID := uuid.New()
	notifRepo.EXPECT().UnreadCount(mock.Anything, userID).Return(7, nil)

	// when
	got, err := svc.UnreadCount(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 7, got)
}

func TestUnreadCount_RepoError(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	userID := uuid.New()
	notifRepo.EXPECT().UnreadCount(mock.Anything, userID).Return(0, errors.New("boom"))

	// when
	got, err := svc.UnreadCount(context.Background(), userID)

	// then
	require.Error(t, err)
	assert.Equal(t, 0, got)
}
