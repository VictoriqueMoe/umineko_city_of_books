package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/email"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	emailSvc := email.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	hub := ws.NewHub()

	settingsSvc.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("Test Site").Maybe()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://test.example").Maybe()

	svc := NewService(notifRepo, userRepo, hub, emailSvc, nil, settingsSvc, nil).(*service)
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
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(0), errors.New("db down"))

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
		HasRecentDuplicate(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ActorID).
		Return(false, errors.New("lookup failed"))
	notifRepo.EXPECT().
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(42), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything, params.RecipientID).
		Return(nil, nil)
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
		HasRecentDuplicate(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ActorID).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(1), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything, params.RecipientID).
		Return(nil, nil)
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
				Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
				Return(int64(1), nil)
			notifRepo.EXPECT().
				GetByID(mock.Anything, mock.Anything, params.RecipientID).
				Return(nil, nil)

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
		HasRecentDuplicate(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ActorID).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(1), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything, params.RecipientID).
		Return(nil, nil)
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
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(1), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything, params.RecipientID).
		Return(nil, nil)

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
		HasRecentDuplicate(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ActorID).
		Return(true, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(1), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything, params.RecipientID).
		Return(nil, nil)

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
		HasRecentDuplicate(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ActorID).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(1), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything, params.RecipientID).
		Return(nil, nil)
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
		HasRecentDuplicate(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ActorID).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(1), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything, params.RecipientID).
		Return(nil, nil)
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
		HasRecentDuplicate(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ActorID).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(1), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything, params.RecipientID).
		Return(nil, nil)
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
		HasRecentDuplicate(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ActorID).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(1), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything, params.RecipientID).
		Return(nil, nil)
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
		HasRecentDuplicate(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ActorID).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(1), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything, params.RecipientID).
		Return(nil, nil)
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
		HasRecentDuplicate(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ActorID).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(1), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything, params.RecipientID).
		Return(nil, nil)
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
		HasRecentDuplicate(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ActorID).
		Return(false, nil)
	notifRepo.EXPECT().
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(1), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything, params.RecipientID).
		Return(nil, nil)
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

func TestNotify_PushNotificationListErrorSilentlyIgnored(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	params := dto.NotifyParams{
		RecipientID: uuid.New(),
		ActorID:     uuid.New(),
		Type:        dto.NotifChatMessage,
		ReferenceID: uuid.New(),
	}
	notifRepo.EXPECT().
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(7), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, 7, params.RecipientID).
		Return(nil, errors.New("lookup failed"))

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
}

func TestNotify_PushNotificationNoMatchingRowNoPanic(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	params := dto.NotifyParams{
		RecipientID: uuid.New(),
		ActorID:     uuid.New(),
		Type:        dto.NotifChatMessage,
		ReferenceID: uuid.New(),
	}
	notifRepo.EXPECT().
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(99), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, 99, params.RecipientID).
		Return(nil, nil)

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
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(123), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, 123, params.RecipientID).
		Return(&model.NotificationRow{ID: 123, UserID: params.RecipientID, Type: params.Type}, nil)

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
}

type fakeOverlayDispatcher struct {
	called      bool
	recipientID uuid.UUID
	resp        dto.NotificationResponse
}

func (f *fakeOverlayDispatcher) DispatchNotification(recipientID uuid.UUID, resp dto.NotificationResponse) {
	f.called = true
	f.recipientID = recipientID
	f.resp = resp
}

func TestNotify_DispatchesToOverlay(t *testing.T) {
	// given
	notifRepo := repository.NewMockNotificationRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	emailSvc := email.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("Test Site").Maybe()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://test.example").Maybe()
	overlay := &fakeOverlayDispatcher{}
	svc := NewService(notifRepo, userRepo, ws.NewHub(), emailSvc, nil, settingsSvc, overlay)
	recipient := uuid.New()
	params := dto.NotifyParams{
		RecipientID: recipient,
		ActorID:     uuid.New(),
		Type:        dto.NotifPostLiked,
		ReferenceID: uuid.New(),
	}
	notifRepo.EXPECT().
		Create(mock.Anything, params.RecipientID, params.Type, params.ReferenceID, params.ReferenceType, params.ActorID, params.Message).
		Return(int64(55), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, 55, params.RecipientID).
		Return(&model.NotificationRow{ID: 55, UserID: params.RecipientID, Type: params.Type}, nil)

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
	assert.True(t, overlay.called)
	assert.Equal(t, recipient, overlay.recipientID)
	assert.Equal(t, dto.NotifPostLiked, overlay.resp.Type)
}

