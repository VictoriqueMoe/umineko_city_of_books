package chat

import (
	"context"
	"errors"
	"strings"
	"testing"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	membersTimeoutUntil = "2099-01-01 00:00:00"
)

type (
	membersIDs struct {
		room   uuid.UUID
		actor  uuid.UUID
		target uuid.UUID
	}
)

func membersExpectRoomAndActorRole(m *testMocks, ids membersIDs, room *model.ChatRoomRow, actorRoomRole string) {
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: ids.room, ViewerID: ids.actor}).Return(room, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: ids.room, UserID: ids.actor}).Return(actorRoomRole, nil)
}

func membersExpectMemberTarget(m *testMocks, ids membersIDs, targetSiteRole role.Role) {
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: ids.room, UserID: ids.target}).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, ids.target).Return(targetSiteRole, nil)
}

func membersExpectTimeoutState(m *testMocks, ids membersIDs, active, byStaff bool) {
	until := ""
	if active {
		until = membersTimeoutUntil
	}

	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, spec.ChatMemberRef{RoomID: ids.room, UserID: ids.target}).Return(active, until, byStaff, nil)
}

func membersExpectAnnouncedUpdate(m *testMocks, ids membersIDs, member model.ChatRoomMemberRow) {
	m.userRepo.EXPECT().GetByID(mock.Anything, ids.actor).Return(sampleUser(ids.actor), nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, ids.target).Return(sampleUser(ids.target), nil)
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.MatchedBy(func(s spec.NewChatMessage) bool {
		return s.RoomID == ids.room && s.SenderID == ids.actor
	})).Return(nil, errors.New("boom"))
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, ids.room).Return([]model.ChatRoomMemberRow{member}, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{ids.target}).Return(nil, nil)
}

func membersResponseIDs(members []dto.ChatRoomMemberResponse) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.User.ID)
	}

	return ids
}

func TestInviteMembers_BotsOnlyJoinRPRooms(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name         string
		isRP         bool
		invitee      func(uuid.UUID) *model.User
		wantRejected bool
	}{
		{name: "a bot invited into a non-RP room is rejected", isRP: false, invitee: botUser, wantRejected: true},
		{name: "a bot invited into an RP room is not rejected", isRP: true, invitee: botUser},
		{name: "a human invited into a non-RP room is not rejected", isRP: false, invitee: sampleUser},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			ids := membersIDs{room: uuid.New(), actor: uuid.New(), target: uuid.New()}
			membersExpectRoomAndActorRole(m, ids, &model.ChatRoomRow{ID: ids.room, Type: dto.RoomTypeGroup, IsRP: tc.isRP}, "host")
			if !tc.isRP {
				m.userRepo.EXPECT().GetByIDs(mock.Anything, []uuid.UUID{ids.target}).Return([]model.User{*tc.invitee(ids.target)}, nil)
			}
			if !tc.wantRejected {
				m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxChatRoomMembers).Return(0)
				m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, ids.room).Return(nil, nil)
				m.userRepo.EXPECT().GetByID(mock.Anything, ids.actor).Return(sampleUser(ids.actor), nil)
				m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: ids.room, UserID: ids.target}).Return("", boom)
			}

			// when
			_, err := svc.InviteMembers(context.Background(), ids.actor, ids.room, []uuid.UUID{ids.target})

			// then
			if tc.wantRejected {
				require.ErrorIs(t, err, ErrBotsRPRoomsOnly)
			} else {
				require.ErrorIs(t, err, boom)
				require.NotErrorIs(t, err, ErrBotsRPRoomsOnly)
			}
			m.chatRepo.AssertNotCalled(t, "AddMemberWithSystemMessage", mock.Anything, mock.Anything)
		})
	}
}

