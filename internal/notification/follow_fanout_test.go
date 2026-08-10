package notification

import (
	"context"
	"errors"
	"strings"
	"testing"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSendFollowerNotification_BuildsOneParamsEntryPerFollower(t *testing.T) {
	// given
	followRepo := repository.NewMockFollowRepository(t)
	notifSvc := NewMockService(t)
	actorID := uuid.New()
	referenceID := uuid.New()
	firstFollower := uuid.New()
	secondFollower := uuid.New()
	followRepo.EXPECT().GetFollowerIDsToNotify(mock.Anything, actorID).Return([]uuid.UUID{firstFollower, secondFollower}, nil)

	var captured []dto.NotifyParams
	notifSvc.EXPECT().NotifyMany(mock.Anything, mock.Anything).Run(func(_ context.Context, params []dto.NotifyParams) {
		captured = params
	}).Return()

	// when
	SendFollowerNotification(context.Background(), followRepo, notifSvc, FollowerNotifyParams{
		ActorID:       actorID,
		Type:          dto.NotifTheoryCreated,
		ReferenceID:   referenceID,
		ReferenceType: "theory",
		Action:        "posted a new theory",
		Title:         "The Golden Land is a metaphor",
	})

	// then
	require.Len(t, captured, 2)
	assert.Equal(t, firstFollower, captured[0].RecipientID)
	assert.Equal(t, secondFollower, captured[1].RecipientID)
	for _, p := range captured {
		assert.Equal(t, dto.NotifTheoryCreated, p.Type)
		assert.Equal(t, referenceID, p.ReferenceID)
		assert.Equal(t, "theory", p.ReferenceType)
		assert.Equal(t, actorID, p.ActorID)
		assert.Equal(t, "posted a new theory: The Golden Land is a metaphor", p.Message)
	}
}

func TestSendFollowerNotification_NeverSendsEmail(t *testing.T) {
	// given
	followRepo := repository.NewMockFollowRepository(t)
	notifSvc := NewMockService(t)
	actorID := uuid.New()
	followRepo.EXPECT().GetFollowerIDsToNotify(mock.Anything, actorID).Return([]uuid.UUID{uuid.New()}, nil)

	var captured []dto.NotifyParams
	notifSvc.EXPECT().NotifyMany(mock.Anything, mock.Anything).Run(func(_ context.Context, params []dto.NotifyParams) {
		captured = params
	}).Return()

	// when
	SendFollowerNotification(context.Background(), followRepo, notifSvc, FollowerNotifyParams{
		ActorID: actorID,
		Type:    dto.NotifStreamLive,
		Action:  "went live",
		Title:   "reading EP4",
	})

	// then
	require.Len(t, captured, 1)
	assert.Empty(t, captured[0].EmailActor)
	assert.Empty(t, captured[0].EmailAction)
	assert.Empty(t, captured[0].EmailTitle)
	assert.Empty(t, captured[0].EmailLink)
}

func TestSendFollowerNotification_SkipsTheActor(t *testing.T) {
	// given
	followRepo := repository.NewMockFollowRepository(t)
	notifSvc := NewMockService(t)
	actorID := uuid.New()
	otherFollower := uuid.New()
	followRepo.EXPECT().GetFollowerIDsToNotify(mock.Anything, actorID).Return([]uuid.UUID{actorID, otherFollower}, nil)

	var captured []dto.NotifyParams
	notifSvc.EXPECT().NotifyMany(mock.Anything, mock.Anything).Run(func(_ context.Context, params []dto.NotifyParams) {
		captured = params
	}).Return()

	// when
	SendFollowerNotification(context.Background(), followRepo, notifSvc, FollowerNotifyParams{
		ActorID: actorID,
		Type:    dto.NotifMysteryCreated,
		Action:  "posted a new mystery",
	})

	// then
	require.Len(t, captured, 1)
	assert.Equal(t, otherFollower, captured[0].RecipientID)
}

func TestSendFollowerNotification_LookupErrorSendsNothing(t *testing.T) {
	// given
	followRepo := repository.NewMockFollowRepository(t)
	notifSvc := NewMockService(t)
	actorID := uuid.New()
	followRepo.EXPECT().GetFollowerIDsToNotify(mock.Anything, actorID).Return(nil, errors.New("db down"))

	// when
	SendFollowerNotification(context.Background(), followRepo, notifSvc, FollowerNotifyParams{
		ActorID: actorID,
		Type:    dto.NotifTheoryCreated,
		Action:  "posted a new theory",
	})

	// then
	notifSvc.AssertNumberOfCalls(t, "NotifyMany", 0)
}

func TestSendFollowerNotification_NoFollowersStillCallsNotifyMany(t *testing.T) {
	// given
	followRepo := repository.NewMockFollowRepository(t)
	notifSvc := NewMockService(t)
	actorID := uuid.New()
	followRepo.EXPECT().GetFollowerIDsToNotify(mock.Anything, actorID).Return(nil, nil)

	var captured []dto.NotifyParams
	notifSvc.EXPECT().NotifyMany(mock.Anything, mock.Anything).Run(func(_ context.Context, params []dto.NotifyParams) {
		captured = params
	}).Return()

	// when
	SendFollowerNotification(context.Background(), followRepo, notifSvc, FollowerNotifyParams{
		ActorID: actorID,
		Type:    dto.NotifTheoryCreated,
		Action:  "posted a new theory",
	})

	// then
	assert.Empty(t, captured)
}

func TestSendFollowerNotification_Message(t *testing.T) {
	tests := []struct {
		name   string
		action string
		title  string
		want   string
	}{
		{
			name:   "title appended after the action",
			action: "posted a new theory",
			title:  "Beatrice does not exist",
			want:   "posted a new theory: Beatrice does not exist",
		},
		{
			name:   "empty title leaves the bare action",
			action: "went live",
			title:  "",
			want:   "went live",
		},
		{
			name:   "over-long title is clamped",
			action: "went live",
			title:  strings.Repeat("z", maxFollowerNotifTitle+40),
			want:   "went live: " + strings.Repeat("z", maxFollowerNotifTitle),
		},
		{
			name:   "clamp counts runes not bytes",
			action: "went live",
			title:  strings.Repeat("ベアトリーチェ", 40),
			want:   "went live: " + string([]rune(strings.Repeat("ベアトリーチェ", 40))[:maxFollowerNotifTitle]),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			followRepo := repository.NewMockFollowRepository(t)
			notifSvc := NewMockService(t)
			actorID := uuid.New()
			followRepo.EXPECT().GetFollowerIDsToNotify(mock.Anything, actorID).Return([]uuid.UUID{uuid.New()}, nil)

			var captured []dto.NotifyParams
			notifSvc.EXPECT().NotifyMany(mock.Anything, mock.Anything).Run(func(_ context.Context, params []dto.NotifyParams) {
				captured = params
			}).Return()

			// when
			SendFollowerNotification(context.Background(), followRepo, notifSvc, FollowerNotifyParams{
				ActorID: actorID,
				Type:    dto.NotifStreamLive,
				Action:  tt.action,
				Title:   tt.title,
			})

			// then
			require.Len(t, captured, 1)
			assert.Equal(t, tt.want, captured[0].Message)
		})
	}
}
