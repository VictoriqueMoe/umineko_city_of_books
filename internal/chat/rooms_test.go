package chat

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"testing"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func roomsExpectViewerRole(m *testMocks, row *model.ChatRoomRow, roomID, viewerID uuid.UUID, memberRole string) {
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: viewerID}).Return(row, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: viewerID}).Return(memberRole, nil)
}

func roomsExpectMemberLeaving(m *testMocks, roomID, userID uuid.UUID, roomType dto.RoomType, remaining int) {
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: userID}).
		Return(&model.ChatRoomRow{ID: roomID, Type: roomType, IsMember: true, ViewerRole: "member"}, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: userID}).Return(nil)
	expectEvictionSideEffects(m, roomID)
	m.chatRepo.EXPECT().CountRoomMembers(mock.Anything, roomID).Return(remaining, nil)
}

func roomsExpectDepartureAnnounced(m *testMocks, roomID, userID uuid.UUID) {
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, spec.NewChatMessage{RoomID: roomID, SenderID: userID, Body: "User left the room."}).
		Return(nil, errors.New("skip"))
}

func roomsExpectJoinChecks(m *testMocks, userID, creatorID uuid.UUID, capacity int) {
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, creatorID).Return(false, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxChatRoomMembers).Return(capacity)
}

func roomsExpectRPRoomWithABot(m *testMocks, roomID, actor, human, bot uuid.UUID) {
	row := editableRoom(roomID)
	row.IsRP = true

	roomsExpectViewerRole(m, row, roomID, actor, "host")
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{human, bot}, nil)
	m.userRepo.EXPECT().GetByIDs(mock.Anything, []uuid.UUID{human, bot}).Return([]model.User{*sampleUser(human), *botUser(bot)}, nil)
}

func roomsJoinAnnouncement(roomID, userID uuid.UUID) spec.ChatMemberJoinAnnouncement {
	return spec.ChatMemberJoinAnnouncement{
		Member:  spec.NewChatRoomMember{RoomID: roomID, UserID: userID, Role: "member"},
		Message: spec.NewChatMessage{RoomID: roomID, SenderID: userID, Body: "User joined the room.", IsSystem: true},
	}
}

func roomsCaptureAuditDetails(m *testMocks, matches func(audit.NewEntry) bool) *string {
	details := new(string)
	m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(matches)).
		Run(func(ctx context.Context, written audit.NewEntry, tx ...*sql.Tx) {
			*details = written.Details
		}).Return(nil)

	return details
}

func TestCreateGroupRoom_BuildsTheRepoSpecOrRefuses(t *testing.T) {
	creator := uuid.New()
	human := uuid.New()
	bot := uuid.New()
	boom := errors.New("boom")

	cases := []struct {
		name     string
		req      dto.CreateGroupRoomRequest
		invitees []model.User
		created  *spec.NewChatGroupRoom
		wantErr  error
	}{
		{name: "a blank name is missing fields", req: dto.CreateGroupRoomRequest{Name: "   "}, wantErr: ErrMissingFields},
		{
			name:    "a create failure surfaces",
			req:     dto.CreateGroupRoomRequest{Name: "Room"},
			created: &spec.NewChatGroupRoom{Name: "Room", CreatedBy: creator, MemberIDs: []uuid.UUID{}},
			wantErr: boom,
		},
		{
			name:    "tags reach the repo sanitised",
			req:     dto.CreateGroupRoomRequest{Name: "Room", Tags: []string{"  Tag1  "}},
			created: &spec.NewChatGroupRoom{Name: "Room", CreatedBy: creator, Tags: []string{"tag1"}, MemberIDs: []uuid.UUID{}},
			wantErr: boom,
		},
		{
			name:     "an invited human reaches the repo and its failure surfaces",
			req:      dto.CreateGroupRoomRequest{Name: "Room", MemberIDs: []uuid.UUID{human}},
			invitees: []model.User{*sampleUser(human)},
			created:  &spec.NewChatGroupRoom{Name: "Room", CreatedBy: creator, MemberIDs: []uuid.UUID{human}},
			wantErr:  boom,
		},
		{
			name:     "a bot outside a roleplay room is rejected before the room exists",
			req:      dto.CreateGroupRoomRequest{Name: "Room", MemberIDs: []uuid.UUID{bot}},
			invitees: []model.User{*botUser(bot)},
			wantErr:  ErrBotsRPRoomsOnly,
		},
		{
			name:    "a bot inside a roleplay room is let through without being looked up",
			req:     dto.CreateGroupRoomRequest{Name: "Room", IsRP: true, MemberIDs: []uuid.UUID{bot}},
			created: &spec.NewChatGroupRoom{Name: "Room", IsRP: true, CreatedBy: creator, MemberIDs: []uuid.UUID{bot}},
			wantErr: boom,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			for _, memberID := range tc.req.MemberIDs {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, creator, memberID).Return(false, nil)
			}
			if tc.invitees != nil {
				m.userRepo.EXPECT().GetByIDs(mock.Anything, tc.req.MemberIDs).Return(tc.invitees, nil)
			}
			if tc.created != nil {
				m.chatRepo.EXPECT().CreateGroupRoom(mock.Anything, *tc.created).Return(nil, boom)
			}

			// when
			_, err := svc.CreateGroupRoom(context.Background(), creator, tc.req)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			if tc.created == nil {
				m.chatRepo.AssertNotCalled(t, "CreateGroupRoom", mock.Anything, mock.Anything)

				return
			}

			require.NotErrorIs(t, err, ErrBotsRPRoomsOnly)
		})
	}
}

