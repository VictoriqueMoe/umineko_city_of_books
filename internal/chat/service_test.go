package chat

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testMocks struct {
	chatRepo       *repository.MockChatRepository
	userRepo       *repository.MockUserRepository
	roleRepo       *repository.MockRoleRepository
	vanityRoleRepo *repository.MockVanityRoleRepository
	authzSvc       *authz.MockService
	notifSvc       *notification.MockService
	blockSvc       *block.MockService
	uploadSvc      *upload.MockService
	settingsSvc    *settings.MockService
	hub            *ws.Hub
}

func newTestService(t *testing.T) (*service, *testMocks) {
	chatRepo := repository.NewMockChatRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	roleRepo := repository.NewMockRoleRepository(t)
	vanityRoleRepo := repository.NewMockVanityRoleRepository(t)
	authzSvc := authz.NewMockService(t)
	notifSvc := notification.NewMockService(t)
	blockSvc := block.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	mediaProc := &media.Processor{}
	hub := ws.NewHub()
	svc := NewService(chatRepo, userRepo, roleRepo, vanityRoleRepo, authzSvc, notifSvc, blockSvc, uploadSvc, settingsSvc, mediaProc, hub, contentfilter.New()).(*service)

	chatRepo.EXPECT().HasGhostMembers(mock.Anything, mock.Anything).Return(false, nil).Maybe()
	chatRepo.EXPECT().IsGhostMember(mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Maybe()
	chatRepo.EXPECT().HasActiveMemberTimeout(mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Maybe()

	return svc, &testMocks{
		chatRepo:       chatRepo,
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		vanityRoleRepo: vanityRoleRepo,
		authzSvc:       authzSvc,
		notifSvc:       notifSvc,
		blockSvc:       blockSvc,
		uploadSvc:      uploadSvc,
		settingsSvc:    settingsSvc,
		hub:            hub,
	}
}

func sampleUser(id uuid.UUID) *model.User {
	return &model.User{
		ID:          id,
		Username:    "user",
		DisplayName: "User",
		AvatarURL:   "avatar.png",
		DmsEnabled:  true,
	}
}

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
		{"empty after sanitize", []string{"---"}, []string{}},
		{"max length 30", []string{"abcdefghijklmnopqrstuvwxyz1234567890"}, []string{"abcdefghijklmnopqrstuvwxyz1234"}},
		{"max 10 tags", []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}, []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			in := tc.in

			// when
			got := sanitizeTags(in)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResolveDMRoom_CannotDMSelf(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	id := uuid.New()

	// when
	_, err := svc.ResolveDMRoom(context.Background(), id, id)

	// then
	require.ErrorIs(t, err, ErrCannotDMSelf)
}

func TestResolveDMRoom_RecipientLookupError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	sender := uuid.New()
	recipient := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, recipient).Return(nil, errors.New("db"))

	// when
	_, err := svc.ResolveDMRoom(context.Background(), sender, recipient)

	// then
	require.Error(t, err)
}

func TestResolveDMRoom_RecipientNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	sender := uuid.New()
	recipient := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, recipient).Return(nil, nil)

	// when
	_, err := svc.ResolveDMRoom(context.Background(), sender, recipient)

	// then
	require.ErrorIs(t, err, ErrUserNotFound)
}

func TestResolveDMRoom_DMsDisabled(t *testing.T) {
	// given
	svc, m := newTestService(t)
	sender := uuid.New()
	recipient := uuid.New()
	u := sampleUser(recipient)
	u.DmsEnabled = false
	m.userRepo.EXPECT().GetByID(mock.Anything, recipient).Return(u, nil)

	// when
	_, err := svc.ResolveDMRoom(context.Background(), sender, recipient)

	// then
	require.ErrorIs(t, err, ErrDmsDisabled)
}

func TestResolveDMRoom_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	sender := uuid.New()
	recipient := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, recipient).Return(sampleUser(recipient), nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, sender, recipient).Return(true, nil)

	// when
	_, err := svc.ResolveDMRoom(context.Background(), sender, recipient)

	// then
	require.ErrorIs(t, err, ErrUserBlocked)
}

func TestResolveDMRoom_FindDMError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	sender := uuid.New()
	recipient := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, recipient).Return(sampleUser(recipient), nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, sender, recipient).Return(false, nil)
	m.chatRepo.EXPECT().FindDMRoom(mock.Anything, sender, recipient).Return(uuid.Nil, errors.New("db"))

	// when
	_, err := svc.ResolveDMRoom(context.Background(), sender, recipient)

	// then
	require.Error(t, err)
}

func TestResolveDMRoom_NoExistingRoom(t *testing.T) {
	// given
	svc, m := newTestService(t)
	sender := uuid.New()
	recipient := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, recipient).Return(sampleUser(recipient), nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, sender, recipient).Return(false, nil)
	m.chatRepo.EXPECT().FindDMRoom(mock.Anything, sender, recipient).Return(uuid.Nil, nil)

	// when
	got, err := svc.ResolveDMRoom(context.Background(), sender, recipient)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Nil(t, got.Room)
}

func TestResolveDMRoom_ExistingRoom(t *testing.T) {
	// given
	svc, m := newTestService(t)
	sender := uuid.New()
	recipient := uuid.New()
	roomID := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, recipient).Return(sampleUser(recipient), nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, sender, recipient).Return(false, nil)
	m.chatRepo.EXPECT().FindDMRoom(mock.Anything, sender, recipient).Return(roomID, nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, sender).Return(&repository.ChatRoomRow{ID: roomID, Type: "dm"}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{sender, recipient}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, sender).Return(sampleUser(sender), nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, recipient).Return(sampleUser(recipient), nil)

	// when
	got, err := svc.ResolveDMRoom(context.Background(), sender, recipient)

	// then
	require.NoError(t, err)
	require.NotNil(t, got.Room)
	assert.Equal(t, roomID, got.Room.ID)
}

func TestSendDMMessage_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.SendDMMessage(context.Background(), uuid.New(), uuid.New(), "", nil)

	// then
	require.ErrorIs(t, err, ErrMissingFields)
}

func TestSendDMMessage_PreconditionFails(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	id := uuid.New()

	// when
	_, err := svc.SendDMMessage(context.Background(), id, id, "hi", nil)

	// then
	require.ErrorIs(t, err, ErrCannotDMSelf)
}

func TestSendDMMessage_CreateRoomError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	sender := uuid.New()
	recipient := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, recipient).Return(sampleUser(recipient), nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, sender, recipient).Return(false, nil)
	m.chatRepo.EXPECT().CreateDMRoomAtomic(mock.Anything, mock.Anything, sender, recipient).Return(uuid.Nil, errors.New("boom"))

	// when
	_, err := svc.SendDMMessage(context.Background(), sender, recipient, "hi", nil)

	// then
	require.Error(t, err)
}

func TestCreateGroupRoom_EmptyName(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := dto.CreateGroupRoomRequest{Name: "   "}

	// when
	_, err := svc.CreateGroupRoom(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrMissingFields)
}

func TestCreateGroupRoom_CreateRoomError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	creator := uuid.New()
	req := dto.CreateGroupRoomRequest{Name: "Room"}
	m.chatRepo.EXPECT().CreateRoom(mock.Anything, mock.Anything, "Room", "", "group", false, false, creator).Return(errors.New("boom"))

	// when
	_, err := svc.CreateGroupRoom(context.Background(), creator, req)

	// then
	require.Error(t, err)
}

func TestCreateGroupRoom_AddTagsError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	creator := uuid.New()
	req := dto.CreateGroupRoomRequest{Name: "Room", Tags: []string{"tag1"}}
	m.chatRepo.EXPECT().CreateRoom(mock.Anything, mock.Anything, "Room", "", "group", false, false, creator).Return(nil)
	m.chatRepo.EXPECT().AddRoomTags(mock.Anything, mock.Anything, []string{"tag1"}).Return(errors.New("boom"))

	// when
	_, err := svc.CreateGroupRoom(context.Background(), creator, req)

	// then
	require.Error(t, err)
}

func TestCreateGroupRoom_AddHostError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	creator := uuid.New()
	req := dto.CreateGroupRoomRequest{Name: "Room"}
	m.chatRepo.EXPECT().CreateRoom(mock.Anything, mock.Anything, "Room", "", "group", false, false, creator).Return(nil)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, mock.Anything, creator, "host", false).Return(errors.New("boom"))

	// when
	_, err := svc.CreateGroupRoom(context.Background(), creator, req)

	// then
	require.Error(t, err)
}

func TestCreateGroupRoom_SkipsBlockedMembers(t *testing.T) {
	// given
	svc, m := newTestService(t)
	creator := uuid.New()
	memberA := uuid.New()
	memberB := uuid.New()
	req := dto.CreateGroupRoomRequest{Name: "Room", MemberIDs: []uuid.UUID{creator, memberA, memberB}}
	m.chatRepo.EXPECT().CreateRoom(mock.Anything, mock.Anything, "Room", "", "group", false, false, creator).Return(nil)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, mock.Anything, creator, "host", false).Return(nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, creator, memberA).Return(true, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, creator, memberB).Return(false, nil)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, mock.Anything, memberB, "member", false).Return(nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, mock.Anything, creator).Return(&repository.ChatRoomRow{Name: "Room"}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, mock.Anything).Return([]uuid.UUID{creator, memberB}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, creator).Return(sampleUser(creator), nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, memberB).Return(sampleUser(memberB), nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://x").Maybe()
	m.userRepo.EXPECT().GetByID(mock.Anything, creator).Return(sampleUser(creator), nil).Maybe()
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

	// when
	got, err := svc.CreateGroupRoom(context.Background(), creator, req)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestCreateGroupRoom_AddMemberError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	creator := uuid.New()
	memberA := uuid.New()
	req := dto.CreateGroupRoomRequest{Name: "Room", MemberIDs: []uuid.UUID{memberA}}
	m.chatRepo.EXPECT().CreateRoom(mock.Anything, mock.Anything, "Room", "", "group", false, false, creator).Return(nil)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, mock.Anything, creator, "host", false).Return(nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, creator, memberA).Return(false, nil)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, mock.Anything, memberA, "member", false).Return(errors.New("boom"))

	// when
	_, err := svc.CreateGroupRoom(context.Background(), creator, req)

	// then
	require.Error(t, err)
}

func TestListPublicRooms_DefaultsAndTrim(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.chatRepo.EXPECT().ListPublicRooms(mock.Anything, "q", false, "tagx", viewer, []uuid.UUID(nil), 20, 0).Return([]repository.ChatRoomRow{{ID: uuid.New()}}, 1, nil)

	// when
	got, err := svc.ListPublicRooms(context.Background(), "q", false, "  TagX  ", viewer, 0, -5)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Rooms, 1)
}

