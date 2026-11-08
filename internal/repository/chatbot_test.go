package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	chatbotRepoMocks struct {
		chatbots    *dao.MockChatbotDAO
		users       *MockUserRepository
		permissions *MockPermissionRepository
		basePrompts *MockChatbotBasePromptRepository
	}
)

func newChatbotRepo(t *testing.T) (ChatbotRepository, chatbotRepoMocks) {
	t.Helper()

	m := chatbotRepoMocks{
		chatbots:    dao.NewMockChatbotDAO(t),
		users:       NewMockUserRepository(t),
		permissions: NewMockPermissionRepository(t),
		basePrompts: NewMockChatbotBasePromptRepository(t),
	}

	return NewChatbotRepo(nil, m.chatbots, m.users, m.permissions, m.basePrompts), m
}

func TestChatbotRepository_DeleteBotDeletesItsAccountThroughTheUserRepository(t *testing.T) {
	// given a bot whose account deletion cascades its chatbot row
	repo, m := newChatbotRepo(t)
	botID, botUserID, promptID := uuid.New(), uuid.New(), uuid.New()
	m.chatbots.EXPECT().GetBot(mock.Anything, botID).Return(&model.Chatbot{ID: botID, UserID: botUserID, BasePromptID: &promptID}, nil).Once()
	m.users.EXPECT().AdminDeleteAccount(mock.Anything, botUserID).Return(nil).Once()
	m.basePrompts.EXPECT().Invalidate(mock.Anything, []uuid.UUID{promptID}).Once()

	// when
	err := repo.DeleteBot(context.Background(), botID)

	// then the user repository runs every user-delete invalidation, and the prompt's bot count is refreshed
	require.NoError(t, err)
}

func TestChatbotRepository_DeleteBotRejectsAnUnknownBot(t *testing.T) {
	// given
	repo, m := newChatbotRepo(t)
	botID := uuid.New()
	m.chatbots.EXPECT().GetBot(mock.Anything, botID).Return(nil, nil).Once()

	// when
	err := repo.DeleteBot(context.Background(), botID)

	// then no account is touched
	require.ErrorIs(t, err, daoerr.ErrBotNotFound)
}

func TestChatbotRepository_FailedAccountDeleteKeepsThePromptCache(t *testing.T) {
	// given
	repo, m := newChatbotRepo(t)
	botID, botUserID := uuid.New(), uuid.New()
	deleteErr := errors.New("db down")
	m.chatbots.EXPECT().GetBot(mock.Anything, botID).Return(&model.Chatbot{ID: botID, UserID: botUserID}, nil).Once()
	m.users.EXPECT().AdminDeleteAccount(mock.Anything, botUserID).Return(deleteErr).Once()

	// when
	err := repo.DeleteBot(context.Background(), botID)

	// then
	require.ErrorIs(t, err, deleteErr)
}

func TestChatbotRepository_CreateBotWithAccountRefreshesTheBadgeCacheAfterCommit(t *testing.T) {
	// given a bot created inside a caller's transaction
	repo, m := newChatbotRepo(t)
	tx := new(sql.Tx)
	botUserID, promptID := uuid.New(), uuid.New()
	s := spec.NewChatbotWithAccount{Account: spec.NewUser{Username: "bot"}, Bot: model.Chatbot{BasePromptID: &promptID}, VanityRoleID: "bot-badge"}
	m.users.EXPECT().Create(mock.Anything, s.Account, []*sql.Tx{tx}).Return(&model.User{ID: botUserID}, nil).Once()
	m.permissions.EXPECT().AssignVanityRoleToUser(mock.Anything, spec.VanityRoleAssignment{UserID: botUserID, RoleID: "bot-badge"}, []*sql.Tx{tx}).Return(nil).Once()
	m.chatbots.EXPECT().CreateBot(mock.Anything, model.Chatbot{UserID: botUserID, BasePromptID: &promptID}, []*sql.Tx{tx}).Return(&model.Chatbot{UserID: botUserID, BasePromptID: &promptID}, nil).Once()
	m.permissions.EXPECT().InvalidateVanityAssignments(mock.Anything, []string{botUserID.String()}).Once()
	m.basePrompts.EXPECT().Invalidate(mock.Anything, []uuid.UUID{promptID}).Once()

	// when
	created, err := repo.CreateBotWithAccount(context.Background(), s, tx)

	// then a read cached between the in-transaction assign and the commit is dropped again
	require.NoError(t, err)
	assert.Equal(t, botUserID, created.UserID)
}

func TestChatbotRepository_UpdateBotRefreshesTheOldAndNewPromptCounts(t *testing.T) {
	// given a bot moving from one base prompt to another
	repo, m := newChatbotRepo(t)
	botID, oldPrompt, newPrompt := uuid.New(), uuid.New(), uuid.New()
	bot := model.Chatbot{ID: botID, BasePromptID: &newPrompt}
	m.chatbots.EXPECT().GetBot(mock.Anything, botID).Return(&model.Chatbot{ID: botID, BasePromptID: &oldPrompt}, nil).Once()
	m.chatbots.EXPECT().UpdateBot(mock.Anything, bot).Return(&bot, nil).Once()
	m.basePrompts.EXPECT().Invalidate(mock.Anything, []uuid.UUID{oldPrompt, newPrompt}).Once()

	// when
	_, err := repo.UpdateBot(context.Background(), bot)

	// then
	require.NoError(t, err)
}
