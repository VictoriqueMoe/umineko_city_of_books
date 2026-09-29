package chat

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	roomSettingsRejection struct {
		name    string
		given   func(m *testMocks, ref spec.ChatMemberRef)
		wantErr error
	}
)

func roomSettingsFixture(t *testing.T) (*service, *testMocks, spec.ChatMemberRef) {
	svc, m := newTestService(t)

	return svc, m, spec.ChatMemberRef{RoomID: uuid.New(), UserID: uuid.New()}
}

func roomSettingsExpectEditPermitted(m *testMocks, ref spec.ChatMemberRef, siteRole role.Role) {
	m.chatRepo.EXPECT().IsMember(mock.Anything, ref).Return(true, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, ref.UserID).Return(siteRole, nil)

	if siteRole == "" {
		m.chatRepo.EXPECT().IsMemberNicknameLocked(mock.Anything, ref).Return(false, nil)
	}
}

func roomSettingsGateRejections() []roomSettingsRejection {
	return []roomSettingsRejection{
		{
			name: "a non-member is turned away",
			given: func(m *testMocks, ref spec.ChatMemberRef) {
				m.chatRepo.EXPECT().IsMember(mock.Anything, ref).Return(false, nil)
			},
			wantErr: ErrNotMember,
		},
		{
			name: "a regular member with a locked nickname is refused",
			given: func(m *testMocks, ref spec.ChatMemberRef) {
				m.chatRepo.EXPECT().IsMember(mock.Anything, ref).Return(true, nil)
				m.authzSvc.EXPECT().GetRole(mock.Anything, ref.UserID).Return("", nil)
				m.chatRepo.EXPECT().IsMemberNicknameLocked(mock.Anything, ref).Return(true, nil)
			},
			wantErr: ErrNicknameLocked,
		},
	}
}

func roomSettingsExpectAvatarSave(m *testMocks, ref spec.ChatMemberRef, reader any, avatarURL string, saveErr error) {
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1024)
	m.uploadSvc.EXPECT().SaveImage(mock.Anything, "chat-avatars/"+ref.RoomID.String(), ref.UserID, int64(3), int64(1024), reader).Return(avatarURL, saveErr)
}

func roomSettingsExpectMemberBroadcast(m *testMocks, ref spec.ChatMemberRef, nickname, avatarURL string) {
	row := model.ChatRoomMemberRow{UserID: ref.UserID, Role: "member", Nickname: nickname, MemberAvatarURL: avatarURL}

	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, ref.RoomID).Return([]model.ChatRoomMemberRow{row}, nil).Once()
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{ref.UserID}).Return(nil, nil)
}