func TestListPublicRooms_LimitClamped(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.chatRepo.EXPECT().ListPublicRooms(mock.Anything, "", false, "", viewer, []uuid.UUID(nil), 100, 0).Return(nil, 0, nil)

	// when
	_, err := svc.ListPublicRooms(context.Background(), "", false, "", viewer, 500, 0)

	// then
	require.NoError(t, err)
}

func TestListPublicRooms_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.chatRepo.EXPECT().ListPublicRooms(mock.Anything, "", false, "", viewer, []uuid.UUID(nil), 20, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListPublicRooms(context.Background(), "", false, "", viewer, 0, 0)

	// then
	require.Error(t, err)
}

func TestListUserGroupRooms_DefaultsAndRoleReset(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.chatRepo.EXPECT().ListUserGroupRooms(mock.Anything, userID, "", false, "tag", "", 20, 0).Return(nil, 0, nil)

	// when
	_, err := svc.ListUserGroupRooms(context.Background(), userID, "", false, "  Tag  ", "bogus", -1, -1)

	// then
	require.NoError(t, err)
}

func TestListUserGroupRooms_ValidRoleHost(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.chatRepo.EXPECT().ListUserGroupRooms(mock.Anything, userID, "", false, "", "host", 100, 0).Return(nil, 0, nil)

	// when
	_, err := svc.ListUserGroupRooms(context.Background(), userID, "", false, "", "host", 500, 0)

	// then
	require.NoError(t, err)
}

func TestListUserGroupRooms_ValidRoleMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.chatRepo.EXPECT().ListUserGroupRooms(mock.Anything, userID, "", false, "", "member", 10, 5).Return(nil, 0, nil)

	// when
	_, err := svc.ListUserGroupRooms(context.Background(), userID, "", false, "", "member", 10, 5)

	// then
	require.NoError(t, err)
}

func TestListUserGroupRooms_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.chatRepo.EXPECT().ListUserGroupRooms(mock.Anything, userID, "", false, "", "", 20, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListUserGroupRooms(context.Background(), userID, "", false, "", "", 0, 0)

	// then
	require.Error(t, err)
}

func TestSetRoomMuted_MembershipCheckError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(false, errors.New("boom"))

	// when
	err := svc.SetRoomMuted(context.Background(), roomID, userID, true)

	// then
	require.Error(t, err)
}

func TestSetRoomMuted_NotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(false, nil)

	// when
	err := svc.SetRoomMuted(context.Background(), roomID, userID, true)

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestSetRoomMuted_SetMutedError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().SetMuted(mock.Anything, roomID, userID, true).Return(errors.New("boom"))

	// when
	err := svc.SetRoomMuted(context.Background(), roomID, userID, true)

	// then
	require.Error(t, err)
}

func TestSetRoomMuted_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().SetMuted(mock.Anything, roomID, userID, false).Return(nil)

	// when
	err := svc.SetRoomMuted(context.Background(), roomID, userID, false)

	// then
	require.NoError(t, err)
}

func TestIsRoomMuted_Delegates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMuted(mock.Anything, roomID, userID).Return(true, nil)

	// when
	got, err := svc.IsRoomMuted(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
	assert.True(t, got)
}

func TestJoinRoom_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(nil, errors.New("boom"))

	// when
	_, err := svc.JoinRoom(context.Background(), roomID, userID, false)

	// then
	require.Error(t, err)
}

func TestJoinRoom_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(nil, nil)

	// when
	_, err := svc.JoinRoom(context.Background(), roomID, userID, false)

	// then
	require.ErrorIs(t, err, ErrRoomNotFound)
}

func TestJoinRoom_NotGroup(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{ID: roomID, Type: "dm"}, nil)

	// when
	_, err := svc.JoinRoom(context.Background(), roomID, userID, false)

	// then
	require.ErrorIs(t, err, ErrNotGroupRoom)
}

func TestJoinRoom_SystemRoom(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{ID: roomID, Type: "group", IsSystem: true}, nil)

	// when
	_, err := svc.JoinRoom(context.Background(), roomID, userID, false)

	// then
	require.ErrorIs(t, err, ErrSystemRoom)
}

func TestJoinRoom_NotPublic(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{ID: roomID, Type: "group", IsPublic: false}, nil)

	// when
	_, err := svc.JoinRoom(context.Background(), roomID, userID, false)

	// then
	require.ErrorIs(t, err, ErrNotPublic)
}

func TestJoinRoom_AlreadyMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	row := &repository.ChatRoomRow{ID: roomID, Type: "group", IsPublic: true, IsMember: true}
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(row, nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(row, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil)

	// when
	got, err := svc.JoinRoom(context.Background(), roomID, userID, false)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestJoinRoom_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	creatorID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{ID: roomID, Type: "group", IsPublic: true, CreatedBy: creatorID}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, creatorID).Return(true, nil)

	// when
	_, err := svc.JoinRoom(context.Background(), roomID, userID, false)

	// then
	require.ErrorIs(t, err, ErrUserBlocked)
}

func TestJoinRoom_RoomFull(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	creatorID := uuid.New()
	row := &repository.ChatRoomRow{ID: roomID, Type: "group", IsPublic: true, CreatedBy: creatorID, MemberCount: 10}
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(row, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, creatorID).Return(false, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxChatRoomMembers).Return(10)

	// when
	_, err := svc.JoinRoom(context.Background(), roomID, userID, false)

	// then
	require.ErrorIs(t, err, ErrRoomFull)
}

func TestJoinRoom_AddMemberError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	creatorID := uuid.New()
	row := &repository.ChatRoomRow{ID: roomID, Type: "group", IsPublic: true, CreatedBy: creatorID}
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(row, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, creatorID).Return(false, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxChatRoomMembers).Return(0)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, roomID, userID, "member", false).Return(errors.New("boom"))

	// when
	_, err := svc.JoinRoom(context.Background(), roomID, userID, false)

	// then
	require.Error(t, err)
}

func TestJoinRoom_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	creatorID := uuid.New()
	row := &repository.ChatRoomRow{ID: roomID, Type: "group", IsPublic: true, CreatedBy: creatorID}
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(row, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, creatorID).Return(false, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxChatRoomMembers).Return(100)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, roomID, userID, "member", false).Return(nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(row, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.Anything, roomID, userID, mock.Anything).Return(errors.New("boom"))
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)

	// when
	got, err := svc.JoinRoom(context.Background(), roomID, userID, false)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestLeaveRoom_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(nil, errors.New("boom"))

	// when
	err := svc.LeaveRoom(context.Background(), roomID, userID)

	// then
	require.Error(t, err)
}

func TestLeaveRoom_NotMemberNil(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(nil, nil)

	// when
	err := svc.LeaveRoom(context.Background(), roomID, userID)

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestLeaveRoom_NotMemberFalse(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{ID: roomID, IsMember: false}, nil)

	// when
	err := svc.LeaveRoom(context.Background(), roomID, userID)

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestLeaveRoom_SystemRoom(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{ID: roomID, IsMember: true, IsSystem: true}, nil)

	// when
	err := svc.LeaveRoom(context.Background(), roomID, userID)

	// then
	require.ErrorIs(t, err, ErrSystemRoom)
}

func TestLeaveRoom_CannotLeaveAsHost(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{ID: roomID, IsMember: true, ViewerRole: "host"}, nil)

	// when
	err := svc.LeaveRoom(context.Background(), roomID, userID)

	// then
	require.ErrorIs(t, err, ErrCannotLeaveAsHost)
}

func TestLeaveRoom_RemoveError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{ID: roomID, IsMember: true, ViewerRole: "member"}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, roomID, userID).Return(errors.New("boom"))

	// when
	err := svc.LeaveRoom(context.Background(), roomID, userID)

	// then
	require.Error(t, err)
}

func TestLeaveRoom_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{ID: roomID, IsMember: true, ViewerRole: "member"}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, roomID, userID).Return(nil)
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.Anything, roomID, userID, mock.Anything).Return(errors.New("boom"))
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)

	// when
	err := svc.LeaveRoom(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
}

func TestKickMember_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	hostID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, hostID).Return(nil, errors.New("boom"))

	// when
	err := svc.KickMember(context.Background(), hostID, roomID, targetID)

	// then
	require.Error(t, err)
}

func TestKickMember_RoomNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	hostID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, hostID).Return(nil, nil)

	// when
	err := svc.KickMember(context.Background(), hostID, roomID, targetID)

	// then
	require.ErrorIs(t, err, ErrRoomNotFound)
}

func TestKickMember_SystemRoom(t *testing.T) {
	// given
	svc, m := newTestService(t)
	hostID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, hostID).Return(&repository.ChatRoomRow{IsSystem: true}, nil)

	// when
	err := svc.KickMember(context.Background(), hostID, roomID, targetID)

	// then
	require.ErrorIs(t, err, ErrSystemRoom)
}

func TestKickMember_GetHostRoleError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	hostID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, hostID).Return(&repository.ChatRoomRow{}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, hostID).Return("", errors.New("boom"))

	// when
	err := svc.KickMember(context.Background(), hostID, roomID, targetID)

	// then
	require.Error(t, err)
}

func TestKickMember_NotHost(t *testing.T) {
	// given
	svc, m := newTestService(t)
	hostID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, hostID).Return(&repository.ChatRoomRow{}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, hostID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, hostID).Return("", nil)

	// when
	err := svc.KickMember(context.Background(), hostID, roomID, targetID)

	// then
	require.ErrorIs(t, err, ErrNotHost)
}

func TestKickMember_TargetRoleError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	hostID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, hostID).Return(&repository.ChatRoomRow{}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, hostID).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("", errors.New("boom"))

	// when
	err := svc.KickMember(context.Background(), hostID, roomID, targetID)

	// then
	require.Error(t, err)
}

func TestKickMember_TargetNotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	hostID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, hostID).Return(&repository.ChatRoomRow{}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, hostID).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("", nil)

	// when
	err := svc.KickMember(context.Background(), hostID, roomID, targetID)

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestKickMember_CannotKickHost(t *testing.T) {
	// given
	svc, m := newTestService(t)
	hostID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, hostID).Return(&repository.ChatRoomRow{}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, hostID).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("host", nil)

	// when
	err := svc.KickMember(context.Background(), hostID, roomID, targetID)

	// then
	require.ErrorIs(t, err, ErrCannotKickHost)
}

func TestKickMember_RemoveError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	hostID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, hostID).Return(&repository.ChatRoomRow{}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, hostID).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, targetID).Return("", nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{hostID, targetID}, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, roomID, targetID).Return(errors.New("boom"))

	// when
	err := svc.KickMember(context.Background(), hostID, roomID, targetID)

	// then
	require.Error(t, err)
}

