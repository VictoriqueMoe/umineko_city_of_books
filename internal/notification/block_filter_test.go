package notification

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/email"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newBlockTestService(t *testing.T) (*service, *repository.MockNotificationRepository, *repository.MockBlockRepository) {
	notifRepo := repository.NewMockNotificationRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	blockRepo := repository.NewMockBlockRepository(t)
	emailSvc := email.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("Test Site").Maybe()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://test.example").Maybe()

	svc := NewService(notifRepo, userRepo, blockRepo, ws.NewHub(), emailSvc, nil, settingsSvc, nil).(*service)
	return svc, notifRepo, blockRepo
}

func TestNotify_BlockFiltering(t *testing.T) {
	tests := []struct {
		name        string
		notifType   dto.NotificationType
		blocked     bool
		blockErr    error
		wantCreated bool
	}{
		{
			name:        "mention from a blocked user is dropped",
			notifType:   dto.NotifMention,
			blocked:     true,
			wantCreated: false,
		},
		{
			name:        "mention from a normal user is delivered",
			notifType:   dto.NotifMention,
			blocked:     false,
			wantCreated: true,
		},
		{
			name:        "post like from a blocked user is dropped",
			notifType:   dto.NotifPostLiked,
			blocked:     true,
			wantCreated: false,
		},
		{
			name:        "new follower from a blocked user is dropped",
			notifType:   dto.NotifNewFollower,
			blocked:     true,
			wantCreated: false,
		},
		{
			name:        "ban notice still reaches a user who blocked the moderator",
			notifType:   dto.NotifChatRoomBanned,
			blocked:     true,
			wantCreated: true,
		},
		{
			name:        "report resolution still reaches a user who blocked the moderator",
			notifType:   dto.NotifReportResolved,
			blocked:     true,
			wantCreated: true,
		},
		{
			name:        "your-turn still reaches a user who blocked their opponent",
			notifType:   dto.NotifGameYourTurn,
			blocked:     true,
			wantCreated: true,
		},
		{
			name:        "block lookup failure delivers rather than silently dropping",
			notifType:   dto.NotifMention,
			blockErr:    errors.New("boom"),
			wantCreated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			svc, notifRepo, blockRepo := newBlockTestService(t)
			recipient := uuid.New()
			actor := uuid.New()
			params := dto.NotifyParams{
				RecipientID: recipient,
				ActorID:     actor,
				Type:        tt.notifType,
				ReferenceID: uuid.New(),
			}
			if !survivesBlock(tt.notifType) {
				blockRepo.EXPECT().IsBlockedEither(mock.Anything, recipient, actor).Return(tt.blocked, tt.blockErr)
			}
			if tt.wantCreated {
				notifRepo.EXPECT().
					Create(mock.Anything, recipient, tt.notifType, params.ReferenceID, "", actor, "").
					Return(int64(1), nil)
				notifRepo.EXPECT().
					GetByID(mock.Anything, 1, recipient).
					Return(&model.NotificationRow{ID: 1, UserID: recipient, Type: tt.notifType}, nil)
			}

			// when
			err := svc.Notify(context.Background(), params)

			// then
			require.NoError(t, err)
			if !tt.wantCreated {
				notifRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}
}

func TestNotify_SystemActorSkipsBlockLookup(t *testing.T) {
	// given
	svc, notifRepo, blockRepo := newBlockTestService(t)
	recipient := uuid.New()
	params := dto.NotifyParams{
		RecipientID: recipient,
		ActorID:     uuid.Nil,
		Type:        dto.NotifMention,
		ReferenceID: uuid.New(),
	}
	notifRepo.EXPECT().
		Create(mock.Anything, recipient, dto.NotifMention, params.ReferenceID, "", uuid.Nil, "").
		Return(int64(1), nil)
	notifRepo.EXPECT().
		GetByID(mock.Anything, 1, recipient).
		Return(&model.NotificationRow{ID: 1, UserID: recipient, Type: dto.NotifMention}, nil)

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
	blockRepo.AssertNotCalled(t, "IsBlockedEither", mock.Anything, mock.Anything, mock.Anything)
}
