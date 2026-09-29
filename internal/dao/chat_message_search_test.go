package dao_test

import (
	"context"
	"slices"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	searchHit struct {
		messageID string
		roomID    string
	}
)

func TestChatDAO_SearchMessagesForViewer(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	viewer := daotest.CreateUser(t, repos)
	owner := daotest.CreateUser(t, repos)
	outsider := daotest.CreateUser(t, repos)

	roomA := daotest.CreateChatRoom(t, repos, viewer.ID, daotest.WithRoomMembers(viewer.ID))
	roomB := daotest.CreateChatRoom(t, repos, viewer.ID, daotest.WithRoomMembers(viewer.ID))
	privateRoom := daotest.CreateChatRoom(t, repos, owner.ID, daotest.WithRoomMembers(owner.ID))

	matchA := daotest.SendChatMessage(t, repos, roomA, viewer.ID, "the golden witch beatrice laughs")
	daotest.SendChatMessage(t, repos, roomA, viewer.ID, "an ordinary mundane lunch")
	_, err := repos.Chat.InsertSystemMessage(ctx, spec.NewChatMessage{RoomID: roomA, SenderID: viewer.ID, Body: "beatrice joined the room"})
	require.NoError(t, err)
	matchB := daotest.SendChatMessage(t, repos, roomB, viewer.ID, "beatrice falls")
	privateMatch := daotest.SendChatMessage(t, repos, privateRoom, owner.ID, "secret beatrice plans")

	tests := []struct {
		name     string
		viewerID uuid.UUID
		roomID   uuid.UUID
		want     []searchHit
	}{
		{
			name:     "all member rooms return only real matches",
			viewerID: viewer.ID,
			roomID:   uuid.Nil,
			want:     []searchHit{{messageID: matchA.String(), roomID: roomA.String()}, {messageID: matchB.String(), roomID: roomB.String()}},
		},
		{
			name:     "room filter scopes to one room",
			viewerID: viewer.ID,
			roomID:   roomA,
			want:     []searchHit{{messageID: matchA.String(), roomID: roomA.String()}},
		},
		{
			name:     "private room member finds its own message",
			viewerID: owner.ID,
			roomID:   uuid.Nil,
			want:     []searchHit{{messageID: privateMatch.String(), roomID: privateRoom.String()}},
		},
		{
			name:     "viewer with no memberships sees nothing",
			viewerID: outsider.ID,
			roomID:   uuid.Nil,
			want:     nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			results, total, err := repos.Chat.SearchMessagesForViewer(ctx, spec.ChatMessageSearch{ViewerID: tc.viewerID, RoomID: tc.roomID, Query: "beatrice", Limit: 20, Offset: 0})

			// then
			require.NoError(t, err)
			assert.Equal(t, len(tc.want), total)

			got := make([]searchHit, 0, len(results))
			for _, r := range results {
				require.NotNil(t, r.ParentID)
				hit := searchHit{messageID: r.ID, roomID: *r.ParentID}
				got = append(got, hit)

				roomID, err := uuid.Parse(hit.roomID)
				require.NoError(t, err)

				before, err := repos.Chat.GetMessagesBefore(ctx, spec.ChatMessageCursorPage{RoomID: roomID, ViewerID: uuid.Nil, Before: r.CreatedAt + "|ffffffff-ffff-ffff-ffff-ffffffffffff", Limit: 50})
				require.NoError(t, err)
				assert.True(t, slices.ContainsFunc(before, func(m model.ChatMessageRow) bool {
					return m.ID.String() == hit.messageID
				}), "jump cursor built from the search created_at must include the target message")
			}

			assert.ElementsMatch(t, tc.want, got)
		})
	}
}