func TestKickMember_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	hostID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, hostID).Return(&repository.ChatRoomRow{}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, hostID).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, targetID).Return("", nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{hostID, targetID}, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, roomID, targetID).Return(nil)
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.Anything, roomID, hostID, mock.Anything).Return(errors.New("boom"))

	// when
	err := svc.KickMember(context.Background(), hostID, roomID, targetID)

	// then
	require.NoError(t, err)
}

func TestKickMember_TargetIsSiteMod(t *testing.T) {
	// given
	svc, m := newTestService(t)
	hostID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, hostID).Return(&repository.ChatRoomRow{}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, hostID).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, targetID).Return(authz.RoleAdmin, nil)

	// when
	err := svc.KickMember(context.Background(), hostID, roomID, targetID)

	// then
	require.ErrorIs(t, err, ErrTargetImmune)
}

func TestSetMemberTimeout_InvalidDuration(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actorID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, actorID).Return(&repository.ChatRoomRow{}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, actorID).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return("", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, targetID).Return("", nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, targetID).Return(false, "", false, nil)

	// when
	_, err := svc.SetMemberTimeout(context.Background(), roomID, actorID, targetID, dto.SetMemberTimeoutRequest{Amount: 0, Unit: "hours"})

	// then
	require.ErrorIs(t, err, ErrInvalidTimeoutDuration)
}

func TestSetMemberTimeout_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actorID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, actorID).Return(&repository.ChatRoomRow{}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, actorID).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return("", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, targetID).Return("", nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, targetID).Return(false, "", false, nil)
	m.chatRepo.EXPECT().SetMemberTimeout(mock.Anything, roomID, targetID, mock.Anything, false).Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(sampleUser(actorID), nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, targetID).Return(sampleUser(targetID), nil)
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.Anything, roomID, actorID, mock.Anything).Return(errors.New("boom"))
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{{
		UserID:       targetID,
		Username:     "target",
		DisplayName:  "Target",
		Role:         "member",
		TimeoutUntil: "2099-01-01 00:00:00",
	}}, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{targetID}).Return(nil, nil)

	// when
	got, err := svc.SetMemberTimeout(context.Background(), roomID, actorID, targetID, dto.SetMemberTimeoutRequest{Amount: 1, Unit: "hours"})

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, targetID, got.User.ID)
}

func TestSetMemberTimeout_HostCannotChangeStaffTimeout(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actorID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, actorID).Return(&repository.ChatRoomRow{}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, actorID).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return("", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, targetID).Return("", nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, targetID).Return(true, "2099-01-01 00:00:00", true, nil)

	// when
	_, err := svc.SetMemberTimeout(context.Background(), roomID, actorID, targetID, dto.SetMemberTimeoutRequest{Amount: 2, Unit: "hours"})

	// then
	require.ErrorIs(t, err, ErrTimeoutLockedByStaff)
}

func TestClearMemberTimeout_HostCannotClearStaffTimeout(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actorID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, actorID).Return(&repository.ChatRoomRow{}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, actorID).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return("", nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, targetID).Return(true, "2099-01-01 00:00:00", true, nil)

	// when
	_, err := svc.ClearMemberTimeout(context.Background(), roomID, actorID, targetID)

	// then
	require.ErrorIs(t, err, ErrTimeoutLockedByStaff)
}

func TestClearMemberTimeout_SiteModCanClearHostTimeout(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actorID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, actorID).Return(&repository.ChatRoomRow{}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, actorID).Return("", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(authz.RoleAdmin, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("member", nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, targetID).Return(true, "2099-01-01 00:00:00", false, nil)
	m.chatRepo.EXPECT().ClearMemberTimeout(mock.Anything, roomID, targetID).Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(sampleUser(actorID), nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, targetID).Return(sampleUser(targetID), nil)
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.Anything, roomID, actorID, mock.Anything).Return(errors.New("boom"))
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{{
		UserID:      targetID,
		Username:    "target",
		DisplayName: "Target",
		Role:        "member",
	}}, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{targetID}).Return(nil, nil)

	// when
	got, err := svc.ClearMemberTimeout(context.Background(), roomID, actorID, targetID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, targetID, got.User.ID)
}

func TestGetMembers_IsMemberError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewerID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, viewerID).Return(false, errors.New("boom"))

	// when
	_, err := svc.GetMembers(context.Background(), viewerID, roomID)

	// then
	require.Error(t, err)
}

func TestGetMembers_NotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewerID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, viewerID).Return(false, nil)

	// when
	_, err := svc.GetMembers(context.Background(), viewerID, roomID)

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestGetMembers_DetailedError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewerID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, viewerID).Return(true, nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetMembers(context.Background(), viewerID, roomID)

	// then
	require.Error(t, err)
}

func TestGetMembers_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewerID := uuid.New()
	roomID := uuid.New()
	memberID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, viewerID).Return(true, nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{{UserID: memberID, Username: "u", DisplayName: "d", Role: "member"}}, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{memberID}).Return(nil, nil)

	// when
	got, err := svc.GetMembers(context.Background(), viewerID, roomID)

	// then
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, memberID, got[0].User.ID)
}

func TestGetMembers_SiteMod_NicknameLockedFalse(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewerID := uuid.New()
	roomID := uuid.New()
	memberID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, viewerID).Return(true, nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{
		{UserID: memberID, Username: "admin", DisplayName: "Admin", Role: "member", AuthorRole: string(authz.RoleAdmin), AuthorRoleTyped: authz.RoleAdmin, NicknameLocked: true},
	}, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{memberID}).Return(nil, nil)

	// when
	got, err := svc.GetMembers(context.Background(), viewerID, roomID)

	// then
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.False(t, got[0].NicknameLocked)
}

func TestListRooms_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomsByUser(mock.Anything, userID).Return(nil, errors.New("boom"))

	// when
	_, err := svc.ListRooms(context.Background(), userID)

	// then
	require.Error(t, err)
}

func TestListRooms_MembersError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().GetRoomsByUser(mock.Anything, userID).Return([]repository.ChatRoomRow{{ID: roomID}}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, errors.New("boom"))

	// when
	_, err := svc.ListRooms(context.Background(), userID)

	// then
	require.Error(t, err)
}

func TestListRooms_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().GetRoomsByUser(mock.Anything, userID).Return([]repository.ChatRoomRow{{ID: roomID, Type: "group"}}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)

	// when
	got, err := svc.ListRooms(context.Background(), userID)

	// then
	require.NoError(t, err)
	require.Len(t, got.Rooms, 1)
	assert.Equal(t, roomID, got.Rooms[0].ID)
}

func TestEnsureSystemRooms_GetModsError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.EnsureSystemRooms(context.Background())

	// then
	require.Error(t, err)
}

func TestEnsureSystemRooms_GetAdminsError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(uuid.Nil, nil)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindAdmins).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.EnsureSystemRooms(context.Background())

	// then
	require.Error(t, err)
}

func TestEnsureSystemRooms_BothExist(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(uuid.New(), nil)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindAdmins).Return(uuid.New(), nil)

	// when
	err := svc.EnsureSystemRooms(context.Background())

	// then
	require.NoError(t, err)
}

func TestEnsureSystemRooms_NoSuperAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(uuid.Nil, nil)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindAdmins).Return(uuid.Nil, nil)
	m.roleRepo.EXPECT().GetUsersByRoles(mock.Anything, []role.Role{authz.RoleSuperAdmin}).Return(nil, nil)

	// when
	err := svc.EnsureSystemRooms(context.Background())

	// then
	require.NoError(t, err)
}

func TestEnsureSystemRooms_SuperAdminLookupError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(uuid.Nil, nil)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindAdmins).Return(uuid.Nil, nil)
	m.roleRepo.EXPECT().GetUsersByRoles(mock.Anything, []role.Role{authz.RoleSuperAdmin}).Return(nil, errors.New("boom"))

	// when
	err := svc.EnsureSystemRooms(context.Background())

	// then
	require.Error(t, err)
}

func TestEnsureSystemRooms_CreatesBoth(t *testing.T) {
	// given
	svc, m := newTestService(t)
	super := uuid.New()
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(uuid.Nil, nil)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindAdmins).Return(uuid.Nil, nil)
	m.roleRepo.EXPECT().GetUsersByRoles(mock.Anything, []role.Role{authz.RoleSuperAdmin}).Return([]uuid.UUID{super}, nil)
	m.chatRepo.EXPECT().CreateSystemRoom(mock.Anything, mock.Anything, systemModsName, systemModsDesc, SystemKindMods, super).Return(nil)
	m.chatRepo.EXPECT().CreateSystemRoom(mock.Anything, mock.Anything, systemAdminsName, systemAdminsDesc, SystemKindAdmins, super).Return(nil)
	m.roleRepo.EXPECT().GetUsersByRoles(mock.Anything, []role.Role{authz.RoleModerator, authz.RoleAdmin, authz.RoleSuperAdmin}).Return(nil, nil)

	// when
	err := svc.EnsureSystemRooms(context.Background())

	// then
	require.NoError(t, err)
}

func TestEnsureSystemRooms_CreateModsError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	super := uuid.New()
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(uuid.Nil, nil)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindAdmins).Return(uuid.Nil, nil)
	m.roleRepo.EXPECT().GetUsersByRoles(mock.Anything, []role.Role{authz.RoleSuperAdmin}).Return([]uuid.UUID{super}, nil)
	m.chatRepo.EXPECT().CreateSystemRoom(mock.Anything, mock.Anything, systemModsName, systemModsDesc, SystemKindMods, super).Return(errors.New("boom"))

	// when
	err := svc.EnsureSystemRooms(context.Background())

	// then
	require.Error(t, err)
}

func TestSyncSystemRoomMembership_GetModsError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.SyncSystemRoomMembership(context.Background(), userID, authz.RoleAdmin)

	// then
	require.Error(t, err)
}

func TestSyncSystemRoomMembership_GetAdminsError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(uuid.New(), nil)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindAdmins).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.SyncSystemRoomMembership(context.Background(), userID, authz.RoleAdmin)

	// then
	require.Error(t, err)
}

func TestSyncSystemRoomMembership_AdminAdded(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	modsID := uuid.New()
	adminsID := uuid.New()
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(modsID, nil)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindAdmins).Return(adminsID, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, modsID, userID).Return("", nil)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, modsID, userID, "member", false).Return(nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, adminsID, userID).Return("", nil)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, adminsID, userID, "member", false).Return(nil)

	// when
	err := svc.SyncSystemRoomMembership(context.Background(), userID, authz.RoleAdmin)

	// then
	require.NoError(t, err)
}

