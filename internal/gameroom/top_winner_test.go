package gameroom

import (
	"context"
	"database/sql"
	"errors"
	"maps"
	"testing"

	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/ws"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRecordTopWinners(t *testing.T) {
	cases := []struct {
		name     string
		seed     map[dto.GameType][]string
		gameType dto.GameType
		ids      []string
		want     bool
	}{
		{name: "the first result after a restart is announced", gameType: dto.GameTypeChess, ids: []string{"a"}, want: true},
		{name: "the same winners in another order are not announced", seed: map[dto.GameType][]string{dto.GameTypeChess: {"a", "b"}}, gameType: dto.GameTypeChess, ids: []string{"b", "a"}, want: false},
		{name: "a new winner is announced", seed: map[dto.GameType][]string{dto.GameTypeChess: {"a"}}, gameType: dto.GameTypeChess, ids: []string{"b"}, want: true},
		{name: "losing the only winner is announced", seed: map[dto.GameType][]string{dto.GameTypeChess: {"a"}}, gameType: dto.GameTypeChess, ids: nil, want: true},
		{name: "each game keeps its own winners", seed: map[dto.GameType][]string{dto.GameTypeChess: {"a"}}, gameType: dto.GameTypeOthello, ids: []string{"a"}, want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			m := newTestService(t)
			maps.Copy(m.svc.topWinners, tc.seed)

			// when
			got := m.svc.recordTopWinners(tc.gameType, tc.ids)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBroadcastTopWinner_OnlyRecordsWhatTheLookupReturned(t *testing.T) {
	cases := []struct {
		name      string
		gameType  dto.GameType
		ids       []string
		lookupErr error
		wantCalls int
		wantKnown bool
	}{
		{name: "a known game records its winners", gameType: dto.GameTypeChess, ids: []string{"a"}, wantCalls: 1, wantKnown: true},
		{name: "a failed lookup records nothing", gameType: dto.GameTypeChess, lookupErr: errors.New("boom"), wantCalls: 1, wantKnown: false},
		{name: "a game with no crown is never looked up", gameType: dto.GameType("pong"), wantCalls: 0, wantKnown: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			roomRepo := repository.NewMockGameRoomRepository(t)
			roomRepo.EXPECT().GetTopWinnerIDs(mock.Anything, string(tc.gameType)).
				RunAndReturn(func(_ context.Context, _ string, _ ...*sql.Tx) ([]string, error) {
					return tc.ids, tc.lookupErr
				}).
				Maybe()
			svc := NewService(roomRepo, repository.NewMockUserRepository(t), repository.NewMockBlockRepository(t), nil, ws.NewHub(), contentfilter.New(), nil).(*service)

			// when
			svc.broadcastTopWinner(tc.gameType)

			// then
			roomRepo.AssertNumberOfCalls(t, "GetTopWinnerIDs", tc.wantCalls)
			_, known := svc.topWinners[tc.gameType]
			assert.Equal(t, tc.wantKnown, known)
		})
	}
}
