package controllers

import (
	"database/sql"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"umineko_city_of_books/internal/db/dbtest"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sitemapBaseURL = "https://example.test"

func newSitemapDB(t *testing.T) *sql.DB {
	t.Helper()
	db, _ := dbtest.NewEmptyDatabase(t)

	schema := []string{
		`CREATE TABLE theories  (id TEXT PRIMARY KEY, created_at TEXT)`,
		`CREATE TABLE posts     (id TEXT PRIMARY KEY, created_at TEXT)`,
		`CREATE TABLE art       (id TEXT PRIMARY KEY, created_at TEXT)`,
		`CREATE TABLE users     (username TEXT PRIMARY KEY, created_at TEXT)`,
		`CREATE TABLE mysteries (id TEXT PRIMARY KEY, created_at TEXT)`,
		`CREATE TABLE ships     (id TEXT PRIMARY KEY, created_at TEXT)`,
		`CREATE TABLE fanfics   (id TEXT PRIMARY KEY, created_at TEXT)`,
	}
	for _, stmt := range schema {
		_, err := db.Exec(stmt)
		require.NoError(t, err)
	}
	return db
}

func newSitemapApp(t *testing.T, db *sql.DB) *fiber.App {
	t.Helper()
	app := fiber.New()
	handler := NewSitemapHandler(db, sitemapBaseURL)
	handler.Register(app)
	return app
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
	db := newSitemapDB(t)
	app := newSitemapApp(t, db)

	// when
	status, body, contentType := doSitemapRequest(t, app, "/sitemap.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, contentType, "application/xml")

	var idx sitemapIndex
	require.NoError(t, xml.Unmarshal(body, &idx))
	assert.Equal(t, "http://www.sitemaps.org/schemas/sitemap/0.9", idx.XMLNS)
	require.Len(t, idx.Sitemaps, 8)

	want := []string{
		sitemapBaseURL + "/sitemap-static.xml",
		sitemapBaseURL + "/sitemap-theories.xml",
		sitemapBaseURL + "/sitemap-posts.xml",
		sitemapBaseURL + "/sitemap-art.xml",
		sitemapBaseURL + "/sitemap-users.xml",
		sitemapBaseURL + "/sitemap-mysteries.xml",
		sitemapBaseURL + "/sitemap-ships.xml",
		sitemapBaseURL + "/sitemap-fanfics.xml",
	}
	got := make([]string, 0, len(idx.Sitemaps))
	for _, s := range idx.Sitemaps {
		got = append(got, s.Loc)
	}
	assert.Equal(t, want, got)
}

func TestSitemap_Static_OK(t *testing.T) {
	// given
	db := newSitemapDB(t)
	app := newSitemapApp(t, db)

	// when
	status, body, contentType := doSitemapRequest(t, app, "/sitemap-static.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, contentType, "application/xml")

	var set sitemapURLSet
	require.NoError(t, xml.Unmarshal(body, &set))
	assert.Equal(t, "http://www.sitemaps.org/schemas/sitemap/0.9", set.XMLNS)
	require.NotEmpty(t, set.URLs)

	locs := make(map[string]bool, len(set.URLs))
	for _, u := range set.URLs {
		locs[u.Loc] = true
		assert.NotEmpty(t, u.LastMod, "static URL should have lastmod")
	}
	for _, expected := range []string{
		sitemapBaseURL,
		sitemapBaseURL + "/theories",
		sitemapBaseURL + "/game-board",
		sitemapBaseURL + "/gallery",
		sitemapBaseURL + "/quotes",
		sitemapBaseURL + "/mysteries",
		sitemapBaseURL + "/ships",
		sitemapBaseURL + "/fanfiction",
		sitemapBaseURL + "/suggestions",
		sitemapBaseURL + "/login",
	} {
		assert.Truef(t, locs[expected], "expected static sitemap to contain %q", expected)
	}
}

func TestSitemap_Theories_OK(t *testing.T) {
	// given
	db := newSitemapDB(t)
	_, err := db.Exec(
		`INSERT INTO theories (id, created_at) VALUES ($1, $2), ($3, $4)`,
		"theory-a", "2024-01-02 10:00:00",
		"theory-b", "2024-02-03 11:30:00",
	)
	require.NoError(t, err)
	app := newSitemapApp(t, db)

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

func TestSitemap_Theories_DBError(t *testing.T) {
	// given
	db := newSitemapDB(t)
	_, err := db.Exec(`DROP TABLE theories`)
	require.NoError(t, err)
	app := newSitemapApp(t, db)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-theories.xml")

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to query theories")
}

func TestSitemap_Posts_OK(t *testing.T) {
	// given
	db := newSitemapDB(t)
	_, err := db.Exec(
		`INSERT INTO posts (id, created_at) VALUES ($1, $2)`,
		"post-1", "2024-05-01 09:15:00",
	)
	require.NoError(t, err)
	app := newSitemapApp(t, db)

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

func TestSitemap_Posts_DBError(t *testing.T) {
	// given
	db := newSitemapDB(t)
	_, err := db.Exec(`DROP TABLE posts`)
	require.NoError(t, err)
	app := newSitemapApp(t, db)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-posts.xml")

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to query posts")
}

func TestSitemap_Art_OK(t *testing.T) {
	// given
	db := newSitemapDB(t)
	_, err := db.Exec(
		`INSERT INTO art (id, created_at) VALUES ($1, $2)`,
		"art-xyz", "2024-06-07 08:00:00",
	)
	require.NoError(t, err)
	app := newSitemapApp(t, db)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-art.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	var set sitemapURLSet
	require.NoError(t, xml.Unmarshal(body, &set))
	require.Len(t, set.URLs, 1)
	assert.Equal(t, sitemapBaseURL+"/gallery/art/art-xyz", set.URLs[0].Loc)
	assert.Equal(t, "2024-06-07", set.URLs[0].LastMod)
}

func TestSitemap_Art_DBError(t *testing.T) {
	// given
	db := newSitemapDB(t)
	_, err := db.Exec(`DROP TABLE art`)
	require.NoError(t, err)
	app := newSitemapApp(t, db)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-art.xml")

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to query art")
}

func TestSitemap_Users_OK(t *testing.T) {
	// given
	db := newSitemapDB(t)
	_, err := db.Exec(
		`INSERT INTO users (username, created_at) VALUES ($1, $2), ($3, $4)`,
		"alice", "2024-01-01 00:00:00",
		"bob", "2024-01-02 00:00:00",
	)
	require.NoError(t, err)
	app := newSitemapApp(t, db)

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

func TestSitemap_Users_DBError(t *testing.T) {
	// given
	db := newSitemapDB(t)
	_, err := db.Exec(`DROP TABLE users`)
	require.NoError(t, err)
	app := newSitemapApp(t, db)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-users.xml")

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to query users")
}

func TestSitemap_Mysteries_OK(t *testing.T) {
	// given
	db := newSitemapDB(t)
	_, err := db.Exec(
		`INSERT INTO mysteries (id, created_at) VALUES ($1, $2)`,
		"mystery-1", "2024-07-08 12:34:56",
	)
	require.NoError(t, err)
	app := newSitemapApp(t, db)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-mysteries.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	var set sitemapURLSet
	require.NoError(t, xml.Unmarshal(body, &set))
	require.Len(t, set.URLs, 1)
	assert.Equal(t, sitemapBaseURL+"/mystery/mystery-1", set.URLs[0].Loc)
	assert.Equal(t, "2024-07-08", set.URLs[0].LastMod)
}

func TestSitemap_Mysteries_DBError(t *testing.T) {
	// given
	db := newSitemapDB(t)
	_, err := db.Exec(`DROP TABLE mysteries`)
	require.NoError(t, err)
	app := newSitemapApp(t, db)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-mysteries.xml")

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to query mysteries")
}

func TestSitemap_Ships_OK(t *testing.T) {
	// given
	db := newSitemapDB(t)
	_, err := db.Exec(
		`INSERT INTO ships (id, created_at) VALUES ($1, $2)`,
		"ship-1", "2024-08-09 01:02:03",
	)
	require.NoError(t, err)
	app := newSitemapApp(t, db)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-ships.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	var set sitemapURLSet
	require.NoError(t, xml.Unmarshal(body, &set))
	require.Len(t, set.URLs, 1)
	assert.Equal(t, sitemapBaseURL+"/ships/ship-1", set.URLs[0].Loc)
	assert.Equal(t, "2024-08-09", set.URLs[0].LastMod)
}

func TestSitemap_Ships_DBError(t *testing.T) {
	// given
	db := newSitemapDB(t)
	_, err := db.Exec(`DROP TABLE ships`)
	require.NoError(t, err)
	app := newSitemapApp(t, db)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-ships.xml")

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to query ships")
}

func TestSitemap_Fanfics_OK(t *testing.T) {
	// given
	db := newSitemapDB(t)
	_, err := db.Exec(
		`INSERT INTO fanfics (id, created_at) VALUES ($1, $2)`,
		"fic-1", "2024-09-10 04:05:06",
	)
	require.NoError(t, err)
	app := newSitemapApp(t, db)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-fanfics.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	var set sitemapURLSet
	require.NoError(t, xml.Unmarshal(body, &set))
	require.Len(t, set.URLs, 1)
	assert.Equal(t, sitemapBaseURL+"/fanfiction/fic-1", set.URLs[0].Loc)
	assert.Equal(t, "2024-09-10", set.URLs[0].LastMod)
}

func TestSitemap_Fanfics_DBError(t *testing.T) {
	// given
	db := newSitemapDB(t)
	_, err := db.Exec(`DROP TABLE fanfics`)
	require.NoError(t, err)
	app := newSitemapApp(t, db)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap-fanfics.xml")

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to query fanfics")
}

func TestSitemap_Empty_ReturnsEmptyURLSet(t *testing.T) {
	// given
	db := newSitemapDB(t)
	app := newSitemapApp(t, db)

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
	db := newSitemapDB(t)
	app := newSitemapApp(t, db)

	// when
	status, body, _ := doSitemapRequest(t, app, "/sitemap.xml")

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), `<?xml version="1.0" encoding="UTF-8"?>`)
}
