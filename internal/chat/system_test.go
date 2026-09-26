package chat

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func systemExpectRoomIDs(m *testMocks, modsID, adminsID uuid.UUID) {
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(modsID, nil)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindAdmins).Return(adminsID, nil)
}

func systemExpectUnseeded(m *testMocks, supers []uuid.UUID, supersErr error) {
	systemExpectRoomIDs(m, uuid.Nil, uuid.Nil)
	m.roleRepo.EXPECT().GetUsersByRoles(mock.Anything, []role.Role{authz.RoleSuperAdmin}).Return(supers, supersErr)
}

func TestEnsureSystemRooms(t *testing.T) {
	boom := errors.New("boom")

	cases := []struct {
		name    string
		setup   func(m *testMocks)
		wantErr string
	}{
		{
			name: "mods room lookup error",
			setup: func(m *testMocks) {
				m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(uuid.Nil, boom)
			},
			wantErr: "get mods room: boom",
		},
		{
			name: "admins room lookup error",
			setup: func(m *testMocks) {
				m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(uuid.Nil, nil)
				m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindAdmins).Return(uuid.Nil, boom)
			},
			wantErr: "get admins room: boom",
		},
		{
			name: "both rooms already exist",
			setup: func(m *testMocks) {
				systemExpectRoomIDs(m, uuid.New(), uuid.New())
			},
		},
		{
			name: "no super admin to own the rooms",
			setup: func(m *testMocks) {
				systemExpectUnseeded(m, nil, nil)
			},
		},
		{
			name: "super admin lookup error",
			setup: func(m *testMocks) {
				systemExpectUnseeded(m, nil, boom)
			},
			wantErr: "find super admin: boom",
		},
		{
			name: "create rooms error",
			setup: func(m *testMocks) {
				systemExpectUnseeded(m, []uuid.UUID{uuid.New()}, nil)
				m.chatRepo.EXPECT().CreateSystemRooms(mock.Anything, mock.Anything).Return(boom)
			},
			wantErr: "boom",
		},
		{
			name: "creates both rooms owned by the first super admin",
			setup: func(m *testMocks) {
				super := uuid.New()
				systemExpectUnseeded(m, []uuid.UUID{super, uuid.New()}, nil)
				m.chatRepo.EXPECT().CreateSystemRooms(mock.Anything, mock.MatchedBy(func(specs []spec.NewChatSystemRoom) bool {
					return len(specs) == 2 &&
						specs[0].Name == systemModsName && specs[0].Description == systemModsDesc && specs[0].SystemKind == SystemKindMods && specs[0].CreatedBy == super &&
						specs[1].Name == systemAdminsName && specs[1].Description == systemAdminsDesc && specs[1].SystemKind == SystemKindAdmins && specs[1].CreatedBy == super
				})).Return(nil)
				m.roleRepo.EXPECT().GetUsersByRoles(mock.Anything, []role.Role{authz.RoleModerator, authz.RoleAdmin, authz.RoleSuperAdmin}).Return(nil, nil)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			tc.setup(m)

			// when
			err := svc.EnsureSystemRooms(context.Background())

			// then
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorIs(t, err, boom)
			assert.EqualError(t, err, tc.wantErr)
		})
	}
}

func TestSyncSystemRoomMembership(t *testing.T) {
	cases := []struct {
		name       string
		newRole    role.Role
		wantMods   bool
		wantAdmins bool
		wantRole   string
		joined     []string
		left       []string
	}{
		{
			name:       "admin joins both rooms as member",
			newRole:    authz.RoleAdmin,
			wantMods:   true,
			wantAdmins: true,
			wantRole:   "member",
			joined:     []string{SystemKindMods, SystemKindAdmins},
		},
		{
			name:       "super admin is host of both rooms",
			newRole:    authz.RoleSuperAdmin,
			wantMods:   true,
			wantAdmins: true,
			wantRole:   "host",
		},
		{
			name:     "moderator is removed from the admins room",
			newRole:  authz.RoleModerator,
			wantMods: true,
			wantRole: "member",
			left:     []string{SystemKindAdmins},
		},
		{
			name:     "demoted user is removed from both rooms",
			newRole:  "user",
			wantRole: "member",
			left:     []string{SystemKindMods},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			roomIDs := map[string]uuid.UUID{SystemKindMods: uuid.New(), SystemKindAdmins: uuid.New()}
			systemExpectRoomIDs(m, roomIDs[SystemKindMods], roomIDs[SystemKindAdmins])

			var changes []model.SystemRoomMembershipChange
			for _, kind := range tc.joined {
				changes = append(changes, model.SystemRoomMembershipChange{RoomID: roomIDs[kind], Joined: true})
			}
			for _, kind := range tc.left {
				m.hub.JoinRoom(roomIDs[kind], userID)
				changes = append(changes, model.SystemRoomMembershipChange{RoomID: roomIDs[kind], Left: true})
			}

			m.chatRepo.EXPECT().SyncSystemRoomMembership(mock.Anything, []spec.SystemRoomMembership{
				{RoomID: roomIDs[SystemKindMods], UserID: userID, ShouldBeMember: tc.wantMods, DesiredRole: tc.wantRole},
				{RoomID: roomIDs[SystemKindAdmins], UserID: userID, ShouldBeMember: tc.wantAdmins, DesiredRole: tc.wantRole},
			}).Return(changes, nil)

			// when
			err := svc.SyncSystemRoomMembership(context.Background(), userID, tc.newRole)

			// then
			require.NoError(t, err)
			for _, kind := range tc.joined {
				assert.True(t, m.hub.IsUserInRoom(roomIDs[kind], userID), kind)
			}
			for _, kind := range tc.left {
				assert.False(t, m.hub.IsUserInRoom(roomIDs[kind], userID), kind)
			}
		})
	}
}

func TestSyncSystemRoomMembership_Errors(t *testing.T) {
	boom := errors.New("boom")

	cases := []struct {
		name    string
		setup   func(m *testMocks)
		wantErr string
	}{
		{
			name: "mods room lookup error",
			setup: func(m *testMocks) {
				m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(uuid.Nil, boom)
			},
			wantErr: "get mods room: boom",
		},
		{
			name: "admins room lookup error",
			setup: func(m *testMocks) {
				m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(uuid.New(), nil)
				m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindAdmins).Return(uuid.Nil, boom)
			},
			wantErr: "get admins room: boom",
		},
		{
			name: "membership sync error",
			setup: func(m *testMocks) {
				systemExpectRoomIDs(m, uuid.New(), uuid.New())
				m.chatRepo.EXPECT().SyncSystemRoomMembership(mock.Anything, mock.Anything).Return(nil, boom)
			},
			wantErr: "boom",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			tc.setup(m)

			// when
			err := svc.SyncSystemRoomMembership(context.Background(), uuid.New(), authz.RoleAdmin)

			// then
			require.ErrorIs(t, err, boom)
			assert.EqualError(t, err, tc.wantErr)
		})
	}
}