func TestCreateGroupRoom_SkipsBlockedMembers(t *testing.T) {
	// given
	svc, m := newTestService(t)
	creator := uuid.New()
	memberA := uuid.New()
	memberB := uuid.New()
	roomID := uuid.New()
	req := dto.CreateGroupRoomRequest{Name: "Room", MemberIDs: []uuid.UUID{creator, memberA, memberB}}
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, creator, memberA).Return(true, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, creator, memberB).Return(false, nil)
	m.userRepo.EXPECT().GetByIDs(mock.Anything, []uuid.UUID{memberB}).Return([]model.User{*sampleUser(memberB)}, nil)
	m.chatRepo.EXPECT().CreateGroupRoom(mock.Anything, spec.NewChatGroupRoom{Name: "Room", CreatedBy: creator, MemberIDs: []uuid.UUID{memberB}}).
		Return(&model.ChatRoomRow{ID: roomID}, nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: creator}).Return(&model.ChatRoomRow{Name: "Room"}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{creator, memberB}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, creator).Return(sampleUser(creator), nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, memberB).Return(sampleUser(memberB), nil)

	// when
	got, err := svc.CreateGroupRoom(context.Background(), creator, req)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, m.hub.IsUserInRoom(roomID, memberB))
	assert.False(t, m.hub.IsUserInRoom(roomID, memberA))
}

func TestCreateGroupRoom_AFailedBlockLookupRefusesTheRoom(t *testing.T) {
	// given
	svc, m := newTestService(t)
	creator := uuid.New()
	memberA := uuid.New()
	boom := errors.New("block lookup failed")
	req := dto.CreateGroupRoomRequest{Name: "Room", MemberIDs: []uuid.UUID{memberA}}
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, creator, memberA).Return(false, boom)

	// when
	got, err := svc.CreateGroupRoom(context.Background(), creator, req)

	// then
	require.ErrorIs(t, err, boom, "an unknown block state must not add the member anyway")
	assert.Nil(t, got)
	m.chatRepo.AssertNotCalled(t, "CreateGroupRoom", mock.Anything, mock.Anything)
}

func TestListPublicRooms_SurfacesAFailedFilterLookup(t *testing.T) {
	boom := errors.New("boom")

	cases := []struct {
		name      string
		blockErr  error
		bannedErr error
	}{
		{name: "the viewer's block list", blockErr: boom},
		{name: "the viewer's room bans", bannedErr: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name+" failing is surfaced instead of listing unfiltered rooms", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			viewerID := uuid.New()
			m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, tc.blockErr)
			if tc.blockErr == nil {
				m.chatRepo.EXPECT().ListPublicRooms(mock.Anything, mock.Anything).Return([]model.ChatRoomRow{{ID: uuid.New()}}, 1, nil)
				unsetDefaults(&m.banRepo.Mock, "BannedRoomIDsForUser")
				m.banRepo.EXPECT().BannedRoomIDsForUser(mock.Anything, viewerID).Return(nil, tc.bannedErr)
			}

			// when
			got, err := svc.ListPublicRooms(context.Background(), "", false, "", viewerID, false, 10, 0)

			// then
			require.ErrorIs(t, err, boom)
			assert.Nil(t, got)
		})
	}
}

func TestUpdateGroupRoom_Guards(t *testing.T) {
	roomID := uuid.New()
	actor := uuid.New()

	cases := []struct {
		name    string
		req     dto.UpdateGroupRoomRequest
		setup   func(m *testMocks)
		wantErr error
	}{
		{
			name: "room not found",
			req:  editRequest("New name"),
			setup: func(m *testMocks) {
				m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: actor}).Return(nil, nil)
			},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "system room",
			req:  editRequest("New name"),
			setup: func(m *testMocks) {
				row := editableRoom(roomID)
				row.IsSystem = true
				row.SystemKind = "announcements"
				m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: actor}).Return(row, nil)
			},
			wantErr: ErrSystemRoom,
		},
		{
			name: "neither host nor site staff",
			req:  editRequest("New name"),
			setup: func(m *testMocks) {
				roomsExpectViewerRole(m, editableRoom(roomID), roomID, actor, "member")
				m.authzSvc.EXPECT().GetRole(mock.Anything, actor).Return("", nil)
			},
			wantErr: ErrNotHost,
		},
		{
			name: "a dm is not editable even for site staff",
			req:  editRequest("New name"),
			setup: func(m *testMocks) {
				row := editableRoom(roomID)
				row.Type = dto.RoomTypeDM
				roomsExpectViewerRole(m, row, roomID, actor, "member")
				m.authzSvc.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleModerator, nil)
			},
			wantErr: ErrNotGroupRoom,
		},
		{
			name: "blank name",
			req:  editRequest("   "),
			setup: func(m *testMocks) {
				roomsExpectViewerRole(m, editableRoom(roomID), roomID, actor, "host")
			},
			wantErr: ErrMissingFields,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			tc.setup(m)

			// when
			resp, err := svc.UpdateGroupRoom(context.Background(), roomID, actor, tc.req)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, resp)
			m.chatRepo.AssertNotCalled(t, "UpdateGroupRoom", mock.Anything, mock.Anything)
		})
	}
}

