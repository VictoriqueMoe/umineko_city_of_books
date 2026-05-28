package sitemap_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/sitemap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testBaseURL = "https://example.test"

func newTestService(t *testing.T) (sitemap.Service, *repository.MockSitemapRepository, *settings.MockService) {
	t.Helper()
	repo := repository.NewMockSitemapRepository(t)
	settingsSvc := settings.NewMockService(t)
	return sitemap.NewService(repo, settingsSvc, testBaseURL), repo, settingsSvc
}

func TestService_BaseURL(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)

	// when
	got := svc.BaseURL()

	// then
	assert.Equal(t, testBaseURL, got)
}

func TestService_IndexEntries_HasAllSitemaps(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)

	// when
	entries := svc.IndexEntries()

	// then
	require.Len(t, entries, 9)
	assert.Equal(t, testBaseURL+"/sitemap-static.xml", entries[0].Loc)
	assert.Equal(t, testBaseURL+"/sitemap-journals.xml", entries[8].Loc)
}

func TestService_StaticEntries_NoRulesPage(t *testing.T) {
	// given
	svc, _, settingsSvc := newTestService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingRulesPage).Return("")

	// when
	entries := svc.StaticEntries(context.Background())

	// then
	urls := make(map[string]bool)
	for _, e := range entries {
		urls[e.URL] = true
		assert.NotEmpty(t, e.LastMod)
	}
	assert.True(t, urls[testBaseURL])
	assert.True(t, urls[testBaseURL+"/login"])
	assert.False(t, urls[testBaseURL+"/rules"])
}

func TestService_StaticEntries_WithRulesPage(t *testing.T) {
	// given
	svc, _, settingsSvc := newTestService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingRulesPage).Return("Rules go here")

	// when
	entries := svc.StaticEntries(context.Background())

	// then
	urls := make(map[string]bool)
	for _, e := range entries {
		urls[e.URL] = true
	}
	assert.True(t, urls[testBaseURL+"/rules"])
}

func TestService_Theories_BuildsURLs(t *testing.T) {
	// given
	svc, repo, _ := newTestService(t)
	when := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)
	repo.EXPECT().ListTheories(mock.Anything).Return([]repository.SitemapEntry{
		{ID: "theory-a", LastMod: when},
	}, nil)

	// when
	entries, err := svc.Theories(context.Background())

	// then
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, testBaseURL+"/theory/theory-a", entries[0].URL)
	assert.Equal(t, "2024-01-02", entries[0].LastMod)
}

func TestService_Theories_RepoError(t *testing.T) {
	// given
	svc, repo, _ := newTestService(t)
	repo.EXPECT().ListTheories(mock.Anything).Return(nil, errors.New("boom"))

	// when
	_, err := svc.Theories(context.Background())

	// then
	require.Error(t, err)
}

func TestService_Users_OmitsLastMod(t *testing.T) {
	// given
	svc, repo, _ := newTestService(t)
	repo.EXPECT().ListUsernames(mock.Anything).Return([]string{"alice", "bob"}, nil)

	// when
	entries, err := svc.Users(context.Background())

	// then
	require.NoError(t, err)
	require.Len(t, entries, 2)
	for _, e := range entries {
		assert.Empty(t, e.LastMod)
	}
}

func TestService_Journals_DedupesJournalAddsEntries(t *testing.T) {
	// given
	svc, repo, _ := newTestService(t)
	jUpdated := time.Date(2024, 3, 4, 0, 0, 0, 0, time.UTC)
	eUpdated := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)
	repo.EXPECT().ListJournalRows(mock.Anything).Return([]repository.SitemapJournalRow{
		{
			JournalID:        "journal-1",
			JournalUpdatedAt: jUpdated,
			EntryNumber:      sql.NullInt64{Valid: true, Int64: 1},
			EntryUpdatedAt:   sql.NullTime{Valid: true, Time: eUpdated},
		},
		{
			JournalID:        "journal-1",
			JournalUpdatedAt: jUpdated,
			EntryNumber:      sql.NullInt64{Valid: true, Int64: 2},
			EntryUpdatedAt:   sql.NullTime{Valid: false},
		},
	}, nil)

	// when
	entries, err := svc.Journals(context.Background())

	// then
	require.NoError(t, err)
	require.Len(t, entries, 3)
	assert.Equal(t, testBaseURL+"/journals/journal-1", entries[0].URL)
	assert.Equal(t, "2024-03-04", entries[0].LastMod)
	assert.Equal(t, testBaseURL+"/journals/journal-1/entry/1", entries[1].URL)
	assert.Equal(t, "2024-03-05", entries[1].LastMod)
	assert.Equal(t, testBaseURL+"/journals/journal-1/entry/2", entries[2].URL)
	assert.Equal(t, "2024-03-04", entries[2].LastMod)
}
