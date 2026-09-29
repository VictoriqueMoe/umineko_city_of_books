package controllers

import (
	"net/http"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/chatbot"
	"umineko_city_of_books/internal/controllers/utils/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAdminChatbotRoutes_ServerFailuresDoNotLeakTheCause(t *testing.T) {
	botID := uuid.New()

	cases := []struct {
		name    string
		method  string
		path    string
		expect  func(m *chatbot.MockAdminService)
		wantErr string
	}{
		{
			name:   "list chatbots",
			method: "GET",
			path:   "/admin/chatbots",
			expect: func(m *chatbot.MockAdminService) {
				m.EXPECT().List(mock.Anything).Return(nil, assert.AnError)
			},
			wantErr: "failed to list chatbots",
		},
		{
			name:   "chatbot usage",
			method: "GET",
			path:   "/admin/chatbots/usage",
			expect: func(m *chatbot.MockAdminService) {
				m.EXPECT().Usage(mock.Anything, mock.Anything).Return(nil, assert.AnError)
			},
			wantErr: "failed to load chatbot usage",
		},
		{
			name:   "list base prompts",
			method: "GET",
			path:   "/admin/chatbots/base-prompts",
			expect: func(m *chatbot.MockAdminService) {
				m.EXPECT().ListBasePrompts(mock.Anything).Return(nil, assert.AnError)
			},
			wantErr: "failed to list base prompts",
		},
		{
			name:   "delete chatbot",
			method: "DELETE",
			path:   "/admin/chatbots/" + botID.String(),
			expect: func(m *chatbot.MockAdminService) {
				m.EXPECT().Delete(mock.Anything, mock.Anything, botID).Return(assert.AnError)
			},
			wantErr: "chatbot action failed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h := testutil.NewHarness(t)
			adminSvc := chatbot.NewMockAdminService(t)
			actorID := uuid.New()
			h.ExpectValidSession("valid-cookie", actorID)
			h.ExpectHasPermission(actorID, authz.PermManageSettings, true)
			tc.expect(adminSvc)

			s := &Service{
				ChatbotAdminService: adminSvc,
				AuthSession:         h.SessionManager,
				AuthzService:        h.AuthzService,
			}
			for _, setup := range s.getAllAdminChatbotRoutes() {
				setup(h.App)
			}

			// when
			status, body := h.NewRequest(tc.method, tc.path).WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, http.StatusInternalServerError, status)
			assert.JSONEq(t, `{"error":"`+tc.wantErr+`"}`, string(body))
		})
	}
}
