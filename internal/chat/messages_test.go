package chat

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	messagesGuardCase struct {
		name      string
		emptyBody bool
		setup     func(m *testMocks, ref spec.ChatMemberRef)
		wantErr   error
		wantText  string
	}
)

func messagesMembershipGuardCases(boom error) []messagesGuardCase {
	return []messagesGuardCase{
		{
			name: "a failed membership lookup is returned",
			setup: func(m *testMocks, ref spec.ChatMemberRef) {
				m.chatRepo.EXPECT().IsMember(mock.Anything, ref).Return(false, boom)
			},
			wantErr: boom,
		},
		{
			name: "a non-member is refused",
			setup: func(m *testMocks, ref spec.ChatMemberRef) {
				m.chatRepo.EXPECT().IsMember(mock.Anything, ref).Return(false, nil)
			},
			wantErr: ErrNotMember,
		},
	}
}

func messagesExpectSenderMaySpeak(m *testMocks, ref spec.ChatMemberRef) {
	m.chatRepo.EXPECT().IsMember(mock.Anything, ref).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, ref).Return(false, "", false, nil)
	m.bannedWordRepo.EXPECT().ListApplicable(mock.Anything, ref.RoomID).Return(nil, nil).Maybe()
}

func messagesExpectRoom(m *testMocks, ref spec.ChatMemberRef, members []uuid.UUID, room *model.ChatRoomSendContext) {
	messagesExpectSenderMaySpeak(m, ref)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, ref.RoomID).Return(members, nil)
	m.chatRepo.EXPECT().GetRoomSendContext(mock.Anything, ref.RoomID).Return(room, nil)
}

func messagesExpectInsert(m *testMocks, message spec.NewChatMessage, detailed []model.ChatRoomMemberRow) uuid.UUID {
	msgID := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, message.SenderID).Return(sampleUser(message.SenderID), nil)
	m.chatRepo.EXPECT().InsertMessageAndMarkRead(mock.Anything, message).Return(&model.ChatMessageRow{ID: msgID}, nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, message.RoomID).Return(detailed, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUser(mock.Anything, message.SenderID).Return(nil, nil)

	return msgID
}

func messagesExpectHydration(m *testMocks, viewerID uuid.UUID, messageIDs []uuid.UUID) {
	m.chatRepo.EXPECT().GetMessageMediaBatch(mock.Anything, messageIDs).Return(nil, nil)
	m.chatRepo.EXPECT().GetReactionsBatch(mock.Anything, spec.ChatReactionsQuery{MessageIDs: messageIDs, ViewerID: viewerID}).Return(nil, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, mock.Anything).Return(nil, nil)
}

func messagesStoredMessage(messageID, roomID, senderID uuid.UUID) *model.ChatMessageRow {
	return &model.ChatMessageRow{ID: messageID, RoomID: roomID, SenderID: senderID, Body: "old"}
}

func TestGetMessages_Errors(t *testing.T) {
	boom := errors.New("boom")
	cases := append(messagesMembershipGuardCases(boom), messagesGuardCase{
		name: "a failed page load is returned",
		setup: func(m *testMocks, ref spec.ChatMemberRef) {
			m.chatRepo.EXPECT().IsMember(mock.Anything, ref).Return(true, nil)
			m.chatRepo.EXPECT().GetMessagesForViewer(mock.Anything, spec.ChatMessagePage{RoomID: ref.RoomID, ViewerID: ref.UserID, Limit: 10, Offset: 0}).Return(nil, 0, boom)
		},
		wantErr: boom,
	})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			ref := spec.ChatMemberRef{RoomID: uuid.New(), UserID: uuid.New()}
			tc.setup(m, ref)

			// when
			_, err := svc.GetMessages(context.Background(), ref.UserID, ref.RoomID, 10, 0)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestGetMessages_SurfacesAFailedHydration(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name        string
		mediaErr    error
		reactionErr error
		vanityErr   error
	}{
		{name: "the media", mediaErr: boom},
		{name: "the reactions", reactionErr: boom},
		{name: "the senders' vanity roles", vanityErr: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name+" failing is surfaced, not rendered as missing", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			ref := spec.ChatMemberRef{RoomID: uuid.New(), UserID: uuid.New()}
			msgID := uuid.New()
			m.chatRepo.EXPECT().IsMember(mock.Anything, ref).Return(true, nil)
			m.chatRepo.EXPECT().GetMessagesForViewer(mock.Anything, mock.Anything).Return([]model.ChatMessageRow{{ID: msgID, RoomID: ref.RoomID}}, 1, nil)
			m.chatRepo.EXPECT().GetMessageMediaBatch(mock.Anything, []uuid.UUID{msgID}).Return(nil, tc.mediaErr)
			if tc.mediaErr == nil {
				m.chatRepo.EXPECT().GetReactionsBatch(mock.Anything, mock.Anything).Return(nil, tc.reactionErr)
			}
			if tc.mediaErr == nil && tc.reactionErr == nil {
				m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, mock.Anything).Return(nil, tc.vanityErr)
			}

			// when
			got, err := svc.GetMessages(context.Background(), ref.UserID, ref.RoomID, 10, 0)

			// then
			require.ErrorIs(t, err, boom)
			assert.Nil(t, got)
		})
	}
}

