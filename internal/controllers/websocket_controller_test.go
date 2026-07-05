package controllers

import (
	"net/http"
	"testing"

	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/gameroom"
	"umineko_city_of_books/internal/ws"

	"github.com/stretchr/testify/assert"
)

func TestWebSocketController_RegistersGetWS(t *testing.T) {
	// given
	h := testutil.NewHarness(t)
	s := &Service{
		Hub:             ws.NewHub(),
		AuthSession:     h.SessionManager,
		ChatService:     chat.NewMockService(t),
		GameRoomService: gameroom.NewMockService(t),
		SettingsService: h.SettingsService,
	}

	// when
	for _, setup := range s.getAllWebSocketRoutes() {
		setup(h.App)
	}

	// then
	found := false
	for _, r := range h.App.GetRoutes(true) {
		if r.Method == http.MethodGet && r.Path == "/ws" {
			found = true
		}
	}
	assert.True(t, found)
}
