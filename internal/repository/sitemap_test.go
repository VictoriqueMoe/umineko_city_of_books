package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository/repotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSitemapRepository_ListPosts(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "body")

	// when
	entries, err := repos.Sitemap.ListPosts(context.Background())

	// then
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, postID.String(), entries[0].ID)
	assert.False(t, entries[0].LastMod.IsZero())
}

func TestSitemapRepository_ListUsernames(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	usernames, err := repos.Sitemap.ListUsernames(context.Background())

	// then
	require.NoError(t, err)
	assert.Contains(t, usernames, user.Username)
}

func TestSitemapRepository_ListJournalRows_NullUpdatedAt(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	journalID, err := repos.Journal.Create(context.Background(), user.ID, dto.CreateJournalRequest{Title: "Reading Umineko"})
	require.NoError(t, err)

	// when
	rows, err := repos.Sitemap.ListJournalRows(context.Background())

	// then
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, journalID.String(), rows[0].JournalID)
	assert.False(t, rows[0].JournalUpdatedAt.IsZero())
}

func TestSitemapRepository_ListPosts_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	entries, err := repos.Sitemap.ListPosts(context.Background())

	// then
	require.NoError(t, err)
	assert.Empty(t, entries)
}
