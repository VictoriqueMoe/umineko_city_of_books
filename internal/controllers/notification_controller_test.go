package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"testing"

	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	notifsvc "umineko_city_of_books/internal/notification"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newNotificationHarness(t *testing.T) (*testutil.Harness, *notifsvc.MockService) {
	h := testutil.NewHarness(t)
	ns := notifsvc.NewMockService(t)

	s := &Service{
		NotificationService: ns,
		AuthSession:         h.SessionManager,
		AuthzService:        h.AuthzService,
	}
	for _, setup := range s.getAllNotificationRoutes() {
		setup(h.App)
	}
	return h, ns
}

func TestListNotifications_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newNotificationHarness, "GET", "/notifications", nil)
}

func TestListNotifications_OK_Defaults(t *testing.T) {
	// given
	h, ns := newNotificationHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	expected := &dto.NotificationListResponse{
		Notifications: []dto.NotificationResponse{{ID: 1, Message: "hi"}},
		Total:         1,
		Limit:         20,
		Offset:        0,
	}
	ns.EXPECT().List(mock.Anything, userID, 20, 0).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/notifications").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.NotificationListResponse](t, body)
	assert.Equal(t, expected.Total, got.Total)
	assert.Equal(t, expected.Limit, got.Limit)
	assert.Equal(t, expected.Offset, got.Offset)
	require.Len(t, got.Notifications, 1)
	assert.Equal(t, 1, got.Notifications[0].ID)
}

func TestListNotifications_OK_CustomPaging(t *testing.T) {
	// given
	h, ns := newNotificationHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	expected := &dto.NotificationListResponse{
		Notifications: []dto.NotificationResponse{},
		Total:         0,
		Limit:         5,
		Offset:        10,
	}
	ns.EXPECT().List(mock.Anything, userID, 5, 10).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/notifications?limit=5&offset=10").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.NotificationListResponse](t, body)
	assert.Equal(t, 5, got.Limit)
	assert.Equal(t, 10, got.Offset)
}

func TestListNotifications_ServiceError(t *testing.T) {
	// given
	h, ns := newNotificationHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ns.EXPECT().List(mock.Anything, userID, 20, 0).Return(nil, errors.New("db down"))

	// when
	status, body := h.NewRequest("GET", "/notifications").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list notifications")
}

func TestMarkNotificationRead_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newNotificationHarness, "POST", "/notifications/42/read", nil)
}

func TestMarkNotificationRead_OK(t *testing.T) {
	// given
	h, ns := newNotificationHarness(t)
	userID := uuid.New()
	notifID := 42
	h.ExpectValidSession("valid-cookie", userID)
	ns.EXPECT().MarkRead(mock.Anything, notifID, userID).Return(nil)

	// when
	status, body := h.NewRequest("POST", "/notifications/"+strconv.Itoa(notifID)+"/read").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), "ok")
}

func TestMarkNotificationRead_InvalidID(t *testing.T) {
	// given — non-int id should not match the :id<int> route constraint.
	h, _ := newNotificationHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/notifications/not-a-number/read").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
}

func TestMarkNotificationRead_ServiceError(t *testing.T) {
	// given
	h, ns := newNotificationHarness(t)
	userID := uuid.New()
	notifID := 7
	h.ExpectValidSession("valid-cookie", userID)
	ns.EXPECT().MarkRead(mock.Anything, notifID, userID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("POST", "/notifications/"+strconv.Itoa(notifID)+"/read").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to mark notification as read")
}

func TestMarkAllNotificationsRead_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newNotificationHarness, "POST", "/notifications/read", nil)
}

func TestMarkAllNotificationsRead_OK(t *testing.T) {
	// given
	h, ns := newNotificationHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ns.EXPECT().MarkAllRead(mock.Anything, userID).Return(nil)

	// when
	status, body := h.NewRequest("POST", "/notifications/read").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), "ok")
}

func TestMarkAllNotificationsRead_ServiceError(t *testing.T) {
	// given
	h, ns := newNotificationHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ns.EXPECT().MarkAllRead(mock.Anything, userID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("POST", "/notifications/read").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to mark all notifications as read")
}

func TestUnreadCount_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newNotificationHarness, "GET", "/notifications/unread-count", nil)
}

func TestUnreadCount_OK(t *testing.T) {
	// given
	h, ns := newNotificationHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ns.EXPECT().UnreadCount(mock.Anything, userID).Return(17, nil)

	// when
	status, body := h.NewRequest("GET", "/notifications/unread-count").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string]int](t, body)
	assert.Equal(t, 17, got["count"])
}

func TestUnreadCount_ServiceError(t *testing.T) {
	// given
	h, ns := newNotificationHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ns.EXPECT().UnreadCount(mock.Anything, userID).Return(0, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/notifications/unread-count").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get unread count")
}
