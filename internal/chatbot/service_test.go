package chatbot

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/openai"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShutdown_DrainsTheBacklogItCanAndStopsAtTheDeadline(t *testing.T) {
	// given a worker that is already past its quit check, with a backlog it must not start
	svc := &service{
		jobs: make(chan job, 4),
		quit: make(chan struct{}),
	}

	for range 3 {
		svc.jobs <- job{}
	}

	expired, cancel := context.WithCancel(context.Background())
	cancel()

	svc.deadline = expired.Done()

	// when the deadline has already passed
	svc.drain(0)

	// then not one queued job was started, so shutdown cannot overrun the budget it shares
	assert.Len(t, svc.jobs, 3, "an expired deadline must stop the drain before it starts new work")
}

func TestShutdown_IsIdempotent(t *testing.T) {
	// given
	svc := &service{
		jobs: make(chan job, 1),
		quit: make(chan struct{}),
	}

	// when
	first := svc.Shutdown(context.Background())
	second := svc.Shutdown(context.Background())

	// then the second call must not panic on a second close of quit
	require.NoError(t, first)
	require.NoError(t, second)
}

func TestListing_OnlyExposesBotsWhenTheFeatureIsUsable(t *testing.T) {
	botA := repository.Chatbot{UserID: uuid.New(), Username: "bern", DisplayName: "Bernkastel"}
	botB := repository.Chatbot{UserID: uuid.New(), Username: "beato", DisplayName: "Beatrice"}

	cases := []struct {
		name          string
		enabled       bool
		openaiEnabled bool
		bots          []repository.Chatbot
		want          []string
	}{
		{"enabled and configured lists every bot", true, true, []repository.Chatbot{botA, botB}, []string{"Beatrice", "Bernkastel"}},
		{"feature switched off lists nothing", false, true, []repository.Chatbot{botA}, nil},
		{"no provider key lists nothing", true, false, []repository.Chatbot{botA}, nil},
		{"no bots lists nothing", true, true, nil, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			openaiSvc := openai.NewMockService(t)
			openaiSvc.EXPECT().Enabled().Return(tc.openaiEnabled).Maybe()

			svc := &service{openaiSvc: openaiSvc, bots: make(map[uuid.UUID]repository.Chatbot)}
			svc.tune = tuning{enabled: tc.enabled}
			for _, bot := range tc.bots {
				svc.bots[bot.UserID] = bot
			}

			// when
			got := svc.Listing()

			// then
			names := make([]string, 0, len(got))
			for i := range got {
				names = append(names, got[i].DisplayName)
			}

			assert.Equal(t, tc.want, nilIfEmpty(names), "listing must be alphabetical and gated on the feature")
		})
	}
}

func nilIfEmpty(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	return values
}