func TestGetMessages_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	ref := spec.ChatMemberRef{RoomID: uuid.New(), UserID: uuid.New()}
	msgID := uuid.New()
	m.chatRepo.EXPECT().IsMember(mock.Anything, ref).Return(true, nil)
	m.chatRepo.EXPECT().GetMessagesForViewer(mock.Anything, spec.ChatMessagePage{RoomID: ref.RoomID, ViewerID: ref.UserID, Limit: 10, Offset: 0}).
		Return([]model.ChatMessageRow{{ID: msgID, RoomID: ref.RoomID, Body: "hi"}}, 1, nil)
	messagesExpectHydration(m, ref.UserID, []uuid.UUID{msgID})

	// when
	got, err := svc.GetMessages(context.Background(), ref.UserID, ref.RoomID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Messages, 1)
}

func TestGetMessagesBefore_Errors(t *testing.T) {
	boom := errors.New("boom")
	cases := append(messagesMembershipGuardCases(boom), messagesGuardCase{
		name: "a failed page load is returned",
		setup: func(m *testMocks, ref spec.ChatMemberRef) {
			m.chatRepo.EXPECT().IsMember(mock.Anything, ref).Return(true, nil)
			m.chatRepo.EXPECT().GetMessagesBefore(mock.Anything, spec.ChatMessageCursorPage{RoomID: ref.RoomID, ViewerID: ref.UserID, Before: "x", Limit: 50}).Return(nil, boom)
		},
		wantErr: boom,
	})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			ref := spec.ChatMemberRef{RoomID: uuid.New(), UserID: uuid.New()}
			tc.setup(m, ref)

			// when
			_, err := svc.GetMessagesBefore(context.Background(), ref.UserID, ref.RoomID, "x", 50)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestGetMessagesBefore_BoundsTheLimit(t *testing.T) {
	cases := []struct {
		name      string
		limit     int
		wantLimit int
	}{
		{name: "a zero limit falls back to the default", limit: 0, wantLimit: bounds.DefaultLimit},
		{name: "an oversized limit is clamped to the maximum", limit: 500, wantLimit: bounds.MaxLimit},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			ref := spec.ChatMemberRef{RoomID: uuid.New(), UserID: uuid.New()}
			m.chatRepo.EXPECT().IsMember(mock.Anything, ref).Return(true, nil)
			m.chatRepo.EXPECT().GetMessagesBefore(mock.Anything, spec.ChatMessageCursorPage{RoomID: ref.RoomID, ViewerID: ref.UserID, Before: "x", Limit: tc.wantLimit}).Return(nil, nil)
			messagesExpectHydration(m, ref.UserID, []uuid.UUID{})

			// when
			got, err := svc.GetMessagesBefore(context.Background(), ref.UserID, ref.RoomID, "x", tc.limit)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantLimit, got.Limit)
		})
	}
}

func TestSendMessage_Errors(t *testing.T) {
	boom := errors.New("boom")
	cases := append(messagesMembershipGuardCases(boom),
		messagesGuardCase{
			name:      "an empty body is rejected before any lookup",
			emptyBody: true,
			wantErr:   ErrMissingFields,
		},
		messagesGuardCase{
			name: "a timed-out sender is refused and told when the timeout ends",
			setup: func(m *testMocks, ref spec.ChatMemberRef) {
				m.chatRepo.EXPECT().IsMember(mock.Anything, ref).Return(true, nil)
				m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, ref).Return(true, "2099-01-01 00:00:00", false, nil)
			},
			wantErr:  ErrTimedOut,
			wantText: "01 January 2099 00:00 UTC",
		},
		messagesGuardCase{
			name: "a failed member lookup is returned",
			setup: func(m *testMocks, ref spec.ChatMemberRef) {
				messagesExpectSenderMaySpeak(m, ref)
				m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, ref.RoomID).Return(nil, boom)
			},
			wantErr: boom,
		},
		messagesGuardCase{
			name: "a dm is refused when either side has blocked the other",
			setup: func(m *testMocks, ref spec.ChatMemberRef) {
				otherID := uuid.New()
				messagesExpectRoom(m, ref, []uuid.UUID{ref.UserID, otherID}, &model.ChatRoomSendContext{Type: dto.RoomTypeDM})
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, ref.UserID, otherID).Return(true, nil)
			},
			wantErr: ErrUserBlocked,
		},
		messagesGuardCase{
			name: "a failed sender lookup is returned",
			setup: func(m *testMocks, ref spec.ChatMemberRef) {
				messagesExpectRoom(m, ref, []uuid.UUID{ref.UserID}, &model.ChatRoomSendContext{Type: dto.RoomTypeGroup, CreatedBy: ref.UserID})
				m.userRepo.EXPECT().GetByID(mock.Anything, ref.UserID).Return(nil, boom)
			},
			wantErr: boom,
		},
		messagesGuardCase{
			name: "a sender without an account is refused",
			setup: func(m *testMocks, ref spec.ChatMemberRef) {
				messagesExpectRoom(m, ref, []uuid.UUID{ref.UserID}, &model.ChatRoomSendContext{Type: dto.RoomTypeGroup, CreatedBy: ref.UserID})
				m.userRepo.EXPECT().GetByID(mock.Anything, ref.UserID).Return(nil, nil)
			},
			wantErr: ErrUserNotFound,
		},
		messagesGuardCase{
			name: "a failed insert is returned",
			setup: func(m *testMocks, ref spec.ChatMemberRef) {
				messagesExpectRoom(m, ref, []uuid.UUID{ref.UserID}, &model.ChatRoomSendContext{Type: dto.RoomTypeGroup, CreatedBy: ref.UserID})
				m.userRepo.EXPECT().GetByID(mock.Anything, ref.UserID).Return(sampleUser(ref.UserID), nil)
				m.chatRepo.EXPECT().InsertMessageAndMarkRead(mock.Anything, spec.NewChatMessage{RoomID: ref.RoomID, SenderID: ref.UserID, Body: "hi"}).Return(nil, boom)
			},
			wantErr: boom,
		},
	)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			ref := spec.ChatMemberRef{RoomID: uuid.New(), UserID: uuid.New()}
			if tc.setup != nil {
				tc.setup(m, ref)
			}

			req := dto.SendMessageRequest{Body: "hi"}
			if tc.emptyBody {
				req.Body = ""
			}

			// when
			_, err := svc.SendMessage(context.Background(), ref.UserID, ref.RoomID, req, nil)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			if tc.wantText != "" {
				assert.ErrorContains(t, err, tc.wantText)
			}
		})
	}
}