func TestInviteMembers_AFailedLookupRefusesTheInviteInsteadOfSkippingThePerson(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name       string
		membersErr error
		inviterErr error
		userErr    error
		blockErr   error
	}{
		{name: "the room's member list", membersErr: boom},
		{name: "the inviter's profile", inviterErr: boom},
		{name: "the invitee's profile", userErr: boom},
		{name: "the block check between host and invitee", blockErr: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name+" failing is surfaced", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			ids := membersIDs{room: uuid.New(), actor: uuid.New(), target: uuid.New()}
			membersExpectRoomAndActorRole(m, ids, &model.ChatRoomRow{ID: ids.room, Type: dto.RoomTypeGroup, IsRP: true}, "host")
			m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxChatRoomMembers).Return(0)
			m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, ids.room).Return(nil, tc.membersErr)
			if tc.membersErr == nil {
				m.userRepo.EXPECT().GetByID(mock.Anything, ids.actor).Return(sampleUser(ids.actor), tc.inviterErr)
			}
			if tc.membersErr == nil && tc.inviterErr == nil {
				m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: ids.room, UserID: ids.target}).Return("", nil)
				m.userRepo.EXPECT().GetByID(mock.Anything, ids.target).Return(sampleUser(ids.target), tc.userErr)
			}
			if tc.membersErr == nil && tc.inviterErr == nil && tc.userErr == nil {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, ids.actor, ids.target).Return(false, tc.blockErr)
			}

			// when
			_, err := svc.InviteMembers(context.Background(), ids.actor, ids.room, []uuid.UUID{ids.target})

			// then
			require.ErrorIs(t, err, boom)
			m.chatRepo.AssertNotCalled(t, "AddMemberWithSystemMessage", mock.Anything, mock.Anything)
		})
	}
}

