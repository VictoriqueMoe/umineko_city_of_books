package chat

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSanitizeTags(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil", nil, nil},
		{"empty", []string{}, nil},
		{"trim spaces and lowercase", []string{"  Hello World  "}, []string{"hello-world"}},
		{"dedup", []string{"tag", "Tag", "TAG"}, []string{"tag"}},
		{"strip invalid chars", []string{"tag!@#$%"}, []string{"tag"}},
		{"empty after sanitise", []string{"---"}, []string{}},
		{"max length 30", []string{"abcdefghijklmnopqrstuvwxyz1234567890"}, []string{"abcdefghijklmnopqrstuvwxyz1234"}},
		{"max 10 tags", []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}, []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given / when
			got := sanitizeTags(tc.in)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResolveSenderName(t *testing.T) {
	cases := []struct {
		name     string
		nickname string
		display  string
		username string
		want     string
	}{
		{"nickname wins", "alias", "Display", "user", "alias"},
		{"display when no nickname", "", "Display", "user", "Display"},
		{"display when whitespace nickname", "   ", "Display", "user", "Display"},
		{"username when both empty", "", "", "user", "user"},
		{"username when both whitespace", " ", "  ", "user", "user"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given / when
			got := resolveSenderName(tc.nickname, tc.display, tc.username)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsUnread(t *testing.T) {
	cases := []struct {
		name     string
		last     sql.NullString
		read     sql.NullString
		expected bool
	}{
		{"no messages", sql.NullString{}, sql.NullString{}, false},
		{"never read", sql.NullString{Valid: true, String: "2024-01-01"}, sql.NullString{}, true},
		{"message newer", sql.NullString{Valid: true, String: "2024-01-02"}, sql.NullString{Valid: true, String: "2024-01-01"}, true},
		{"read newer", sql.NullString{Valid: true, String: "2024-01-01"}, sql.NullString{Valid: true, String: "2024-01-02"}, false},
		{"equal", sql.NullString{Valid: true, String: "2024-01-01"}, sql.NullString{Valid: true, String: "2024-01-01"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given / when
			got := isUnread(tc.last, tc.read)

			// then
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestNullStr(t *testing.T) {
	// given / when
	gotValid := nullStr(sql.NullString{Valid: true, String: "x"})
	gotInvalid := nullStr(sql.NullString{})

	// then
	assert.Equal(t, "x", gotValid)
	assert.Equal(t, "", gotInvalid)
}

func TestMessageRowToResponse_ReplyPreview(t *testing.T) {
	replyID := uuid.New()
	replySenderID := uuid.New()

	cases := []struct {
		name      string
		row       model.ChatMessageRow
		wantReply *dto.ChatMessageReplyPreview
	}{
		{
			name: "no reply leaves the preview nil",
			row:  model.ChatMessageRow{ID: uuid.New(), RoomID: uuid.New(), SenderID: uuid.New(), Body: "hi"},
		},
		{
			name: "reply body over 140 runes is clamped with an ellipsis",
			row: model.ChatMessageRow{
				ID:                uuid.New(),
				RoomID:            uuid.New(),
				SenderID:          uuid.New(),
				Body:              "hi",
				ReplyToID:         &replyID,
				ReplyToSenderID:   &replySenderID,
				ReplyToSenderName: new("Sender"),
				ReplyToBody:       new(strings.Repeat("x", 200)),
			},
			wantReply: &dto.ChatMessageReplyPreview{
				ID:          replyID,
				SenderID:    replySenderID,
				SenderName:  "Sender",
				BodyPreview: strings.Repeat("x", 140) + "...",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, _ := newTestService(t)

			// when
			got := svc.messageRowToResponse(tc.row, nil, nil, nil)

			// then
			assert.Equal(t, tc.wantReply, got.ReplyTo)
			assert.Equal(t, "hi", got.Body)
		})
	}
}

func TestRoomActionMessageBody_WithheldWhenTheTimeoutLookupFails(t *testing.T) {
	// given an actor whose timeout state cannot be read
	svc, m := newTestService(t)
	roomID := uuid.New()
	actorID := uuid.New()
	unsetDefaults(&m.chatRepo.Mock, "HasActiveMemberTimeout")
	m.chatRepo.EXPECT().HasActiveMemberTimeout(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: actorID}).Return(false, errors.New("boom"))

	// when
	got := svc.roomActionMessageBody(context.Background(), roomID, actorID, "Kujo joined the room.")

	// then a possibly timed-out actor is not allowed to speak through an action message
	assert.Empty(t, got)
}

func TestCanModerateRoom(t *testing.T) {
	cases := []struct {
		name       string
		memberRole string
		siteRole   *role.Role
		want       bool
	}{
		{name: "room host is decided without a site role lookup", memberRole: "host", want: true},
		{name: "site admin", memberRole: "member", siteRole: new(authz.RoleAdmin), want: true},
		{name: "site mod", memberRole: "member", siteRole: new(authz.RoleModerator), want: true},
		{name: "super admin", memberRole: "member", siteRole: new(authz.RoleSuperAdmin), want: true},
		{name: "regular member", memberRole: "member", siteRole: new(role.Role("")), want: false},
		{name: "non member", memberRole: "", siteRole: new(role.Role("")), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			roomID := uuid.New()
			userID := uuid.New()
			m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: userID}).Return(tc.memberRole, nil)
			if tc.siteRole != nil {
				m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(*tc.siteRole, nil)
			}

			// when
			got, err := svc.canModerateRoom(context.Background(), roomID, userID)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