func TestSendMessage_NotificationLadder(t *testing.T) {
	cases := []struct {
		name        string
		roomType    dto.RoomType
		mention     bool
		reply       bool
		muted       bool
		viewing     bool
		wantNotify  bool
		wantType    dto.NotificationType
		wantMessage string
	}{
		{name: "a dm mention notifies as a mention and pierces the mute", roomType: dto.RoomTypeDM, mention: true, muted: true, wantNotify: true, wantType: dto.NotifChatMention},
		{name: "a dm reply notifies as a reply", roomType: dto.RoomTypeDM, reply: true, wantNotify: true, wantType: dto.NotifChatReply},
		{name: "a muted dm reply notifies nothing", roomType: dto.RoomTypeDM, reply: true, muted: true},
		{name: "a plain dm notifies as a direct message", roomType: dto.RoomTypeDM, wantNotify: true, wantType: dto.NotifChatMessage},
		{name: "a muted plain dm notifies nothing", roomType: dto.RoomTypeDM, muted: true},
		{name: "a dm recipient with the thread open is neither notified nor counted", roomType: dto.RoomTypeDM, viewing: true},
		{name: "a room mention notifies as a mention and pierces the mute", roomType: dto.RoomTypeGroup, mention: true, muted: true, wantNotify: true, wantType: dto.NotifChatMention},
		{name: "a room reply notifies as a reply", roomType: dto.RoomTypeGroup, reply: true, wantNotify: true, wantType: dto.NotifChatReply},
		{name: "a muted room reply notifies nothing", roomType: dto.RoomTypeGroup, reply: true, muted: true},
		{name: "a plain room message notifies as a room message", roomType: dto.RoomTypeGroup, wantNotify: true, wantType: dto.NotifChatRoomMessage, wantMessage: "sent a message in G"},
		{name: "a muted plain room message notifies nothing", roomType: dto.RoomTypeGroup, muted: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			senderID := uuid.New()
			recipientID := uuid.New()
			roomID := uuid.New()
			replyMsgID := uuid.New()

			body := "hi"
			if tc.mention {
				body = "hey @bob"
			}

			req := dto.SendMessageRequest{Body: body}
			if tc.reply {
				req.ReplyToID = &replyMsgID
				m.chatRepo.EXPECT().GetMessageByID(mock.Anything, replyMsgID).
					Return(&model.ChatMessageRow{ID: replyMsgID, RoomID: roomID, SenderID: recipientID, Body: "original"}, nil)
			}
			if tc.mention {
				m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"bob"}).
					Return([]model.User{{ID: recipientID, Username: "bob"}}, nil)
			}
			if tc.roomType == dto.RoomTypeDM || tc.mention {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, recipientID).Return(false, nil)
			}

			messagesExpectRoom(m, spec.ChatMemberRef{RoomID: roomID, UserID: senderID}, []uuid.UUID{senderID, recipientID},
				&model.ChatRoomSendContext{Type: tc.roomType, Name: "G", CreatedBy: senderID, LastMessageAt: ongoingThread()})
			msgID := messagesExpectInsert(m, spec.NewChatMessage{RoomID: roomID, SenderID: senderID, Body: body, ReplyToID: req.ReplyToID}, nil)

			if !tc.viewing {
				m.chatRepo.EXPECT().IsMuted(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: recipientID}).Return(tc.muted, nil)
			}

			counted := tc.roomType == dto.RoomTypeDM && !tc.muted && !tc.viewing
			if counted {
				m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, recipientID).Return(1, nil)
			}

			m.hub.JoinRoom(roomID, recipientID)
			if tc.viewing {
				m.hub.AddViewer(roomID, recipientID)
			}

			// when
			got, err := svc.SendMessage(context.Background(), senderID, roomID, req, nil)
			svc.sideEffectsWG.Wait()

			// then
			require.NoError(t, err)
			assert.Equal(t, body, got.Body)
			assert.Equal(t, senderID, got.Sender.ID)
			if !counted {
				m.chatRepo.AssertNotCalled(t, "CountUnreadRoomsForUser", mock.Anything, recipientID)
			}

			if !tc.wantNotify {
				m.notifSvc.AssertNotCalled(t, "Notify", mock.Anything, mock.Anything)

				return
			}

			m.notifSvc.AssertCalled(t, "Notify", mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
				return p.RecipientID == recipientID &&
					p.ActorID == senderID &&
					p.Type == tc.wantType &&
					p.ReferenceID == roomID &&
					p.ReferenceType == "chat_message:"+msgID.String() &&
					p.Message == tc.wantMessage
			}))
		})
	}
}

