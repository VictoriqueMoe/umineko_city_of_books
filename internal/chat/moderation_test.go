package chat

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func stubRoom(id uuid.UUID) *repository.ChatRoomRow {
	return &repository.ChatRoomRow{
		ID:       id,
		Type:     "group",
		IsPublic: true,
	}
}

func TestBanMember_RejectsStaffTarget(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	room := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, room, actor).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, room, actor).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, room, target).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, target).Return(authz.RoleModerator, nil)

	err := svc.BanMember(context.Background(), actor, room, target, "spam")
	require.ErrorIs(t, err, ErrCannotBanStaff)
}

func TestBanMember_Succeeds(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	room := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, room, actor).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, room, actor).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, room, target).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, target).Return(role.Role(""), nil)

	stubSystemMessage(m, actor, target)

	m.banRepo.EXPECT().Ban(mock.Anything, room, target, &actor, "spam").Return(nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, room).Return(nil, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, room, target).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "chat_room_ban", "chat_room", room.String(), mock.Anything).Return(nil)

	err := svc.BanMember(context.Background(), actor, room, target, "spam")
	require.NoError(t, err)
}

func TestUnbanMember_AuditsOnSuccess(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	room := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, room, actor).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, room, actor).Return("host", nil)
	m.banRepo.EXPECT().Unban(mock.Anything, room, target).Return(nil)
	stubSystemMessage(m, actor, target)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "chat_room_unban", "chat_room", room.String(), mock.Anything).Return(nil)

	err := svc.UnbanMember(context.Background(), actor, room, target)
	require.NoError(t, err)
}

func stubSystemMessage(m *testMocks, actor, target uuid.UUID) {
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(nil, nil).Maybe()
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.Anything, mock.Anything, actor, mock.Anything).Return(errors.New("skip")).Maybe()
}

func TestBanMember_PostsSystemMessageWithReason(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	room := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, room, actor).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, room, actor).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, room, target).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, target).Return(role.Role(""), nil)

	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{
		ID: target, Username: "bar", DisplayName: "Bar",
	}, nil)
	m.chatRepo.EXPECT().
		InsertSystemMessage(mock.Anything, mock.Anything, room, actor, "Bar was banned because spamming links.").
		Return(errors.New("skip"))

	m.banRepo.EXPECT().Ban(mock.Anything, room, target, &actor, "spamming links").Return(nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, room).Return(nil, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, room, target).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "chat_room_ban", "chat_room", room.String(), mock.Anything).Return(nil)

	err := svc.BanMember(context.Background(), actor, room, target, "spamming links")
	require.NoError(t, err)
}

func TestBanMember_PostsSystemMessageWithoutReason(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	room := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, room, actor).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, room, actor).Return("host", nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, room, target).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, target).Return(role.Role(""), nil)

	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{
		ID: target, Username: "bar", DisplayName: "Bar",
	}, nil)
	m.chatRepo.EXPECT().
		InsertSystemMessage(mock.Anything, mock.Anything, room, actor, "Bar was banned.").
		Return(errors.New("skip"))

	m.banRepo.EXPECT().Ban(mock.Anything, room, target, &actor, "").Return(nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, room).Return(nil, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, room, target).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "chat_room_ban", "chat_room", room.String(), mock.Anything).Return(nil)

	err := svc.BanMember(context.Background(), actor, room, target, "")
	require.NoError(t, err)
}

