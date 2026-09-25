package dao_test

import (
	"context"
	"database/sql"
	"slices"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func messageRowIDs(rows []model.ChatMessageRow) []uuid.UUID {
	var ids []uuid.UUID
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	return ids
}

func TestChatDAO_InsertMessage(t *testing.T) {
	tests := []struct {
		name       string
		insert     func(repository.ChatRepository, context.Context, spec.NewChatMessage, ...*sql.Tx) (*model.ChatMessageRow, error)
		wantSystem bool
	}{
		{name: "a plain message is stored unflagged, readable through every lookup, and stamps room activity", insert: repository.ChatRepository.InsertMessageAndMarkRead, wantSystem: false},
		{name: "a system message is stored flagged, readable through every lookup, and stamps room activity", insert: repository.ChatRepository.InsertSystemMessage, wantSystem: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			user := daotest.CreateUser(t, repos)
			roomID := daotest.CreateChatRoom(t, repos, user.ID, daotest.WithRoomMembers(user.ID))

			// when
			msg, err := tc.insert(repos.Chat, ctx, spec.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "hello"})

			// then
			require.NoError(t, err)

			got, err := repos.Chat.GetMessageByID(ctx, msg.ID)
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "hello", got.Body)
			assert.Equal(t, user.ID, got.SenderID)
			assert.Equal(t, tc.wantSystem, got.IsSystem)
			assert.Nil(t, got.ReplyToID)
			assert.Nil(t, got.ReplyToSenderID)
			assert.Nil(t, got.ReplyToBody)
			assert.Nil(t, got.ReplyToSenderName)

			senderID, err := repos.Chat.GetMessageSenderID(ctx, msg.ID)
			require.NoError(t, err)
			assert.Equal(t, user.ID, senderID)

			messageRoomID, err := repos.Chat.GetMessageRoomID(ctx, msg.ID)
			require.NoError(t, err)
			assert.Equal(t, roomID, messageRoomID)

			room, err := repos.Chat.GetRoomByID(ctx, spec.ChatRoomViewer{RoomID: roomID, ViewerID: user.ID})
			require.NoError(t, err)
			require.NotNil(t, room)
			assert.True(t, room.LastMessageAt.Valid)
		})
	}
}

func TestChatDAO_InsertMessage_ReplyPreview(t *testing.T) {
	tests := []struct {
		name        string
		parentAlias string
		want        string
	}{
		{name: "falls back to the parent author's display name when they have no room alias", parentAlias: "", want: "RealName"},
		{name: "prefers the parent author's room alias over their display name", parentAlias: "Battler", want: "Battler"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			parentAuthor := daotest.CreateUser(t, repos, daotest.WithDisplayName("RealName"))
			replier := daotest.CreateUser(t, repos, daotest.WithDisplayName("Replier"))
			roomID := daotest.CreateChatRoom(t, repos, parentAuthor.ID, daotest.WithRoomMembers(parentAuthor.ID, replier.ID))
			require.NoError(t, repos.Chat.SetMemberNickname(ctx, spec.ChatMemberNicknameUpdate{RoomID: roomID, UserID: parentAuthor.ID, Nickname: tc.parentAlias}))
			parentID := daotest.SendChatMessage(t, repos, roomID, parentAuthor.ID, "parent")

			// when
			reply, err := repos.Chat.InsertMessageAndMarkRead(ctx, spec.NewChatMessage{RoomID: roomID, SenderID: replier.ID, Body: "reply", ReplyToID: &parentID})

			// then
			require.NoError(t, err)

			got, err := repos.Chat.GetMessageByID(ctx, reply.ID)
			require.NoError(t, err)
			require.NotNil(t, got)
			require.NotNil(t, got.ReplyToID)
			assert.Equal(t, parentID, *got.ReplyToID)
			require.NotNil(t, got.ReplyToSenderID)
			assert.Equal(t, parentAuthor.ID, *got.ReplyToSenderID)
			require.NotNil(t, got.ReplyToBody)
			assert.Equal(t, "parent", *got.ReplyToBody)
			require.NotNil(t, got.ReplyToSenderName)
			assert.Equal(t, tc.want, *got.ReplyToSenderName)
		})
	}
}

