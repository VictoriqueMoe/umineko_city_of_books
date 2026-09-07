package repository

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	valkeymock "github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
)

func newCachedChatbotRepo(t *testing.T) (ChatbotRepository, *dao.MockChatbotDAO, *valkeymock.Client) {
	t.Helper()

	client := valkeymock.NewClient(gomock.NewController(t))
	chatbotDAO := dao.NewMockChatbotDAO(t)
	basePrompts := NewMockBasePromptInvalidator(t)
	basePrompts.EXPECT().InvalidateList(mock.Anything).Return().Maybe()

	return NewChatbotRepo(nil, chatbotDAO, nil, nil, basePrompts, cache.NewManager(engines.NewValkeyWithClient(client))), chatbotDAO, client
}

func TestDeleteBot_InvalidatesBotUserKeys(t *testing.T) {
	botID, botUserID, otherID := uuid.New(), uuid.New(), uuid.New()

	tests := []struct {
		name     string
		bots     []model.Chatbot
		listErr  error
		expected []string
	}{
		{
			name:    "bot user resolved",
			bots:    []model.Chatbot{{ID: otherID, UserID: uuid.New()}, {ID: botID, UserID: botUserID}},
			listErr: nil,
			expected: []string{
				"DEL",
				cache.VanityAssignments.Key(),
				cache.UserRole.Key(botUserID.String()),
				cache.UserVanityRoleIDs.Key(botUserID.String()),
			},
		},
		{
			name:     "lookup fails so only the site wide key is busted",
			bots:     nil,
			listErr:  errors.New("db down"),
			expected: []string{"DEL", cache.VanityAssignments.Key()},
		},
		{
			name:     "bot missing from listing so only the site wide key is busted",
			bots:     []model.Chatbot{{ID: otherID, UserID: uuid.New()}},
			listErr:  nil,
			expected: []string{"DEL", cache.VanityAssignments.Key()},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo, chatbotDAO, client := newCachedChatbotRepo(t)
			chatbotDAO.EXPECT().ListBots(mock.Anything).Return(tc.bots, tc.listErr)
			chatbotDAO.EXPECT().DeleteBot(mock.Anything, botID).Return(nil)

			var commands []string
			captureDel(client, &commands)

			// when
			err := repo.DeleteBot(context.Background(), botID)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.expected, commands)
		})
	}
}

func TestDeleteBot_DaoErrorSkipsInvalidation(t *testing.T) {
	// given
	repo, chatbotDAO, client := newCachedChatbotRepo(t)
	botID := uuid.New()
	chatbotDAO.EXPECT().ListBots(mock.Anything).Return(nil, nil)
	chatbotDAO.EXPECT().DeleteBot(mock.Anything, botID).Return(dao.ErrBotNotFound)
	client.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

	// when
	err := repo.DeleteBot(context.Background(), botID)

	// then
	require.ErrorIs(t, err, dao.ErrBotNotFound)
}