func TestNotifyMany_IteratesAllParamsAndSwallowsErrors(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	recipient := uuid.New()
	actor := uuid.New()
	ref := uuid.New()
	paramsList := []dto.NotifyParams{
		{RecipientID: recipient, ActorID: actor, Type: dto.NotifChatMessage, ReferenceID: ref},
		{RecipientID: recipient, ActorID: actor, Type: dto.NotifChatMessage, ReferenceID: ref},
	}
	notifRepo.EXPECT().
		Create(mock.Anything, recipient, dto.NotifChatMessage, ref, "", actor, "").
		Return(int64(0), errors.New("boom")).Once()
	notifRepo.EXPECT().
		Create(mock.Anything, recipient, dto.NotifChatMessage, ref, "", actor, "").
		Return(int64(1), nil).Once()
	notifRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything, recipient).
		Return(nil, nil).Once()

	// when
	svc.NotifyMany(context.Background(), paramsList)

	// then
	notifRepo.AssertExpectations(t)
}

func TestNotifyMany_EmptyListIsNoop(t *testing.T) {
	// given
	svc, _, _, _, _ := newTestService(t)

	// when
	svc.NotifyMany(context.Background(), nil)

	// then
}

func TestList_OK(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	userID := uuid.New()
	rows := []model.NotificationRow{
		{ID: 1, UserID: userID, Type: dto.NotifPostLiked, ActorUsername: "alice"},
		{ID: 2, UserID: userID, Type: dto.NotifMention, ActorUsername: "bob"},
	}
	notifRepo.EXPECT().ListByUser(mock.Anything, userID, 10, 0).Return(rows, 2, nil)

	// when
	got, err := svc.List(context.Background(), userID, 10, 0)

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
	notifRepo.EXPECT().ListByUser(mock.Anything, userID, 5, 10).Return(nil, 0, nil)

	// when
	got, err := svc.List(context.Background(), userID, 5, 10)

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
	notifRepo.EXPECT().ListByUser(mock.Anything, userID, 10, 0).Return(nil, 0, errors.New("db down"))

	// when
	got, err := svc.List(context.Background(), userID, 10, 0)

	// then
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestMarkRead_Delegates(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	userID := uuid.New()
	notifRepo.EXPECT().MarkRead(mock.Anything, 42, userID).Return(nil)

	// when
	err := svc.MarkRead(context.Background(), 42, userID)

	// then
	require.NoError(t, err)
}

func TestMarkRead_RepoError(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	userID := uuid.New()
	notifRepo.EXPECT().MarkRead(mock.Anything, 1, userID).Return(errors.New("boom"))

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
		DeleteOlderThanBatch(mock.Anything, mock.MatchedBy(func(cutoff time.Time) bool {
			return time.Since(cutoff) > 89*24*time.Hour
		}), pruneBatchSize).
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
	notifRepo.EXPECT().DeleteOlderThanBatch(mock.Anything, mock.Anything, pruneBatchSize).Return(int64(pruneBatchSize), nil).Twice()
	notifRepo.EXPECT().DeleteOlderThanBatch(mock.Anything, mock.Anything, pruneBatchSize).Return(int64(42), nil).Once()

	// when
	total, err := svc.PruneOld(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 2*pruneBatchSize+42, total)
}

func TestPruneOld_ReturnsErrorFromRepo(t *testing.T) {
	// given
	svc, notifRepo, _, _, _ := newTestService(t)
	notifRepo.EXPECT().DeleteOlderThanBatch(mock.Anything, mock.Anything, pruneBatchSize).Return(int64(0), errors.New("boom")).Once()

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