func TestChatDAO_GetMessages(t *testing.T) {
	tests := []struct {
		name    string
		sent    int
		limit   int
		wantLen int
	}{
		{name: "under the limit every message is returned with the sender's per-room overrides", sent: 3, limit: 20, wantLen: 3},
		{name: "over the limit one page is returned while the total still counts every message", sent: 5, limit: 2, wantLen: 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			user := daotest.CreateUser(t, repos)
			roomID := daotest.CreateChatRoom(t, repos, user.ID, daotest.WithRoomMembers(user.ID))
			require.NoError(t, repos.Chat.SetMemberNickname(ctx, spec.ChatMemberNicknameUpdate{RoomID: roomID, UserID: user.ID, Nickname: "Beato"}))
			require.NoError(t, repos.Chat.SetMemberAvatar(ctx, spec.ChatMemberAvatarUpdate{RoomID: roomID, UserID: user.ID, AvatarURL: "/custom.png"}))

			for range tc.sent {
				daotest.SendChatMessage(t, repos, roomID, user.ID, "m")
			}

			// when
			msgs, total, err := repos.Chat.GetMessages(ctx, spec.ChatMessagePage{RoomID: roomID, Limit: tc.limit, Offset: 0})

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.sent, total)
			require.Len(t, msgs, tc.wantLen)

			for _, msg := range msgs {
				assert.Equal(t, "Beato", msg.SenderNickname)
				assert.Equal(t, "/custom.png", msg.SenderMemberAvatar)
			}
		})
	}
}

func TestChatDAO_GetMessagesBefore(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, user.ID, daotest.WithRoomMembers(user.ID))
	olderID := daotest.SendChatMessage(t, repos, roomID, user.ID, "older")
	newerID := daotest.SendChatMessage(t, repos, roomID, user.ID, "newer")
	daotest.BackdateChatMessage(t, repos, olderID, "2024-01-01 00:30:00")
	daotest.BackdateChatMessage(t, repos, newerID, "2024-01-01 02:00:00")

	tests := []struct {
		name   string
		before string
		want   []uuid.UUID
	}{
		{name: "a space-separated cursor after every message returns them all oldest first", before: "2099-01-01 00:00:00", want: []uuid.UUID{olderID, newerID}},
		{name: "a cursor before every message returns nothing", before: "2000-01-01 00:00:00", want: nil},
		{name: "an RFC3339 cursor compares as a datetime, so a later message cannot slip past it as a string", before: "2024-01-01T01:00:00Z", want: []uuid.UUID{olderID}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			msgs, err := repos.Chat.GetMessagesBefore(ctx, spec.ChatMessageCursorPage{RoomID: roomID, ViewerID: uuid.Nil, Before: tc.before, Limit: 20})

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, messageRowIDs(msgs))
		})
	}
}

func TestChatDAO_GetMessagesBefore_CursorWithIDPaginatesSameSecondMessages(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, user.ID, daotest.WithRoomMembers(user.ID))

	ids := make([]string, 0, 3)
	for range 3 {
		id := daotest.SendChatMessage(t, repos, roomID, user.ID, "m")
		daotest.BackdateChatMessage(t, repos, id, "2024-01-01 00:00:00")
		ids = append(ids, id.String())
	}
	slices.Sort(ids)

	// when
	firstPage, total, err := repos.Chat.GetMessages(ctx, spec.ChatMessagePage{RoomID: roomID, Limit: 2, Offset: 0})
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Len(t, firstPage, 2)
	assert.Equal(t, ids[1:], []string{firstPage[0].ID.String(), firstPage[1].ID.String()})

	cursor := firstPage[0].CreatedAt + "|" + firstPage[0].ID.String()
	secondPage, err := repos.Chat.GetMessagesBefore(ctx, spec.ChatMessageCursorPage{RoomID: roomID, ViewerID: uuid.Nil, Before: cursor, Limit: 2})

	// then
	require.NoError(t, err)
	require.Len(t, secondPage, 1)
	assert.Equal(t, ids[0], secondPage[0].ID.String())
}