func TestSyncSystemRoomMembership_SuperAdminAsHost(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	modsID := uuid.New()
	adminsID := uuid.New()
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(modsID, nil)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindAdmins).Return(adminsID, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, modsID, userID).Return("", nil)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, modsID, userID, "host", false).Return(nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, adminsID, userID).Return("", nil)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, adminsID, userID, "host", false).Return(nil)

	// when
	err := svc.SyncSystemRoomMembership(context.Background(), userID, authz.RoleSuperAdmin)

	// then
	require.NoError(t, err)
}

func TestSyncSystemRoomMembership_ModRemovedFromAdmins(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	modsID := uuid.New()
	adminsID := uuid.New()
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(modsID, nil)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindAdmins).Return(adminsID, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, modsID, userID).Return("member", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, adminsID, userID).Return("member", nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, adminsID, userID).Return(nil)

	// when
	err := svc.SyncSystemRoomMembership(context.Background(), userID, authz.RoleModerator)

	// then
	require.NoError(t, err)
}

func TestSyncSystemRoomMembership_DemotedUserRemovedFromBoth(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	modsID := uuid.New()
	adminsID := uuid.New()
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(modsID, nil)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindAdmins).Return(adminsID, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, modsID, userID).Return("member", nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, modsID, userID).Return(nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, adminsID, userID).Return("", nil)

	// when
	err := svc.SyncSystemRoomMembership(context.Background(), userID, role.Role("user"))

	// then
	require.NoError(t, err)
}

func TestSyncSystemRoomMembership_RoleUpgradedToHost(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	modsID := uuid.New()
	adminsID := uuid.New()
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(modsID, nil)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindAdmins).Return(adminsID, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, modsID, userID).Return("member", nil)
	m.chatRepo.EXPECT().SetMemberRole(mock.Anything, modsID, userID, "host").Return(nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, adminsID, userID).Return("member", nil)
	m.chatRepo.EXPECT().SetMemberRole(mock.Anything, adminsID, userID, "host").Return(nil)

	// when
	err := svc.SyncSystemRoomMembership(context.Background(), userID, authz.RoleSuperAdmin)

	// then
	require.NoError(t, err)
}

func TestSyncSystemRoomMembership_GetRoleError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	modsID := uuid.New()
	adminsID := uuid.New()
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindMods).Return(modsID, nil)
	m.chatRepo.EXPECT().GetSystemRoomID(mock.Anything, SystemKindAdmins).Return(adminsID, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, modsID, userID).Return("", errors.New("boom"))

	// when
	err := svc.SyncSystemRoomMembership(context.Background(), userID, authz.RoleAdmin)

	// then
	require.Error(t, err)
}

func TestGetMessages_IsMemberError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(false, errors.New("boom"))

	// when
	_, err := svc.GetMessages(context.Background(), userID, roomID, 10, 0)

	// then
	require.Error(t, err)
}

func TestGetMessages_NotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(false, nil)

	// when
	_, err := svc.GetMessages(context.Background(), userID, roomID, 10, 0)

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestGetMessages_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().GetMessages(mock.Anything, roomID, 10, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.GetMessages(context.Background(), userID, roomID, 10, 0)

	// then
	require.Error(t, err)
}

func TestGetMessages_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	roomID := uuid.New()
	msgID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().GetMessages(mock.Anything, roomID, 10, 0).Return([]repository.ChatMessageRow{{ID: msgID, RoomID: roomID, Body: "hi"}}, 1, nil)
	m.chatRepo.EXPECT().GetMessageMediaBatch(mock.Anything, []uuid.UUID{msgID}).Return(nil, nil)
	m.chatRepo.EXPECT().GetReactionsBatch(mock.Anything, []uuid.UUID{msgID}, userID).Return(nil, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, mock.Anything).Return(nil, nil)

	// when
	got, err := svc.GetMessages(context.Background(), userID, roomID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Messages, 1)
}

func TestGetMessagesBefore_IsMemberError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(false, errors.New("boom"))

	// when
	_, err := svc.GetMessagesBefore(context.Background(), userID, roomID, "x", 50)

	// then
	require.Error(t, err)
}

func TestGetMessagesBefore_NotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(false, nil)

	// when
	_, err := svc.GetMessagesBefore(context.Background(), userID, roomID, "x", 50)

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestGetMessagesBefore_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().GetMessagesBefore(mock.Anything, roomID, "x", 50).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetMessagesBefore(context.Background(), userID, roomID, "x", 50)

	// then
	require.Error(t, err)
}

func TestGetMessagesBefore_DefaultsApplied(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().GetMessagesBefore(mock.Anything, roomID, "x", 50).Return(nil, nil)
	m.chatRepo.EXPECT().GetMessageMediaBatch(mock.Anything, []uuid.UUID{}).Return(nil, nil)
	m.chatRepo.EXPECT().GetReactionsBatch(mock.Anything, []uuid.UUID{}, userID).Return(nil, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, mock.Anything).Return(nil, nil)

	// when
	got, err := svc.GetMessagesBefore(context.Background(), userID, roomID, "x", 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 50, got.Limit)
}

func TestGetMessagesBefore_LimitClamped(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().GetMessagesBefore(mock.Anything, roomID, "x", 200).Return(nil, nil)
	m.chatRepo.EXPECT().GetMessageMediaBatch(mock.Anything, []uuid.UUID{}).Return(nil, nil)
	m.chatRepo.EXPECT().GetReactionsBatch(mock.Anything, []uuid.UUID{}, userID).Return(nil, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, mock.Anything).Return(nil, nil)

	// when
	got, err := svc.GetMessagesBefore(context.Background(), userID, roomID, "x", 500)

	// then
	require.NoError(t, err)
	assert.Equal(t, 200, got.Limit)
}

func TestSendMessage_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.SendMessage(context.Background(), uuid.New(), uuid.New(), dto.SendMessageRequest{Body: ""}, nil)

	// then
	require.ErrorIs(t, err, ErrMissingFields)
}

func TestSendMessage_IsMemberError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(false, errors.New("boom"))

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"}, nil)

	// then
	require.Error(t, err)
}

func TestSendMessage_NotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(false, nil)

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"}, nil)

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestSendMessage_TimedOut(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, senderID).Return(true, "2099-01-01 00:00:00", false, nil)

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"}, nil)

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTimedOut)
	assert.Contains(t, err.Error(), "01 January 2099 00:00 UTC")
}

func TestSendMessage_MembersError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, senderID).Return(false, "", false, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, errors.New("boom"))

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"}, nil)

	// then
	require.Error(t, err)
}

func TestSendMessage_BlockedByRecipient(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	otherID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, senderID).Return(false, "", false, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, otherID}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, otherID).Return(true, nil)

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"}, nil)

	// then
	require.ErrorIs(t, err, ErrUserBlocked)
}

func TestSendMessage_SenderLookupError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, senderID).Return(false, "", false, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(nil, errors.New("boom"))

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"}, nil)

	// then
	require.Error(t, err)
}

func TestSendMessage_SenderNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, senderID).Return(false, "", false, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(nil, nil)

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"}, nil)

	// then
	require.ErrorIs(t, err, ErrUserNotFound)
}

func TestSendMessage_InsertError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, senderID).Return(false, "", false, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(sampleUser(senderID), nil)
	m.chatRepo.EXPECT().InsertMessage(mock.Anything, mock.Anything, roomID, senderID, "hi", (*uuid.UUID)(nil)).Return(errors.New("boom"))

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"}, nil)

	// then
	require.Error(t, err)
}

func TestSendMessage_MarkReadError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, senderID).Return(false, "", false, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(sampleUser(senderID), nil)
	m.chatRepo.EXPECT().InsertMessage(mock.Anything, mock.Anything, roomID, senderID, "hi", (*uuid.UUID)(nil)).Return(nil)
	m.chatRepo.EXPECT().MarkRoomRead(mock.Anything, roomID, senderID).Return(errors.New("boom"))

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"}, nil)

	// then
	require.Error(t, err)
}

func TestSendMessage_DMSuccess(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	recipientID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, senderID).Return(false, "", false, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, recipientID}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, recipientID).Return(false, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(sampleUser(senderID), nil)
	m.chatRepo.EXPECT().InsertMessage(mock.Anything, mock.Anything, roomID, senderID, "hi", (*uuid.UUID)(nil)).Return(nil)
	m.chatRepo.EXPECT().MarkRoomRead(mock.Anything, roomID, senderID).Return(nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return(nil, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUser(mock.Anything, senderID).Return(nil, nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, senderID).Return(&repository.ChatRoomRow{Type: "dm"}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, recipientID}, nil)
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool { return p.Type == dto.NotifChatMessage })).Return(nil)
	m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, recipientID).Return(1, nil)

	// when
	got, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, "hi", got.Body)
	assert.Equal(t, senderID, got.Sender.ID)
}

func TestSendMessage_GroupWithMentionAndReply(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	mentionedID := uuid.New()
	replyAuthorID := uuid.New()
	replyMsgID := uuid.New()
	body := "hey @bob check this"
	req := dto.SendMessageRequest{Body: body, ReplyToID: &replyMsgID}
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, senderID).Return(false, "", false, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, mentionedID, replyAuthorID}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, mentionedID).Return(false, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, replyAuthorID).Return(false, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(sampleUser(senderID), nil)
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, replyMsgID).Return(&repository.ChatMessageRow{ID: replyMsgID, RoomID: roomID, SenderID: replyAuthorID, SenderDisplayName: "Parent", Body: "original"}, nil)
	m.chatRepo.EXPECT().InsertMessage(mock.Anything, mock.Anything, roomID, senderID, body, &replyMsgID).Return(nil)
	m.chatRepo.EXPECT().MarkRoomRead(mock.Anything, roomID, senderID).Return(nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return(nil, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUser(mock.Anything, senderID).Return(nil, nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, senderID).Return(&repository.ChatRoomRow{Type: "group", Name: "G"}, nil)
	m.userRepo.EXPECT().GetByUsername(mock.Anything, "bob").Return(&model.User{ID: mentionedID, Username: "bob"}, nil)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, mentionedID).Return(true, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, mentionedID).Return(false, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, mentionedID, replyAuthorID}, nil)
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool { return p.Type == dto.NotifChatMention && p.RecipientID == mentionedID })).Return(nil)
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool { return p.Type == dto.NotifChatReply && p.RecipientID == replyAuthorID })).Return(nil)
	m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, mentionedID).Return(1, nil)
	m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, replyAuthorID).Return(1, nil)

	// when
	got, err := svc.SendMessage(context.Background(), senderID, roomID, req, nil)

	// then
	require.NoError(t, err)
	require.NotNil(t, got.ReplyTo)
	assert.Equal(t, replyMsgID, got.ReplyTo.ID)
}

