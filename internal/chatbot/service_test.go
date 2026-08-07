package chatbot

import (
	"testing"

	"umineko_city_of_books/internal/openai"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

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