func TestChatDAO_DMRejoinHidesEarlierMessagesAndPinsFromTheRejoinerOnly(t *testing.T) {
	// given a dm pair with messages and a pinned message backdated before the leaver rejoined
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	stayer := daotest.CreateUser(t, repos)
	leaver := daotest.CreateUser(t, repos)
	roomID := daotest.CreateDMRoom(t, repos, stayer.ID, leaver.ID)
	daotest.BackdateChatRoomJoins(t, repos, roomID, "2024-01-01 00:00:00")

	beforeID := daotest.SendChatMessage(t, repos, roomID, leaver.ID, "before")
	secretID := daotest.SendChatMessage(t, repos, roomID, stayer.ID, "secret")
	daotest.BackdateChatMessage(t, repos, beforeID, "2024-01-01 01:00:00")
	daotest.BackdateChatMessage(t, repos, secretID, "2024-01-01 01:00:01")

	require.NoError(t, repos.Chat.PinMessage(ctx, spec.ChatMessagePin{MessageID: secretID, PinnedBy: stayer.ID}))

	// when the leaver leaves and rejoins the same pair, then posts again
	require.NoError(t, repos.Chat.RemoveMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: leaver.ID}))

	reopened, err := repos.Chat.CreateDMRoomAtomic(ctx, spec.ChatDMPair{UserA: leaver.ID, UserB: stayer.ID})
	require.NoError(t, err)
	require.Equal(t, roomID, reopened.ID)

	afterID := daotest.SendChatMessage(t, repos, roomID, leaver.ID, "after")
	daotest.BackdateChatMessage(t, repos, afterID, "2099-01-01 00:00:00")

	// then the rejoiner sees no messages or pins from before, and the other party still sees them
	mine, err := repos.Chat.GetMessagesForMember(ctx, spec.ChatMessagePage{RoomID: roomID, ViewerID: leaver.ID, Limit: 50})
	require.NoError(t, err)
	require.Len(t, mine, 1, "the member who deleted the chat only sees what came after")
	assert.Equal(t, "after", mine[0].Body)

	theirs, err := repos.Chat.GetMessagesForMember(ctx, spec.ChatMessagePage{RoomID: roomID, ViewerID: stayer.ID, Limit: 50})
	require.NoError(t, err)

	var theirBodies []string
	for _, msg := range theirs {
		theirBodies = append(theirBodies, msg.Body)
	}
	assert.Equal(t, []string{"before", "secret", "after"}, theirBodies, "the other member keeps the whole conversation")

	forLeaver, err := repos.Chat.ListPinnedMessages(ctx, spec.ChatRoomViewer{RoomID: roomID, ViewerID: leaver.ID})
	require.NoError(t, err)
	assert.Empty(t, forLeaver, "a pin from before the rejoin must not leak to the rejoiner")

	forStayer, err := repos.Chat.ListPinnedMessages(ctx, spec.ChatRoomViewer{RoomID: roomID, ViewerID: stayer.ID})
	require.NoError(t, err)
	require.Len(t, forStayer, 1)
	assert.Equal(t, secretID, forStayer[0].ID)
}