func TestSendMessage_GroupUnmutedRoomMessage(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	otherID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, senderID).Return(false, "", false, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, otherID}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, otherID).Return(false, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(sampleUser(senderID), nil)
	m.chatRepo.EXPECT().InsertMessage(mock.Anything, mock.Anything, roomID, senderID, "hi", (*uuid.UUID)(nil)).Return(nil)
	m.chatRepo.EXPECT().MarkRoomRead(mock.Anything, roomID, senderID).Return(nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return(nil, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUser(mock.Anything, senderID).Return(nil, nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, senderID).Return(&repository.ChatRoomRow{Type: "group", Name: "G"}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, otherID}, nil)
	m.chatRepo.EXPECT().IsMuted(mock.Anything, roomID, otherID).Return(false, nil)
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool { return p.Type == dto.NotifChatRoomMessage })).Return(nil)
	m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, otherID).Return(1, nil)

	// when
	got, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, "hi", got.Body)
}

func TestSendMessage_GroupMutedNoNotify(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	otherID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, senderID).Return(false, "", false, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, otherID}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, otherID).Return(false, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(sampleUser(senderID), nil)
	m.chatRepo.EXPECT().InsertMessage(mock.Anything, mock.Anything, roomID, senderID, "hi", (*uuid.UUID)(nil)).Return(nil)
	m.chatRepo.EXPECT().MarkRoomRead(mock.Anything, roomID, senderID).Return(nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return(nil, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUser(mock.Anything, senderID).Return(nil, nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, senderID).Return(&repository.ChatRoomRow{Type: "group", Name: "G"}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, otherID}, nil)
	m.chatRepo.EXPECT().IsMuted(mock.Anything, roomID, otherID).Return(true, nil)
	m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, otherID).Return(1, nil)

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"}, nil)

	// then
	require.NoError(t, err)
}

func TestGetRoomsByUser_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomsByUser(mock.Anything, userID).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetRoomsByUser(context.Background(), userID)

	// then
	require.Error(t, err)
}

func TestGetRoomsByUser_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	r1 := uuid.New()
	r2 := uuid.New()
	m.chatRepo.EXPECT().GetRoomsByUser(mock.Anything, userID).Return([]repository.ChatRoomRow{{ID: r1}, {ID: r2}}, nil)

	// when
	got, err := svc.GetRoomsByUser(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{r1, r2}, got)
}

func TestDeleteChat_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(nil, errors.New("boom"))

	// when
	err := svc.DeleteChat(context.Background(), roomID, userID)

	// then
	require.Error(t, err)
}

func TestDeleteChat_NotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(nil, nil)

	// when
	err := svc.DeleteChat(context.Background(), roomID, userID)

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestDeleteChat_SystemRoom(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{IsMember: true, IsSystem: true}, nil)

	// when
	err := svc.DeleteChat(context.Background(), roomID, userID)

	// then
	require.ErrorIs(t, err, ErrSystemRoom)
}

func TestDeleteChat_GroupHost_DeleteMessagesError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{IsMember: true, Type: "group", ViewerRole: "host"}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, userID).Return("host", nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
	m.chatRepo.EXPECT().DeleteMessages(mock.Anything, roomID).Return(errors.New("boom"))

	// when
	err := svc.DeleteChat(context.Background(), roomID, userID)

	// then
	require.Error(t, err)
}

func TestDeleteChat_GroupHost_DeleteRoomError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{IsMember: true, Type: "group", ViewerRole: "host"}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, userID).Return("host", nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
	m.chatRepo.EXPECT().DeleteMessages(mock.Anything, roomID).Return(nil)
	m.chatRepo.EXPECT().DeleteRoom(mock.Anything, roomID).Return(errors.New("boom"))

	// when
	err := svc.DeleteChat(context.Background(), roomID, userID)

	// then
	require.Error(t, err)
}

func TestDeleteChat_GroupHost_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{IsMember: true, Type: "group", ViewerRole: "host"}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, userID).Return("host", nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
	m.chatRepo.EXPECT().DeleteMessages(mock.Anything, roomID).Return(nil)
	m.chatRepo.EXPECT().DeleteRoom(mock.Anything, roomID).Return(nil)

	// when
	err := svc.DeleteChat(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteChat_DM_RemoveError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{IsMember: true, Type: "dm"}, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, roomID, userID).Return(errors.New("boom"))

	// when
	err := svc.DeleteChat(context.Background(), roomID, userID)

	// then
	require.Error(t, err)
}

func TestDeleteChat_DM_CountError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{IsMember: true, Type: "dm"}, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, roomID, userID).Return(nil)
	m.chatRepo.EXPECT().CountRoomMembers(mock.Anything, roomID).Return(0, errors.New("boom"))

	// when
	err := svc.DeleteChat(context.Background(), roomID, userID)

	// then
	require.Error(t, err)
}

func TestDeleteChat_DM_LastMemberDeletesRoom(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{IsMember: true, Type: "dm"}, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, roomID, userID).Return(nil)
	m.chatRepo.EXPECT().CountRoomMembers(mock.Anything, roomID).Return(0, nil)
	m.chatRepo.EXPECT().DeleteMessages(mock.Anything, roomID).Return(nil)
	m.chatRepo.EXPECT().DeleteRoom(mock.Anything, roomID).Return(nil)

	// when
	err := svc.DeleteChat(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteChat_DM_StillHasMembers(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{IsMember: true, Type: "dm"}, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, roomID, userID).Return(nil)
	m.chatRepo.EXPECT().CountRoomMembers(mock.Anything, roomID).Return(1, nil)

	// when
	err := svc.DeleteChat(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
}

func TestGetUnreadCount_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, userID).Return(0, errors.New("boom"))

	// when
	_, err := svc.GetUnreadCount(context.Background(), userID)

	// then
	require.Error(t, err)
}

func TestGetUnreadCount_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, userID).Return(3, nil)

	// when
	got, err := svc.GetUnreadCount(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, got)
}

func TestMarkRead_IsMemberError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(false, errors.New("boom"))

	// when
	err := svc.MarkRead(context.Background(), roomID, userID)

	// then
	require.Error(t, err)
}

func TestMarkRead_NotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(false, nil)

	// when
	err := svc.MarkRead(context.Background(), roomID, userID)

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestMarkRead_MarkError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().MarkRoomRead(mock.Anything, roomID, userID).Return(errors.New("boom"))

	// when
	err := svc.MarkRead(context.Background(), roomID, userID)

	// then
	require.Error(t, err)
}

func TestMarkRead_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().MarkRoomRead(mock.Anything, roomID, userID).Return(nil)
	m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, userID).Return(0, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)

	// when
	err := svc.MarkRead(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
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
			// given
			a := tc.last
			b := tc.read

			// when
			got := isUnread(a, b)

			// then
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestNullStr(t *testing.T) {
	// given
	valid := sql.NullString{Valid: true, String: "x"}
	invalid := sql.NullString{}

	// when
	gotValid := nullStr(valid)
	gotInvalid := nullStr(invalid)

	// then
	assert.Equal(t, "x", gotValid)
	assert.Equal(t, "", gotInvalid)
}

func TestEligibleForMods(t *testing.T) {
	assert.True(t, eligibleForMods(authz.RoleModerator))
	assert.True(t, eligibleForMods(authz.RoleAdmin))
	assert.True(t, eligibleForMods(authz.RoleSuperAdmin))
	assert.False(t, eligibleForMods(role.Role("user")))
}

func TestEligibleForAdmins(t *testing.T) {
	assert.False(t, eligibleForAdmins(authz.RoleModerator))
	assert.True(t, eligibleForAdmins(authz.RoleAdmin))
	assert.True(t, eligibleForAdmins(authz.RoleSuperAdmin))
}

func TestMemberRoleForSystem(t *testing.T) {
	assert.Equal(t, "host", memberRoleForSystem(authz.RoleSuperAdmin))
	assert.Equal(t, "member", memberRoleForSystem(authz.RoleAdmin))
	assert.Equal(t, "member", memberRoleForSystem(authz.RoleModerator))
}

func TestMessageRowToResponse_ReplyTruncation(t *testing.T) {
	// given
	msgID := uuid.New()
	senderID := uuid.New()
	replyID := uuid.New()
	longBody := ""
	for i := 0; i < 200; i++ {
		longBody += "x"
	}
	row := repository.ChatMessageRow{
		ID:                msgID,
		RoomID:            uuid.New(),
		SenderID:          senderID,
		SenderUsername:    "u",
		SenderDisplayName: "d",
		Body:              "body",
		ReplyToID:         &replyID,
		ReplyToSenderID:   new(uuid.New()),
		ReplyToSenderName: new("Sender"),
		ReplyToBody:       &longBody,
	}

	// when
	svc, _ := newTestService(t)
	got := svc.messageRowToResponse(row, nil, nil, nil)

	// then
	require.NotNil(t, got.ReplyTo)
	assert.Equal(t, replyID, got.ReplyTo.ID)
	assert.Equal(t, 143, len(got.ReplyTo.BodyPreview))
}

func TestMessageRowToResponse_NoReply(t *testing.T) {
	// given
	row := repository.ChatMessageRow{ID: uuid.New(), RoomID: uuid.New(), SenderID: uuid.New(), Body: "hi"}

	// when
	svc, _ := newTestService(t)
	got := svc.messageRowToResponse(row, nil, nil, nil)

	// then
	assert.Nil(t, got.ReplyTo)
	assert.Equal(t, "hi", got.Body)
}

func TestSetRoomNickname_HappyPath(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().IsMemberNicknameLocked(mock.Anything, roomID, userID).Return(false, nil)
	m.chatRepo.EXPECT().SetMemberNickname(mock.Anything, roomID, userID, "Alice").Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.Anything, roomID, userID, mock.Anything).Return(errors.New("boom"))
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{
		{UserID: userID, Role: "member", Nickname: "Alice"},
	}, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{userID}).Return(nil, nil)

	// when
	got, err := svc.SetRoomNickname(context.Background(), roomID, userID, "Alice")

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Alice", got.Nickname)
}

func TestSetRoomNickname_TrimsAndCapsAt32(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	input := "  "
	for i := 0; i < 50; i++ {
		input += "a"
	}
	input += "  "
	expected := ""
	for i := 0; i < 32; i++ {
		expected += "a"
	}
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().IsMemberNicknameLocked(mock.Anything, roomID, userID).Return(false, nil)
	m.chatRepo.EXPECT().SetMemberNickname(mock.Anything, roomID, userID, expected).Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.Anything, roomID, userID, mock.Anything).Return(errors.New("boom"))
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{
		{UserID: userID, Role: "member", Nickname: expected},
	}, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{userID}).Return(nil, nil)

	// when
	got, err := svc.SetRoomNickname(context.Background(), roomID, userID, input)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, expected, got.Nickname)
}