func TestSendMessage_AFailedReplyLookupIsReturnedBeforeTheMessageIsSaved(t *testing.T) {
	// given a reply whose parent cannot be read
	svc, m := newTestService(t)
	ref := spec.ChatMemberRef{RoomID: uuid.New(), UserID: uuid.New()}
	replyMsgID := uuid.New()
	boom := errors.New("boom")
	messagesExpectRoom(m, ref, []uuid.UUID{ref.UserID}, &model.ChatRoomSendContext{Type: dto.RoomTypeGroup, CreatedBy: ref.UserID})
	m.userRepo.EXPECT().GetByID(mock.Anything, ref.UserID).Return(sampleUser(ref.UserID), nil)
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, replyMsgID).Return(nil, boom)

	// when
	_, err := svc.SendMessage(context.Background(), ref.UserID, ref.RoomID, dto.SendMessageRequest{Body: "hi", ReplyToID: &replyMsgID}, nil)

	// then the message is refused instead of silently losing its reply
	require.ErrorIs(t, err, boom)
	m.chatRepo.AssertNotCalled(t, "InsertMessageAndMarkRead", mock.Anything, mock.Anything)
}

func TestSendMessage_AFailedLookupNeverPingsAnyone(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name     string
		body     string
		blockErr error
		muteErr  error
		wantType dto.NotificationType
	}{
		{name: "a mention whose block check fails is not delivered as a mention", body: "hey @bob", blockErr: boom, wantType: dto.NotifChatRoomMessage},
		{name: "a recipient whose mute state cannot be read is treated as muted", body: "hi", muteErr: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			senderID := uuid.New()
			recipientID := uuid.New()
			roomID := uuid.New()
			messagesExpectRoom(m, spec.ChatMemberRef{RoomID: roomID, UserID: senderID}, []uuid.UUID{senderID, recipientID},
				&model.ChatRoomSendContext{Type: dto.RoomTypeGroup, Name: "G", CreatedBy: senderID, LastMessageAt: ongoingThread()})
			if tc.blockErr != nil {
				m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"bob"}).Return([]model.User{{ID: recipientID, Username: "bob"}}, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, recipientID).Return(false, tc.blockErr)
			}
			messagesExpectInsert(m, spec.NewChatMessage{RoomID: roomID, SenderID: senderID, Body: tc.body}, nil)
			m.chatRepo.EXPECT().IsMuted(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: recipientID}).Return(false, tc.muteErr)
			m.hub.JoinRoom(roomID, recipientID)

			// when
			_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: tc.body}, nil)
			svc.sideEffectsWG.Wait()

			// then
			require.NoError(t, err)
			m.notifSvc.AssertNotCalled(t, "Notify", mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool { return p.Type == dto.NotifChatMention }))
			if tc.wantType == "" {
				m.notifSvc.AssertNotCalled(t, "Notify", mock.Anything, mock.Anything)

				return
			}

			m.notifSvc.AssertCalled(t, "Notify", mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool { return p.Type == tc.wantType && p.RecipientID == recipientID }))
		})
	}
}