func TestChatDAO_EditMessage(t *testing.T) {
	tests := []struct {
		name          string
		editsExisting bool
		wantBody      string
		wantEdited    bool
	}{
		{name: "editing a message rewrites its body and stamps edited_at in single and list reads", editsExisting: true, wantBody: "updated body", wantEdited: true},
		{name: "editing an unknown id is a silent no-op that leaves existing messages untouched", editsExisting: false, wantBody: "original", wantEdited: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			user := daotest.CreateUser(t, repos)
			roomID := daotest.CreateChatRoom(t, repos, user.ID, daotest.WithRoomMembers(user.ID))
			msgID := daotest.SendChatMessage(t, repos, roomID, user.ID, "original")

			// when
			target := uuid.New()
			if tc.editsExisting {
				target = msgID
			}

			err := repos.Chat.EditMessage(ctx, spec.ChatMessageUpdate{MessageID: target, Body: "updated body"})

			// then
			require.NoError(t, err)

			got, err := repos.Chat.GetMessageByID(ctx, msgID)
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tc.wantBody, got.Body)

			if tc.wantEdited {
				require.NotNil(t, got.EditedAt)
				assert.NotEmpty(t, *got.EditedAt)
			} else {
				assert.Nil(t, got.EditedAt)
			}

			msgs, _, err := repos.Chat.GetMessages(ctx, spec.ChatMessagePage{RoomID: roomID, Limit: 10, Offset: 0})
			require.NoError(t, err)
			require.Len(t, msgs, 1)
			assert.Equal(t, tc.wantBody, msgs[0].Body)
			assert.Equal(t, tc.wantEdited, msgs[0].EditedAt != nil)
		})
	}
}

func TestChatDAO_DeleteMessage(t *testing.T) {
	t.Run("DeleteMessage removes only that message and reading it back maps not-found to nil", func(t *testing.T) {
		// given
		repos := daotest.NewRepos(t)
		ctx := context.Background()
		user := daotest.CreateUser(t, repos)
		roomID := daotest.CreateChatRoom(t, repos, user.ID, daotest.WithRoomMembers(user.ID))
		keptID := daotest.SendChatMessage(t, repos, roomID, user.ID, "keep")
		goneID := daotest.SendChatMessage(t, repos, roomID, user.ID, "gone")

		// when
		err := repos.Chat.DeleteMessage(ctx, goneID)

		// then
		require.NoError(t, err)

		got, err := repos.Chat.GetMessageByID(ctx, goneID)
		require.NoError(t, err)
		assert.Nil(t, got)

		kept, err := repos.Chat.GetMessageByID(ctx, keptID)
		require.NoError(t, err)
		assert.NotNil(t, kept)
	})

	t.Run("DeleteMessages clears only the given room", func(t *testing.T) {
		// given
		repos := daotest.NewRepos(t)
		ctx := context.Background()
		user := daotest.CreateUser(t, repos)
		roomA := daotest.CreateChatRoom(t, repos, user.ID, daotest.WithRoomName("A"), daotest.WithRoomMembers(user.ID))
		roomB := daotest.CreateChatRoom(t, repos, user.ID, daotest.WithRoomName("B"), daotest.WithRoomMembers(user.ID))
		daotest.SendChatMessage(t, repos, roomA, user.ID, "a1")
		daotest.SendChatMessage(t, repos, roomA, user.ID, "a2")
		daotest.SendChatMessage(t, repos, roomB, user.ID, "b1")

		// when
		err := repos.Chat.DeleteMessages(ctx, roomA)

		// then
		require.NoError(t, err)

		msgsA, totalA, err := repos.Chat.GetMessages(ctx, spec.ChatMessagePage{RoomID: roomA, Limit: 20, Offset: 0})
		require.NoError(t, err)
		assert.Equal(t, 0, totalA)
		assert.Empty(t, msgsA)

		_, totalB, err := repos.Chat.GetMessages(ctx, spec.ChatMessagePage{RoomID: roomB, Limit: 20, Offset: 0})
		require.NoError(t, err)
		assert.Equal(t, 1, totalB)
	})
}

