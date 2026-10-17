package repository

import (
	"context"
	"database/sql"
	"testing"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	bannedWordWriteCase struct {
		name             string
		write            func(repo ChatBannedWordRepository, bannedWords *dao.MockChatBannedWordDAO, auditLog *MockAuditLogRepository) error
		wantGlobalReload bool
		wantRoomReload   bool
	}
)

func newCachedBannedWordRepo(t *testing.T) (ChatBannedWordRepository, *dao.MockChatBannedWordDAO, *MockAuditLogRepository) {
	t.Helper()

	bannedWords := dao.NewMockChatBannedWordDAO(t)
	auditLog := NewMockAuditLogRepository(t)

	return NewChatBannedWordRepo(nil, bannedWords, auditLog, cache.NewManager(engines.NewInMemory(0))), bannedWords, auditLog
}

func TestChatBannedWordRepo_ListApplicableServesRepeatSendsFromTheCache(t *testing.T) {
	// given
	repo, bannedWords, _ := newCachedBannedWordRepo(t)
	ctx := context.Background()
	roomID := uuid.New()
	global := model.ChatBannedWordRow{ID: uuid.New(), Scope: "global", Pattern: "witch", MatchMode: "substring", Action: "delete"}
	room := model.ChatBannedWordRow{ID: uuid.New(), Scope: "room", RoomID: &roomID, Pattern: "cat", MatchMode: "whole_word", Action: "kick"}
	bannedWords.EXPECT().ListGlobal(mock.Anything).Return([]model.ChatBannedWordRow{global}, nil).Once()
	bannedWords.EXPECT().ListForRoom(mock.Anything, roomID).Return([]model.ChatBannedWordRow{room}, nil).Once()

	// when
	first, firstErr := repo.ListApplicable(ctx, roomID)
	second, secondErr := repo.ListApplicable(ctx, roomID)

	// then
	require.NoError(t, firstErr)
	require.NoError(t, secondErr)
	assert.Equal(t, []model.ChatBannedWordRow{global, room}, first)
	assert.Equal(t, first, second)
}

func TestChatBannedWordRepo_WritesDropOnlyTheListTheyChanged(t *testing.T) {
	roomID := uuid.New()
	ruleID := uuid.New()
	tx := new(sql.Tx)
	globalRule := &model.ChatBannedWordRow{ID: ruleID, Scope: "global", Pattern: "witch"}
	roomRule := &model.ChatBannedWordRow{ID: ruleID, Scope: "room", RoomID: &roomID, Pattern: "cat"}
	deletion := spec.ChatBannedWordDeletion{ID: ruleID, Audit: audit.NewEntry{Action: audit.ActionChatRoomBannedWordDelete}}
	creation := spec.ChatBannedWordCreation{Scope: "global", Pattern: "witch", Audit: audit.NewEntry{Action: audit.ActionChatGlobalBannedWordCreate}}

	cases := []bannedWordWriteCase{
		{
			name: "creating a global rule reloads the global list",
			write: func(repo ChatBannedWordRepository, bannedWords *dao.MockChatBannedWordDAO, _ *MockAuditLogRepository) error {
				bannedWords.EXPECT().Create(mock.Anything, spec.ChatBannedWordSpec{Scope: "global", Pattern: "witch"}).Return(globalRule, nil)

				_, err := repo.Create(context.Background(), spec.ChatBannedWordSpec{Scope: "global", Pattern: "witch"})

				return err
			},
			wantGlobalReload: true,
		},
		{
			name: "creating a global rule with its audit entry reloads the global list",
			write: func(repo ChatBannedWordRepository, bannedWords *dao.MockChatBannedWordDAO, auditLog *MockAuditLogRepository) error {
				bannedWords.EXPECT().Create(mock.Anything, creation.ChatBannedWordSpec, []*sql.Tx{tx}).Return(globalRule, nil)
				auditLog.EXPECT().Create(mock.Anything, audit.NewEntry{Action: audit.ActionChatGlobalBannedWordCreate, TargetID: ruleID.String()}, []*sql.Tx{tx}).Return(nil)

				_, err := repo.CreateWithAudit(context.Background(), creation, tx)

				return err
			},
			wantGlobalReload: true,
		},
		{
			name: "editing a room rule reloads that room's list",
			write: func(repo ChatBannedWordRepository, bannedWords *dao.MockChatBannedWordDAO, _ *MockAuditLogRepository) error {
				update := spec.ChatBannedWordUpdate{ID: ruleID, Pattern: "dog"}
				bannedWords.EXPECT().GetByID(mock.Anything, ruleID).Return(roomRule, nil)
				bannedWords.EXPECT().Update(mock.Anything, update).Return(nil)

				return repo.Update(context.Background(), update)
			},
			wantRoomReload: true,
		},
		{
			name: "deleting a global rule reloads the global list",
			write: func(repo ChatBannedWordRepository, bannedWords *dao.MockChatBannedWordDAO, _ *MockAuditLogRepository) error {
				bannedWords.EXPECT().GetByID(mock.Anything, ruleID).Return(globalRule, nil)
				bannedWords.EXPECT().Delete(mock.Anything, ruleID).Return(nil)

				return repo.Delete(context.Background(), ruleID)
			},
			wantGlobalReload: true,
		},
		{
			name: "deleting a room rule with its audit entry reloads that room's list",
			write: func(repo ChatBannedWordRepository, bannedWords *dao.MockChatBannedWordDAO, auditLog *MockAuditLogRepository) error {
				bannedWords.EXPECT().GetByID(mock.Anything, ruleID, []*sql.Tx{tx}).Return(roomRule, nil)
				bannedWords.EXPECT().Delete(mock.Anything, ruleID, []*sql.Tx{tx}).Return(nil)
				auditLog.EXPECT().Create(mock.Anything, deletion.Audit, []*sql.Tx{tx}).Return(nil)

				return repo.DeleteWithAudit(context.Background(), deletion, tx)
			},
			wantRoomReload: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a warm cache for the room
			repo, bannedWords, auditLog := newCachedBannedWordRepo(t)
			ctx := context.Background()
			globalLoads := 1
			if tc.wantGlobalReload {
				globalLoads = 2
			}
			roomLoads := 1
			if tc.wantRoomReload {
				roomLoads = 2
			}
			bannedWords.EXPECT().ListGlobal(mock.Anything).Return(nil, nil).Times(globalLoads)
			bannedWords.EXPECT().ListForRoom(mock.Anything, roomID).Return(nil, nil).Times(roomLoads)
			_, err := repo.ListApplicable(ctx, roomID)
			require.NoError(t, err)

			// when
			writeErr := tc.write(repo, bannedWords, auditLog)
			_, readErr := repo.ListApplicable(ctx, roomID)

			// then
			require.NoError(t, writeErr)
			require.NoError(t, readErr)
		})
	}
}
