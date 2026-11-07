package dao_test

import (
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mediaRoomAttachmentIDs(t *testing.T, repos *repository.Repositories, roomID, viewerID uuid.UUID, kind model.AttachmentKind) []uuid.UUID {
	t.Helper()

	rows, err := repos.Chat.ListRoomAttachments(context.Background(), spec.ChatRoomAttachmentQuery{RoomID: roomID, ViewerID: viewerID, Before: "", Limit: 50, Kind: kind})
	require.NoError(t, err)

	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}

	return ids
}

func mediaAttach(t *testing.T, repos *repository.Repositories, media spec.NewMedia) int64 {
	t.Helper()

	id, err := repos.Chat.AddMessageMedia(context.Background(), spec.NewChatMessageMedia{NewMedia: media})
	require.NoError(t, err)

	return id
}

func TestChatDAO_GetMessageMediaBatch_RoundTripsAddedMediaInSortOrder(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, user.ID, daotest.WithRoomMembers(user.ID))
	msgID := daotest.SendChatMessage(t, repos, roomID, user.ID, "m")
	bareID := daotest.SendChatMessage(t, repos, roomID, user.ID, "no media")

	// when
	spoilerID, err := repos.Chat.AddMessageMedia(ctx, spec.NewChatMessageMedia{TargetID: msgID, MediaURL: "/b", MediaType: "video", ThumbnailURL: "/b_thumb", Filename: "b.mp4", SortOrder: 2, IsSpoiler: true, Width: 640, Height: 480})
	require.NoError(t, err)

	plainID, err := repos.Chat.AddMessageMedia(ctx, spec.NewChatMessageMedia{TargetID: msgID, MediaURL: "/a", MediaType: "image", ThumbnailURL: "/a_thumb", SortOrder: 1})
	require.NoError(t, err)

	media, err := repos.Chat.GetMessageMediaBatch(ctx, []uuid.UUID{msgID, bareID})

	// then
	require.NoError(t, err)
	assert.Equal(t, map[uuid.UUID][]dto.PostMediaResponse{
		msgID: {
			{ID: int(plainID), MediaURL: "/a", MediaType: "image", ThumbnailURL: "/a_thumb", SortOrder: 1},
			{ID: int(spoilerID), MediaURL: "/b", MediaType: "video", ThumbnailURL: "/b_thumb", Filename: "b.mp4", SortOrder: 2, Width: 640, Height: 480, IsSpoiler: true},
		},
	}, media)

	t.Run("nil ids return an empty map", func(t *testing.T) {
		// when
		none, err := repos.Chat.GetMessageMediaBatch(ctx, nil)

		// then
		require.NoError(t, err)
		assert.Empty(t, none)
	})
}

func TestChatRepository_InsertMessageAndMarkRead_ClaimsItsUploadsInOrder(t *testing.T) {
	// given two finished uploads, the second marked as a spoiler
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, user.ID, daotest.WithRoomMembers(user.ID))
	first := daotest.ReadyUpload(t, repos, user.ID, "image", "/uploads/media/a.webp")
	second := daotest.ReadyUpload(t, repos, user.ID, "video", "/uploads/media/b.mp4")
	claim := spec.ClaimedUploads{UserID: user.ID, Uploads: []spec.UploadRef{{ID: first}, {ID: second, IsSpoiler: true}}, MediaTypes: []string{"image", "video", "audio"}, MaxAge: time.Hour}

	// when
	msg, media, err := repos.Chat.InsertMessageAndMarkRead(ctx, spec.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "look", Media: claim})

	// then the message carries both files with their stored dimensions, and the uploads are no longer pending
	require.NoError(t, err)
	require.Len(t, media, 2)
	assert.Equal(t, "/uploads/media/a.webp", media[0].MediaURL)
	assert.Equal(t, 0, media[0].SortOrder)
	assert.Equal(t, 4, media[0].Width)
	assert.False(t, media[0].IsSpoiler)
	assert.Equal(t, "/uploads/media/b.mp4", media[1].MediaURL)
	assert.True(t, media[1].IsSpoiler)

	stored, err := repos.Chat.GetMessageMediaBatch(ctx, []uuid.UUID{msg.ID})
	require.NoError(t, err)
	assert.Equal(t, media, stored[msg.ID])

	pending, err := repos.PendingUpload.List(ctx, spec.UserUploads{UserID: user.ID, IDs: []uuid.UUID{first, second}})
	require.NoError(t, err)
	assert.Empty(t, pending)
}

func TestChatRepository_InsertMessageAndMarkRead_AnUnclaimableUploadWritesNothing(t *testing.T) {
	// given one finished upload and one that does not exist
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, user.ID, daotest.WithRoomMembers(user.ID))
	ready := daotest.ReadyUpload(t, repos, user.ID, "image", "/uploads/media/a.webp")
	claim := spec.ClaimedUploads{UserID: user.ID, Uploads: []spec.UploadRef{{ID: ready}, {ID: uuid.New()}}, MediaTypes: []string{"image", "video", "audio"}, MaxAge: time.Hour}

	// when
	msg, _, err := repos.Chat.InsertMessageAndMarkRead(ctx, spec.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "look", Media: claim})

	// then no message exists and the ready upload is still pending
	require.ErrorIs(t, err, daoerr.ErrUploadUnavailable)
	assert.Nil(t, msg)
	pending, err := repos.PendingUpload.List(ctx, spec.UserUploads{UserID: user.ID, IDs: []uuid.UUID{ready}})
	require.NoError(t, err)
	assert.Len(t, pending, 1)
}