func TestKickMember_Refusals(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name    string
		given   func(m *testMocks, ids membersIDs)
		wantErr error
	}{
		{
			name: "a room lookup failure is returned",
			given: func(m *testMocks, ids membersIDs) {
				m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: ids.room, ViewerID: ids.actor}).Return(nil, boom)
			},
			wantErr: boom,
		},
		{
			name: "a missing room is not found",
			given: func(m *testMocks, ids membersIDs) {
				m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: ids.room, ViewerID: ids.actor}).Return(nil, nil)
			},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "nobody can be kicked from a system room",
			given: func(m *testMocks, ids membersIDs) {
				m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: ids.room, ViewerID: ids.actor}).Return(&model.ChatRoomRow{IsSystem: true}, nil)
			},
			wantErr: ErrSystemRoom,
		},
		{
			name: "an actor role lookup failure is returned",
			given: func(m *testMocks, ids membersIDs) {
				m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: ids.room, ViewerID: ids.actor}).Return(&model.ChatRoomRow{}, nil)
				m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: ids.room, UserID: ids.actor}).Return("", boom)
			},
			wantErr: boom,
		},
		{
			name: "an actor who is neither the host nor site staff is refused",
			given: func(m *testMocks, ids membersIDs) {
				membersExpectRoomAndActorRole(m, ids, &model.ChatRoomRow{}, "member")
				m.authzSvc.EXPECT().GetRole(mock.Anything, ids.actor).Return("", nil)
			},
			wantErr: ErrNotHost,
		},
		{
			name: "a target role lookup failure is returned",
			given: func(m *testMocks, ids membersIDs) {
				membersExpectRoomAndActorRole(m, ids, &model.ChatRoomRow{}, "host")
				m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: ids.room, UserID: ids.target}).Return("", boom)
			},
			wantErr: boom,
		},
		{
			name: "a target who is not a member is refused",
			given: func(m *testMocks, ids membersIDs) {
				membersExpectRoomAndActorRole(m, ids, &model.ChatRoomRow{}, "host")
				m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: ids.room, UserID: ids.target}).Return("", nil)
			},
			wantErr: ErrNotMember,
		},
		{
			name: "the host cannot be kicked",
			given: func(m *testMocks, ids membersIDs) {
				membersExpectRoomAndActorRole(m, ids, &model.ChatRoomRow{}, "host")
				m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: ids.room, UserID: ids.target}).Return("host", nil)
			},
			wantErr: ErrCannotKickHost,
		},
		{
			name: "a target who is site staff is immune",
			given: func(m *testMocks, ids membersIDs) {
				membersExpectRoomAndActorRole(m, ids, &model.ChatRoomRow{}, "host")
				membersExpectMemberTarget(m, ids, authz.RoleAdmin)
			},
			wantErr: ErrTargetImmune,
		},
		{
			name: "a member list failure is returned before anyone is removed",
			given: func(m *testMocks, ids membersIDs) {
				membersExpectRoomAndActorRole(m, ids, &model.ChatRoomRow{}, "host")
				membersExpectMemberTarget(m, ids, "")
				m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, ids.room).Return(nil, boom)
			},
			wantErr: boom,
		},
		{
			name: "a member removal failure is returned",
			given: func(m *testMocks, ids membersIDs) {
				membersExpectRoomAndActorRole(m, ids, &model.ChatRoomRow{}, "host")
				membersExpectMemberTarget(m, ids, "")
				m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, ids.room).Return([]uuid.UUID{ids.actor, ids.target}, nil)
				m.chatRepo.EXPECT().RemoveMember(mock.Anything, spec.ChatMemberRef{RoomID: ids.room, UserID: ids.target}).Return(boom)
			},
			wantErr: boom,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			ids := membersIDs{room: uuid.New(), actor: uuid.New(), target: uuid.New()}
			tc.given(m, ids)

			// when
			err := svc.KickMember(context.Background(), ids.actor, ids.room, ids.target)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestKickMember_RemovesTheTarget(t *testing.T) {
	cases := []struct {
		name          string
		public        bool
		actorRoomRole string
		actorSiteRole role.Role
	}{
		{name: "a host kicks a member from a private room without auditing it", actorRoomRole: "host"},
		{name: "site staff can kick without being the host", actorRoomRole: "member", actorSiteRole: authz.RoleModerator},
		{name: "a kick from a public room is audited", public: true, actorRoomRole: "host"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			ids := membersIDs{room: uuid.New(), actor: uuid.New(), target: uuid.New()}
			room := &model.ChatRoomRow{}
			if tc.public {
				room = &model.ChatRoomRow{ID: ids.room, Type: dto.RoomTypeGroup, IsPublic: true}
				m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry audit.NewEntry) bool {
					return entry.ActorID == ids.actor && entry.Action == audit.ActionChatRoomKick && entry.TargetType == audit.TargetChatRoom && entry.TargetID == ids.room.String() && entry.SubjectID == ids.target
				})).Return(nil)
			}
			membersExpectRoomAndActorRole(m, ids, room, tc.actorRoomRole)
			if tc.actorSiteRole != "" {
				m.authzSvc.EXPECT().GetRole(mock.Anything, ids.actor).Return(tc.actorSiteRole, nil)
			}
			membersExpectMemberTarget(m, ids, "")
			m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, ids.room).Return([]uuid.UUID{ids.actor, ids.target}, nil)
			m.chatRepo.EXPECT().RemoveMember(mock.Anything, spec.ChatMemberRef{RoomID: ids.room, UserID: ids.target}).Return(nil)
			expectEvictionSideEffects(m, ids.room)
			m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.MatchedBy(func(s spec.NewChatMessage) bool {
				return s.RoomID == ids.room && s.SenderID == ids.actor
			})).Return(nil, errors.New("boom"))
			m.hub.JoinRoom(ids.room, ids.target)

			// when
			err := svc.KickMember(context.Background(), ids.actor, ids.room, ids.target)

			// then
			require.NoError(t, err)
			assert.False(t, m.hub.IsUserInRoom(ids.room, ids.target))
		})
	}
}