func TestSetRoomMuted(t *testing.T) {
	errBoom := errors.New("boom")
	cases := []struct {
		name      string
		isMember  bool
		memberErr error
		muted     bool
		writeErr  error
		wantErr   error
	}{
		{name: "the membership check failing is returned", memberErr: errBoom, muted: true, wantErr: errBoom},
		{name: "a non-member is turned away", muted: true, wantErr: ErrNotMember},
		{name: "the mute write failing is returned", isMember: true, muted: true, writeErr: errBoom, wantErr: errBoom},
		{name: "a member unmutes the room", isMember: true, muted: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m, ref := roomSettingsFixture(t)
			m.chatRepo.EXPECT().IsMember(mock.Anything, ref).Return(tc.isMember, tc.memberErr)
			if tc.isMember {
				m.chatRepo.EXPECT().SetMuted(mock.Anything, spec.ChatMemberMuteUpdate{RoomID: ref.RoomID, UserID: ref.UserID, Muted: tc.muted}).Return(tc.writeErr)
			}

			// when
			err := svc.SetRoomMuted(context.Background(), ref.RoomID, ref.UserID, tc.muted)

			// then
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestIsRoomMuted_Delegates(t *testing.T) {
	// given
	svc, m, ref := roomSettingsFixture(t)
	m.chatRepo.EXPECT().IsMuted(mock.Anything, ref).Return(true, nil)

	// when
	got, err := svc.IsRoomMuted(context.Background(), ref.RoomID, ref.UserID)

	// then
	require.NoError(t, err)
	assert.True(t, got)
}

func TestSetRoomNickname(t *testing.T) {
	cases := []struct {
		name     string
		siteRole role.Role
		input    string
		want     string
	}{
		{name: "a member sets their room nickname", input: "Alice", want: "Alice"},
		{name: "the nickname is trimmed and capped at 32 characters", input: "  " + strings.Repeat("a", 50) + "  ", want: strings.Repeat("a", 32)},
		{name: "a site moderator bypasses the nickname lock", siteRole: authz.RoleModerator, input: "Alice", want: "Alice"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m, ref := roomSettingsFixture(t)
			roomSettingsExpectEditPermitted(m, ref, tc.siteRole)
			m.chatRepo.EXPECT().SetMemberNickname(mock.Anything, spec.ChatMemberNicknameUpdate{RoomID: ref.RoomID, UserID: ref.UserID, Nickname: tc.want}).Return(nil)
			m.userRepo.EXPECT().GetByID(mock.Anything, ref.UserID).Return(sampleUser(ref.UserID), nil)
			m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.MatchedBy(func(s spec.NewChatMessage) bool {
				return s.RoomID == ref.RoomID && s.SenderID == ref.UserID
			})).Return(nil, errors.New("boom"))
			roomSettingsExpectMemberBroadcast(m, ref, tc.want, "")

			// when
			got, err := svc.SetRoomNickname(context.Background(), ref.RoomID, ref.UserID, tc.input)

			// then
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tc.want, got.Nickname)
		})
	}
}

func TestSetRoomNickname_Rejected(t *testing.T) {
	errBoom := errors.New("boom")
	cases := append(roomSettingsGateRejections(),
		roomSettingsRejection{
			name: "the membership check failing is returned",
			given: func(m *testMocks, ref spec.ChatMemberRef) {
				m.chatRepo.EXPECT().IsMember(mock.Anything, ref).Return(false, errBoom)
			},
			wantErr: errBoom,
		},
		roomSettingsRejection{
			name: "the nickname write failing is returned",
			given: func(m *testMocks, ref spec.ChatMemberRef) {
				roomSettingsExpectEditPermitted(m, ref, "")
				m.chatRepo.EXPECT().SetMemberNickname(mock.Anything, spec.ChatMemberNicknameUpdate{RoomID: ref.RoomID, UserID: ref.UserID, Nickname: "nick"}).Return(errBoom)
			},
			wantErr: errBoom,
		},
	)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m, ref := roomSettingsFixture(t)
			tc.given(m, ref)

			// when
			got, err := svc.SetRoomNickname(context.Background(), ref.RoomID, ref.UserID, "nick")

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, got)
		})
	}
}

func TestSetRoomAvatar(t *testing.T) {
	cases := []struct {
		name     string
		siteRole role.Role
	}{
		{name: "a member uploads a room avatar"},
		{name: "a site admin bypasses the nickname lock", siteRole: authz.RoleAdmin},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m, ref := roomSettingsFixture(t)
			data := bytes.NewReader([]byte("img"))
			roomSettingsExpectEditPermitted(m, ref, tc.siteRole)
			roomSettingsExpectAvatarSave(m, ref, data, "avatar.png", nil)
			m.chatRepo.EXPECT().SetMemberAvatar(mock.Anything, spec.ChatMemberAvatarUpdate{RoomID: ref.RoomID, UserID: ref.UserID, AvatarURL: "avatar.png"}).Return(nil)
			roomSettingsExpectMemberBroadcast(m, ref, "", "avatar.png")

			// when
			got, err := svc.SetRoomAvatar(context.Background(), ref.RoomID, ref.UserID, "image/png", 3, data)

			// then
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "avatar.png", got.MemberAvatarURL)
		})
	}
}