func TestUpdateGroupRoom_ContentFilterRejectionPropagates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	svc.contentFilter = contentfilter.New(rejectRoomEditRule{})
	roomID := uuid.New()
	actor := uuid.New()
	roomsExpectViewerRole(m, editableRoom(roomID), roomID, actor, "host")

	// when
	_, err := svc.UpdateGroupRoom(context.Background(), roomID, actor, editRequest("New name"))

	// then
	var rej *contentfilter.RejectedError
	require.ErrorAs(t, err, &rej)
	assert.Equal(t, "slur", rej.Rejection.Detail)
	m.chatRepo.AssertNotCalled(t, "UpdateGroupRoom", mock.Anything, mock.Anything)
}

func TestUpdateGroupRoom_RenameAuditsBroadcastsAndStaysSilentInTheRoom(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	actor := uuid.New()
	roomsExpectViewerRole(m, editableRoom(roomID), roomID, actor, "host")
	m.chatRepo.EXPECT().UpdateGroupRoom(mock.Anything, spec.UpdateChatRoom{RoomID: roomID, Name: "New name", Description: "old description", Tags: []string{"tag"}, IsPublic: true}).
		Return(nil)
	details := roomsCaptureAuditDetails(m, func(entry audit.NewEntry) bool {
		return entry.ActorID == actor &&
			entry.Action == audit.ActionChatRoomUpdate &&
			entry.TargetType == audit.TargetChatRoom &&
			entry.TargetID == roomID.String()
	})

	memberReads := 0
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).
		Run(func(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) {
			memberReads++
		}).Return(nil, nil)

	// when
	resp, err := svc.UpdateGroupRoom(context.Background(), roomID, actor, editRequest("New name"))

	// then
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, *details, `name="Old name"->"New name"`)
	assert.NotContains(t, *details, "is_public")
	assert.NotContains(t, *details, "is_rp")
	assert.Equal(t, 2, memberReads, "the response build reads the members once and the chat_room_updated broadcast reads them again, so one read means the broadcast never fired")
	m.chatRepo.AssertNotCalled(t, "InsertSystemMessage", mock.Anything, mock.Anything)
}

func TestUpdateGroupRoom_PrivacyChangePostsASystemMessage(t *testing.T) {
	cases := []struct {
		name        string
		wasPublic   bool
		nowPublic   bool
		wantBody    string
		wantDetails string
	}{
		{
			name:        "private to public",
			wasPublic:   false,
			nowPublic:   true,
			wantBody:    "User made this room public. Anyone who joins can read the history.",
			wantDetails: "is_public=false->true",
		},
		{
			name:        "public to private",
			wasPublic:   true,
			nowPublic:   false,
			wantBody:    "User made this room private.",
			wantDetails: "is_public=true->false",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			roomID := uuid.New()
			actor := uuid.New()
			row := editableRoom(roomID)
			row.IsPublic = tc.wasPublic
			req := editRequest("Old name")
			req.IsPublic = tc.nowPublic

			roomsExpectViewerRole(m, row, roomID, actor, "host")
			m.chatRepo.EXPECT().UpdateGroupRoom(mock.Anything, spec.UpdateChatRoom{RoomID: roomID, Name: "Old name", Description: "old description", Tags: []string{"tag"}, IsPublic: tc.nowPublic}).
				Return(nil)
			m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil)
			m.userRepo.EXPECT().GetByID(mock.Anything, actor).Return(sampleUser(actor), nil)
			m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, spec.NewChatMessage{RoomID: roomID, SenderID: actor, Body: tc.wantBody}).
				Return(nil, errors.New("skip"))
			details := roomsCaptureAuditDetails(m, func(entry audit.NewEntry) bool {
				return entry.Action == audit.ActionChatRoomUpdate
			})

			// when
			resp, err := svc.UpdateGroupRoom(context.Background(), roomID, actor, req)

			// then
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Contains(t, *details, tc.wantDetails, "a privacy flip must never be the one edit that goes unlogged")
		})
	}
}

func TestUpdateGroupRoom_TurningRPOffWithABotNeedsConfirmation(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	actor := uuid.New()
	human := uuid.New()
	bot := uuid.New()
	roomsExpectRPRoomWithABot(m, roomID, actor, human, bot)

	// when
	resp, err := svc.UpdateGroupRoom(context.Background(), roomID, actor, editRequest("Old name"))

	// then
	kicked, ok := errors.AsType[*ErrBotsWillBeKicked](err)
	require.True(t, ok)
	require.Len(t, kicked.Bots, 1)
	assert.Equal(t, bot, kicked.Bots[0].ID)
	assert.Equal(t, "Beatrice", kicked.Bots[0].DisplayName)
	assert.Nil(t, resp)
	m.chatRepo.AssertNotCalled(t, "UpdateGroupRoom", mock.Anything, mock.Anything)
	m.chatRepo.AssertNotCalled(t, "RemoveMember", mock.Anything, mock.Anything)
}