func TestSetMemberTimeout_ByHost(t *testing.T) {
	cases := []struct {
		name               string
		public             bool
		staffTimeoutActive bool
		amount             int
		wantErr            error
	}{
		{name: "a zero amount is an invalid duration", amount: 0, wantErr: ErrInvalidTimeoutDuration},
		{name: "a host cannot change a timeout staff set", staffTimeoutActive: true, amount: 2, wantErr: ErrTimeoutLockedByStaff},
		{name: "a host times out a member of a private room without auditing it", amount: 1},
		{name: "a timeout in a public room is audited with its expiry and duration", public: true, amount: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			ids := membersIDs{room: uuid.New(), actor: uuid.New(), target: uuid.New()}
			room := &model.ChatRoomRow{}
			if tc.public {
				room = &model.ChatRoomRow{ID: ids.room, Type: dto.RoomTypeGroup, IsPublic: true}
				m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry audit.NewEntry) bool {
					return entry.ActorID == ids.actor && entry.Action == audit.ActionChatRoomTimeout && entry.TargetType == audit.TargetChatRoom && entry.TargetID == ids.room.String() && entry.SubjectID == ids.target && strings.HasPrefix(entry.Details, "until=") && strings.HasSuffix(entry.Details, " duration=1 hour")
				})).Return(nil)
			}
			membersExpectRoomAndActorRole(m, ids, room, "host")
			m.authzSvc.EXPECT().GetRole(mock.Anything, ids.actor).Return("", nil)
			membersExpectMemberTarget(m, ids, "")
			membersExpectTimeoutState(m, ids, tc.staffTimeoutActive, tc.staffTimeoutActive)
			if tc.wantErr == nil {
				m.chatRepo.EXPECT().SetMemberTimeout(mock.Anything, mock.MatchedBy(func(s spec.ChatMemberTimeout) bool {
					return s.RoomID == ids.room && s.UserID == ids.target && !s.ByStaff
				})).Return(nil)
				membersExpectAnnouncedUpdate(m, ids, model.ChatRoomMemberRow{UserID: ids.target, Username: "target", DisplayName: "Target", Role: "member", TimeoutUntil: membersTimeoutUntil})
			}

			// when
			got, err := svc.SetMemberTimeout(context.Background(), ids.room, ids.actor, ids.target, dto.SetMemberTimeoutRequest{Amount: tc.amount, Unit: "hours"})

			// then
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, ids.target, got.User.ID)
			}
		})
	}
}

func TestClearMemberTimeout_StaffLock(t *testing.T) {
	cases := []struct {
		name           string
		actorRoomRole  string
		actorSiteRole  role.Role
		timeoutByStaff bool
		wantErr        error
	}{
		{name: "a host cannot clear a timeout staff set", actorRoomRole: "host", timeoutByStaff: true, wantErr: ErrTimeoutLockedByStaff},
		{name: "site staff outside the room can clear a timeout a host set", actorRoomRole: "", actorSiteRole: authz.RoleAdmin, timeoutByStaff: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			ids := membersIDs{room: uuid.New(), actor: uuid.New(), target: uuid.New()}
			membersExpectRoomAndActorRole(m, ids, &model.ChatRoomRow{}, tc.actorRoomRole)
			m.authzSvc.EXPECT().GetRole(mock.Anything, ids.actor).Return(tc.actorSiteRole, nil)
			m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: ids.room, UserID: ids.target}).Return("member", nil)
			membersExpectTimeoutState(m, ids, true, tc.timeoutByStaff)
			if tc.wantErr == nil {
				m.chatRepo.EXPECT().ClearMemberTimeout(mock.Anything, spec.ChatMemberRef{RoomID: ids.room, UserID: ids.target}).Return(nil)
				membersExpectAnnouncedUpdate(m, ids, model.ChatRoomMemberRow{UserID: ids.target, Username: "target", DisplayName: "Target", Role: "member"})
			}

			// when
			got, err := svc.ClearMemberTimeout(context.Background(), ids.room, ids.actor, ids.target)

			// then
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, ids.target, got.User.ID)
			}
		})
	}
}

