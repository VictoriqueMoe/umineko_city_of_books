package controllers

import (
	"net/http"
	"testing"

	"umineko_city_of_books/internal/chatbot"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newChatbotHarness(t *testing.T) (*testutil.Harness, *chatbot.MockService) {
	t.Helper()

	h := testutil.NewHarness(t)
	botSvc := chatbot.NewMockService(t)

	s := &Service{
		ChatbotService: botSvc,
		AuthSession:    h.SessionManager,
		AuthzService:   h.AuthzService,
	}
	for _, setup := range s.getAllChatbotRoutes() {
		setup(h.App)
	}

	return h, botSvc
}

func TestListChatbots(t *testing.T) {
	cases := []struct {
		name     string
		listing  []dto.ChatbotSummary
		wantBody string
	}{
		{
			name: "returns the bots the site is offering",
			listing: []dto.ChatbotSummary{
				{UserID: uuid.New(), Username: "beato", DisplayName: "Beatrice"},
			},
			wantBody: `"username":"beato"`,
		},
		{
			name:     "returns an empty list rather than null when there are none",
			listing:  []dto.ChatbotSummary{},
			wantBody: `"chatbots":[]`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, botSvc := newChatbotHarness(t)
			botSvc.EXPECT().Listing().Return(tc.listing).Once()

			// when
			status, body := h.NewRequest("GET", "/chatbots").Do()

			// then
			require.Equal(t, http.StatusOK, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}