func TestSendMessage_LiveStreamRoomSkipsNotifications(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	roomID := uuid.New()
	streamerID := uuid.New()
	messagesExpectRoom(m, spec.ChatMemberRef{RoomID: roomID, UserID: senderID}, []uuid.UUID{senderID, streamerID}, &model.ChatRoomSendContext{
		Type: dto.RoomTypeGroup, IsSystem: true, SystemKind: SystemKindLiveStream, CreatedBy: streamerID,
	})
	m.blockSvc.EXPECT().IsBlocked(mock.Anything, streamerID, senderID).Return(false, nil)
	messagesExpectInsert(m, spec.NewChatMessage{RoomID: roomID, SenderID: senderID, Body: "hi"}, nil)

	// when
	got, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"}, nil)
	svc.sideEffectsWG.Wait()

	// then
	require.NoError(t, err)
	assert.Equal(t, "hi", got.Body)
	m.notifSvc.AssertNotCalled(t, "Notify", mock.Anything, mock.Anything)
	m.chatRepo.AssertNotCalled(t, "CountUnreadRoomsForUser", mock.Anything, mock.Anything)
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
	messagesExpectRoom(m, spec.ChatMemberRef{RoomID: roomID, UserID: senderID}, []uuid.UUID{senderID, mentionedID, replyAuthorID},
		&model.ChatRoomSendContext{Type: dto.RoomTypeGroup, Name: "G", CreatedBy: senderID})
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, mentionedID).Return(false, nil)
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, replyMsgID).Return(&model.ChatMessageRow{ID: replyMsgID, RoomID: roomID, SenderID: replyAuthorID, SenderDisplayName: "Parent", Body: "original"}, nil)
	m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"bob"}).Return([]model.User{{ID: mentionedID, Username: "bob"}}, nil)
	messagesExpectInsert(m, spec.NewChatMessage{RoomID: roomID, SenderID: senderID, Body: body, ReplyToID: &replyMsgID}, nil)
	m.chatRepo.EXPECT().IsMuted(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: mentionedID}).Return(false, nil)
	m.chatRepo.EXPECT().IsMuted(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: replyAuthorID}).Return(false, nil)
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool { return p.Type == dto.NotifChatMention && p.RecipientID == mentionedID })).Return(nil)
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool { return p.Type == dto.NotifChatReply && p.RecipientID == replyAuthorID })).Return(nil)

	// when
	got, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: body, ReplyToID: &replyMsgID}, nil)
	svc.sideEffectsWG.Wait()

	// then
	require.NoError(t, err)
	require.NotNil(t, got.ReplyTo)
	assert.Equal(t, replyMsgID, got.ReplyTo.ID)
}

func TestSendMessage_BotTriggerCarriesTheRoomAlias(t *testing.T) {
	cases := []struct {
		name     string
		nickname string
		want     string
	}{
		{"the room alias is used when the sender has one", "Feather", "Feather"},
		{"the display name is used when there is no alias", "", "User"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			obs := &captureMessageObserver{}
			svc.SetMessageObserver(obs)

			senderID := uuid.New()
			roomID := uuid.New()
			messagesExpectRoom(m, spec.ChatMemberRef{RoomID: roomID, UserID: senderID}, []uuid.UUID{senderID},
				&model.ChatRoomSendContext{Type: dto.RoomTypeGroup, Name: "G", CreatedBy: senderID})
			messagesExpectInsert(m, spec.NewChatMessage{RoomID: roomID, SenderID: senderID, Body: "who am i?"},
				[]model.ChatRoomMemberRow{{UserID: senderID, Nickname: tc.nickname}})

			// when
			_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "who am i?"}, nil)
			svc.sideEffectsWG.Wait()

			// then
			require.NoError(t, err)
			require.Len(t, obs.events, 1)
			assert.Equal(t, tc.want, obs.events[0].SenderName, "the bot must be told who is talking to it")
		})
	}
}

func TestSendMessage_DMMentionStillCarriesTheBotAudience(t *testing.T) {
	// given
	svc, m := newTestService(t)
	obs := &captureMessageObserver{}
	svc.SetMessageObserver(obs)

	senderID := uuid.New()
	botID := uuid.New()
	roomID := uuid.New()
	body := "@beato hello"
	messagesExpectRoom(m, spec.ChatMemberRef{RoomID: roomID, UserID: senderID}, []uuid.UUID{senderID, botID},
		&model.ChatRoomSendContext{Type: dto.RoomTypeDM, LastMessageAt: ongoingThread()})
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, botID).Return(false, nil)
	m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"beato"}).Return([]model.User{{ID: botID, Username: "beato"}}, nil)
	messagesExpectInsert(m, spec.NewChatMessage{RoomID: roomID, SenderID: senderID, Body: body}, nil)
	m.chatRepo.EXPECT().IsMuted(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: botID}).Return(false, nil)
	m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, botID).Return(1, nil)

	// when
	_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: body}, nil)
	svc.sideEffectsWG.Wait()

	// then
	require.NoError(t, err)
	require.Len(t, obs.events, 1)
	assert.Equal(t, []uuid.UUID{senderID, botID}, obs.events[0].Members, "dm bot selection scans the members, so resolving mentions must never narrow the audience")
	assert.Contains(t, obs.events[0].MentionedIDs, botID)
}