func TestSetRoomNickname_NotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(false, nil)

	// when
	got, err := svc.SetRoomNickname(context.Background(), roomID, userID, "nick")

	// then
	require.ErrorIs(t, err, ErrNotMember)
	assert.Nil(t, got)
}

func TestSetRoomNickname_IsMemberError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(false, errors.New("db"))

	// when
	got, err := svc.SetRoomNickname(context.Background(), roomID, userID, "nick")

	// then
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestSetRoomNickname_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().IsMemberNicknameLocked(mock.Anything, roomID, userID).Return(false, nil)
	m.chatRepo.EXPECT().SetMemberNickname(mock.Anything, roomID, userID, "nick").Return(errors.New("db"))

	// when
	got, err := svc.SetRoomNickname(context.Background(), roomID, userID, "nick")

	// then
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestSetRoomAvatar_HappyPath(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	data := bytes.NewReader([]byte("img"))
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().IsMemberNicknameLocked(mock.Anything, roomID, userID).Return(false, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1024)
	m.uploadSvc.EXPECT().SaveImage(mock.Anything, mock.Anything, userID, int64(3), int64(1024), data).Return("avatar.png", nil)
	m.chatRepo.EXPECT().SetMemberAvatar(mock.Anything, roomID, userID, "avatar.png").Return(nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{
		{UserID: userID, Role: "member", MemberAvatarURL: "avatar.png"},
	}, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{userID}).Return(nil, nil)

	// when
	got, err := svc.SetRoomAvatar(context.Background(), roomID, userID, "image/png", 3, data)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "avatar.png", got.MemberAvatarURL)
}

func TestSetRoomAvatar_NotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(false, nil)

	// when
	_, err := svc.SetRoomAvatar(context.Background(), roomID, userID, "image/png", 3, bytes.NewReader([]byte("x")))

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestSetRoomAvatar_UploadError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().IsMemberNicknameLocked(mock.Anything, roomID, userID).Return(false, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1024)
	m.uploadSvc.EXPECT().SaveImage(mock.Anything, mock.Anything, userID, int64(3), int64(1024), mock.Anything).Return("", errors.New("too big"))

	// when
	_, err := svc.SetRoomAvatar(context.Background(), roomID, userID, "image/png", 3, bytes.NewReader([]byte("img")))

	// then
	require.Error(t, err)
}

func TestSetRoomAvatar_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().IsMemberNicknameLocked(mock.Anything, roomID, userID).Return(false, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1024)
	m.uploadSvc.EXPECT().SaveImage(mock.Anything, mock.Anything, userID, int64(3), int64(1024), mock.Anything).Return("avatar.png", nil)
	m.chatRepo.EXPECT().SetMemberAvatar(mock.Anything, roomID, userID, "avatar.png").Return(errors.New("db"))

	// when
	_, err := svc.SetRoomAvatar(context.Background(), roomID, userID, "image/png", 3, bytes.NewReader([]byte("img")))

	// then
	require.Error(t, err)
}

func TestClearRoomAvatar_HappyPath(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().IsMemberNicknameLocked(mock.Anything, roomID, userID).Return(false, nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{
		{UserID: userID, Role: "member", MemberAvatarURL: "old.png"},
	}, nil).Once()
	m.uploadSvc.EXPECT().Delete("old.png").Return(nil)
	m.chatRepo.EXPECT().SetMemberAvatar(mock.Anything, roomID, userID, "").Return(nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{
		{UserID: userID, Role: "member", MemberAvatarURL: ""},
	}, nil).Once()
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{userID}).Return(nil, nil)

	// when
	got, err := svc.ClearRoomAvatar(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "", got.MemberAvatarURL)
}

func TestClearRoomAvatar_NoExistingAvatar(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().IsMemberNicknameLocked(mock.Anything, roomID, userID).Return(false, nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{
		{UserID: userID, Role: "member", MemberAvatarURL: ""},
	}, nil).Twice()
	m.chatRepo.EXPECT().SetMemberAvatar(mock.Anything, roomID, userID, "").Return(nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{userID}).Return(nil, nil)

	// when
	got, err := svc.ClearRoomAvatar(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestClearRoomAvatar_NotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(false, nil)

	// when
	got, err := svc.ClearRoomAvatar(context.Background(), roomID, userID)

	// then
	require.ErrorIs(t, err, ErrNotMember)
	assert.Nil(t, got)
}

func TestClearRoomAvatar_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().IsMemberNicknameLocked(mock.Anything, roomID, userID).Return(false, nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return(nil, errors.New("db"))
	m.chatRepo.EXPECT().SetMemberAvatar(mock.Anything, roomID, userID, "").Return(errors.New("db"))

	// when
	got, err := svc.ClearRoomAvatar(context.Background(), roomID, userID)

	// then
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestPinMessage_HappyPath(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, userID).Return("host", nil)
	m.chatRepo.EXPECT().PinMessage(mock.Anything, messageID, userID).Return(nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil).Maybe()

	// when
	err := svc.PinMessage(context.Background(), messageID, userID)

	// then
	require.NoError(t, err)
}

func TestPinMessage_MessageNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(nil, nil)

	// when
	err := svc.PinMessage(context.Background(), messageID, userID)

	// then
	require.ErrorIs(t, err, ErrRoomNotFound)
}

func TestPinMessage_GetMessageError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(nil, errors.New("db"))

	// when
	err := svc.PinMessage(context.Background(), messageID, userID)

	// then
	require.Error(t, err)
}

func TestPinMessage_NotHost(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, userID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return("", nil)

	// when
	err := svc.PinMessage(context.Background(), messageID, userID)

	// then
	require.ErrorIs(t, err, ErrNotHost)
}

func TestPinMessage_GetMemberRoleError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, userID).Return("", errors.New("db"))

	// when
	err := svc.PinMessage(context.Background(), messageID, userID)

	// then
	require.Error(t, err)
}

func TestPinMessage_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, userID).Return("host", nil)
	m.chatRepo.EXPECT().PinMessage(mock.Anything, messageID, userID).Return(errors.New("db"))

	// when
	err := svc.PinMessage(context.Background(), messageID, userID)

	// then
	require.Error(t, err)
}

func TestUnpinMessage_HappyPath(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID, PinnedAt: new("2024-01-01T00:00:00Z")}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, userID).Return("host", nil)
	m.chatRepo.EXPECT().UnpinMessage(mock.Anything, messageID).Return(nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil).Maybe()

	// when
	err := svc.UnpinMessage(context.Background(), messageID, userID)

	// then
	require.NoError(t, err)
}

func TestUnpinMessage_MessageNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(nil, nil)

	// when
	err := svc.UnpinMessage(context.Background(), messageID, userID)

	// then
	require.ErrorIs(t, err, ErrRoomNotFound)
}

func TestUnpinMessage_NotPinned(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID, PinnedAt: nil}, nil)

	// when
	err := svc.UnpinMessage(context.Background(), messageID, userID)

	// then
	require.ErrorIs(t, err, ErrMessageNotPinned)
}

func TestUnpinMessage_NotHost(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID, PinnedAt: new("2024-01-01T00:00:00Z")}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, userID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return("", nil)

	// when
	err := svc.UnpinMessage(context.Background(), messageID, userID)

	// then
	require.ErrorIs(t, err, ErrNotHost)
}

func TestUnpinMessage_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID, PinnedAt: new("2024-01-01T00:00:00Z")}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, userID).Return("host", nil)
	m.chatRepo.EXPECT().UnpinMessage(mock.Anything, messageID).Return(errors.New("db"))

	// when
	err := svc.UnpinMessage(context.Background(), messageID, userID)

	// then
	require.Error(t, err)
}

func TestListPinnedMessages_HappyPath(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	viewerID := uuid.New()
	msgID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, viewerID).Return(true, nil)
	m.chatRepo.EXPECT().ListPinnedMessages(mock.Anything, roomID).Return([]repository.ChatMessageRow{
		{ID: msgID, RoomID: roomID, SenderID: uuid.New(), Body: "pinned"},
	}, nil)
	m.chatRepo.EXPECT().GetMessageMediaBatch(mock.Anything, []uuid.UUID{msgID}).Return(nil, nil)
	m.chatRepo.EXPECT().GetReactionsBatch(mock.Anything, []uuid.UUID{msgID}, viewerID).Return(nil, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, mock.Anything).Return(nil, nil)

	// when
	res, err := svc.ListPinnedMessages(context.Background(), roomID, viewerID)

	// then
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 1, res.Total)
	assert.Equal(t, "pinned", res.Messages[0].Body)
}

func TestListPinnedMessages_NotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	viewerID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, viewerID).Return(false, nil)

	// when
	_, err := svc.ListPinnedMessages(context.Background(), roomID, viewerID)

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestListPinnedMessages_IsMemberError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	viewerID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, viewerID).Return(false, errors.New("db"))

	// when
	_, err := svc.ListPinnedMessages(context.Background(), roomID, viewerID)

	// then
	require.Error(t, err)
}

func TestListPinnedMessages_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	viewerID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, viewerID).Return(true, nil)
	m.chatRepo.EXPECT().ListPinnedMessages(mock.Anything, roomID).Return(nil, errors.New("db"))

	// when
	_, err := svc.ListPinnedMessages(context.Background(), roomID, viewerID)

	// then
	require.Error(t, err)
}

func TestAddReaction_HappyPath(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID}, nil)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, userID).Return(false, "", false, nil)
	m.chatRepo.EXPECT().AddReaction(mock.Anything, messageID, userID, "👍").Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil).Maybe()
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return(nil, nil).Maybe()
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil).Maybe()

	// when
	err := svc.AddReaction(context.Background(), messageID, userID, "👍")

	// then
	require.NoError(t, err)
}

func TestAddReaction_TimedOut(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID}, nil)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, userID).Return(true, "2099-01-01T00:00:00Z", false, nil)

	// when
	err := svc.AddReaction(context.Background(), messageID, userID, "👍")

	// then
	require.ErrorIs(t, err, ErrTimedOut)
}

func TestAddReaction_EmptyEmoji(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	messageID := uuid.New()
	userID := uuid.New()

	// when
	err := svc.AddReaction(context.Background(), messageID, userID, "")

	// then
	require.ErrorIs(t, err, ErrInvalidEmoji)
}

