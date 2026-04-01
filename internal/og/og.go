package og

import (
	"context"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

type (
	Resolver struct {
		theoryRepo repository.TheoryRepository
		userRepo   repository.UserRepository
		postRepo   repository.PostRepository
		baseHTML   string
		baseURL    string
	}

	Meta struct {
		Title       string
		Description string
		Image       string
		URL         string
	}
)

const (
	defaultTitle       = "Umineko City of Books"
	defaultDescription = "A social platform for Umineko no Naku Koro ni fans. Declare fan theories as blue truth, debate with evidence, and earn credibility through community response."
)

func NewResolver(theoryRepo repository.TheoryRepository, userRepo repository.UserRepository, postRepo repository.PostRepository, baseHTML, baseURL string) *Resolver {
	return &Resolver{
		theoryRepo: theoryRepo,
		userRepo:   userRepo,
		postRepo:   postRepo,
		baseHTML:   baseHTML,
		baseURL:    baseURL,
	}
}

func (r *Resolver) Resolve(ctx context.Context, path string) string {
	meta := r.metaForPath(ctx, path)
	if meta == nil {
		return r.baseHTML
	}
	return r.inject(*meta)
}

func (r *Resolver) metaForPath(ctx context.Context, path string) *Meta {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 2 && parts[0] == "theory" {
		return r.theoryMeta(ctx, parts[1])
	}

	if len(parts) == 2 && parts[0] == "user" {
		return r.profileMeta(ctx, parts[1])
	}

	if len(parts) >= 2 && parts[0] == "game-board" {
		return r.postMeta(ctx, parts[1])
	}

	return nil
}

func (r *Resolver) theoryMeta(ctx context.Context, idStr string) *Meta {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil
	}

	theory, err := r.theoryRepo.GetByID(ctx, id)
	if err != nil || theory == nil {
		return nil
	}

	desc := theory.Body
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}

	title := fmt.Sprintf("%s - %s's Blue Truth", theory.Title, theory.Author.DisplayName)

	return &Meta{
		Title:       title,
		Description: desc,
		Image:       theory.Author.AvatarURL,
		URL:         fmt.Sprintf("%s/theory/%s", r.baseURL, idStr),
	}
}

func (r *Resolver) profileMeta(ctx context.Context, username string) *Meta {
	u, _, err := r.userRepo.GetProfileByUsername(ctx, username)
	if err != nil || u == nil {
		return nil
	}

	desc := u.Bio
	if desc == "" {
		desc = fmt.Sprintf("%s's profile on Umineko City of Books", u.DisplayName)
	}
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}

	return &Meta{
		Title:       fmt.Sprintf("%s (@%s)", u.DisplayName, u.Username),
		Description: desc,
		Image:       u.AvatarURL,
		URL:         fmt.Sprintf("%s/user/%s", r.baseURL, username),
	}
}

func (r *Resolver) postMeta(ctx context.Context, idStr string) *Meta {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil
	}

	post, err := r.postRepo.GetByID(ctx, id, uuid.Nil)
	if err != nil || post == nil {
		return nil
	}

	desc := post.Body
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}

	title := fmt.Sprintf("%s on Game Board", post.AuthorDisplayName)

	meta := &Meta{
		Title:       title,
		Description: desc,
		Image:       post.AuthorAvatarURL,
		URL:         fmt.Sprintf("%s/game-board/%s", r.baseURL, idStr),
	}

	media, _ := r.postRepo.GetMedia(ctx, id)
	if len(media) > 0 {
		first := media[0]
		if first.MediaType == "video" && first.ThumbnailURL != "" {
			meta.Image = first.ThumbnailURL
		} else if first.MediaType == "image" {
			meta.Image = first.MediaURL
		}
	}

	return meta
}

func (r *Resolver) inject(meta Meta) string {
	html := r.baseHTML
	html = replaceMetaContent(html, "property", "og:title", defaultTitle, escapeAttr(meta.Title))
	html = replaceMetaContent(html, "property", "og:description", defaultDescription, escapeAttr(meta.Description))
	html = replaceMetaContent(html, "name", "twitter:title", defaultTitle, escapeAttr(meta.Title))
	html = replaceMetaContent(html, "name", "twitter:description", defaultDescription, escapeAttr(meta.Description))

	if meta.URL != "" {
		html = replaceMetaContent(html, "property", "og:url", "https://meta.auaurora.moe/", meta.URL)
	}

	if meta.Image != "" {
		html = strings.Replace(html,
			`<meta name="twitter:card" content="summary_large_image">`,
			`<meta name="twitter:card" content="summary_large_image">`+
				"\n    "+`<meta property="og:image" content="`+meta.Image+`">`+
				"\n    "+`<meta name="twitter:image" content="`+meta.Image+`">`,
			1,
		)
	}

	return html
}

func replaceMetaContent(html, attrName, attrValue, oldContent, newContent string) string {
	old := attrName + `="` + attrValue + `" content="` + oldContent + `"`
	repl := attrName + `="` + attrValue + `" content="` + newContent + `"`
	return strings.Replace(html, old, repl, 1)
}

func escapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
