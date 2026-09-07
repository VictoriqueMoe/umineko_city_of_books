package chat

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func stubRoom(id uuid.UUID) *model.ChatRoomRow {
	return &model.ChatRoomRow{
		ID:       id,
		Type:     "group",
		IsPublic: true,
	}
}

func expectLiveKitUnconfigured(m *testMocks) {
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingLiveKitURL).Return("").Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingLiveKitAPIKey).Return("").Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingLiveKitAPISecret).Return("").Maybe()
}

func expectEvictionSideEffects(m *testMocks, roomID uuid.UUID) {
	m.watchPartyRepo.EXPECT().ListActiveByRoom(mock.Anything, roomID).Return(nil, nil)
	expectLiveKitUnconfigured(m)
}

func TestBanMember_RejectsStaffTarget(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	room := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: room, ViewerID: actor}).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: actor}).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: target}).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, target).Return(authz.RoleModerator, nil)

	err := svc.BanMember(context.Background(), actor, room, target, "spam")
	require.ErrorIs(t, err, ErrCannotBanStaff)
}

func TestBanMember_Succeeds(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	room := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: room, ViewerID: actor}).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: actor}).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: target}).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, target).Return("", nil)

	stubSystemMessage(m, target, spec.NewChatMessage{RoomID: room, SenderID: actor, Body: "A user was banned because spam."})

	m.banRepo.EXPECT().Ban(mock.Anything, spec.NewChatRoomBan{RoomID: room, UserID: target, BannedBy: &actor, Reason: "spam"}).Return(nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, room).Return(nil, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: target}).Return(nil)
	expectEvictionSideEffects(m, room)
	m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry audit.NewEntry) bool {
		return entry.ActorID == actor && entry.Action == audit.ActionChatRoomBan && entry.TargetType == audit.TargetChatRoom && entry.TargetID == room.String() && entry.SubjectID == target
	})).Return(nil)

	err := svc.BanMember(context.Background(), actor, room, target, "spam")
	require.NoError(t, err)
}

func TestUnbanMember_AuditsOnSuccess(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	room := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: room, ViewerID: actor}).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: actor}).Return("host", nil)
	m.banRepo.EXPECT().UnbanWithAudit(mock.Anything, spec.ChatRoomUnban{
		ChatMemberRef: spec.ChatMemberRef{RoomID: room, UserID: target},
		ActorID:       actor,
	}).Return(nil)
	stubSystemMessage(m, target, spec.NewChatMessage{RoomID: room, SenderID: actor, Body: "A user was unbanned."})

	err := svc.UnbanMember(context.Background(), actor, room, target)
	require.NoError(t, err)
}

func stubSystemMessage(m *testMocks, target uuid.UUID, message spec.NewChatMessage) {
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(nil, nil).Maybe()
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, message).Return(nil, errors.New("skip")).Maybe()
}

func TestBanMember_PostsSystemMessageWithReason(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	room := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: room, ViewerID: actor}).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: actor}).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: target}).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, target).Return("", nil)

	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{
		ID: target, Username: "bar", DisplayName: "Bar",
	}, nil)
	m.chatRepo.EXPECT().
		InsertSystemMessage(mock.Anything, spec.NewChatMessage{RoomID: room, SenderID: actor, Body: "Bar was banned because spamming links."}).
		Return(nil, errors.New("skip"))

	m.banRepo.EXPECT().Ban(mock.Anything, spec.NewChatRoomBan{RoomID: room, UserID: target, BannedBy: &actor, Reason: "spamming links"}).Return(nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, room).Return(nil, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: target}).Return(nil)
	expectEvictionSideEffects(m, room)
	m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry audit.NewEntry) bool {
		return entry.ActorID == actor && entry.Action == audit.ActionChatRoomBan && entry.TargetType == audit.TargetChatRoom && entry.TargetID == room.String() && entry.SubjectID == target
	})).Return(nil)

	err := svc.BanMember(context.Background(), actor, room, target, "spamming links")
	require.NoError(t, err)
}

func TestBanMember_PostsSystemMessageWithoutReason(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	room := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: room, ViewerID: actor}).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: actor}).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: target}).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, target).Return("", nil)

	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{
		ID: target, Username: "bar", DisplayName: "Bar",
	}, nil)
	m.chatRepo.EXPECT().
		InsertSystemMessage(mock.Anything, spec.NewChatMessage{RoomID: room, SenderID: actor, Body: "Bar was banned."}).
		Return(nil, errors.New("skip"))

	m.banRepo.EXPECT().Ban(mock.Anything, spec.NewChatRoomBan{RoomID: room, UserID: target, BannedBy: &actor, Reason: ""}).Return(nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, room).Return(nil, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: target}).Return(nil)
	expectEvictionSideEffects(m, room)
	m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry audit.NewEntry) bool {
		return entry.ActorID == actor && entry.Action == audit.ActionChatRoomBan && entry.TargetType == audit.TargetChatRoom && entry.TargetID == room.String() && entry.SubjectID == target
	})).Return(nil)

	err := svc.BanMember(context.Background(), actor, room, target, "")
	require.NoError(t, err)
}