func TestSendMessage_RecipientOptInGatesNewThreadsOnTheRoomRoute(t *testing.T) {
	cases := []struct {
		name          string
		roomType      dto.RoomType
		lastMessageAt sql.NullString
		dmsEnabled    bool
		wantErr       error
	}{
		{name: "a first message into a pair whose recipient has dms off is refused", roomType: dto.RoomTypeDM, wantErr: ErrDmsDisabled},
		{name: "a first message into a pair whose recipient has dms on is allowed", roomType: dto.RoomTypeDM, dmsEnabled: true},
		{name: "an existing thread keeps working after the recipient turns dms off", roomType: dto.RoomTypeDM, lastMessageAt: ongoingThread()},
		{name: "a group room never consults the opt-in", roomType: dto.RoomTypeGroup},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a room of this kind reached by room id, which is the single send path
			svc, m := newTestService(t)
			senderID := uuid.New()
			recipientID := uuid.New()
			roomID := uuid.New()
			messagesExpectRoom(m, spec.ChatMemberRef{RoomID: roomID, UserID: senderID}, []uuid.UUID{senderID, recipientID}, &model.ChatRoomSendContext{
				ID: roomID, Type: tc.roomType, CreatedBy: senderID, LastMessageAt: tc.lastMessageAt,
			})
			m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, recipientID).Return(false, nil).Maybe()

			if tc.roomType == dto.RoomTypeDM && !tc.lastMessageAt.Valid {
				recipient := sampleUser(recipientID)
				recipient.DmsEnabled = tc.dmsEnabled
				m.userRepo.EXPECT().GetByIDs(mock.Anything, []uuid.UUID{recipientID}).Return([]model.User{*recipient}, nil)
			}
			if tc.wantErr == nil {
				messagesExpectInsert(m, spec.NewChatMessage{RoomID: roomID, SenderID: senderID, Body: "hi"}, nil)
				m.chatRepo.EXPECT().IsMuted(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: recipientID}).Return(true, nil).Maybe()
				m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, recipientID).Return(1, nil).Maybe()
			}

			// when the message is sent
			_, err := svc.SendMessage(context.Background(), senderID, roomID, dto.SendMessageRequest{Body: "hi"}, nil)
			svc.sideEffectsWG.Wait()

			// then the opt-in gates new threads only, and the room-id route no longer bypasses it
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestGetRoomsByUser(t *testing.T) {
	boom := errors.New("boom")
	first := uuid.New()
	second := uuid.New()
	cases := []struct {
		name    string
		rows    []model.ChatRoomRow
		repoErr error
		want    []uuid.UUID
	}{
		{name: "a failed lookup is returned", repoErr: boom},
		{name: "each room is reduced to its id, in order", rows: []model.ChatRoomRow{{ID: first}, {ID: second}}, want: []uuid.UUID{first, second}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			m.chatRepo.EXPECT().GetRoomsByUser(mock.Anything, userID).Return(tc.rows, tc.repoErr)

			// when
			got, err := svc.GetRoomsByUser(context.Background(), userID)

			// then
			require.ErrorIs(t, err, tc.repoErr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetUnreadCount(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name    string
		count   int
		repoErr error
	}{
		{name: "a failed count is returned", repoErr: boom},
		{name: "the unread room count is returned as is", count: 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, userID).Return(tc.count, tc.repoErr)

			// when
			got, err := svc.GetUnreadCount(context.Background(), userID)

			// then
			require.ErrorIs(t, err, tc.repoErr)
			assert.Equal(t, tc.count, got)
		})
	}
}

func TestMarkRead_Errors(t *testing.T) {
	boom := errors.New("boom")
	cases := append(messagesMembershipGuardCases(boom), messagesGuardCase{
		name: "a failed read mark is returned",
		setup: func(m *testMocks, ref spec.ChatMemberRef) {
			m.chatRepo.EXPECT().IsMember(mock.Anything, ref).Return(true, nil)
			m.chatRepo.EXPECT().MarkRoomRead(mock.Anything, ref).Return(boom)
		},
		wantErr: boom,
	})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			ref := spec.ChatMemberRef{RoomID: uuid.New(), UserID: uuid.New()}
			tc.setup(m, ref)

			// when
			err := svc.MarkRead(context.Background(), ref.RoomID, ref.UserID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestMarkRead_ClearsTheThreadsNotifications(t *testing.T) {
	cases := []struct {
		name                   string
		roomType               dto.RoomType
		recountsAndFansReceipt bool
	}{
		{name: "a pair recounts the badge and fans out read receipts", roomType: dto.RoomTypeDM, recountsAndFansReceipt: true},
		{name: "a room, where seen-by is never displayed, skips the badge recount and the receipt fan-out", roomType: dto.RoomTypeGroup},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a member opening a thread of this kind
			svc, m := newTestService(t)
			ref := spec.ChatMemberRef{RoomID: uuid.New(), UserID: uuid.New()}
			m.chatRepo.EXPECT().IsMember(mock.Anything, ref).Return(true, nil)
			m.chatRepo.EXPECT().MarkRoomRead(mock.Anything, ref).Return(nil)
			m.notifSvc.EXPECT().MarkChatRoomRead(mock.Anything, ref.UserID, ref.RoomID).Return(nil)
			m.chatRepo.EXPECT().GetRoomSendContext(mock.Anything, ref.RoomID).
				Return(&model.ChatRoomSendContext{ID: ref.RoomID, Type: tc.roomType}, nil)
			if tc.recountsAndFansReceipt {
				m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, ref.UserID).Return(0, nil)
				m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, ref.RoomID).Return([]uuid.UUID{ref.UserID}, nil)
			}

			// when the thread is marked read
			err := svc.MarkRead(context.Background(), ref.RoomID, ref.UserID)

			// then the bell stops carrying mentions, replies and messages for a thread already on screen, and in a room no receipt fan-out happens (GetRoomMembers is never called) and the DM-only badge is never recounted
			require.NoError(t, err)
			m.notifSvc.AssertCalled(t, "MarkChatRoomRead", mock.Anything, ref.UserID, ref.RoomID)
			if !tc.recountsAndFansReceipt {
				m.chatRepo.AssertNotCalled(t, "CountUnreadRoomsForUser", mock.Anything, ref.UserID)
			}
		})
	}
}