func TestUpdateGroupRoom_ConfirmedRPOffRemovesTheBotAndAppliesTheEdit(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	actor := uuid.New()
	human := uuid.New()
	bot := uuid.New()
	req := editRequest("Old name")
	req.ConfirmBotRemoval = true

	roomsExpectRPRoomWithABot(m, roomID, actor, human, bot)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: bot}).Return(nil)
	expectEvictionSideEffects(m, roomID)
	m.chatRepo.EXPECT().UpdateGroupRoom(mock.Anything, spec.UpdateChatRoom{RoomID: roomID, Name: "Old name", Description: "old description", Tags: []string{"tag"}, IsPublic: true}).
		Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, human).Return(sampleUser(human), nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, bot).Return(botUser(bot), nil)
	m.chatRepo.EXPECT().
		InsertSystemMessage(mock.Anything, spec.NewChatMessage{RoomID: roomID, SenderID: actor, Body: "Beatrice was removed because this room is no longer a roleplay room."}).
		Return(nil, errors.New("skip"))
	details := roomsCaptureAuditDetails(m, func(entry audit.NewEntry) bool {
		return entry.Action == audit.ActionChatRoomUpdate
	})

	// when
	resp, err := svc.UpdateGroupRoom(context.Background(), roomID, actor, req)

	// then
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, *details, "is_rp=true->false")
	assert.Contains(t, *details, "bots_removed=1")
	m.chatRepo.AssertCalled(t, "RemoveMember", mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: bot})
	m.chatRepo.AssertNotCalled(t, "RemoveMember", mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: human})
}

func TestListPublicRooms_NormalisesTheFilter(t *testing.T) {
	viewer := uuid.New()
	boom := errors.New("boom")

	cases := []struct {
		name    string
		search  string
		tag     string
		limit   int
		offset  int
		want    spec.ChatRoomFilter
		rows    []model.ChatRoomRow
		total   int
		repoErr error
	}{
		{
			name:   "defaults the page and trims and lowercases the tag",
			search: "q",
			tag:    "  TagX  ",
			offset: -5,
			want:   spec.ChatRoomFilter{Search: "q", Tag: "tagx", Limit: 20},
			rows:   []model.ChatRoomRow{{ID: uuid.New()}},
			total:  1,
		},
		{name: "clamps an oversized limit", limit: 500, want: spec.ChatRoomFilter{Limit: 100}},
		{name: "a repo failure surfaces", want: spec.ChatRoomFilter{Limit: 20}, repoErr: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
			m.chatRepo.EXPECT().ListPublicRooms(mock.Anything, spec.ChatPublicRoomFilter{ChatRoomFilter: tc.want, ViewerID: viewer}).
				Return(tc.rows, tc.total, tc.repoErr)

			// when
			got, err := svc.ListPublicRooms(context.Background(), tc.search, false, tc.tag, viewer, false, tc.limit, tc.offset)

			// then
			if tc.repoErr != nil {
				require.ErrorIs(t, err, tc.repoErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.total, got.Total)
			assert.Len(t, got.Rooms, len(tc.rows))
		})
	}
}

func TestListUserGroupRooms_NormalisesTheFilter(t *testing.T) {
	userID := uuid.New()
	boom := errors.New("boom")

	cases := []struct {
		name     string
		tag      string
		role     string
		limit    int
		offset   int
		want     spec.ChatRoomFilter
		wantRole string
		repoErr  error
	}{
		{name: "defaults the page, trims the tag and drops an unknown role", tag: "  Tag  ", role: "bogus", limit: -1, offset: -1, want: spec.ChatRoomFilter{Tag: "tag", Limit: 20}},
		{name: "keeps the host role and clamps the limit", role: "host", limit: 500, want: spec.ChatRoomFilter{Limit: 100}, wantRole: "host"},
		{name: "keeps the member role and the page", role: "member", limit: 10, offset: 5, want: spec.ChatRoomFilter{Limit: 10, Offset: 5}, wantRole: "member"},
		{name: "a repo failure surfaces", want: spec.ChatRoomFilter{Limit: 20}, repoErr: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.chatRepo.EXPECT().ListUserGroupRooms(mock.Anything, spec.ChatUserRoomFilter{ChatRoomFilter: tc.want, UserID: userID, Role: tc.wantRole}).
				Return(nil, 0, tc.repoErr)

			// when
			_, err := svc.ListUserGroupRooms(context.Background(), userID, "", false, tc.tag, tc.role, false, tc.limit, tc.offset)

			// then
			if tc.repoErr != nil {
				require.ErrorIs(t, err, tc.repoErr)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestJoinRoom_Refusals(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	creatorID := uuid.New()
	boom := errors.New("boom")
	publicRoom := &model.ChatRoomRow{ID: roomID, Type: dto.RoomTypeGroup, IsPublic: true, CreatedBy: creatorID}

	cases := []struct {
		name    string
		row     *model.ChatRoomRow
		getErr  error
		ghost   bool
		setup   func(m *testMocks)
		wantErr error
	}{
		{name: "a lookup failure surfaces", getErr: boom, wantErr: boom},
		{name: "an unknown room is not found", wantErr: ErrRoomNotFound},
		{name: "a dm is not a group room", row: &model.ChatRoomRow{ID: roomID, Type: dto.RoomTypeDM}, wantErr: ErrNotGroupRoom},
		{name: "a system room cannot be joined", row: &model.ChatRoomRow{ID: roomID, Type: dto.RoomTypeGroup, IsSystem: true}, wantErr: ErrSystemRoom},
		{name: "a private room cannot be joined", row: &model.ChatRoomRow{ID: roomID, Type: dto.RoomTypeGroup, IsPublic: false}, wantErr: ErrNotPublic},
		{
			name:  "a ghost join needs site staff",
			row:   publicRoom,
			ghost: true,
			setup: func(m *testMocks) {
				m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return("", nil)
			},
			wantErr: ErrGhostRequiresStaff,
		},
		{
			name: "a block between the joiner and the creator refuses the join",
			row:  publicRoom,
			setup: func(m *testMocks) {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, creatorID).Return(true, nil)
			},
			wantErr: ErrUserBlocked,
		},
		{
			name: "a failed block lookup refuses the join",
			row:  publicRoom,
			setup: func(m *testMocks) {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, creatorID).Return(false, boom)
			},
			wantErr: boom,
		},
		{
			name: "a room at capacity is full",
			row:  &model.ChatRoomRow{ID: roomID, Type: dto.RoomTypeGroup, IsPublic: true, CreatedBy: creatorID, MemberCount: 10},
			setup: func(m *testMocks) {
				roomsExpectJoinChecks(m, userID, creatorID, 10)
			},
			wantErr: ErrRoomFull,
		},
		{
			name: "a joiner profile failure refuses the join before anyone is added",
			row:  publicRoom,
			setup: func(m *testMocks) {
				roomsExpectJoinChecks(m, userID, creatorID, 0)
				m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, boom)
			},
			wantErr: boom,
		},
		{
			name: "an add failure surfaces",
			row:  publicRoom,
			setup: func(m *testMocks) {
				roomsExpectJoinChecks(m, userID, creatorID, 0)
				m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)
				m.chatRepo.EXPECT().AddMemberWithSystemMessage(mock.Anything, roomsJoinAnnouncement(roomID, userID)).Return(nil, boom)
			},
			wantErr: boom,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: userID}).Return(tc.row, tc.getErr)
			if tc.setup != nil {
				tc.setup(m)
			}

			// when
			_, err := svc.JoinRoom(context.Background(), roomID, userID, tc.ghost)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestJoinRoom_AlreadyMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: userID}).
		Return(&model.ChatRoomRow{ID: roomID, Type: dto.RoomTypeGroup, IsPublic: true, IsMember: true}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil)

	// when
	got, err := svc.JoinRoom(context.Background(), roomID, userID, false)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, m.hub.IsUserInRoom(roomID, userID))
}