func TestSetRoomAvatar_Rejected(t *testing.T) {
	errBoom := errors.New("boom")
	cases := append(roomSettingsGateRejections(),
		roomSettingsRejection{
			name: "the upload failing is returned",
			given: func(m *testMocks, ref spec.ChatMemberRef) {
				roomSettingsExpectEditPermitted(m, ref, "")
				roomSettingsExpectAvatarSave(m, ref, mock.Anything, "", errBoom)
			},
			wantErr: errBoom,
		},
		roomSettingsRejection{
			name: "the avatar write failing is returned",
			given: func(m *testMocks, ref spec.ChatMemberRef) {
				roomSettingsExpectEditPermitted(m, ref, "")
				roomSettingsExpectAvatarSave(m, ref, mock.Anything, "avatar.png", nil)
				m.chatRepo.EXPECT().SetMemberAvatar(mock.Anything, spec.ChatMemberAvatarUpdate{RoomID: ref.RoomID, UserID: ref.UserID, AvatarURL: "avatar.png"}).Return(errBoom)
			},
			wantErr: errBoom,
		},
	)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m, ref := roomSettingsFixture(t)
			tc.given(m, ref)

			// when
			got, err := svc.SetRoomAvatar(context.Background(), ref.RoomID, ref.UserID, "image/png", 3, bytes.NewReader([]byte("img")))

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, got)
		})
	}
}

func TestClearRoomAvatar(t *testing.T) {
	cases := []struct {
		name          string
		siteRole      role.Role
		currentAvatar string
	}{
		{name: "a member's existing avatar file is deleted", currentAvatar: "old.png"},
		{name: "nothing is deleted when the member has no avatar"},
		{name: "a site super admin bypasses the nickname lock", siteRole: authz.RoleSuperAdmin},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m, ref := roomSettingsFixture(t)
			roomSettingsExpectEditPermitted(m, ref, tc.siteRole)
			m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, ref.RoomID).Return([]model.ChatRoomMemberRow{
				{UserID: ref.UserID, Role: "member", MemberAvatarURL: tc.currentAvatar},
			}, nil).Once()
			if tc.currentAvatar != "" {
				m.uploadSvc.EXPECT().Delete([]string{tc.currentAvatar}).Return()
			}
			m.chatRepo.EXPECT().SetMemberAvatar(mock.Anything, spec.ChatMemberAvatarUpdate{RoomID: ref.RoomID, UserID: ref.UserID, AvatarURL: ""}).Return(nil)
			roomSettingsExpectMemberBroadcast(m, ref, "", "")

			// when
			got, err := svc.ClearRoomAvatar(context.Background(), ref.RoomID, ref.UserID)

			// then
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "", got.MemberAvatarURL)
		})
	}
}

func TestClearRoomAvatar_Rejected(t *testing.T) {
	errBoom := errors.New("boom")
	cases := append(roomSettingsGateRejections(),
		roomSettingsRejection{
			name: "the current avatar lookup failing is returned before anything is cleared",
			given: func(m *testMocks, ref spec.ChatMemberRef) {
				roomSettingsExpectEditPermitted(m, ref, "")
				m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, ref.RoomID).Return(nil, errBoom)
			},
			wantErr: errBoom,
		},
		roomSettingsRejection{
			name: "the avatar write failing is returned and the file it still points at is kept",
			given: func(m *testMocks, ref spec.ChatMemberRef) {
				roomSettingsExpectEditPermitted(m, ref, "")
				m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, ref.RoomID).Return([]model.ChatRoomMemberRow{
					{UserID: ref.UserID, Role: "member", MemberAvatarURL: "old.png"},
				}, nil)
				m.chatRepo.EXPECT().SetMemberAvatar(mock.Anything, spec.ChatMemberAvatarUpdate{RoomID: ref.RoomID, UserID: ref.UserID, AvatarURL: ""}).Return(errBoom)
			},
			wantErr: errBoom,
		},
	)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m, ref := roomSettingsFixture(t)
			tc.given(m, ref)

			// when
			got, err := svc.ClearRoomAvatar(context.Background(), ref.RoomID, ref.UserID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, got)
		})
	}
}