func TestGetMembers_Refusals(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name        string
		isMember    bool
		isMemberErr error
		listErr     error
		wantErr     error
	}{
		{name: "a membership lookup failure is returned", isMemberErr: boom, wantErr: boom},
		{name: "a viewer who is not a member is refused", wantErr: ErrNotMember},
		{name: "a member list failure is returned", isMember: true, listErr: boom, wantErr: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			viewerID := uuid.New()
			roomID := uuid.New()
			m.chatRepo.EXPECT().IsMember(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: viewerID}).Return(tc.isMember, tc.isMemberErr)
			if tc.isMember {
				m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return(nil, tc.listErr)
			}

			// when
			got, err := svc.GetMembers(context.Background(), viewerID, roomID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, got)
		})
	}
}

func TestGetMembers_ListsBotsAsOnlineAndStaffAliasesAsUnlocked(t *testing.T) {
	// given an always-online bot that is not viewing the room, a nickname-locked human and a nickname-locked site admin
	svc, m := newTestService(t)
	viewerID := uuid.New()
	roomID := uuid.New()
	botID := uuid.New()
	humanID := uuid.New()
	adminID := uuid.New()
	m.hub.SetAlwaysOnline([]uuid.UUID{botID})
	m.chatRepo.EXPECT().IsMember(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: viewerID}).Return(true, nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]model.ChatRoomMemberRow{
		{UserID: botID, Username: "beatrice", DisplayName: "Beatrice", Role: "member"},
		{UserID: humanID, Username: "battler", DisplayName: "Battler", Role: "member", NicknameLocked: true},
		{UserID: adminID, Username: "admin", DisplayName: "Admin", Role: "member", AuthorRole: string(authz.RoleAdmin), AuthorRoleTyped: authz.RoleAdmin, NicknameLocked: true},
	}, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{botID, humanID, adminID}).Return(nil, nil)

	// when
	got, err := svc.GetMembers(context.Background(), viewerID, roomID)

	// then
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{botID, humanID, adminID}, membersResponseIDs(got))
	assert.Equal(t, ws.ViewerStateActive, got[0].Presence)
	assert.Empty(t, got[1].Presence)
	assert.True(t, got[1].NicknameLocked)
	assert.False(t, got[2].NicknameLocked)
}

func TestGetMembers_Ghosts(t *testing.T) {
	cases := []struct {
		name           string
		viewerSiteRole role.Role
		ghostVisible   bool
	}{
		{name: "a viewer who is not site staff never sees ghost members", viewerSiteRole: ""},
		{name: "site staff see ghost members, flagged as ghosts", viewerSiteRole: authz.RoleModerator, ghostVisible: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			viewerID := uuid.New()
			roomID := uuid.New()
			normalID := uuid.New()
			ghostID := uuid.New()
			wantIDs := []uuid.UUID{normalID}
			if tc.ghostVisible {
				wantIDs = append(wantIDs, ghostID)
			}
			m.chatRepo.EXPECT().IsMember(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: viewerID}).Return(true, nil)
			m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]model.ChatRoomMemberRow{
				{UserID: normalID, Username: "a", DisplayName: "A", Role: "member"},
				{UserID: ghostID, Username: "g", DisplayName: "G", Role: "member", Ghost: true},
			}, nil)
			m.authzSvc.EXPECT().GetRole(mock.Anything, viewerID).Return(tc.viewerSiteRole, nil)
			m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, wantIDs).Return(nil, nil)

			// when
			got, err := svc.GetMembers(context.Background(), viewerID, roomID)

			// then
			require.NoError(t, err)
			require.Equal(t, wantIDs, membersResponseIDs(got))
			for _, member := range got {
				assert.Equal(t, member.User.ID == ghostID, member.Ghost)
			}
		})
	}
}

func TestGetMembers_SurfacesAFailedLookupInsteadOfGuessingWhoMaySeeGhosts(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name      string
		roleErr   error
		vanityErr error
	}{
		{name: "the viewer's site role", roleErr: boom},
		{name: "the members' vanity roles", vanityErr: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name+" failing is surfaced", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			viewerID := uuid.New()
			roomID := uuid.New()
			m.chatRepo.EXPECT().IsMember(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: viewerID}).Return(true, nil)
			m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]model.ChatRoomMemberRow{
				{UserID: uuid.New(), Username: "g", DisplayName: "G", Role: "member", Ghost: true},
			}, nil)
			m.authzSvc.EXPECT().GetRole(mock.Anything, viewerID).Return(authz.RoleModerator, tc.roleErr)
			if tc.roleErr == nil {
				m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, mock.Anything).Return(nil, tc.vanityErr)
			}

			// when
			got, err := svc.GetMembers(context.Background(), viewerID, roomID)

			// then
			require.ErrorIs(t, err, boom)
			assert.Nil(t, got)
		})
	}
}

