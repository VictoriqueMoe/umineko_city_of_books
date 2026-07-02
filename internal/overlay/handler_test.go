package overlay

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http/httptest"
	"testing"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/ws"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newHandlerApp(t *testing.T) (*fiber.App, *repository.MockOverlayTokenRepository) {
	repo := repository.NewMockOverlayTokenRepository(t)
	svc := NewService(repo, ws.NewHub(), settings.NewMockService(t))
	app := fiber.New()
	app.Get("/api/v1/overlay", svc.Handler())
	return app, repo
}

func TestHandler_MissingToken(t *testing.T) {
	// given
	app, _ := newHandlerApp(t)

	// when
	resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/overlay", nil))

	// then
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestHandler_InvalidToken(t *testing.T) {
	// given
	app, repo := newHandlerApp(t)
	repo.EXPECT().GetUserByToken(mock.Anything, "bad").Return(uuid.Nil, nil)

	// when
	resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/overlay?token=bad", nil))

	// then
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestHandler_ValidationError(t *testing.T) {
	// given
	app, repo := newHandlerApp(t)
	repo.EXPECT().GetUserByToken(mock.Anything, "boom").Return(uuid.Nil, errors.New("db down"))

	// when
	resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/overlay?token=boom", nil))

	// then
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestHandler_LiveDispatch(t *testing.T) {
	if raceEnabled {
		t.Skip("ws.Client writer and the fasthttp websocket close path race on the hijacked conn; CI runs without -race")
	}

	// given
	repo := repository.NewMockOverlayTokenRepository(t)
	userID := uuid.New()
	repo.EXPECT().GetUserByToken(mock.Anything, "good-token").Return(userID, nil)
	svc := NewService(repo, ws.NewHub(), settings.NewMockService(t))

	app := fiber.New()
	app.Get("/api/v1/overlay", svc.Handler())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() {
		_ = app.Listener(ln)
	}()
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = app.ShutdownWithContext(shutdownCtx)
	})

	wsURL := "ws://" + ln.Addr().String() + "/api/v1/overlay?token=good-token"

	// when: a SAMMI connector dials in
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
	})

	// then: the streamer is reported online
	require.Eventually(t, func() bool {
		return svc.IsConnected(userID)
	}, 3*time.Second, 25*time.Millisecond)

	type overlayEvent struct {
		Type string `json:"type"`
		Data struct {
			Actor string `json:"actor"`
			Text  string `json:"text"`
		} `json:"data"`
	}

	// when: a test overlay is fired
	require.NoError(t, svc.TestFire(userID))

	// then: it arrives over the socket
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, raw, err := conn.ReadMessage()
	require.NoError(t, err)
	var fired overlayEvent
	require.NoError(t, json.Unmarshal(raw, &fired))
	assert.Equal(t, "test", fired.Type)

	// when: a real allowlisted notification is dispatched
	svc.DispatchNotification(userID, dto.NotificationResponse{
		Type:      dto.NotifPostLiked,
		Actor:     dto.UserResponse{Username: "lambda", DisplayName: "Lambda"},
		CreatedAt: "2026-06-29T00:00:00Z",
	})

	// then: it arrives as a post_liked overlay event with the mapped action text
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, raw, err = conn.ReadMessage()
	require.NoError(t, err)
	var liked overlayEvent
	require.NoError(t, json.Unmarshal(raw, &liked))
	assert.Equal(t, "post_liked", liked.Type)
	assert.Equal(t, "liked your post", liked.Data.Text)
	assert.Equal(t, "lambda", liked.Data.Actor)
}