func TestUnbanMember_PostsSystemMessage(t *testing.T) {
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	room := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, room, actor).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, room, actor).Return("host", nil)
	m.banRepo.EXPECT().Unban(mock.Anything, room, target).Return(nil)

	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{
		ID: target, Username: "bar", DisplayName: "Bar",
	}, nil)
	m.chatRepo.EXPECT().
		InsertSystemMessage(mock.Anything, mock.Anything, room, actor, "Bar was unbanned.").
		Return(errors.New("skip"))

	m.auditRepo.EXPECT().Create(mock.Anything, actor, "chat_room_unban", "chat_room", room.String(), mock.Anything).Return(nil)

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

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, room, actor).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, room, actor).Return("host", nil)
	m.bannedWordRepo.EXPECT().GetByID(mock.Anything, ruleID).Return(&repository.ChatBannedWordRow{
		ID: ruleID, Scope: "room", RoomID: &room, Pattern: "old",
	}, nil).Once()
	m.bannedWordRepo.EXPECT().Update(mock.Anything, ruleID, repository.ChatBannedWordUpdate{
		Pattern:       "new",
		MatchMode:     contentfilter.MatchModeWholeWord,
		CaseSensitive: true,
		Action:        contentfilter.BannedWordActionKick,
	}).Return(nil)
	m.bannedWordRepo.EXPECT().GetByID(mock.Anything, ruleID).Return(&repository.ChatBannedWordRow{
		ID:            ruleID,
		Scope:         "room",
		RoomID:        &room,
		Pattern:       "new",
		MatchMode:     contentfilter.MatchModeWholeWord,
		CaseSensitive: true,
		Action:        contentfilter.BannedWordActionKick,
	}, nil).Once()
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "chat_room_banned_word_update", "chat_room", room.String(), mock.Anything).Return(nil)

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
	other := uuid.New()
	ruleID := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, room, actor).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, room, actor).Return("host", nil)
	m.bannedWordRepo.EXPECT().GetByID(mock.Anything, ruleID).Return(&repository.ChatBannedWordRow{
		ID: ruleID, Scope: "room", RoomID: &other,
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
	room := uuid.New()
	ruleID := uuid.New()

	m.authzSvc.EXPECT().Can(mock.Anything, actor, authz.PermManageBannedWords).Return(true)
	m.bannedWordRepo.EXPECT().GetByID(mock.Anything, ruleID).Return(&repository.ChatBannedWordRow{
		ID: ruleID, Scope: "room", RoomID: &room,
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
	m.bannedWordRepo.EXPECT().GetByID(mock.Anything, ruleID).Return(&repository.ChatBannedWordRow{
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
	other := uuid.New()
	ruleID := uuid.New()

	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, room, actor).Return(stubRoom(room), nil)
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, room, actor).Return("host", nil)
	m.bannedWordRepo.EXPECT().GetByID(mock.Anything, ruleID).Return(&repository.ChatBannedWordRow{
		ID: ruleID, Scope: "room", RoomID: &other,
	}, nil)

	err := svc.DeleteRoomBannedWord(context.Background(), actor, room, ruleID)
	require.ErrorIs(t, err, ErrBannedWordRuleMismatch)
}

func TestSendMessage_BannedWordKickFires(t *testing.T) {
	svc, m := newTestService(t)
	sender := uuid.New()
	room := uuid.New()

	m.chatRepo.EXPECT().IsMember(mock.Anything, room, sender).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, room, sender).Return(false, "", false, nil)
	m.bannedWordRepo.EXPECT().ListApplicable(mock.Anything, room).Return([]repository.ChatBannedWordRow{
		{ID: uuid.New(), Pattern: "dogs", MatchMode: contentfilter.MatchModeSubstring,
			Action: contentfilter.BannedWordActionKick, Scope: "room", RoomID: &room},
	}, nil).Maybe()
	m.chatRepo.EXPECT().GetMemberRole(mock.Anything, room, sender).Return("member", nil)
	m.authzSvc.EXPECT().GetRole(mock.Anything, sender).Return(role.Role(""), nil)
	m.auditRepo.EXPECT().CreateSystem(mock.Anything, "chat_word_filter_kick", "chat_room", room.String(), mock.Anything).Return(nil)
	stubSystemMessage(m, sender, sender)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, room).Return(nil, nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, room, sender).Return(nil)

	_, err := svc.SendMessage(context.Background(), sender, room, dto.SendMessageRequest{Body: "I love dogs"}, nil)
	require.Error(t, err)
	var bw *ErrBannedWordMatch
	assert.True(t, errors.As(err, &bw))
	assert.Equal(t, "dogs", bw.Pattern)
	assert.Equal(t, contentfilter.BannedWordActionKick, bw.Action)
}