func TestGetRoomMemberResponses_SurfacesAFailedLookupInsteadOfLeakingGhosts(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name     string
		hasErr   error
		roleErr  error
		ghostErr error
		userErr  error
	}{
		{name: "whether the room has ghosts", hasErr: boom},
		{name: "the viewer's site role", roleErr: boom},
		{name: "whether a member is a ghost", ghostErr: boom},
		{name: "a member's profile", userErr: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name+" failing is surfaced", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			unsetDefaults(&m.chatRepo.Mock, "HasGhostMembers", "IsGhostMember")
			viewerID := uuid.New()
			roomID := uuid.New()
			memberID := uuid.New()
			m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{memberID}, nil)
			m.chatRepo.EXPECT().HasGhostMembers(mock.Anything, roomID).Return(true, tc.hasErr)
			if tc.hasErr == nil {
				m.authzSvc.EXPECT().GetRole(mock.Anything, viewerID).Return("", tc.roleErr)
			}
			if tc.hasErr == nil && tc.roleErr == nil {
				m.chatRepo.EXPECT().IsGhostMember(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: memberID}).Return(false, tc.ghostErr)
			}
			if tc.hasErr == nil && tc.roleErr == nil && tc.ghostErr == nil {
				m.userRepo.EXPECT().GetByID(mock.Anything, memberID).Return(nil, tc.userErr)
			}

			// when
			got, _, err := svc.getRoomMemberResponses(context.Background(), roomID, viewerID)

			// then
			require.ErrorIs(t, err, boom)
			assert.Nil(t, got)
		})
	}
}

func TestSetMemberNicknameAsMod_Refusals(t *testing.T) {
	dbErr := errors.New("db")
	cases := []struct {
		name          string
		actorSiteRole role.Role
		given         func(m *testMocks, ids membersIDs)
		wantErr       error
	}{
		{
			name:          "an actor who is not site staff is refused, even a room host",
			actorSiteRole: "",
			given:         func(*testMocks, membersIDs) {},
			wantErr:       ErrModRoleRequired,
		},
		{
			name:          "a target who is not a member is refused",
			actorSiteRole: authz.RoleAdmin,
			given: func(m *testMocks, ids membersIDs) {
				m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: ids.room, UserID: ids.target}).Return("", nil)
			},
			wantErr: ErrNotMember,
		},
		{
			name:          "a target who is site staff is immune",
			actorSiteRole: authz.RoleAdmin,
			given: func(m *testMocks, ids membersIDs) {
				membersExpectMemberTarget(m, ids, authz.RoleModerator)
			},
			wantErr: ErrTargetImmune,
		},
		{
			name:          "a nickname write failure is returned",
			actorSiteRole: authz.RoleAdmin,
			given: func(m *testMocks, ids membersIDs) {
				membersExpectMemberTarget(m, ids, "")
				m.chatRepo.EXPECT().SetMemberNicknameWithLock(mock.Anything, spec.ChatMemberNicknameUpdate{RoomID: ids.room, UserID: ids.target, Nickname: "x", Locked: true}).Return(dbErr)
			},
			wantErr: dbErr,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			ids := membersIDs{room: uuid.New(), actor: uuid.New(), target: uuid.New()}
			m.authzSvc.EXPECT().GetRole(mock.Anything, ids.actor).Return(tc.actorSiteRole, nil)
			tc.given(m, ids)

			// when
			got, err := svc.SetMemberNicknameAsMod(context.Background(), ids.room, ids.actor, ids.target, "x")

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, got)
		})
	}
}

