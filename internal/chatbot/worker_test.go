package chatbot

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOverQuota(t *testing.T) {
	cases := []struct {
		name          string
		perUserPerDay int
		perDay        int
		userUsed      int
		userErr       error
		siteUsed      int
		siteErr       error
		want          bool
		wantErr       bool
	}{
		{name: "both ceilings disabled"},
		{name: "under both ceilings", perUserPerDay: 20, perDay: 500, userUsed: 3, siteUsed: 40},
		{name: "per user ceiling reached", perUserPerDay: 20, perDay: 500, userUsed: 20, want: true},
		{name: "site ceiling reached", perUserPerDay: 20, perDay: 500, userUsed: 1, siteUsed: 500, want: true},
		{name: "per user count failure is an error", perUserPerDay: 20, userErr: errors.New("db down"), wantErr: true},
		{name: "site count failure is an error", perDay: 500, siteErr: errors.New("db down"), wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			userID := uuid.New()
			botRepo := repository.NewMockChatbotRepository(t)
			if tc.perUserPerDay > 0 {
				botRepo.EXPECT().CountUserInvocationsToday(mock.Anything, userID).Return(tc.userUsed, tc.userErr).Once()
			}
			if tc.perDay > 0 {
				botRepo.EXPECT().CountInvocationsToday(mock.Anything).Return(tc.siteUsed, tc.siteErr).Maybe()
			}

			svc := &service{botRepo: botRepo}

			// when
			over, err := svc.overQuota(context.Background(), userID, tuning{perUserPerDay: tc.perUserPerDay, perDay: tc.perDay})

			// then
			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, over)
		})
	}
}

func TestStartTyping_OnlyChatFansOut(t *testing.T) {
	cases := []struct {
		name    string
		surface Surface
	}{
		{"post body trigger sends no typing", SurfacePost},
		{"post comment trigger sends no typing", SurfacePostComment},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc := &service{quit: make(chan struct{})}
			ev := botEvent{Surface: tc.surface, ScopeID: uuid.New(), Audience: []uuid.UUID{uuid.New()}}

			// when
			stop := svc.startTyping(ev, uuid.New())

			// then
			require.NotNil(t, stop)
			stop()
		})
	}
}