func TestJoinRoom_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	creatorID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: userID}).
		Return(&model.ChatRoomRow{ID: roomID, Type: dto.RoomTypeGroup, IsPublic: true, CreatedBy: creatorID}, nil)
	roomsExpectJoinChecks(m, userID, creatorID, 100)
	m.chatRepo.EXPECT().AddMemberWithSystemMessage(mock.Anything, roomsJoinAnnouncement(roomID, userID)).Return(&model.ChatMessageRow{ID: uuid.New()}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUser(mock.Anything, userID).Return(nil, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)

	// when
	got, err := svc.JoinRoom(context.Background(), roomID, userID, false)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, m.hub.IsUserInRoom(roomID, userID))
}

func TestJoinRoom_Ghost_StaffAllowedAndSilent(t *testing.T) {
	// given
	svc, m := newTestService(t)
	unsetDefaults(&m.chatRepo.Mock, "HasGhostMembers", "IsGhostMember")
	roomID := uuid.New()
	userID := uuid.New()
	creatorID := uuid.New()
	otherMember := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: userID}).
		Return(&model.ChatRoomRow{ID: roomID, Type: dto.RoomTypeGroup, IsPublic: true, CreatedBy: creatorID}, nil).Once()
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return("moderator", nil)
	roomsExpectJoinChecks(m, userID, creatorID, 0)
	m.chatRepo.EXPECT().AddMemberWithSystemMessage(mock.Anything, spec.ChatMemberJoinAnnouncement{
		Member:  spec.NewChatRoomMember{RoomID: roomID, UserID: userID, Role: "member", Ghost: true},
		Message: spec.NewChatMessage{RoomID: roomID, SenderID: userID, IsSystem: true},
	}).Return(nil, nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: userID}).
		Return(&model.ChatRoomRow{ID: roomID, Type: dto.RoomTypeGroup, IsMember: true}, nil).Once()
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID, otherMember}, nil)
	m.chatRepo.EXPECT().HasGhostMembers(mock.Anything, roomID).Return(true, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, otherMember).Return(sampleUser(otherMember), nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, otherMember).Return("", nil).Maybe()

	// when
	_, err := svc.JoinRoom(context.Background(), roomID, userID, true)

	// then
	require.NoError(t, err)
	m.chatRepo.AssertNotCalled(t, "InsertSystemMessage", mock.Anything, mock.Anything)
}