func TestChatDAO_ListRoomAttachments(t *testing.T) {
	// given a room holding one message with media, one with a link, one plain and one system message carrying both
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, user.ID, daotest.WithRoomMembers(user.ID))

	withMediaID := daotest.SendChatMessage(t, repos, roomID, user.ID, "look")
	mediaAttach(t, repos, spec.NewMedia{TargetID: withMediaID, MediaURL: "/uploads/a.png", MediaType: "image"})

	withLinkID := daotest.SendChatMessage(t, repos, roomID, user.ID, "see https://example.com/page")
	daotest.SendChatMessage(t, repos, roomID, user.ID, "just talking")

	system, _, err := repos.Chat.InsertMessageAndMarkRead(ctx, spec.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "system https://example.com/sys", IsSystem: true})
	require.NoError(t, err)

	mediaAttach(t, repos, spec.NewMedia{TargetID: system.ID, MediaURL: "/uploads/sys.png", MediaType: "image"})

	tests := []struct {
		name string
		kind model.AttachmentKind
		want []uuid.UUID
	}{
		{name: "media returns only messages carrying a media row", kind: model.AttachmentKindMedia, want: []uuid.UUID{withMediaID}},
		{name: "links returns only bodies holding a url", kind: model.AttachmentKindLinks, want: []uuid.UUID{withLinkID}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			ids := mediaRoomAttachmentIDs(t, repos, roomID, user.ID, tc.kind)

			// then
			assert.Equal(t, tc.want, ids)
		})
	}

	t.Run("an unknown kind is an error rather than a silent default", func(t *testing.T) {
		// when
		_, err := repos.Chat.ListRoomAttachments(ctx, spec.ChatRoomAttachmentQuery{RoomID: roomID, ViewerID: user.ID, Before: "", Limit: 50, Kind: model.AttachmentKind("files")})

		// then
		require.Error(t, err)
	})
}

func TestChatDAO_ListRoomAttachments_HidesMessagesFromBeforeADMRejoin(t *testing.T) {
	// given a dm pair with a linked, media-carrying message backdated before the leaver rejoined
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	stayer := daotest.CreateUser(t, repos)
	leaver := daotest.CreateUser(t, repos)
	roomID := daotest.CreateDMRoom(t, repos, stayer.ID, leaver.ID)
	daotest.BackdateChatRoomJoins(t, repos, roomID, "2024-01-01 00:00:00")

	oldID := daotest.SendChatMessage(t, repos, roomID, stayer.ID, "secret https://example.com/old")
	mediaAttach(t, repos, spec.NewMedia{TargetID: oldID, MediaURL: "/uploads/old.png", MediaType: "image"})
	daotest.BackdateChatMessage(t, repos, oldID, "2024-01-01 01:00:00")

	// when the leaver leaves and rejoins the same pair
	require.NoError(t, repos.Chat.RemoveMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: leaver.ID}))
	_, err := repos.Chat.CreateDMRoomAtomic(ctx, spec.ChatDMPair{UserA: leaver.ID, UserB: stayer.ID})
	require.NoError(t, err)

	tests := []struct {
		name string
		kind model.AttachmentKind
	}{
		{name: "media", kind: model.AttachmentKindMedia},
		{name: "links", kind: model.AttachmentKindLinks},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// then the rejoiner sees nothing from before, and the other party still sees it
			assert.Empty(t, mediaRoomAttachmentIDs(t, repos, roomID, leaver.ID, tc.kind), "an attachment from before the rejoin must not leak to the rejoiner")
			assert.Equal(t, []uuid.UUID{oldID}, mediaRoomAttachmentIDs(t, repos, roomID, stayer.ID, tc.kind))
		})
	}
}

func TestChatDAO_DeleteMessageWithMedia_ReturnsThatMessagesFiles(t *testing.T) {
	// given two messages in the same room, each carrying media
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, user.ID, daotest.WithRoomMembers(user.ID))

	targetID := daotest.SendChatMessage(t, repos, roomID, user.ID, "mine")
	mediaAttach(t, repos, spec.NewMedia{TargetID: targetID, MediaURL: "/uploads/chat/target.webp", MediaType: "image", ThumbnailURL: "/uploads/chat/target_thumb.webp", SortOrder: 0})
	mediaAttach(t, repos, spec.NewMedia{TargetID: targetID, MediaURL: "/uploads/chat/target2.mp4", MediaType: "video", ThumbnailURL: "", SortOrder: 1})

	otherID := daotest.SendChatMessage(t, repos, roomID, user.ID, "theirs")
	mediaAttach(t, repos, spec.NewMedia{TargetID: otherID, MediaURL: "/uploads/chat/other.webp", MediaType: "image", ThumbnailURL: "/uploads/chat/other_thumb.webp", SortOrder: 0})

	// when
	paths, err := repos.Chat.DeleteMessageWithMedia(ctx, targetID)

	// then only the deleted message's files come back
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/uploads/chat/target.webp",
		"/uploads/chat/target_thumb.webp",
		"/uploads/chat/target2.mp4",
	}, paths)

	gone, err := repos.Chat.GetMessageByID(ctx, targetID)
	require.NoError(t, err)
	assert.Nil(t, gone)

	media, err := repos.Chat.GetMessageMediaBatch(ctx, []uuid.UUID{targetID, otherID})
	require.NoError(t, err)
	assert.NotContains(t, media, targetID)
	require.Len(t, media[otherID], 1)
	assert.Equal(t, "/uploads/chat/other.webp", media[otherID][0].MediaURL)
}
