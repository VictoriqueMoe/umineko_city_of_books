package favourite

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*service, *repository.MockGiphyFavouriteRepository) {
	repo := repository.NewMockGiphyFavouriteRepository(t)
	svc := NewService(repo).(*service)
	return svc, repo
}

func TestAdd_RejectsMissingGiphyID(t *testing.T) {
	svc, _ := newTestService(t)
	err := svc.Add(context.Background(), uuid.New(), Favourite{URL: "https://media.giphy.com/abc.gif"})
	assert.ErrorIs(t, err, ErrGiphyIDRequired)
}

func TestAdd_RejectsMissingURL(t *testing.T) {
	svc, _ := newTestService(t)
	err := svc.Add(context.Background(), uuid.New(), Favourite{GiphyID: "abc"})
	assert.ErrorIs(t, err, ErrURLRequired)
}

func TestAdd_PassesAllFieldsToRepo(t *testing.T) {
	svc, repo := newTestService(t)
	userID := uuid.New()
	fav := Favourite{
		GiphyID:    "abc",
		URL:        "https://media.giphy.com/abc.gif",
		Title:      "cat",
		PreviewURL: "https://media.giphy.com/abc-preview.gif",
		Width:      200,
		Height:     150,
	}
	repo.EXPECT().Add(mock.Anything, userID, repository.GiphyFavourite{
		GiphyID:    "abc",
		URL:        "https://media.giphy.com/abc.gif",
		Title:      "cat",
		PreviewURL: "https://media.giphy.com/abc-preview.gif",
		Width:      200,
		Height:     150,
	}).Return(nil)

	require.NoError(t, svc.Add(context.Background(), userID, fav))
}

func TestAdd_PropagatesRepoError(t *testing.T) {
	svc, repo := newTestService(t)
	repo.EXPECT().Add(mock.Anything, mock.Anything, mock.Anything).Return(errors.New("boom"))

	err := svc.Add(context.Background(), uuid.New(), Favourite{GiphyID: "x", URL: "https://x"})
	assert.EqualError(t, err, "boom")
}

func TestRemove_DelegatesToRepo(t *testing.T) {
	svc, repo := newTestService(t)
	userID := uuid.New()
	repo.EXPECT().Remove(mock.Anything, userID, "abc").Return(nil)

	require.NoError(t, svc.Remove(context.Background(), userID, "abc"))
}

func TestList_MapsRepoRowsAndTotal(t *testing.T) {
	svc, repo := newTestService(t)
	userID := uuid.New()
	repo.EXPECT().List(mock.Anything, userID, 10, 5).Return([]repository.GiphyFavourite{
		{GiphyID: "a", URL: "urlA", Title: "A", PreviewURL: "pA", Width: 100, Height: 50},
		{GiphyID: "b", URL: "urlB", Title: "B", PreviewURL: "pB", Width: 200, Height: 75},
	}, 42, nil)

	favs, total, err := svc.List(context.Background(), userID, 10, 5)
	require.NoError(t, err)
	assert.Equal(t, 42, total)
	assert.Equal(t, []Favourite{
		{GiphyID: "a", URL: "urlA", Title: "A", PreviewURL: "pA", Width: 100, Height: 50},
		{GiphyID: "b", URL: "urlB", Title: "B", PreviewURL: "pB", Width: 200, Height: 75},
	}, favs)
}

func TestList_PropagatesRepoError(t *testing.T) {
	svc, repo := newTestService(t)
	repo.EXPECT().List(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, 0, errors.New("db down"))

	_, _, err := svc.List(context.Background(), uuid.New(), 10, 0)
	assert.EqualError(t, err, "db down")
}

func TestListIDs_DelegatesToRepo(t *testing.T) {
	svc, repo := newTestService(t)
	userID := uuid.New()
	repo.EXPECT().ListIDs(mock.Anything, userID).Return([]string{"a", "b"}, nil)

	ids, err := svc.ListIDs(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, ids)
}
