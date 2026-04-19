package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatBannedWordRepository_CreateUpdateDelete(t *testing.T) {
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()

	id, err := repos.ChatBannedWord.Create(ctx, repository.ChatBannedWordSpec{
		Scope:         "global",
		Pattern:       "old",
		MatchMode:     "substring",
		CaseSensitive: false,
		Action:        "delete",
		CreatedBy:     &user.ID,
	})
	require.NoError(t, err)

	got, err := repos.ChatBannedWord.GetByID(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "old", got.Pattern)
	assert.Equal(t, "delete", got.Action)
	assert.False(t, got.CaseSensitive)

	err = repos.ChatBannedWord.Update(ctx, id, repository.ChatBannedWordUpdate{
		Pattern:       "new",
		MatchMode:     "whole_word",
		CaseSensitive: true,
		Action:        "kick",
	})
	require.NoError(t, err)

	updated, err := repos.ChatBannedWord.GetByID(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "new", updated.Pattern)
	assert.Equal(t, "whole_word", updated.MatchMode)
	assert.True(t, updated.CaseSensitive)
	assert.Equal(t, "kick", updated.Action)

	require.NoError(t, repos.ChatBannedWord.Delete(ctx, id))
	gone, err := repos.ChatBannedWord.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Nil(t, gone)
}

func TestChatBannedWordRepository_Update_MissingRow(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	err := repos.ChatBannedWord.Update(ctx, uuid.New(), repository.ChatBannedWordUpdate{
		Pattern: "x", MatchMode: "substring", Action: "delete",
	})
	require.Error(t, err)
}
