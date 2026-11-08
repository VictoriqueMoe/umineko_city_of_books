package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"umineko_city_of_books/internal/dao"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newChatRepoWithBannedWords(t *testing.T) (ChatRepository, *dao.MockChatDAO, *MockChatBannedWordRepository) {
	t.Helper()

	chatDAO := dao.NewMockChatDAO(t)
	bannedWords := NewMockChatBannedWordRepository(t)

	return NewChatRepo(nil, chatDAO, nil, nil, bannedWords), chatDAO, bannedWords
}

func TestChatRepository_DeleteRoomWithMessages_DropsTheRoomsCachedBannedWords(t *testing.T) {
	// given a room whose banned words cascade away with it
	repo, chatDAO, bannedWords := newChatRepoWithBannedWords(t)
	roomID := uuid.New()
	tx := new(sql.Tx)
	chatDAO.EXPECT().ListRoomMediaURLs(mock.Anything, roomID, []*sql.Tx{tx}).Return([]string{"media.webp"}, nil).Once()
	chatDAO.EXPECT().ListRoomMemberAvatarURLs(mock.Anything, roomID, []*sql.Tx{tx}).Return([]string{"avatar.webp"}, nil).Once()
	chatDAO.EXPECT().DeleteMessages(mock.Anything, roomID, []*sql.Tx{tx}).Return(nil).Once()
	chatDAO.EXPECT().DeleteRoom(mock.Anything, roomID, []*sql.Tx{tx}).Return(nil).Once()
	bannedWords.EXPECT().InvalidateApplicable(mock.Anything, &roomID).Once()

	// when
	paths, err := repo.DeleteRoomWithMessages(context.Background(), roomID, tx)

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"media.webp", "avatar.webp"}, paths)
}

func TestChatRepository_DeleteRoomWithMessages_FailedDeleteKeepsTheBannedWordCache(t *testing.T) {
	// given
	repo, chatDAO, _ := newChatRepoWithBannedWords(t)
	roomID := uuid.New()
	tx := new(sql.Tx)
	deleteErr := errors.New("db down")
	chatDAO.EXPECT().ListRoomMediaURLs(mock.Anything, roomID, []*sql.Tx{tx}).Return(nil, nil).Once()
	chatDAO.EXPECT().ListRoomMemberAvatarURLs(mock.Anything, roomID, []*sql.Tx{tx}).Return(nil, nil).Once()
	chatDAO.EXPECT().DeleteMessages(mock.Anything, roomID, []*sql.Tx{tx}).Return(nil).Once()
	chatDAO.EXPECT().DeleteRoom(mock.Anything, roomID, []*sql.Tx{tx}).Return(deleteErr).Once()

	// when
	_, err := repo.DeleteRoomWithMessages(context.Background(), roomID, tx)

	// then the room still exists, so its banned words stay cached
	require.ErrorIs(t, err, deleteErr)
}