func TestLeaveRoom_Refusals(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	boom := errors.New("boom")

	cases := []struct {
		name    string
		row     *model.ChatRoomRow
		getErr  error
		setup   func(m *testMocks)
		wantErr error
	}{
		{name: "a lookup failure surfaces", getErr: boom, wantErr: boom},
		{name: "an unknown room means not a member", wantErr: ErrNotMember},
		{name: "a room the user is not in means not a member", row: &model.ChatRoomRow{ID: roomID, IsMember: false}, wantErr: ErrNotMember},
		{name: "a system room cannot be left", row: &model.ChatRoomRow{ID: roomID, IsMember: true, IsSystem: true}, wantErr: ErrSystemRoom},
		{name: "the host cannot leave", row: &model.ChatRoomRow{ID: roomID, IsMember: true, ViewerRole: "host"}, wantErr: ErrCannotLeaveAsHost},
		{
			name: "a remove failure surfaces",
			row:  &model.ChatRoomRow{ID: roomID, IsMember: true, ViewerRole: "member"},
			setup: func(m *testMocks) {
				m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
				m.chatRepo.EXPECT().RemoveMember(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: userID}).Return(boom)
			},
			wantErr: boom,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: userID}).Return(tc.row, tc.getErr)
			if tc.setup != nil {
				tc.setup(m)
			}

			// when
			err := svc.LeaveRoom(context.Background(), roomID, userID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestLeaveRoom_AFailedGhostCheckRefusesTheLeaveRatherThanAnnouncingAGhost(t *testing.T) {
	boom := errors.New("boom")

	cases := []struct {
		name        string
		audienceErr error
		hasErr      error
		ghostErr    error
	}{
		{name: "the audience", audienceErr: boom},
		{name: "whether the room has ghosts", hasErr: boom},
		{name: "whether the leaver is a ghost", ghostErr: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name+" failing refuses the leave before anything is removed", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			roomID := uuid.New()
			userID := uuid.New()
			unsetDefaults(&m.chatRepo.Mock, "HasGhostMembers", "IsGhostMember")
			m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: userID}).Return(&model.ChatRoomRow{ID: roomID, Type: dto.RoomTypeGroup, IsMember: true, ViewerRole: "member"}, nil)
			m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, tc.audienceErr)
			if tc.audienceErr == nil {
				m.chatRepo.EXPECT().HasGhostMembers(mock.Anything, roomID).Return(true, tc.hasErr)
			}
			if tc.audienceErr == nil && tc.hasErr == nil {
				m.chatRepo.EXPECT().IsGhostMember(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: userID}).Return(false, tc.ghostErr)
			}

			// when
			err := svc.LeaveRoom(context.Background(), roomID, userID)

			// then
			require.ErrorIs(t, err, boom)
			m.chatRepo.AssertNotCalled(t, "RemoveMember", mock.Anything, mock.Anything)
		})
	}
}

func TestLeaveRoom_Ghost_Silent(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	unsetDefaults(&m.chatRepo.Mock, "HasGhostMembers", "IsGhostMember")
	roomsExpectMemberLeaving(m, roomID, userID, dto.RoomTypeGroup, 1)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
	m.chatRepo.EXPECT().HasGhostMembers(mock.Anything, roomID).Return(true, nil).Once()
	m.chatRepo.EXPECT().IsGhostMember(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: userID}).Return(true, nil).Once()
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return("", nil).Maybe()

	// when
	err := svc.LeaveRoom(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
	m.chatRepo.AssertNotCalled(t, "InsertSystemMessage", mock.Anything, mock.Anything)
}

func TestLeaveRoom_SoftLeaveNeverAnnouncesInsideAPair(t *testing.T) {
	cases := []struct {
		name         string
		roomType     dto.RoomType
		wantAnnounce bool
	}{
		{
			name:         "a group room still posts the departure and broadcasts chat_member_left",
			roomType:     dto.RoomTypeGroup,
			wantAnnounce: true,
		},
		{
			name:     "a pair is left silently, because the only audience is the other party",
			roomType: dto.RoomTypeDM,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a member of a room of this kind
			svc, m := newTestService(t)
			roomID := uuid.New()
			userID := uuid.New()
			roomsExpectMemberLeaving(m, roomID, userID, tc.roomType, 1)
			m.hub.JoinRoom(roomID, userID)

			if tc.wantAnnounce {
				roomsExpectDepartureAnnounced(m, roomID, userID)
			}

			// when they leave
			err := svc.LeaveRoom(context.Background(), roomID, userID)

			// then the row is soft-left either way and only a room announces it
			require.NoError(t, err)
			assert.False(t, m.hub.IsUserInRoom(roomID, userID))
			m.chatRepo.AssertCalled(t, "RemoveMember", mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: userID})
			m.chatRepo.AssertNotCalled(t, "DeleteRoomWithMessages", mock.Anything, roomID)

			if tc.wantAnnounce {
				return
			}

			m.chatRepo.AssertNotCalled(t, "InsertSystemMessage", mock.Anything, mock.Anything)
			m.chatRepo.AssertNotCalled(t, "GetRoomMembers", mock.Anything, roomID)
			m.userRepo.AssertNotCalled(t, "GetByID", mock.Anything, userID)
		})
	}
}