func TestUnbanMember_PostsSystemMessage(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	room := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: room, ViewerID: actor}).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: actor}).Return("host", nil)
	m.banRepo.EXPECT().UnbanWithAudit(mock.Anything, spec.ChatRoomUnban{
		ChatMemberRef: spec.ChatMemberRef{RoomID: room, UserID: target},
		ActorID:       actor,
	}).Return(nil)

	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{
		ID: target, Username: "bar", DisplayName: "Bar",
	}, nil)
	m.chatRepo.EXPECT().
		InsertSystemMessage(mock.Anything, spec.NewChatMessage{RoomID: room, SenderID: actor, Body: "Bar was unbanned."}).
		Return(nil, errors.New("skip"))

	err := svc.UnbanMember(context.Background(), actor, room, target)
	require.NoError(t, err)
}

func TestCreateGlobalBannedWord_RequiresPermission(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()

	m.authzSvc.EXPECT().Can(mock.Anything, actor, authz.PermManageBannedWords).Return(false)

	_, err := svc.CreateGlobalBannedWord(context.Background(), actor, dto.CreateBannedWordRequest{
		Pattern: "dogs", MatchMode: contentfilter.MatchModeSubstring, Action: contentfilter.BannedWordActionDelete,
	})
	require.ErrorIs(t, err, ErrModRoleRequired)
}

func TestCreateGlobalBannedWord_RejectsInvalidRegex(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()

	m.authzSvc.EXPECT().Can(mock.Anything, actor, authz.PermManageBannedWords).Return(true)

	_, err := svc.CreateGlobalBannedWord(context.Background(), actor, dto.CreateBannedWordRequest{
		Pattern: `\b(foo`, MatchMode: contentfilter.MatchModeRegex, Action: contentfilter.BannedWordActionDelete,
	})
	require.ErrorIs(t, err, ErrInvalidBannedWordRegex)
}

func TestUpdateRoomBannedWord_Succeeds(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	room := uuid.New()
	ruleID := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: room, ViewerID: actor}).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: actor}).Return("host", nil)
	m.bannedWordRepo.EXPECT().GetByID(mock.Anything, ruleID).Return(&model.ChatBannedWordRow{
		ID: ruleID, Scope: "room", RoomID: &room, Pattern: "old",
	}, nil).Once()
	m.bannedWordRepo.EXPECT().Update(mock.Anything, spec.ChatBannedWordUpdate{
		ID:            ruleID,
		Pattern:       "new",
		MatchMode:     contentfilter.MatchModeWholeWord,
		CaseSensitive: true,
		Action:        contentfilter.BannedWordActionKick,
	}).Return(nil)
	m.bannedWordRepo.EXPECT().GetByID(mock.Anything, ruleID).Return(&model.ChatBannedWordRow{
		ID:            ruleID,
		Scope:         "room",
		RoomID:        &room,
		Pattern:       "new",
		MatchMode:     contentfilter.MatchModeWholeWord,
		CaseSensitive: true,
		Action:        contentfilter.BannedWordActionKick,
	}, nil).Once()
	m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry audit.NewEntry) bool {
		return entry.ActorID == actor && entry.Action == audit.ActionChatRoomBannedWordUpdate && entry.TargetType == audit.TargetChatRoom && entry.TargetID == room.String()
	})).Return(nil)

	resp, err := svc.UpdateRoomBannedWord(context.Background(), actor, room, ruleID, dto.UpdateBannedWordRequest{
		Pattern: "new", MatchMode: contentfilter.MatchModeWholeWord, CaseSensitive: true, Action: contentfilter.BannedWordActionKick,
	})
	require.NoError(t, err)
	assert.Equal(t, "new", resp.Pattern)
	assert.Equal(t, contentfilter.BannedWordActionKick, resp.Action)
	assert.True(t, resp.CaseSensitive)
}

func TestUpdateRoomBannedWord_RejectsMismatchedScope(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	room := uuid.New()
	ruleID := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: room, ViewerID: actor}).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: actor}).Return("host", nil)
	m.bannedWordRepo.EXPECT().GetByID(mock.Anything, ruleID).Return(&model.ChatBannedWordRow{
		ID: ruleID, Scope: "room", RoomID: new(uuid.New()),
	}, nil)

	_, err := svc.UpdateRoomBannedWord(context.Background(), actor, room, ruleID, dto.UpdateBannedWordRequest{
		Pattern: "x", MatchMode: contentfilter.MatchModeSubstring, Action: contentfilter.BannedWordActionDelete,
	})
	require.ErrorIs(t, err, ErrBannedWordRuleMismatch)
}

