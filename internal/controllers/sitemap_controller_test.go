package controllers

import (
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"umineko_city_of_books/internal/sitemap"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const sitemapBaseURL = "https://example.test"

func newSitemapApp(t *testing.T, svc sitemap.Service) *fiber.App {
	t.Helper()
	app := fiber.New()
	NewSitemapHandler(svc).Register(app)
	return app
}

func newSitemapMock(t *testing.T) *sitemap.MockService {
	t.Helper()
	m := sitemap.NewMockService(t)
	m.EXPECT().BaseURL().Return(sitemapBaseURL).Maybe()
	return m
}

func doSitemapRequest(t *testing.T, app *fiber.App, path string) (int, []byte, string) {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, body, resp.Header.Get("Content-Type")
}

func TestSitemap_Index_OK(t *testing.T) {
	// given
	svc := newSitemapMock(t)
	svc.EXPECT().IndexEntries().Return([]sitemap.IndexEntry{
		{Loc: sitemapBaseURL + "/sitemap-static.xml"},
		{Loc: sitemapBaseURL + "/sitemap-theories.xml"},
	})
	app := newSitemapApp(t, svc)

	// when
	status, body, contentType := doSitemapRequest(t, app, "/sitemap.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, contentType, "application/xml")
	var idx sitemapIndex
	require.NoError(t, xml.Unmarshal(body, &idx))
	assert.Equal(t, "http://www.sitemaps.org/schemas/sitemap/0.9", idx.XMLNS)
	require.Len(t, idx.Sitemaps, 2)
	assert.Equal(t, sitemapBaseURL+"/sitemap-static.xml", idx.Sitemaps[0].Loc)
}

func TestSitemap_Static_OK(t *testing.T) {
	// given
	svc := newSitemapMock(t)
	svc.EXPECT().StaticEntries(mock.Anything).Return([]sitemap.Entry{
		{URL: sitemapBaseURL, LastMod: "2026-05-28"},
		{URL: sitemapBaseURL + "/welcome", LastMod: "2026-05-28"},
	})
	app := newSitemapApp(t, svc)

	// when
	status, body, contentType := doSitemapRequest(t, app, "/sitemap-static.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, contentType, "application/xml")
	var set sitemapURLSet
	require.NoError(t, xml.Unmarshal(body, &set))
	require.Len(t, set.URLs, 2)
	assert.Equal(t, sitemapBaseURL, set.URLs[0].Loc)
	assert.Equal(t, "2026-05-28", set.URLs[0].LastMod)
}

func TestSitemap_Theories_OK(t *testing.T) {
	// given
	svc := newSitemapMock(t)
	svc.EXPECT().Theories(mock.Anything).Return([]sitemap.Entry{
		{URL: sitemapBaseURL + "/theory/theory-a", LastMod: "2024-01-02"},
		{URL: sitemapBaseURL + "/theory/theory-b", LastMod: "2024-02-03"},
	}, nil)
	app := newSitemapApp(t, svc)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-theories.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	var set sitemapURLSet
	require.NoError(t, xml.Unmarshal(body, &set))
	require.Len(t, set.URLs, 2)
	locs := map[string]string{}
	for _, u := range set.URLs {
		locs[u.Loc] = u.LastMod
	}
	assert.Equal(t, "2024-01-02", locs[sitemapBaseURL+"/theory/theory-a"])
	assert.Equal(t, "2024-02-03", locs[sitemapBaseURL+"/theory/theory-b"])
}

func TestSitemap_Theories_ServiceError(t *testing.T) {
	// given
	svc := newSitemapMock(t)
	svc.EXPECT().Theories(mock.Anything).Return(nil, errors.New("boom"))
	app := newSitemapApp(t, svc)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-theories.xml")

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to query theories")
}

func TestSitemap_Posts_OK(t *testing.T) {
	// given
	svc := newSitemapMock(t)
	svc.EXPECT().Posts(mock.Anything).Return([]sitemap.Entry{
		{URL: sitemapBaseURL + "/game-board/post-1", LastMod: "2024-05-01"},
	}, nil)
	app := newSitemapApp(t, svc)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-posts.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	var set sitemapURLSet
	require.NoError(t, xml.Unmarshal(body, &set))
	require.Len(t, set.URLs, 1)
	assert.Equal(t, sitemapBaseURL+"/game-board/post-1", set.URLs[0].Loc)
	assert.Equal(t, "2024-05-01", set.URLs[0].LastMod)
}

func TestSitemap_Posts_ServiceError(t *testing.T) {
	// given
	svc := newSitemapMock(t)
	svc.EXPECT().Posts(mock.Anything).Return(nil, errors.New("boom"))
	app := newSitemapApp(t, svc)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-posts.xml")

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to query posts")
}

func TestSitemap_Art_OK(t *testing.T) {
	// given
	svc := newSitemapMock(t)
	svc.EXPECT().Art(mock.Anything).Return([]sitemap.Entry{
		{URL: sitemapBaseURL + "/gallery/art/art-xyz", LastMod: "2024-06-07"},
	}, nil)
	app := newSitemapApp(t, svc)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-art.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	var set sitemapURLSet
	require.NoError(t, xml.Unmarshal(body, &set))
	require.Len(t, set.URLs, 1)
	assert.Equal(t, sitemapBaseURL+"/gallery/art/art-xyz", set.URLs[0].Loc)
}

func TestSitemap_Users_OK(t *testing.T) {
	// given
	svc := newSitemapMock(t)
	svc.EXPECT().Users(mock.Anything).Return([]sitemap.Entry{
		{URL: sitemapBaseURL + "/user/alice"},
		{URL: sitemapBaseURL + "/user/bob"},
	}, nil)
	app := newSitemapApp(t, svc)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-users.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	var set sitemapURLSet
	require.NoError(t, xml.Unmarshal(body, &set))
	require.Len(t, set.URLs, 2)
	locs := map[string]bool{}
	for _, u := range set.URLs {
		locs[u.Loc] = true
		assert.Empty(t, u.LastMod, "users sitemap should omit lastmod")
	}
	assert.True(t, locs[sitemapBaseURL+"/user/alice"])
	assert.True(t, locs[sitemapBaseURL+"/user/bob"])
}

func TestSitemap_Mysteries_OK(t *testing.T) {
	// given
	svc := newSitemapMock(t)
	svc.EXPECT().Mysteries(mock.Anything).Return([]sitemap.Entry{
		{URL: sitemapBaseURL + "/mystery/mystery-1", LastMod: "2024-07-08"},
	}, nil)
	app := newSitemapApp(t, svc)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-mysteries.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	var set sitemapURLSet
	require.NoError(t, xml.Unmarshal(body, &set))
	require.Len(t, set.URLs, 1)
	assert.Equal(t, sitemapBaseURL+"/mystery/mystery-1", set.URLs[0].Loc)
}

func TestSitemap_Ships_OK(t *testing.T) {
	// given
	svc := newSitemapMock(t)
	svc.EXPECT().Ships(mock.Anything).Return([]sitemap.Entry{
		{URL: sitemapBaseURL + "/ships/ship-1", LastMod: "2024-08-09"},
	}, nil)
	app := newSitemapApp(t, svc)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-ships.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	var set sitemapURLSet
	require.NoError(t, xml.Unmarshal(body, &set))
	require.Len(t, set.URLs, 1)
	assert.Equal(t, sitemapBaseURL+"/ships/ship-1", set.URLs[0].Loc)
}

func TestSitemap_Fanfics_OK(t *testing.T) {
	// given
	svc := newSitemapMock(t)
	svc.EXPECT().Fanfics(mock.Anything).Return([]sitemap.Entry{
		{URL: sitemapBaseURL + "/fanfiction/fic-1", LastMod: "2024-09-10"},
	}, nil)
	app := newSitemapApp(t, svc)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-fanfics.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	var set sitemapURLSet
	require.NoError(t, xml.Unmarshal(body, &set))
	require.Len(t, set.URLs, 1)
	assert.Equal(t, sitemapBaseURL+"/fanfiction/fic-1", set.URLs[0].Loc)
}

func TestSitemap_Journals_OK(t *testing.T) {
	// given
	svc := newSitemapMock(t)
	svc.EXPECT().Journals(mock.Anything).Return([]sitemap.Entry{
		{URL: sitemapBaseURL + "/journals/journal-1", LastMod: "2024-10-11"},
		{URL: sitemapBaseURL + "/journals/journal-1/entry/1", LastMod: "2024-10-11"},
	}, nil)
	app := newSitemapApp(t, svc)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-journals.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	var set sitemapURLSet
	require.NoError(t, xml.Unmarshal(body, &set))
	require.Len(t, set.URLs, 2)
}

func TestSitemap_Empty_ReturnsEmptyURLSet(t *testing.T) {
	// given
	svc := newSitemapMock(t)
	svc.EXPECT().Theories(mock.Anything).Return(nil, nil)
	app := newSitemapApp(t, svc)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-theories.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	var set sitemapURLSet
	require.NoError(t, xml.Unmarshal(body, &set))
	assert.Empty(t, set.URLs)
	assert.Equal(t, "http://www.sitemaps.org/schemas/sitemap/0.9", set.XMLNS)
}

func TestSitemap_ResponseHasXMLHeader(t *testing.T) {
	// given
	svc := newSitemapMock(t)
	svc.EXPECT().IndexEntries().Return(nil)
	app := newSitemapApp(t, svc)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), `<?xml version="1.0" encoding="UTF-8"?>`)
}

var _ = time.Time{}