func TestSetMemberNicknameAsMod_LocksTheAlias(t *testing.T) {
	cases := []struct {
		name          string
		actorSiteRole role.Role
		nickname      string
		want          string
	}{
		{name: "a site moderator sets and locks a member's alias", actorSiteRole: authz.RoleModerator, nickname: "Silence", want: "Silence"},
		{name: "the alias is trimmed and capped at 32 characters", actorSiteRole: authz.RoleAdmin, nickname: "  " + strings.Repeat("a", 50), want: strings.Repeat("a", 32)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			ids := membersIDs{room: uuid.New(), actor: uuid.New(), target: uuid.New()}
			m.authzSvc.EXPECT().GetRole(mock.Anything, ids.actor).Return(tc.actorSiteRole, nil)
			membersExpectMemberTarget(m, ids, "")
			m.chatRepo.EXPECT().SetMemberNicknameWithLock(mock.Anything, spec.ChatMemberNicknameUpdate{RoomID: ids.room, UserID: ids.target, Nickname: tc.want, Locked: true}).Return(nil)
			membersExpectAnnouncedUpdate(m, ids, model.ChatRoomMemberRow{UserID: ids.target, Username: "t", DisplayName: "T", Role: "member", Nickname: tc.want, NicknameLocked: true})

			// when
			got, err := svc.SetMemberNicknameAsMod(context.Background(), ids.room, ids.actor, ids.target, tc.nickname)

			// then
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tc.want, got.Nickname)
			assert.True(t, got.NicknameLocked)
		})
	}
}

func TestUnlockMemberNickname_Refusals(t *testing.T) {
	dbErr := errors.New("db")
	cases := []struct {
		name          string
		actorSiteRole role.Role
		given         func(m *testMocks, ids membersIDs)
		wantErr       error
	}{
		{
			name:          "an actor who is not site staff is refused",
			actorSiteRole: "",
			given:         func(*testMocks, membersIDs) {},
			wantErr:       ErrModRoleRequired,
		},
		{
			name:          "a target who is site staff is immune",
			actorSiteRole: authz.RoleAdmin,
			given: func(m *testMocks, ids membersIDs) {
				membersExpectMemberTarget(m, ids, authz.RoleAdmin)
			},
			wantErr: ErrTargetImmune,
		},
		{
			name:          "a nickname reset failure is returned",
			actorSiteRole: authz.RoleAdmin,
			given: func(m *testMocks, ids membersIDs) {
				membersExpectMemberTarget(m, ids, "")
				m.chatRepo.EXPECT().SetMemberNicknameWithLock(mock.Anything, spec.ChatMemberNicknameUpdate{RoomID: ids.room, UserID: ids.target, Nickname: "", Locked: false}).Return(dbErr)
			},
			wantErr: dbErr,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			ids := membersIDs{room: uuid.New(), actor: uuid.New(), target: uuid.New()}
			m.authzSvc.EXPECT().GetRole(mock.Anything, ids.actor).Return(tc.actorSiteRole, nil)
			tc.given(m, ids)

			// when
			got, err := svc.UnlockMemberNickname(context.Background(), ids.room, ids.actor, ids.target)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, got)
		})
	}
}

func TestUnlockMemberNickname_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	ids := membersIDs{room: uuid.New(), actor: uuid.New(), target: uuid.New()}
	m.authzSvc.EXPECT().GetRole(mock.Anything, ids.actor).Return(authz.RoleModerator, nil)
	membersExpectMemberTarget(m, ids, "")
	m.chatRepo.EXPECT().SetMemberNicknameWithLock(mock.Anything, spec.ChatMemberNicknameUpdate{RoomID: ids.room, UserID: ids.target, Nickname: "", Locked: false}).Return(nil)
	membersExpectAnnouncedUpdate(m, ids, model.ChatRoomMemberRow{UserID: ids.target, Role: "member"})

	// when
	got, err := svc.UnlockMemberNickname(context.Background(), ids.room, ids.actor, ids.target)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.False(t, got.NicknameLocked)
}