func TestDeleteMessage_WhoMayDeleteAndWhatIsAudited(t *testing.T) {
	cases := []struct {
		name          string
		byAuthor      bool
		memberRole    string
		siteRole      role.Role
		roomType      dto.RoomType
		isPublic      bool
		wantErr       error
		wantAction    audit.Action
		wantDetailsBy string
	}{
		{name: "an author deletes their own message in a private room without an audit row", byAuthor: true, roomType: dto.RoomTypeGroup},
		{name: "an author deleting in a public room is audited as a plain delete", byAuthor: true, roomType: dto.RoomTypeGroup, isPublic: true, wantAction: audit.ActionChatMessageDelete},
		{name: "an author deleting in a dm is never audited, where an audit row would leak that the conversation exists", byAuthor: true, roomType: dto.RoomTypeDM},
		{name: "a host deletes another member's message in a private room without an audit row", memberRole: "host", roomType: dto.RoomTypeGroup},
		{name: "a host deleting in a public room is audited as a moderator delete by host", memberRole: "host", roomType: dto.RoomTypeGroup, isPublic: true, wantAction: audit.ActionChatMessageDeleteMod, wantDetailsBy: " by=host"},
		{name: "site staff delete another member's message in a private room without an audit row", memberRole: "member", siteRole: authz.RoleModerator, roomType: dto.RoomTypeGroup},
		{name: "site staff deleting in a public room are audited as a moderator delete by staff", memberRole: "member", siteRole: authz.RoleModerator, roomType: dto.RoomTypeGroup, isPublic: true, wantAction: audit.ActionChatMessageDeleteMod, wantDetailsBy: " by=staff"},
		{name: "a plain member may not delete another member's message", memberRole: "member", wantErr: ErrMessageDeletePermission},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			messageID := uuid.New()
			roomID := uuid.New()
			senderID := uuid.New()
			m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&model.ChatMessageRow{ID: messageID, RoomID: roomID, SenderID: senderID}, nil)

			actorID := senderID
			if !tc.byAuthor {
				actorID = uuid.New()
				m.chatRepo.EXPECT().GetMemberRole(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: actorID}).Return(tc.memberRole, nil)
			}
			if tc.memberRole == "member" {
				m.authzSvc.EXPECT().GetRole(mock.Anything, actorID).Return(tc.siteRole, nil)
			}

			if tc.wantErr == nil {
				m.chatRepo.EXPECT().DeleteMessageWithMedia(mock.Anything, messageID).Return(nil, nil)
				m.uploadSvc.EXPECT().Delete().Return()
				m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil).Maybe()
				m.chatRepo.EXPECT().GetRoomSendContext(mock.Anything, roomID).Return(&model.ChatRoomSendContext{ID: roomID, Type: tc.roomType, IsPublic: tc.isPublic}, nil)
			}
			if tc.wantAction != "" {
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    actorID,
					Action:     tc.wantAction,
					TargetType: audit.TargetChatRoom,
					TargetID:   roomID.String(),
					Details:    "message=" + messageID.String() + tc.wantDetailsBy,
					SubjectID:  senderID,
				}).Return(nil)
			}

			// when
			err := svc.DeleteMessage(context.Background(), messageID, actorID)

			// then the strict audit mock records no expectation for an unaudited case, so any row would fail it
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestDeleteMessage_UnlinksMediaAfterTheRowIsGone(t *testing.T) {
	// given a message carrying two uploaded files
	svc, m := newTestService(t)
	messageID := uuid.New()
	roomID := uuid.New()
	authorID := uuid.New()
	var order []string

	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(&model.ChatMessageRow{ID: messageID, RoomID: roomID, SenderID: authorID}, nil)
	m.chatRepo.EXPECT().DeleteMessageWithMedia(mock.Anything, messageID).
		Run(func(ctx context.Context, id uuid.UUID, _ ...*sql.Tx) { order = append(order, "delete-message") }).
		Return([]string{"/uploads/chat/a.webp", "/uploads/chat/a_thumb.webp"}, nil)
	m.uploadSvc.EXPECT().Delete([]string{"/uploads/chat/a.webp", "/uploads/chat/a_thumb.webp"}).
		Run(func(urlPaths ...string) { order = append(order, "delete-files") }).Return()
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil).Maybe()
	m.chatRepo.EXPECT().GetRoomSendContext(mock.Anything, roomID).Return(&model.ChatRoomSendContext{ID: roomID, Type: dto.RoomTypeGroup}, nil)

	// when
	err := svc.DeleteMessage(context.Background(), messageID, authorID)

	// then both files are unlinked, and only once the row delete has committed
	require.NoError(t, err)
	require.Equal(t, []string{"delete-message", "delete-files"}, order)
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

var (
	errMessagesReload = errors.New("reload failed")
)