func TestLeaveRoom_HardDeletesOnlyWhenTheLastMemberIsGone(t *testing.T) {
	cases := []struct {
		name       string
		roomType   dto.RoomType
		remaining  int
		wantDelete bool
	}{
		{
			name:      "a pair the other party is still in keeps every message for them",
			roomType:  dto.RoomTypeDM,
			remaining: 1,
		},
		{
			name:       "a pair both parties have left is destroyed",
			roomType:   dto.RoomTypeDM,
			remaining:  0,
			wantDelete: true,
		},
		{
			name:       "a room emptied through the leave route is destroyed too, which it never was before",
			roomType:   dto.RoomTypeGroup,
			remaining:  0,
			wantDelete: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a leaver in a room of this kind
			svc, m := newTestService(t)
			roomID := uuid.New()
			userID := uuid.New()
			roomsExpectMemberLeaving(m, roomID, userID, tc.roomType, tc.remaining)

			if tc.roomType != dto.RoomTypeDM {
				roomsExpectDepartureAnnounced(m, roomID, userID)
			}
			if tc.wantDelete {
				m.chatRepo.EXPECT().DeleteRoomWithMessages(mock.Anything, roomID).Return([]string{"chat/a.png"}, nil)
				m.uploadSvc.EXPECT().Delete([]string{"chat/a.png"}).Return()
			}

			// when they leave
			err := svc.LeaveRoom(context.Background(), roomID, userID)

			// then the room only dies once nobody is left in it
			require.NoError(t, err)
			if tc.wantDelete {
				return
			}

			m.chatRepo.AssertNotCalled(t, "DeleteRoomWithMessages", mock.Anything, roomID)
		})
	}
}

func TestListRooms_ReadsEachRoomsMembers(t *testing.T) {
	userID := uuid.New()
	roomID := uuid.New()
	boom := errors.New("boom")

	cases := []struct {
		name       string
		roomsErr   error
		membersErr error
	}{
		{name: "each room carries its members"},
		{name: "a room lookup failure surfaces", roomsErr: boom},
		{name: "a member lookup failure surfaces", membersErr: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.chatRepo.EXPECT().GetRoomsByUser(mock.Anything, userID).Return([]model.ChatRoomRow{{ID: roomID, Type: dto.RoomTypeGroup}}, tc.roomsErr)
			if tc.roomsErr == nil {
				m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, tc.membersErr)
			}
			if tc.roomsErr == nil && tc.membersErr == nil {
				m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)
			}

			// when
			got, err := svc.ListRooms(context.Background(), userID)

			// then
			wantErr := cmp.Or(tc.roomsErr, tc.membersErr)
			if wantErr != nil {
				require.ErrorIs(t, err, wantErr)

				return
			}

			require.NoError(t, err)
			require.Len(t, got.Rooms, 1)
			assert.Equal(t, roomID, got.Rooms[0].ID)
		})
	}
}