func TestChatDAO_PinAndUnpinMessage(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	pinner := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, author.ID, daotest.WithRoomMembers(author.ID, pinner.ID))
	firstID := daotest.SendChatMessage(t, repos, roomID, author.ID, "first")
	secondID := daotest.SendChatMessage(t, repos, roomID, author.ID, "second")
	viewer := spec.ChatRoomViewer{RoomID: roomID, ViewerID: author.ID}

	// when
	require.NoError(t, repos.Chat.PinMessage(ctx, spec.ChatMessagePin{MessageID: firstID, PinnedBy: pinner.ID}))

	_, err := repos.DB().ExecContext(ctx, `UPDATE chat_messages SET pinned_at = pinned_at - INTERVAL '1 hour' WHERE id = $1`, firstID)
	require.NoError(t, err)

	require.NoError(t, repos.Chat.PinMessage(ctx, spec.ChatMessagePin{MessageID: secondID, PinnedBy: pinner.ID}))

	pinned, err := repos.Chat.ListPinnedMessages(ctx, viewer)

	// then
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{secondID, firstID}, messageRowIDs(pinned))

	for _, msg := range pinned {
		require.NotNil(t, msg.PinnedAt)
		require.NotNil(t, msg.PinnedBy)
		assert.Equal(t, pinner.ID, *msg.PinnedBy)
	}

	// when
	require.NoError(t, repos.Chat.UnpinMessage(ctx, secondID))

	after, err := repos.Chat.ListPinnedMessages(ctx, viewer)

	// then
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{firstID}, messageRowIDs(after))
}

func TestChatDAO_Reactions(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	userA := daotest.CreateUser(t, repos)
	userB := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, userA.ID, daotest.WithRoomMembers(userA.ID, userB.ID))
	msgID := daotest.SendChatMessage(t, repos, roomID, userA.ID, "hi")
	thumbsFromA := spec.ChatMessageReaction{MessageID: msgID, UserID: userA.ID, Emoji: "👍"}
	thumbsFromB := spec.ChatMessageReaction{MessageID: msgID, UserID: userB.ID, Emoji: "👍"}
	laughFromA := spec.ChatMessageReaction{MessageID: msgID, UserID: userA.ID, Emoji: "😂"}
	query := spec.ChatReactionsQuery{MessageIDs: []uuid.UUID{msgID}, ViewerID: userB.ID}

	// when
	first, err := repos.Chat.AddReaction(ctx, thumbsFromA)
	require.NoError(t, err)

	dup, err := repos.Chat.AddReaction(ctx, thumbsFromA)
	require.NoError(t, err)

	fromB, err := repos.Chat.AddReaction(ctx, thumbsFromB)
	require.NoError(t, err)

	laugh, err := repos.Chat.AddReaction(ctx, laughFromA)
	require.NoError(t, err)

	groups, err := repos.Chat.GetReactionsBatch(ctx, query)

	// then
	require.NoError(t, err)
	assert.True(t, first)
	assert.False(t, dup)
	assert.True(t, fromB)
	assert.True(t, laugh)

	require.Len(t, groups[msgID], 2)
	thumbsGroup := groups[msgID][0]
	assert.Equal(t, "👍", thumbsGroup.Emoji)
	assert.Equal(t, 2, thumbsGroup.Count)
	assert.True(t, thumbsGroup.ViewerReacted)
	laughGroup := groups[msgID][1]
	assert.Equal(t, "😂", laughGroup.Emoji)
	assert.Equal(t, 1, laughGroup.Count)
	assert.False(t, laughGroup.ViewerReacted)

	// when
	removed, err := repos.Chat.RemoveReaction(ctx, laughFromA)
	require.NoError(t, err)

	again, err := repos.Chat.RemoveReaction(ctx, laughFromA)
	require.NoError(t, err)

	after, err := repos.Chat.GetReactionsBatch(ctx, query)

	// then
	require.NoError(t, err)
	assert.True(t, removed)
	assert.False(t, again)

	require.Len(t, after[msgID], 1)
	assert.Equal(t, "👍", after[msgID][0].Emoji)
	assert.Equal(t, 2, after[msgID][0].Count)
	assert.True(t, after[msgID][0].ViewerReacted)
}
