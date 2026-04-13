package chat

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
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
	chatRepo    *repository.MockChatRepository
	userRepo    *repository.MockUserRepository
	roleRepo    *repository.MockRoleRepository
	notifSvc    *notification.MockService
	blockSvc    *block.MockService
	uploadSvc   *upload.MockService
	settingsSvc *settings.MockService
	hub         *ws.Hub
}

func newTestService(t *testing.T) (*service, *testMocks) {
	chatRepo := repository.NewMockChatRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	roleRepo := repository.NewMockRoleRepository(t)
	notifSvc := notification.NewMockService(t)
	blockSvc := block.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	mediaProc := &media.Processor{}
	hub := ws.NewHub()
	svc := NewService(chatRepo, userRepo, roleRepo, notifSvc, blockSvc, uploadSvc, settingsSvc, mediaProc, hub).(*service)
	return svc, &testMocks{
		chatRepo:    chatRepo,
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		notifSvc:    notifSvc,
		blockSvc:    blockSvc,
		uploadSvc:   uploadSvc,
		settingsSvc: settingsSvc,
		hub:         hub,
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

// --- sanitizeTags ---

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

// --- ResolveDMRoom ---

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

// --- SendDMMessage ---

func TestSendDMMessage_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.SendDMMessage(context.Background(), uuid.New(), uuid.New(), "")

	// then
	require.ErrorIs(t, err, ErrMissingFields)
}

func TestSendDMMessage_PreconditionFails(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	id := uuid.New()

	// when
	_, err := svc.SendDMMessage(context.Background(), id, id, "hi")

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
	_, err := svc.SendDMMessage(context.Background(), sender, recipient, "hi")

	// then
	require.Error(t, err)
}

// --- CreateGroupRoom ---

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
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, mock.Anything, creator, "host").Return(errors.New("boom"))

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
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, mock.Anything, creator, "host").Return(nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, creator, memberA).Return(true, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, creator, memberB).Return(false, nil)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, mock.Anything, memberB, "member").Return(nil)
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
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, mock.Anything, creator, "host").Return(nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, creator, memberA).Return(false, nil)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, mock.Anything, memberA, "member").Return(errors.New("boom"))

	// when
	_, err := svc.CreateGroupRoom(context.Background(), creator, req)

	// then
	require.Error(t, err)
}

// --- ListPublicRooms ---

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

// --- ListUserGroupRooms ---

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

// --- SetRoomMuted ---

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

// --- IsRoomMuted ---

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

// --- JoinRoom ---

func TestJoinRoom_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(nil, errors.New("boom"))

	// when
	_, err := svc.JoinRoom(context.Background(), roomID, userID)

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
	_, err := svc.JoinRoom(context.Background(), roomID, userID)

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
	_, err := svc.JoinRoom(context.Background(), roomID, userID)

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
	_, err := svc.JoinRoom(context.Background(), roomID, userID)

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
	_, err := svc.JoinRoom(context.Background(), roomID, userID)

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
	got, err := svc.JoinRoom(context.Background(), roomID, userID)

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
	_, err := svc.JoinRoom(context.Background(), roomID, userID)

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
	_, err := svc.JoinRoom(context.Background(), roomID, userID)

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
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, roomID, userID, "member").Return(errors.New("boom"))

	// when
	_, err := svc.JoinRoom(context.Background(), roomID, userID)

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
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, roomID, userID, "member").Return(nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(row, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{userID}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)

	// when
	got, err := svc.JoinRoom(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
}

// --- LeaveRoom ---

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
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(sampleUser(userID), nil)

	// when
	err := svc.LeaveRoom(context.Background(), roomID, userID)

	// then
	require.NoError(t, err)
}

// --- KickMember ---

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
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{hostID, targetID}, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, roomID, targetID).Return(nil)

	// when
	err := svc.KickMember(context.Background(), hostID, roomID, targetID)

	// then
	require.NoError(t, err)
}

// --- GetMembers ---

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

	// when
	got, err := svc.GetMembers(context.Background(), viewerID, roomID)

	// then
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, memberID, got[0].User.ID)
}

// --- ListRooms ---

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

// --- EnsureSystemRooms ---

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

// --- SyncSystemRoomMembership ---

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
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, modsID, userID, "member").Return(nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, adminsID, userID).Return("", nil)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, adminsID, userID, "member").Return(nil)

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
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, modsID, userID, "host").Return(nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, adminsID, userID).Return("", nil)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, adminsID, userID, "host").Return(nil)

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

// --- GetMessages ---

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

	// when
	got, err := svc.GetMessages(context.Background(), userID, roomID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Messages, 1)
}

// --- GetMessagesBefore ---

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

	// when
	got, err := svc.GetMessagesBefore(context.Background(), userID, roomID, "x", 500)

	// then
	require.NoError(t, err)
	assert.Equal(t, 200, got.Limit)
}

// --- SendMessage ---