func TestUpdateGlobalBannedWord_RequiresPermission(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	ruleID := uuid.New()

	m.authzSvc.EXPECT().Can(mock.Anything, actor, authz.PermManageBannedWords).Return(false)

	_, err := svc.UpdateGlobalBannedWord(context.Background(), actor, ruleID, dto.UpdateBannedWordRequest{
		Pattern: "x", MatchMode: contentfilter.MatchModeSubstring, Action: contentfilter.BannedWordActionDelete,
	})
	require.ErrorIs(t, err, ErrModRoleRequired)
}

func TestUpdateGlobalBannedWord_RejectsRoomScopeRule(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	ruleID := uuid.New()

	m.authzSvc.EXPECT().Can(mock.Anything, actor, authz.PermManageBannedWords).Return(true)
	m.bannedWordRepo.EXPECT().GetByID(mock.Anything, ruleID).Return(&model.ChatBannedWordRow{
		ID: ruleID, Scope: "room", RoomID: new(uuid.New()),
	}, nil)

	_, err := svc.UpdateGlobalBannedWord(context.Background(), actor, ruleID, dto.UpdateBannedWordRequest{
		Pattern: "x", MatchMode: contentfilter.MatchModeSubstring, Action: contentfilter.BannedWordActionDelete,
	})
	require.ErrorIs(t, err, ErrBannedWordRuleMismatch)
}

func TestUpdateGlobalBannedWord_RejectsInvalidRegex(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	ruleID := uuid.New()

	m.authzSvc.EXPECT().Can(mock.Anything, actor, authz.PermManageBannedWords).Return(true)
	m.bannedWordRepo.EXPECT().GetByID(mock.Anything, ruleID).Return(&model.ChatBannedWordRow{
		ID: ruleID, Scope: "global",
	}, nil)

	_, err := svc.UpdateGlobalBannedWord(context.Background(), actor, ruleID, dto.UpdateBannedWordRequest{
		Pattern: `\b(foo`, MatchMode: contentfilter.MatchModeRegex, Action: contentfilter.BannedWordActionDelete,
	})
	require.ErrorIs(t, err, ErrInvalidBannedWordRegex)
}

func TestDeleteRoomBannedWord_RejectsMismatchedScope(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	room := uuid.New()
	ruleID := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: room, ViewerID: actor}).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: actor}).Return("host", nil)
	m.bannedWordRepo.EXPECT().GetByID(mock.Anything, ruleID).Return(&model.ChatBannedWordRow{
		ID: ruleID, Scope: "room", RoomID: new(uuid.New()),
	}, nil)

	err := svc.DeleteRoomBannedWord(context.Background(), actor, room, ruleID)
	require.ErrorIs(t, err, ErrBannedWordRuleMismatch)
}

func TestCreateRoomBannedWord_RejectsSystemRoom(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	room := uuid.New()

	systemRoom := stubRoom(room)
	systemRoom.IsSystem = true
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: room, ViewerID: actor}).Return(systemRoom, nil)

	_, err := svc.CreateRoomBannedWord(context.Background(), actor, room, dto.CreateBannedWordRequest{
		Pattern: "dogs", MatchMode: contentfilter.MatchModeSubstring, Action: contentfilter.BannedWordActionKick,
	})
	require.ErrorIs(t, err, ErrSystemRoom)
}

func TestEditMessage_BannedWordBlocked(t *testing.T) {
	svc, m := newTestService(t)
	author := uuid.New()
	room := uuid.New()
	messageID := uuid.New()

	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&model.ChatMessageRow{
		ID: messageID, RoomID: room, SenderID: author, Body: "ok",
	}, nil)
	m.chatRepo.EXPECT().IsMember(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: author}).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: author}).Return(false, "", false, nil)
	m.bannedWordRepo.EXPECT().ListApplicable(mock.Anything, room).Return([]model.ChatBannedWordRow{
		{ID: uuid.New(), Pattern: "dogs", MatchMode: contentfilter.MatchModeSubstring,
			Action: contentfilter.BannedWordActionDelete, Scope: "room", RoomID: &room},
	}, nil).Maybe()
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: author}).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, author).Return("", nil)
	m.auditRepo.EXPECT().CreateSystem(mock.Anything, mock.MatchedBy(func(entry audit.NewEntry) bool {
		return entry.Action == audit.ActionChatWordFilterDelete && entry.TargetType == audit.TargetChatRoom && entry.TargetID == room.String() && entry.SubjectID == author
	})).Return(nil)

	resp, err := svc.EditMessage(context.Background(), messageID, author, "I love dogs")
	require.Error(t, err)
	assert.Nil(t, resp)
	var bw *ErrBannedWordMatch
	assert.True(t, errors.As(err, &bw))
	assert.Equal(t, "dogs", bw.Pattern)
}

