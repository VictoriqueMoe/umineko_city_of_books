package notification

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/email"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
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
				blockRepo.EXPECT().IsBlockedEither(mock.Anything, spec.BlockPairSpec{UserA: recipient, UserB: actor}).Return(tt.blocked, tt.blockErr)
			}
			if tt.wantCreated {
				notifRepo.EXPECT().
					Create(mock.Anything, spec.NewNotification{
						UserID:      recipient,
						Type:        tt.notifType,
						ReferenceID: params.ReferenceID,
						ActorID:     actor,
					}).
					Return(&model.NotificationRow{ID: 1, UserID: recipient, Type: tt.notifType}, nil)
			}

			// when
			err := svc.Notify(context.Background(), params)

			// then
			require.NoError(t, err)
			if !tt.wantCreated {
				notifRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
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
		Create(mock.Anything, spec.NewNotification{
			UserID:      recipient,
			Type:        dto.NotifMention,
			ReferenceID: params.ReferenceID,
			ActorID:     uuid.Nil,
		}).
		Return(&model.NotificationRow{ID: 1, UserID: recipient, Type: dto.NotifMention}, nil)

	// when
	err := svc.Notify(context.Background(), params)

	// then
	require.NoError(t, err)
	blockRepo.AssertNotCalled(t, "IsBlockedEither", mock.Anything, mock.Anything)
}

func TestNotifyMany_BlockFiltering(t *testing.T) {
	tests := []struct {
		name          string
		notifType     dto.NotificationType
		systemActor   bool
		expectLookup  bool
		blockedIDs    func(blocked, unblocked uuid.UUID) []uuid.UUID
		lookupErr     error
		wantBlocked   bool
		wantUnblocked bool
		wantNoCreates bool
	}{
		{
			name:         "a recipient in the actor's block set is dropped",
			notifType:    dto.NotifChatRoomMessage,
			expectLookup: true,
			blockedIDs: func(blocked, unblocked uuid.UUID) []uuid.UUID {
				return []uuid.UUID{blocked}
			},
			wantUnblocked: true,
		},
		{
			name:         "every recipient blocked skips the insert entirely",
			notifType:    dto.NotifChatMention,
			expectLookup: true,
			blockedIDs: func(blocked, unblocked uuid.UUID) []uuid.UUID {
				return []uuid.UUID{blocked, unblocked}
			},
			wantNoCreates: true,
		},
		{
			name:          "block lookup failure delivers rather than silently dropping",
			notifType:     dto.NotifChatRoomMessage,
			expectLookup:  true,
			lookupErr:     errors.New("boom"),
			wantBlocked:   true,
			wantUnblocked: true,
		},
		{
			name:          "a type that survives blocks skips the lookup and reaches a blocked recipient",
			notifType:     dto.NotifGameYourTurn,
			wantBlocked:   true,
			wantUnblocked: true,
		},
		{
			name:          "a system actor skips the lookup",
			notifType:     dto.NotifMention,
			systemActor:   true,
			wantBlocked:   true,
			wantUnblocked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			f := newNotifyManyFixture(t)
			blocked := uuid.New()
			unblocked := uuid.New()
			ref := uuid.New()
			actor := uuid.New()
			if tt.systemActor {
				actor = uuid.Nil
			}
			params := []dto.NotifyParams{
				{RecipientID: blocked, ActorID: actor, Type: tt.notifType, ReferenceID: ref},
				{RecipientID: unblocked, ActorID: actor, Type: tt.notifType, ReferenceID: ref},
			}

			if tt.expectLookup {
				var ids []uuid.UUID
				if tt.blockedIDs != nil {
					ids = tt.blockedIDs(blocked, unblocked)
				}
				f.blockRepo.EXPECT().GetBlockedIDs(mock.Anything, actor).Return(ids, tt.lookupErr).Once()
			}

			want := make([]spec.NewNotification, 0, 2)
			if tt.wantBlocked {
				want = append(want, spec.NewNotification{UserID: blocked, Type: tt.notifType, ReferenceID: ref, ActorID: actor})
			}
			if tt.wantUnblocked {
				want = append(want, spec.NewNotification{UserID: unblocked, Type: tt.notifType, ReferenceID: ref, ActorID: actor})
			}
			if !tt.wantNoCreates {
				f.notifRepo.EXPECT().CreateMany(mock.Anything, want).Return(createdRows(want), nil).Once()
				for _, n := range want {
					f.overlay.EXPECT().DispatchNotification(n.UserID, mock.Anything).Return().Once()
				}
			}

			// when
			f.svc.NotifyMany(context.Background(), params)

			// then
			if tt.wantNoCreates {
				f.notifRepo.AssertNotCalled(t, "CreateMany", mock.Anything, mock.Anything)
			}
			if !tt.expectLookup {
				f.blockRepo.AssertNotCalled(t, "GetBlockedIDs", mock.Anything, mock.Anything)
			}
			f.blockRepo.AssertNotCalled(t, "IsBlockedEither", mock.Anything, mock.Anything)
		})
	}
}

func TestNotifyMany_LooksUpBlocksOncePerDistinctActor(t *testing.T) {
	// given
	f := newNotifyManyFixture(t)
	firstActor := uuid.New()
	secondActor := uuid.New()
	blockedByFirst := uuid.New()
	sharedRecipient := uuid.New()
	ref := uuid.New()
	params := []dto.NotifyParams{
		{RecipientID: blockedByFirst, ActorID: firstActor, Type: dto.NotifPostLiked, ReferenceID: ref},
		{RecipientID: sharedRecipient, ActorID: firstActor, Type: dto.NotifPostLiked, ReferenceID: ref},
		{RecipientID: blockedByFirst, ActorID: secondActor, Type: dto.NotifPostLiked, ReferenceID: ref},
		{RecipientID: sharedRecipient, ActorID: secondActor, Type: dto.NotifPostLiked, ReferenceID: ref},
	}
	f.blockRepo.EXPECT().GetBlockedIDs(mock.Anything, firstActor).Return([]uuid.UUID{blockedByFirst}, nil).Once()
	f.blockRepo.EXPECT().GetBlockedIDs(mock.Anything, secondActor).Return([]uuid.UUID{}, nil).Once()
	want := []spec.NewNotification{
		{UserID: sharedRecipient, Type: dto.NotifPostLiked, ReferenceID: ref, ActorID: firstActor},
		{UserID: blockedByFirst, Type: dto.NotifPostLiked, ReferenceID: ref, ActorID: secondActor},
		{UserID: sharedRecipient, Type: dto.NotifPostLiked, ReferenceID: ref, ActorID: secondActor},
	}
	f.notifRepo.EXPECT().CreateMany(mock.Anything, want).Return(createdRows(want), nil).Once()
	f.overlay.EXPECT().DispatchNotification(sharedRecipient, mock.Anything).Return().Twice()
	f.overlay.EXPECT().DispatchNotification(blockedByFirst, mock.Anything).Return().Once()

	// when
	f.svc.NotifyMany(context.Background(), params)

	// then
}
