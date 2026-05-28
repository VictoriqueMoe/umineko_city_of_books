package controllers

import (
	"encoding/xml"

	"umineko_city_of_books/internal/sitemap"

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
		svc sitemap.Service
	}
)

const sitemapNS = "http://www.sitemaps.org/schemas/sitemap/0.9"

func NewSitemapHandler(svc sitemap.Service) *SitemapHandler {
	return &SitemapHandler{svc: svc}
}

func (h *SitemapHandler) Register(app fiber.Router) {
	app.Get("/sitemap.xml", h.index)
	app.Get("/sitemap-static.xml", h.static)
	app.Get("/sitemap-theories.xml", h.theories)
	app.Get("/sitemap-posts.xml", h.posts)
	app.Get("/sitemap-art.xml", h.art)
	app.Get("/sitemap-users.xml", h.users)
	app.Get("/sitemap-mysteries.xml", h.mysteries)
	app.Get("/sitemap-ships.xml", h.ships)
	app.Get("/sitemap-fanfics.xml", h.fanfics)
	app.Get("/sitemap-journals.xml", h.journals)
}

func (h *SitemapHandler) sendXML(ctx fiber.Ctx, v interface{}) error {
	out, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).SendString("failed to generate sitemap")
	}
	ctx.Set("Content-Type", "application/xml; charset=utf-8")
	return ctx.Send(append([]byte(xml.Header), out...))
}

func toURLs(entries []sitemap.Entry) []sitemapURL {
	urls := make([]sitemapURL, 0, len(entries))
	for _, e := range entries {
		urls = append(urls, sitemapURL{Loc: e.URL, LastMod: e.LastMod})
	}
	return urls
}

func (h *SitemapHandler) index(ctx fiber.Ctx) error {
	indexEntries := h.svc.IndexEntries()
	sitemaps := make([]sitemapIndexURL, 0, len(indexEntries))
	for _, e := range indexEntries {
		sitemaps = append(sitemaps, sitemapIndexURL{Loc: e.Loc})
	}
	return h.sendXML(ctx, sitemapIndex{XMLNS: sitemapNS, Sitemaps: sitemaps})
}

func (h *SitemapHandler) static(ctx fiber.Ctx) error {
	return h.sendXML(ctx, sitemapURLSet{XMLNS: sitemapNS, URLs: toURLs(h.svc.StaticEntries(ctx.Context()))})
}

func (h *SitemapHandler) renderList(ctx fiber.Ctx, entries []sitemap.Entry, err error, label string) error {
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).SendString("failed to query " + label)
	}
	return h.sendXML(ctx, sitemapURLSet{XMLNS: sitemapNS, URLs: toURLs(entries)})
}

func (h *SitemapHandler) theories(ctx fiber.Ctx) error {
	entries, err := h.svc.Theories(ctx.Context())
	return h.renderList(ctx, entries, err, "theories")
}

func (h *SitemapHandler) posts(ctx fiber.Ctx) error {
	entries, err := h.svc.Posts(ctx.Context())
	return h.renderList(ctx, entries, err, "posts")
}

func (h *SitemapHandler) art(ctx fiber.Ctx) error {
	entries, err := h.svc.Art(ctx.Context())
	return h.renderList(ctx, entries, err, "art")
}

func (h *SitemapHandler) users(ctx fiber.Ctx) error {
	entries, err := h.svc.Users(ctx.Context())
	return h.renderList(ctx, entries, err, "users")
}

func (h *SitemapHandler) mysteries(ctx fiber.Ctx) error {
	entries, err := h.svc.Mysteries(ctx.Context())
	return h.renderList(ctx, entries, err, "mysteries")
}

func (h *SitemapHandler) ships(ctx fiber.Ctx) error {
	entries, err := h.svc.Ships(ctx.Context())
	return h.renderList(ctx, entries, err, "ships")
}

func (h *SitemapHandler) fanfics(ctx fiber.Ctx) error {
	entries, err := h.svc.Fanfics(ctx.Context())
	return h.renderList(ctx, entries, err, "fanfics")
}

func (h *SitemapHandler) journals(ctx fiber.Ctx) error {
	entries, err := h.svc.Journals(ctx.Context())
	return h.renderList(ctx, entries, err, "journals")
}