func TestAddReaction_OversizedEmoji(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	messageID := uuid.New()
	userID := uuid.New()
	big := ""
	for i := 0; i < 20; i++ {
		big += "x"
	}

	// when
	err := svc.AddReaction(context.Background(), messageID, userID, big)

	// then
	require.ErrorIs(t, err, ErrInvalidEmoji)
}

func TestAddReaction_MessageNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(nil, nil)

	// when
	err := svc.AddReaction(context.Background(), messageID, userID, "👍")

	// then
	require.ErrorIs(t, err, ErrRoomNotFound)
}

func TestAddReaction_NotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID}, nil)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(false, nil)

	// when
	err := svc.AddReaction(context.Background(), messageID, userID, "👍")

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestAddReaction_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID}, nil)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, userID).Return(false, "", false, nil)
	m.chatRepo.EXPECT().AddReaction(mock.Anything, messageID, userID, "👍").Return(errors.New("db"))

	// when
	err := svc.AddReaction(context.Background(), messageID, userID, "👍")

	// then
	require.Error(t, err)
}

func TestRemoveReaction_HappyPath(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID}, nil)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().RemoveReaction(mock.Anything, messageID, userID, "👍").Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil).Maybe()
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return(nil, nil).Maybe()
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil).Maybe()

	// when
	err := svc.RemoveReaction(context.Background(), messageID, userID, "👍")

	// then
	require.NoError(t, err)
}

func TestRemoveReaction_EmptyEmoji(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.RemoveReaction(context.Background(), uuid.New(), uuid.New(), "")

	// then
	require.ErrorIs(t, err, ErrInvalidEmoji)
}

func TestRemoveReaction_OversizedEmoji(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	big := ""
	for i := 0; i < 20; i++ {
		big += "x"
	}

	// when
	err := svc.RemoveReaction(context.Background(), uuid.New(), uuid.New(), big)

	// then
	require.ErrorIs(t, err, ErrInvalidEmoji)
}

func TestRemoveReaction_MessageNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(nil, nil)

	// when
	err := svc.RemoveReaction(context.Background(), messageID, userID, "👍")

	// then
	require.ErrorIs(t, err, ErrRoomNotFound)
}

func TestRemoveReaction_NotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID}, nil)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(false, nil)

	// when
	err := svc.RemoveReaction(context.Background(), messageID, userID, "👍")

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestRemoveReaction_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID}, nil)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().RemoveReaction(mock.Anything, messageID, userID, "👍").Return(errors.New("db"))

	// when
	err := svc.RemoveReaction(context.Background(), messageID, userID, "👍")

	// then
	require.Error(t, err)
}

func TestCanModerateRoom_Host(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, userID).Return("host", nil)

	// when
	got, err := svc.canModerateRoom(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
	assert.True(t, got)
}

func TestCanModerateRoom_SiteAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, userID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(authz.RoleAdmin, nil)

	// when
	got, err := svc.canModerateRoom(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
	assert.True(t, got)
}

func TestCanModerateRoom_SiteMod(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, userID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(authz.RoleModerator, nil)

	// when
	got, err := svc.canModerateRoom(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
	assert.True(t, got)
}

func TestCanModerateRoom_SuperAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, userID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(authz.RoleSuperAdmin, nil)

	// when
	got, err := svc.canModerateRoom(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
	assert.True(t, got)
}

func TestCanModerateRoom_RegularMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, userID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role(""), nil)

	// when
	got, err := svc.canModerateRoom(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
	assert.False(t, got)
}

func TestCanModerateRoom_NonMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, userID).Return("", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role(""), nil)

	// when
	got, err := svc.canModerateRoom(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
	assert.False(t, got)
}

func TestSetMemberNicknameAsMod_SiteMod_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(authz.RoleModerator, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, targetID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().SetMemberNicknameWithLock(mock.Anything, roomID, targetID, "Silence", true).Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, targetID).Return(sampleUser(targetID), nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(sampleUser(actorID), nil)
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.Anything, roomID, actorID, mock.Anything).Return(errors.New("boom"))
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{
		{UserID: targetID, Username: "t", DisplayName: "T", Role: "member", Nickname: "Silence", NicknameLocked: true},
	}, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{targetID}).Return(nil, nil)

	// when
	got, err := svc.SetMemberNicknameAsMod(context.Background(), roomID, actorID, targetID, "Silence")

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Silence", got.Nickname)
	assert.True(t, got.NicknameLocked)
}

func TestSetMemberNicknameAsMod_HostOnlyRefused(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(role.Role(""), nil)

	// when
	got, err := svc.SetMemberNicknameAsMod(context.Background(), roomID, actorID, targetID, "x")

	// then
	require.ErrorIs(t, err, ErrModRoleRequired)
	assert.Nil(t, got)
}

func TestSetMemberNicknameAsMod_RegularMemberRefused(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(role.Role(""), nil)

	// when
	got, err := svc.SetMemberNicknameAsMod(context.Background(), roomID, actorID, targetID, "x")

	// then
	require.ErrorIs(t, err, ErrModRoleRequired)
	assert.Nil(t, got)
}

func TestSetMemberNicknameAsMod_TargetIsSiteMod(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(authz.RoleAdmin, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, targetID).Return(authz.RoleModerator, nil)

	// when
	got, err := svc.SetMemberNicknameAsMod(context.Background(), roomID, actorID, targetID, "x")

	// then
	require.ErrorIs(t, err, ErrTargetImmune)
	assert.Nil(t, got)
}

func TestSetMemberNicknameAsMod_TargetNotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(authz.RoleAdmin, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("", nil)

	// when
	got, err := svc.SetMemberNicknameAsMod(context.Background(), roomID, actorID, targetID, "x")

	// then
	require.ErrorIs(t, err, ErrNotMember)
	assert.Nil(t, got)
}

func TestSetMemberNicknameAsMod_TrimsAndCapsAt32(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()
	input := "  "
	for i := 0; i < 50; i++ {
		input += "a"
	}
	expected := ""
	for i := 0; i < 32; i++ {
		expected += "a"
	}
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(authz.RoleAdmin, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, targetID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().SetMemberNicknameWithLock(mock.Anything, roomID, targetID, expected, true).Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, targetID).Return(sampleUser(targetID), nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(sampleUser(actorID), nil)
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.Anything, roomID, actorID, mock.Anything).Return(errors.New("boom"))
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{
		{UserID: targetID, Role: "member", Nickname: expected, NicknameLocked: true},
	}, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{targetID}).Return(nil, nil)

	// when
	got, err := svc.SetMemberNicknameAsMod(context.Background(), roomID, actorID, targetID, input)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, expected, got.Nickname)
}

func TestSetMemberNicknameAsMod_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(authz.RoleAdmin, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, targetID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().SetMemberNicknameWithLock(mock.Anything, roomID, targetID, "x", true).Return(errors.New("db"))

	// when
	got, err := svc.SetMemberNicknameAsMod(context.Background(), roomID, actorID, targetID, "x")

	// then
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestUnlockMemberNickname_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(authz.RoleModerator, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, targetID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().SetMemberNicknameWithLock(mock.Anything, roomID, targetID, "", false).Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, targetID).Return(sampleUser(targetID), nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(sampleUser(actorID), nil)
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.Anything, roomID, actorID, mock.Anything).Return(errors.New("boom"))
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{
		{UserID: targetID, Role: "member"},
	}, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{targetID}).Return(nil, nil)

	// when
	got, err := svc.UnlockMemberNickname(context.Background(), roomID, actorID, targetID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.False(t, got.NicknameLocked)
}

func TestUnlockMemberNickname_NonModRefused(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(role.Role(""), nil)

	// when
	got, err := svc.UnlockMemberNickname(context.Background(), roomID, actorID, targetID)

	// then
	require.ErrorIs(t, err, ErrModRoleRequired)
	assert.Nil(t, got)
}

func TestUnlockMemberNickname_TargetIsSiteMod(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(authz.RoleAdmin, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, targetID).Return(authz.RoleAdmin, nil)

	// when
	got, err := svc.UnlockMemberNickname(context.Background(), roomID, actorID, targetID)

	// then
	require.ErrorIs(t, err, ErrTargetImmune)
	assert.Nil(t, got)
}

func TestUnlockMemberNickname_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(authz.RoleAdmin, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, targetID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().SetMemberNicknameWithLock(mock.Anything, roomID, targetID, "", false).Return(errors.New("db"))

	// when
	got, err := svc.UnlockMemberNickname(context.Background(), roomID, actorID, targetID)

	// then
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestSetRoomNickname_Locked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().IsMemberNicknameLocked(mock.Anything, roomID, userID).Return(true, nil)

	// when
	got, err := svc.SetRoomNickname(context.Background(), roomID, userID, "x")

	// then
	require.ErrorIs(t, err, ErrNicknameLocked)
	assert.Nil(t, got)
}

func TestSetRoomNickname_SiteMod_BypassesLock(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(authz.RoleModerator, nil)
	m.chatRepo.EXPECT().SetMemberNickname(mock.Anything, roomID, userID, "Alice").Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.Anything, roomID, userID, mock.Anything).Return(errors.New("boom"))
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{
		{UserID: userID, Role: "member", Nickname: "Alice"},
	}, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{userID}).Return(nil, nil)

	// when
	got, err := svc.SetRoomNickname(context.Background(), roomID, userID, "Alice")

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestSetRoomAvatar_Locked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().IsMemberNicknameLocked(mock.Anything, roomID, userID).Return(true, nil)

	// when
	_, err := svc.SetRoomAvatar(context.Background(), roomID, userID, "image/png", 3, bytes.NewReader([]byte("x")))

	// then
	require.ErrorIs(t, err, ErrNicknameLocked)
}

func TestSetRoomAvatar_SiteMod_BypassesLock(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	data := bytes.NewReader([]byte("img"))
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(authz.RoleAdmin, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1024)
	m.uploadSvc.EXPECT().SaveImage(mock.Anything, mock.Anything, userID, int64(3), int64(1024), data).Return("avatar.png", nil)
	m.chatRepo.EXPECT().SetMemberAvatar(mock.Anything, roomID, userID, "avatar.png").Return(nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{
		{UserID: userID, Role: "member", MemberAvatarURL: "avatar.png"},
	}, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{userID}).Return(nil, nil)

	// when
	got, err := svc.SetRoomAvatar(context.Background(), roomID, userID, "image/png", 3, data)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "avatar.png", got.MemberAvatarURL)
}

func TestClearRoomAvatar_Locked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role(""), nil)
	m.chatRepo.EXPECT().IsMemberNicknameLocked(mock.Anything, roomID, userID).Return(true, nil)

	// when
	got, err := svc.ClearRoomAvatar(context.Background(), roomID, userID)

	// then
	require.ErrorIs(t, err, ErrNicknameLocked)
	assert.Nil(t, got)
}