func TestSendMessage_BannedWordKickFires(t *testing.T) {
	svc, m := newTestService(t)
	sender := uuid.New()
	room := uuid.New()

	m.chatRepo.EXPECT().IsMember(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: sender}).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: sender}).Return(false, "", false, nil)
	m.bannedWordRepo.EXPECT().ListApplicable(mock.Anything, room).Return([]model.ChatBannedWordRow{
		{ID: uuid.New(), Pattern: "dogs", MatchMode: contentfilter.MatchModeSubstring,
			Action: contentfilter.BannedWordActionKick, Scope: "room", RoomID: &room},
	}, nil).Maybe()
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: sender}).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, sender).Return("", nil)
	m.auditRepo.EXPECT().CreateSystem(mock.Anything, mock.MatchedBy(func(entry audit.NewEntry) bool {
		return entry.Action == audit.ActionChatWordFilterKick && entry.TargetType == audit.TargetChatRoom && entry.TargetID == room.String() && entry.SubjectID == sender
	})).Return(nil)
	expectRoomKind(m, room, dto.RoomTypeGroup)
	stubSystemMessage(m, sender, spec.NewChatMessage{RoomID: room, SenderID: sender, Body: "A user was kicked by the word filter."})
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, room).Return(nil, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: sender}).Return(nil)
	expectEvictionSideEffects(m, room)

	_, err := svc.SendMessage(context.Background(), sender, room, dto.SendMessageRequest{Body: "I love dogs"}, nil)
	require.Error(t, err)
	var bw *ErrBannedWordMatch
	assert.True(t, errors.As(err, &bw))
	assert.Equal(t, "dogs", bw.Pattern)
	assert.Equal(t, contentfilter.BannedWordActionKick, bw.Action)
}

func TestSendMessage_BannedWordKickIsDeleteOnlyInsideAPair(t *testing.T) {
	cases := []struct {
		name     string
		roomType dto.RoomType
		wantKick bool
	}{
		{
			name:     "a room evicts the sender the filter caught",
			roomType: dto.RoomTypeGroup,
			wantKick: true,
		},
		{
			name:     "a pair only refuses the message, because nobody is thrown out of their own conversation",
			roomType: dto.RoomTypeDM,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a kick-action banned word in a room of this kind
			svc, m := newTestService(t)
			sender := uuid.New()
			room := uuid.New()

			m.chatRepo.EXPECT().IsMember(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: sender}).Return(true, nil)
			m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: sender}).Return(false, "", false, nil)
			m.bannedWordRepo.EXPECT().ListApplicable(mock.Anything, room).Return([]model.ChatBannedWordRow{
				{ID: uuid.New(), Pattern: "dogs", MatchMode: contentfilter.MatchModeSubstring,
					Action: contentfilter.BannedWordActionKick, Scope: "room", RoomID: &room},
			}, nil).Maybe()
			m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: sender}).Return("member", nil)
			m.authzSvc.EXPECT().GetRole(mock.Anything, sender).Return("", nil)
			m.auditRepo.EXPECT().CreateSystem(mock.Anything, mock.Anything).Return(nil)
			m.userRepo.EXPECT().GetByID(mock.Anything, sender).Return(sampleUser(sender), nil)
			expectRoomKind(m, room, tc.roomType)

			if tc.wantKick {
				m.chatRepo.EXPECT().GetMemberNickname(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: sender}).Return("", nil).Maybe()
				m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, spec.NewChatMessage{RoomID: room, SenderID: sender, Body: "User was kicked by the word filter."}).Return(nil, errors.New("skip"))
				m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, room).Return(nil, nil)
				m.chatRepo.EXPECT().RemoveMember(mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: sender}).Return(nil)
				expectEvictionSideEffects(m, room)
			}

			// when the message is sent
			_, err := svc.SendMessage(context.Background(), sender, room, dto.SendMessageRequest{Body: "I love dogs"}, nil)

			// then the message is refused either way and only a room evicts
			require.Error(t, err)
			var bw *ErrBannedWordMatch
			require.True(t, errors.As(err, &bw))
			assert.Equal(t, contentfilter.BannedWordActionKick, bw.Action)

			if tc.wantKick {
				return
			}

			m.chatRepo.AssertNotCalled(t, "RemoveMember", mock.Anything, spec.ChatMemberRef{RoomID: room, UserID: sender})
			m.chatRepo.AssertNotCalled(t, "InsertSystemMessage", mock.Anything, mock.Anything)
		})
	}
}
