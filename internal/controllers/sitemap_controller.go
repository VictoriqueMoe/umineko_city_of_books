package controllers

import (
	"database/sql"
	"encoding/xml"
	"time"

	"github.com/gofiber/fiber/v3"
)

type (
	sitemapURL struct {
		XMLName xml.Name `xml:"url"`
		Loc     string   `xml:"loc"`
		LastMod string   `xml:"lastmod,omitempty"`
	}

	sitemapURLSet struct {
		XMLName xml.Name     `xml:"urlset"`
		XMLNS   string       `xml:"xmlns,attr"`
		URLs    []sitemapURL `xml:"url"`
	}

	sitemapIndex struct {
		XMLName  xml.Name          `xml:"sitemapindex"`
		XMLNS    string            `xml:"xmlns,attr"`
		Sitemaps []sitemapIndexURL `xml:"sitemap"`
	}

	sitemapIndexURL struct {
		XMLName xml.Name `xml:"sitemap"`
		Loc     string   `xml:"loc"`
	}

	SitemapHandler struct {
		db      *sql.DB
		baseURL string
	}
)

func NewSitemapHandler(db *sql.DB, baseURL string) *SitemapHandler {
	return &SitemapHandler{db: db, baseURL: baseURL}
}

func (h *SitemapHandler) Register(app fiber.Router) {
	app.Get("/sitemap.xml", h.index)
	app.Get("/sitemap-static.xml", h.static)
	app.Get("/sitemap-theories.xml", h.theories)
	app.Get("/sitemap-posts.xml", h.posts)
	app.Get("/sitemap-art.xml", h.art)
	app.Get("/sitemap-users.xml", h.users)
}

func (h *SitemapHandler) sendXML(ctx fiber.Ctx, v interface{}) error {
	out, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).SendString("failed to generate sitemap")
	}

	ctx.Set("Content-Type", "application/xml; charset=utf-8")
	return ctx.Send(append([]byte(xml.Header), out...))
}

func (h *SitemapHandler) index(ctx fiber.Ctx) error {
	idx := sitemapIndex{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		Sitemaps: []sitemapIndexURL{
			{Loc: h.baseURL + "/sitemap-static.xml"},
			{Loc: h.baseURL + "/sitemap-theories.xml"},
			{Loc: h.baseURL + "/sitemap-posts.xml"},
			{Loc: h.baseURL + "/sitemap-art.xml"},
			{Loc: h.baseURL + "/sitemap-users.xml"},
		},
	}
	return h.sendXML(ctx, idx)
}

func (h *SitemapHandler) static(ctx fiber.Ctx) error {
	now := time.Now().Format("2006-01-02")
	set := sitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs: []sitemapURL{
			{Loc: h.baseURL, LastMod: now},
			{Loc: h.baseURL + "/theories", LastMod: now},
			{Loc: h.baseURL + "/game-board", LastMod: now},
			{Loc: h.baseURL + "/game-board/umineko", LastMod: now},
			{Loc: h.baseURL + "/game-board/higurashi", LastMod: now},
			{Loc: h.baseURL + "/game-board/ciconia", LastMod: now},
			{Loc: h.baseURL + "/gallery", LastMod: now},
			{Loc: h.baseURL + "/gallery/umineko", LastMod: now},
			{Loc: h.baseURL + "/gallery/higurashi", LastMod: now},
			{Loc: h.baseURL + "/gallery/ciconia", LastMod: now},
			{Loc: h.baseURL + "/quotes", LastMod: now},
			{Loc: h.baseURL + "/login", LastMod: now},
		},
	}
	return h.sendXML(ctx, set)
}

func (h *SitemapHandler) theories(ctx fiber.Ctx) error {
	rows, err := h.db.QueryContext(ctx.Context(),
		`SELECT id, created_at FROM theories ORDER BY created_at DESC`)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).SendString("failed to query theories")
	}
	defer rows.Close()

	var urls []sitemapURL
	for rows.Next() {
		var id, createdAt string
		if err := rows.Scan(&id, &createdAt); err != nil {
			continue
		}
		t, _ := time.Parse("2006-01-02 15:04:05", createdAt)
		urls = append(urls, sitemapURL{
			Loc:     h.baseURL + "/theory/" + id,
			LastMod: t.Format("2006-01-02"),
		})
	}

	return h.sendXML(ctx, sitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	})
}

func (h *SitemapHandler) posts(ctx fiber.Ctx) error {
	rows, err := h.db.QueryContext(ctx.Context(),
		`SELECT id, created_at FROM posts ORDER BY created_at DESC`)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).SendString("failed to query posts")
	}
	defer rows.Close()

	var urls []sitemapURL
	for rows.Next() {
		var id, createdAt string
		if err := rows.Scan(&id, &createdAt); err != nil {
			continue
		}
		t, _ := time.Parse("2006-01-02 15:04:05", createdAt)
		urls = append(urls, sitemapURL{
			Loc:     h.baseURL + "/game-board/" + id,
			LastMod: t.Format("2006-01-02"),
		})
	}

	return h.sendXML(ctx, sitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	})
}

func (h *SitemapHandler) art(ctx fiber.Ctx) error {
	rows, err := h.db.QueryContext(ctx.Context(),
		`SELECT id, created_at FROM art ORDER BY created_at DESC`)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).SendString("failed to query art")
	}
	defer rows.Close()

	var urls []sitemapURL
	for rows.Next() {
		var id, createdAt string
		if err := rows.Scan(&id, &createdAt); err != nil {
			continue
		}
		t, _ := time.Parse("2006-01-02 15:04:05", createdAt)
		urls = append(urls, sitemapURL{
			Loc:     h.baseURL + "/gallery/art/" + id,
			LastMod: t.Format("2006-01-02"),
		})
	}

	return h.sendXML(ctx, sitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	})
}

func (h *SitemapHandler) users(ctx fiber.Ctx) error {
	rows, err := h.db.QueryContext(ctx.Context(),
		`SELECT username FROM users ORDER BY created_at DESC`)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).SendString("failed to query users")
	}
	defer rows.Close()

	var urls []sitemapURL
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			continue
		}
		urls = append(urls, sitemapURL{
			Loc: h.baseURL + "/user/" + username,
		})
	}

	return h.sendXML(ctx, sitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	})
}