func TestClearRoomAvatar_SiteMod_BypassesLock(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(authz.RoleSuperAdmin, nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{
		{UserID: userID, Role: "member", MemberAvatarURL: ""},
	}, nil)
	m.chatRepo.EXPECT().SetMemberAvatar(mock.Anything, roomID, userID, "").Return(nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{userID}).Return(nil, nil)

	// when
	got, err := svc.ClearRoomAvatar(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestKickMember_SiteMod_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actorID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, actorID).Return(&repository.ChatRoomRow{}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, actorID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(authz.RoleModerator, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, targetID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, targetID).Return("", nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{actorID, targetID}, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, roomID, targetID).Return(nil)
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.Anything, roomID, actorID, mock.Anything).Return(errors.New("boom"))

	// when
	err := svc.KickMember(context.Background(), actorID, roomID, targetID)

	// then
	require.NoError(t, err)
}

func TestDeleteChat_SiteMod_GroupOK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	actorID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, actorID).Return(&repository.ChatRoomRow{Type: "group", IsMember: false}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, actorID).Return("", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(authz.RoleAdmin, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil)
	m.chatRepo.EXPECT().DeleteMessages(mock.Anything, roomID).Return(nil)
	m.chatRepo.EXPECT().DeleteRoom(mock.Anything, roomID).Return(nil)

	// when
	err := svc.DeleteChat(context.Background(), roomID, actorID)

	// then
	require.NoError(t, err)
}

func TestPinMessage_SiteMod_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	actorID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, actorID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(authz.RoleModerator, nil)
	m.chatRepo.EXPECT().PinMessage(mock.Anything, messageID, actorID).Return(nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil).Maybe()

	// when
	err := svc.PinMessage(context.Background(), messageID, actorID)

	// then
	require.NoError(t, err)
}

func TestUnpinMessage_SiteMod_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	actorID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID, PinnedAt: new("2024-01-01T00:00:00Z")}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, actorID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(authz.RoleAdmin, nil)
	m.chatRepo.EXPECT().UnpinMessage(mock.Anything, messageID).Return(nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil).Maybe()

	// when
	err := svc.UnpinMessage(context.Background(), messageID, actorID)

	// then
	require.NoError(t, err)
}

func TestDeleteMessage_Author_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	authorID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID, SenderID: authorID}, nil)
	m.chatRepo.EXPECT().DeleteMessage(mock.Anything, messageID).Return(nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil).Maybe()

	// when
	err := svc.DeleteMessage(context.Background(), messageID, authorID)

	// then
	require.NoError(t, err)
}

func TestDeleteMessage_Host_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	senderID := uuid.New()
	hostID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID, SenderID: senderID}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, hostID).Return("host", nil)
	m.chatRepo.EXPECT().DeleteMessage(mock.Anything, messageID).Return(nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil).Maybe()

	// when
	err := svc.DeleteMessage(context.Background(), messageID, hostID)

	// then
	require.NoError(t, err)
}

func TestDeleteMessage_SiteMod_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	senderID := uuid.New()
	modID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID, SenderID: senderID}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, modID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, modID).Return(authz.RoleModerator, nil)
	m.chatRepo.EXPECT().DeleteMessage(mock.Anything, messageID).Return(nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil).Maybe()

	// when
	err := svc.DeleteMessage(context.Background(), messageID, modID)

	// then
	require.NoError(t, err)
}

func TestDeleteMessage_NotAuthorNotMod_Refused(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	senderID := uuid.New()
	actorID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID, SenderID: senderID}, nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, roomID, actorID).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(role.Role(""), nil)

	// when
	err := svc.DeleteMessage(context.Background(), messageID, actorID)

	// then
	require.ErrorIs(t, err, ErrMessageDeletePermission)
}

func TestDeleteMessage_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	actorID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(nil, nil)

	// when
	err := svc.DeleteMessage(context.Background(), messageID, actorID)

	// then
	require.ErrorIs(t, err, ErrRoomNotFound)
}

func editedAtPtr(s string) *string { return &s }

func TestEditMessage_Author_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	authorID := uuid.New()
	original := &repository.ChatMessageRow{ID: messageID, RoomID: roomID, SenderID: authorID, Body: "old"}
	updated := &repository.ChatMessageRow{ID: messageID, RoomID: roomID, SenderID: authorID, Body: "new", EditedAt: editedAtPtr("2026-04-18T20:00:00Z")}
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(original, nil).Once()
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, authorID).Return(false, "", false, nil)
	m.chatRepo.EXPECT().EditMessage(mock.Anything, messageID, "new").Return(nil)
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(updated, nil).Once()
	m.chatRepo.EXPECT().GetMessageMediaBatch(mock.Anything, []uuid.UUID{messageID}).Return(nil, nil)
	m.chatRepo.EXPECT().GetReactionsBatch(mock.Anything, []uuid.UUID{messageID}, authorID).Return(nil, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUser(mock.Anything, authorID).Return(nil, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil).Maybe()

	// when
	resp, err := svc.EditMessage(context.Background(), messageID, authorID, "new")

	// then
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "new", resp.Body)
	require.NotNil(t, resp.EditedAt)
}

func TestEditMessage_NotAuthor_Refused(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	senderID := uuid.New()
	otherID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID, SenderID: senderID, Body: "old"}, nil)

	// when
	resp, err := svc.EditMessage(context.Background(), messageID, otherID, "new")

	// then
	require.ErrorIs(t, err, ErrMessageEditPermission)
	assert.Nil(t, resp)
}

func TestEditMessage_SystemMessage_Refused(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	authorID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID, SenderID: authorID, Body: "old", IsSystem: true}, nil)

	// when
	resp, err := svc.EditMessage(context.Background(), messageID, authorID, "new")

	// then
	require.ErrorIs(t, err, ErrCannotEditSystemMessage)
	assert.Nil(t, resp)
}

func TestEditMessage_EmptyBody_Rejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	resp, err := svc.EditMessage(context.Background(), uuid.New(), uuid.New(), "")

	// then
	require.ErrorIs(t, err, ErrMissingFields)
	assert.Nil(t, resp)
}

func TestEditMessage_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	actorID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(nil, nil)

	// when
	resp, err := svc.EditMessage(context.Background(), messageID, actorID, "new")

	// then
	require.ErrorIs(t, err, ErrRoomNotFound)
	assert.Nil(t, resp)
}

func TestEditMessage_TimedOut_Refused(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	authorID := uuid.New()
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&repository.ChatMessageRow{ID: messageID, RoomID: roomID, SenderID: authorID, Body: "old"}, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, roomID, authorID).Return(true, "", false, nil)

	// when
	resp, err := svc.EditMessage(context.Background(), messageID, authorID, "new")

	// then
	require.ErrorIs(t, err, ErrTimedOut)
	assert.Nil(t, resp)
}

func TestJoinRoom_Ghost_RequiresStaff(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{
		ID: roomID, Type: "group", IsPublic: true, CreatedBy: uuid.New(),
	}, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role(""), nil)

	// when
	_, err := svc.JoinRoom(context.Background(), roomID, userID, true)

	// then
	require.ErrorIs(t, err, ErrGhostRequiresStaff)
}

func unsetGhostDefaults(repo *repository.MockChatRepository) {
	var kept []*mock.Call
	for _, c := range repo.ExpectedCalls {
		if c.Method == "HasGhostMembers" || c.Method == "IsGhostMember" {
			continue
		}
		kept = append(kept, c)
	}
	repo.ExpectedCalls = kept
}

func TestJoinRoom_Ghost_StaffAllowedAndSilent(t *testing.T) {
	// given
	svc, m := newTestService(t)
	unsetGhostDefaults(m.chatRepo)
	roomID := uuid.New()
	userID := uuid.New()
	otherMember := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{
		ID: roomID, Type: "group", IsPublic: true, CreatedBy: uuid.New(),
	}, nil).Once()
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role("moderator"), nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, mock.Anything).Return(0)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, roomID, userID, "member", true).Return(nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{
		ID: roomID, Type: "group", IsMember: true,
	}, nil).Once()
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID, otherMember}, nil)
	m.chatRepo.EXPECT().HasGhostMembers(mock.Anything, roomID).Return(true, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, otherMember).Return(sampleUser(otherMember), nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, otherMember).Return(role.Role(""), nil).Maybe()

	// when
	_, err := svc.JoinRoom(context.Background(), roomID, userID, true)

	// then
	require.NoError(t, err)
	m.chatRepo.AssertNotCalled(t, "InsertSystemMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestLeaveRoom_Ghost_Silent(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	unsetGhostDefaults(m.chatRepo)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{
		ID: roomID, Type: "group", IsMember: true, ViewerRole: "member",
	}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
	m.chatRepo.EXPECT().HasGhostMembers(mock.Anything, roomID).Return(true, nil).Once()
	m.chatRepo.EXPECT().IsGhostMember(mock.Anything, roomID, userID).Return(true, nil).Once()
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, roomID, userID).Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, userID).Return(role.Role(""), nil).Maybe()

	// when
	err := svc.LeaveRoom(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
	m.chatRepo.AssertNotCalled(t, "InsertSystemMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestGetMembers_FiltersGhostsForNonStaff(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	viewerID := uuid.New()
	ghostID := uuid.New()
	normalID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, viewerID).Return(true, nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{
		{UserID: normalID, Username: "a", DisplayName: "A", Role: "member"},
		{UserID: ghostID, Username: "g", DisplayName: "G", Role: "member", Ghost: true},
	}, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, viewerID).Return(role.Role(""), nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, []uuid.UUID{normalID}).Return(nil, nil)

	// when
	got, err := svc.GetMembers(context.Background(), viewerID, roomID)

	// then
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, normalID, got[0].User.ID)
}

func TestGetMembers_StaffSeesGhosts(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	viewerID := uuid.New()
	ghostID := uuid.New()
	normalID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, viewerID).Return(true, nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return([]repository.ChatRoomMemberRow{
		{UserID: normalID, Username: "a", DisplayName: "A", Role: "member"},
		{UserID: ghostID, Username: "g", DisplayName: "G", Role: "member", Ghost: true},
	}, nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, viewerID).Return(role.Role("moderator"), nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, mock.Anything).Return(nil, nil)

	// when
	got, err := svc.GetMembers(context.Background(), viewerID, roomID)

	// then
	require.NoError(t, err)
	require.Len(t, got, 2)
	var ghostSeen bool
	for _, r := range got {
		if r.Ghost {
			ghostSeen = true
		}
	}
	assert.True(t, ghostSeen)
}