func TestEditMessage_Author_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	messageID := uuid.New()
	author := spec.ChatMemberRef{RoomID: uuid.New(), UserID: uuid.New()}
	updated := &model.ChatMessageRow{ID: messageID, RoomID: author.RoomID, SenderID: author.UserID, Body: "new", EditedAt: new("2026-04-18T20:00:00Z")}
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(messagesStoredMessage(messageID, author.RoomID, author.UserID), nil).Once()
	messagesExpectSenderMaySpeak(m, author)
	m.chatRepo.EXPECT().EditMessage(mock.Anything, spec.ChatMessageUpdate{MessageID: messageID, Body: "new"}).Return(nil)
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(updated, nil).Once()
	m.chatRepo.EXPECT().GetMessageMediaBatch(mock.Anything, []uuid.UUID{messageID}).Return(nil, nil)
	m.chatRepo.EXPECT().GetReactionsBatch(mock.Anything, spec.ChatReactionsQuery{MessageIDs: []uuid.UUID{messageID}, ViewerID: author.UserID}).Return(nil, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUser(mock.Anything, author.UserID).Return(nil, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, author.RoomID).Return(nil, nil).Maybe()

	// when
	resp, err := svc.EditMessage(context.Background(), messageID, author.UserID, "new")

	// then
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "new", resp.Body)
	require.NotNil(t, resp.EditedAt)
}

func TestEditMessage_Errors(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		setup   func(m *testMocks, messageID uuid.UUID, author spec.ChatMemberRef)
		wantErr error
	}{
		{
			name:    "an empty body is rejected before any lookup",
			body:    "",
			wantErr: ErrMissingFields,
		},
		{
			name: "a missing message is not found",
			body: "new",
			setup: func(m *testMocks, messageID uuid.UUID, _ spec.ChatMemberRef) {
				m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(nil, nil)
			},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "a system message cannot be edited, even by its sender",
			body: "new",
			setup: func(m *testMocks, messageID uuid.UUID, author spec.ChatMemberRef) {
				stored := messagesStoredMessage(messageID, author.RoomID, author.UserID)
				stored.IsSystem = true
				m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(stored, nil)
			},
			wantErr: ErrCannotEditSystemMessage,
		},
		{
			name: "another member's message cannot be edited",
			body: "new",
			setup: func(m *testMocks, messageID uuid.UUID, author spec.ChatMemberRef) {
				m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(messagesStoredMessage(messageID, author.RoomID, uuid.New()), nil)
			},
			wantErr: ErrMessageEditPermission,
		},
		{
			name: "an author who has been kicked cannot edit",
			body: "new",
			setup: func(m *testMocks, messageID uuid.UUID, author spec.ChatMemberRef) {
				m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(messagesStoredMessage(messageID, author.RoomID, author.UserID), nil)
				m.chatRepo.EXPECT().IsMember(mock.Anything, author).Return(false, nil)
			},
			wantErr: ErrNotMember,
		},
		{
			name: "a timed-out author cannot edit",
			body: "new",
			setup: func(m *testMocks, messageID uuid.UUID, author spec.ChatMemberRef) {
				m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(messagesStoredMessage(messageID, author.RoomID, author.UserID), nil)
				m.chatRepo.EXPECT().IsMember(mock.Anything, author).Return(true, nil)
				m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, author).Return(true, "", false, nil)
			},
			wantErr: ErrTimedOut,
		},
		{
			name: "a message deleted between the edit and the reload is not found, not a wrapped nil",
			body: "new",
			setup: func(m *testMocks, messageID uuid.UUID, author spec.ChatMemberRef) {
				m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(messagesStoredMessage(messageID, author.RoomID, author.UserID), nil).Once()
				messagesExpectSenderMaySpeak(m, author)
				m.chatRepo.EXPECT().EditMessage(mock.Anything, spec.ChatMessageUpdate{MessageID: messageID, Body: "new"}).Return(nil)
				m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(nil, nil).Once()
			},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "a failed media reload is reported instead of broadcasting the edit without its media",
			body: "new",
			setup: func(m *testMocks, messageID uuid.UUID, author spec.ChatMemberRef) {
				m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(messagesStoredMessage(messageID, author.RoomID, author.UserID), nil).Once()
				messagesExpectSenderMaySpeak(m, author)
				m.chatRepo.EXPECT().EditMessage(mock.Anything, spec.ChatMessageUpdate{MessageID: messageID, Body: "new"}).Return(nil)
				m.chatRepo.EXPECT().GetMessageByID(mock.Anything, messageID).Return(messagesStoredMessage(messageID, author.RoomID, author.UserID), nil).Once()
				m.chatRepo.EXPECT().GetMessageMediaBatch(mock.Anything, []uuid.UUID{messageID}).Return(nil, errMessagesReload)
			},
			wantErr: errMessagesReload,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			messageID := uuid.New()
			author := spec.ChatMemberRef{RoomID: uuid.New(), UserID: uuid.New()}
			if tc.setup != nil {
				tc.setup(m, messageID, author)
			}

			// when
			resp, err := svc.EditMessage(context.Background(), messageID, author.UserID, tc.body)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, resp)
		})
	}
}