func TestDeleteChat_Failures(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	boom := errors.New("boom")
	dmRow := &model.ChatRoomRow{IsMember: true, Type: dto.RoomTypeDM}

	cases := []struct {
		name    string
		row     *model.ChatRoomRow
		getErr  error
		setup   func(m *testMocks)
		wantErr error
	}{
		{name: "a lookup failure surfaces", getErr: boom, wantErr: boom},
		{name: "an unknown room means not a member", wantErr: ErrNotMember},
		{name: "a system room cannot be deleted", row: &model.ChatRoomRow{IsMember: true, IsSystem: true}, wantErr: ErrSystemRoom},
		{
			name: "a remove failure while leaving a dm surfaces",
			row:  dmRow,
			setup: func(m *testMocks) {
				m.chatRepo.EXPECT().RemoveMember(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: userID}).Return(boom)
			},
			wantErr: boom,
		},
		{
			name: "a count failure after leaving a dm surfaces",
			row:  dmRow,
			setup: func(m *testMocks) {
				m.chatRepo.EXPECT().RemoveMember(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: userID}).Return(nil)
				expectEvictionSideEffects(m, roomID)
				m.chatRepo.EXPECT().CountRoomMembers(mock.Anything, roomID).Return(0, boom)
			},
			wantErr: boom,
		},
		{
			name: "a member list failure refuses the destroy before any watch party is torn down",
			row:  &model.ChatRoomRow{IsMember: true, Type: dto.RoomTypeGroup, ViewerRole: "host"},
			setup: func(m *testMocks) {
				m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: userID}).Return("host", nil)
				m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, boom)
			},
			wantErr: boom,
		},
		{
			name: "a delete failure when the host destroys a group room surfaces",
			row:  &model.ChatRoomRow{IsMember: true, Type: dto.RoomTypeGroup, ViewerRole: "host"},
			setup: func(m *testMocks) {
				m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: userID}).Return("host", nil)
				m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
				m.watchPartyRepo.EXPECT().ListActiveByRoom(mock.Anything, roomID).Return(nil, nil)
				m.chatRepo.EXPECT().DeleteRoomWithMessages(mock.Anything, roomID).Return(nil, boom)
			},
			wantErr: boom,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: userID}).Return(tc.row, tc.getErr)
			if tc.setup != nil {
				tc.setup(m)
			}

			// when
			err := svc.DeleteChat(context.Background(), roomID, userID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestDeleteChat_GroupModeratorDestroysTheRoom(t *testing.T) {
	roomID := uuid.New()
	actorID := uuid.New()
	memberID := uuid.New()

	cases := []struct {
		name       string
		row        *model.ChatRoomRow
		memberRole string
		siteRole   role.Role
		members    []uuid.UUID
		wantAudit  string
	}{
		{
			name:     "a site admin who is not a member, in a private room that is not audited",
			row:      &model.ChatRoomRow{Type: dto.RoomTypeGroup, IsMember: false},
			siteRole: authz.RoleAdmin,
		},
		{
			name:       "the host of a public room, audited with its name and member count",
			row:        &model.ChatRoomRow{ID: roomID, Name: "Rokkenjima", IsMember: true, Type: dto.RoomTypeGroup, IsPublic: true, ViewerRole: "host"},
			memberRole: "host",
			members:    []uuid.UUID{actorID, memberID},
			wantAudit:  "name=Rokkenjima members=2",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			roomsExpectViewerRole(m, tc.row, roomID, actorID, tc.memberRole)
			if tc.memberRole != "host" {
				m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(tc.siteRole, nil)
			}
			m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(tc.members, nil)
			m.watchPartyRepo.EXPECT().ListActiveByRoom(mock.Anything, roomID).Return(nil, nil)
			m.chatRepo.EXPECT().DeleteRoomWithMessages(mock.Anything, roomID).Return(nil, nil)
			m.uploadSvc.EXPECT().Delete().Return()
			if tc.wantAudit != "" {
				m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry audit.NewEntry) bool {
					return entry.ActorID == actorID && entry.Action == audit.ActionChatRoomDelete && entry.TargetType == audit.TargetChatRoom && entry.TargetID == roomID.String() && entry.Details == tc.wantAudit
				})).Return(nil)
			}

			// when
			err := svc.DeleteChat(context.Background(), roomID, actorID)

			// then
			require.NoError(t, err)
			if tc.wantAudit == "" {
				m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestDeleteChat_DM_StillHasMembers(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	roomsExpectMemberLeaving(m, roomID, userID, dto.RoomTypeDM, 1)

	// when
	err := svc.DeleteChat(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
	m.chatRepo.AssertNotCalled(t, "DeleteRoomWithMessages", mock.Anything, roomID)
}

func TestDeleteChat_GroupHost_EndsActiveWatchPartiesFirst(t *testing.T) {
	// given a group room a host is deleting while a watch party is still running in it
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	roomsExpectViewerRole(m, &model.ChatRoomRow{IsMember: true, Type: dto.RoomTypeGroup, ViewerRole: "host"}, roomID, userID, "host")
	m.watchPartyRepo.EXPECT().ListActiveByRoom(mock.Anything, roomID).Return([]model.ChatWatchPartySessionRow{
		{ID: sessionID, RoomID: roomID, HyperbeamSessionID: "hb_sess", Status: "active"},
	}, nil)
	m.hyperbeamSvc.EXPECT().TerminateVM(mock.Anything, "hb_sess").Return(nil)
	m.watchPartyRepo.EXPECT().MarkAllParticipantsLeft(mock.Anything, sessionID).Return(nil)
	m.watchPartyRepo.EXPECT().EndSession(mock.Anything, spec.WatchPartySessionEnd{SessionID: sessionID, Reason: "room_deleted"}).Return(nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, sessionID).Return(nil, nil)
	m.chatRepo.EXPECT().DeleteRoomWithMessages(mock.Anything, sessionID).Return(nil, nil)
	m.chatRepo.EXPECT().ClearVoiceForceMutes(mock.Anything, sessionID).Return(nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
	m.chatRepo.EXPECT().DeleteRoomWithMessages(mock.Anything, roomID).Return(nil, nil)
	m.uploadSvc.EXPECT().Delete().Return()

	// when the room is deleted
	err := svc.DeleteChat(context.Background(), roomID, userID)

	// then the party is torn down too, instead of its room being orphaned by the FK cascade
	require.NoError(t, err)
	m.chatRepo.AssertCalled(t, "DeleteRoomWithMessages", mock.Anything, roomID)
	m.uploadSvc.AssertNumberOfCalls(t, "Delete", 2)
	m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestLeaveAndDeleteChatShareOneLeavePathInAPair(t *testing.T) {
	cases := []struct {
		name  string
		leave func(*service, context.Context, uuid.UUID, uuid.UUID) error
	}{
		{name: "the leave route", leave: (*service).LeaveRoom},
		{name: "the delete-chat route", leave: (*service).DeleteChat},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given the last member of a pair
			svc, m := newTestService(t)
			roomID := uuid.New()
			userID := uuid.New()
			roomsExpectMemberLeaving(m, roomID, userID, dto.RoomTypeDM, 0)
			m.chatRepo.EXPECT().DeleteRoomWithMessages(mock.Anything, roomID).Return(nil, nil)
			m.uploadSvc.EXPECT().Delete().Return()

			// when the pair is left through either route
			err := tc.leave(svc, context.Background(), roomID, userID)

			// then both routes clean up and neither announces
			require.NoError(t, err)
			m.chatRepo.AssertNotCalled(t, "InsertSystemMessage", mock.Anything, mock.Anything)
		})
	}
}

func TestDeleteChat_OrdinaryMemberOfARoomNowLeavesItTheSameWayTheLeaveRouteDoes(t *testing.T) {
	// given an ordinary member of a group room using the delete-chat route, which used to vanish them silently
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	roomsExpectMemberLeaving(m, roomID, userID, dto.RoomTypeGroup, 1)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: userID}).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return("", nil)
	roomsExpectDepartureAnnounced(m, roomID, userID)

	// when they delete the chat
	err := svc.DeleteChat(context.Background(), roomID, userID)

	// then the room hears about it, because both routes are now the same leave path
	require.NoError(t, err)
	m.chatRepo.AssertCalled(t, "InsertSystemMessage", mock.Anything, spec.NewChatMessage{RoomID: roomID, SenderID: userID, Body: "User left the room."})
}