func TestSendMessage_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.SendMessage(context.Background(), uuid.New(), uuid.New(), dto.SendMessageRequest{Body: ""})

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
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"})

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
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestSendMessage_MembersError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(true, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, errors.New("boom"))

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"})

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
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, otherID}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, otherID).Return(true, nil)

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, ErrUserBlocked)
}

func TestSendMessage_SenderLookupError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(true, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(nil, errors.New("boom"))

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"})

	// then
	require.Error(t, err)
}

func TestSendMessage_SenderNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(true, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(nil, nil)

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, ErrUserNotFound)
}

func TestSendMessage_InsertError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(true, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(sampleUser(senderID), nil)
	m.chatRepo.EXPECT().InsertMessage(mock.Anything, mock.Anything, roomID, senderID, "hi", (*uuid.UUID)(nil)).Return(errors.New("boom"))

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"})

	// then
	require.Error(t, err)
}

func TestSendMessage_MarkReadError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, senderID).Return(true, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(sampleUser(senderID), nil)
	m.chatRepo.EXPECT().InsertMessage(mock.Anything, mock.Anything, roomID, senderID, "hi", (*uuid.UUID)(nil)).Return(nil)
	m.chatRepo.EXPECT().MarkRoomRead(mock.Anything, roomID, senderID).Return(errors.New("boom"))

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"})

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
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, recipientID}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, recipientID).Return(false, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(sampleUser(senderID), nil)
	m.chatRepo.EXPECT().InsertMessage(mock.Anything, mock.Anything, roomID, senderID, "hi", (*uuid.UUID)(nil)).Return(nil)
	m.chatRepo.EXPECT().MarkRoomRead(mock.Anything, roomID, senderID).Return(nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, senderID).Return(&repository.ChatRoomRow{Type: "dm"}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, recipientID}, nil)
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool { return p.Type == dto.NotifChatMessage })).Return(nil)
	m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, recipientID).Return(1, nil)

	// when
	got, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"})

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
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, mentionedID, replyAuthorID}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, mentionedID).Return(false, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, replyAuthorID).Return(false, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(sampleUser(senderID), nil)
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, replyMsgID).Return(&repository.ChatMessageRow{ID: replyMsgID, RoomID: roomID, SenderID: replyAuthorID, SenderDisplayName: "Parent", Body: "original"}, nil)
	m.chatRepo.EXPECT().InsertMessage(mock.Anything, mock.Anything, roomID, senderID, body, &replyMsgID).Return(nil)
	m.chatRepo.EXPECT().MarkRoomRead(mock.Anything, roomID, senderID).Return(nil)
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
	got, err := svc.SendMessage(context.Background(), senderID, roomID, req)

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
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, otherID}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, otherID).Return(false, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(sampleUser(senderID), nil)
	m.chatRepo.EXPECT().InsertMessage(mock.Anything, mock.Anything, roomID, senderID, "hi", (*uuid.UUID)(nil)).Return(nil)
	m.chatRepo.EXPECT().MarkRoomRead(mock.Anything, roomID, senderID).Return(nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, senderID).Return(&repository.ChatRoomRow{Type: "group", Name: "G"}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, otherID}, nil)
	m.chatRepo.EXPECT().IsMuted(mock.Anything, roomID, otherID).Return(false, nil)
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool { return p.Type == dto.NotifChatRoomMessage })).Return(nil)
	m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, otherID).Return(1, nil)

	// when
	got, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"})

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
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, otherID}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, otherID).Return(false, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(sampleUser(senderID), nil)
	m.chatRepo.EXPECT().InsertMessage(mock.Anything, mock.Anything, roomID, senderID, "hi", (*uuid.UUID)(nil)).Return(nil)
	m.chatRepo.EXPECT().MarkRoomRead(mock.Anything, roomID, senderID).Return(nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, senderID).Return(&repository.ChatRoomRow{Type: "group", Name: "G"}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, otherID}, nil)
	m.chatRepo.EXPECT().IsMuted(mock.Anything, roomID, otherID).Return(true, nil)
	m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, otherID).Return(1, nil)

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"})

	// then
	require.NoError(t, err)
}

// --- GetRoomsByUser ---

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

// --- DeleteChat ---

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

// --- GetUnreadCount ---

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

// --- MarkRead ---

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

// --- Pure helpers ---

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
	replySenderID := uuid.New()
	replySenderName := "Sender"
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
		ReplyToSenderID:   &replySenderID,
		ReplyToSenderName: &replySenderName,
		ReplyToBody:       &longBody,
	}

	// when
	got := messageRowToResponse(row, nil)

	// then
	require.NotNil(t, got.ReplyTo)
	assert.Equal(t, replyID, got.ReplyTo.ID)
	assert.Equal(t, 143, len(got.ReplyTo.BodyPreview))
}

func TestMessageRowToResponse_NoReply(t *testing.T) {
	// given
	row := repository.ChatMessageRow{ID: uuid.New(), RoomID: uuid.New(), SenderID: uuid.New(), Body: "hi"}

	// when
	got := messageRowToResponse(row, nil)

	// then
	assert.Nil(t, got.ReplyTo)
	assert.Equal(t, "hi", got.Body)
}
