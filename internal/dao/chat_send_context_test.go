package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatDAO_GetRoomSendContext(t *testing.T) {
	tests := []struct {
		name           string
		create         func(t *testing.T, repos *repository.Repositories, roomID, creatorID uuid.UUID)
		wantName       string
		wantType       string
		wantIsSystem   bool
		wantSystemKind string
	}{
		{
			name: "group room carries name, type and creator",
			create: func(t *testing.T, repos *repository.Repositories, roomID, creatorID uuid.UUID) {
				require.NoError(t, repos.Chat.CreateRoom(context.Background(), roomID, "Room", "desc", "group", true, false, creatorID))
			},
			wantName: "Room",
			wantType: "group",
		},
		{
			name: "live stream room carries its system kind",
			create: func(t *testing.T, repos *repository.Repositories, roomID, creatorID uuid.UUID) {
				require.NoError(t, repos.Chat.CreateSystemRoom(context.Background(), roomID, "My stream", "", "live_stream", creatorID))
			},
			wantName:       "My stream",
			wantType:       "group",
			wantIsSystem:   true,
			wantSystemKind: "live_stream",
		},
		{
			name: "mods room carries its system kind",
			create: func(t *testing.T, repos *repository.Repositories, roomID, creatorID uuid.UUID) {
				require.NoError(t, repos.Chat.CreateSystemRoom(context.Background(), roomID, "Mods", "", "mods", creatorID))
			},
			wantName:       "Mods",
			wantType:       "group",
			wantIsSystem:   true,
			wantSystemKind: "mods",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			creator := daotest.CreateUser(t, repos)
			roomID := uuid.New()
			tt.create(t, repos, roomID, creator.ID)

			// when
			got, err := repos.Chat.GetRoomSendContext(ctx, roomID)

			// then
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, roomID, got.ID)
			assert.Equal(t, tt.wantName, got.Name)
			assert.Equal(t, tt.wantType, got.Type)
			assert.Equal(t, tt.wantIsSystem, got.IsSystem)
			assert.Equal(t, tt.wantSystemKind, got.SystemKind)
			assert.Equal(t, creator.ID, got.CreatedBy)
		})
	}
}

func TestChatDAO_GetRoomSendContext_DMRoom(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	userA := daotest.CreateUser(t, repos)
	userB := daotest.CreateUser(t, repos)
	roomID, err := repos.Chat.CreateDMRoomAtomic(ctx, uuid.New(), userA.ID, userB.ID)
	require.NoError(t, err)

	// when
	got, err := repos.Chat.GetRoomSendContext(ctx, roomID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "dm", got.Type)
	assert.False(t, got.IsSystem)
	assert.Empty(t, got.SystemKind)
}

func TestChatDAO_GetRoomSendContext_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()

	// when
	got, err := repos.Chat.GetRoomSendContext(ctx, uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestChatDAO_GetRoomSendContext_MatchesGetRoomByID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	creator := daotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateSystemRoom(ctx, roomID, "My stream", "", "live_stream", creator.ID))

	// when
	full, err := repos.Chat.GetRoomByID(ctx, roomID, creator.ID)
	require.NoError(t, err)
	lean, err := repos.Chat.GetRoomSendContext(ctx, roomID)
	require.NoError(t, err)

	// then
	require.NotNil(t, full)
	require.NotNil(t, lean)
	assert.Equal(t, full.ID, lean.ID)
	assert.Equal(t, full.Name, lean.Name)
	assert.Equal(t, full.Type, lean.Type)
	assert.Equal(t, full.IsSystem, lean.IsSystem)
	assert.Equal(t, full.SystemKind, lean.SystemKind)
	assert.Equal(t, full.CreatedBy, lean.CreatedBy)
}
