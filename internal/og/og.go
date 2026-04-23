package og

import (
	"context"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/secrets"

	"github.com/google/uuid"
)

type (
	Resolver struct {
		theoryRepo       repository.TheoryRepository
		userRepo         repository.UserRepository
		postRepo         repository.PostRepository
		artRepo          repository.ArtRepository
		mysteryRepo      repository.MysteryRepository
		shipRepo         repository.ShipRepository
		fanficRepo       repository.FanficRepository
		announcementRepo repository.AnnouncementRepository
		journalRepo      repository.JournalRepository
		chatRepo         repository.ChatRepository
		baseHTML         string
		baseURL          string
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
	defaultDescription = "A social platform for fans of Umineko, Higurashi, and the wider When They Cry series. Post theories, solve mysteries, share fan art, chronicle read-throughs, ship pairings, write fanfiction, and chat in live rooms."
	defaultImagePath   = "/Featherine.jpg"
	baseURLPlaceholder = "__BASE_URL__"
)

func NewResolver(
	theoryRepo repository.TheoryRepository,
	userRepo repository.UserRepository,
	postRepo repository.PostRepository,
	artRepo repository.ArtRepository,
	mysteryRepo repository.MysteryRepository,
	shipRepo repository.ShipRepository,
	fanficRepo repository.FanficRepository,
	announcementRepo repository.AnnouncementRepository,
	journalRepo repository.JournalRepository,
	chatRepo repository.ChatRepository,
	baseHTML, baseURL string,
) *Resolver {
	return &Resolver{
		theoryRepo:       theoryRepo,
		userRepo:         userRepo,
		postRepo:         postRepo,
		artRepo:          artRepo,
		mysteryRepo:      mysteryRepo,
		shipRepo:         shipRepo,
		fanficRepo:       fanficRepo,
		announcementRepo: announcementRepo,
		journalRepo:      journalRepo,
		chatRepo:         chatRepo,
		baseHTML:         strings.ReplaceAll(baseHTML, baseURLPlaceholder, baseURL),
		baseURL:          baseURL,
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

	if len(parts) == 1 && (parts[0] == "" || parts[0] == "welcome") {
		url := r.baseURL + "/"
		if parts[0] == "welcome" {
			url = r.baseURL + "/welcome"
		}
		return &Meta{
			Title:       "Umineko City of Books - Fan Theory Platform",
			Description: "Welcome to the game board. Declare blue truths, solve mysteries, debate pairings, read and write fanfiction, and chronicle your journey through When They Cry.",
			URL:         url,
		}
	}

	if len(parts) == 2 && parts[0] == "theory" {
		return r.theoryMeta(ctx, parts[1])
	}

	if len(parts) == 2 && parts[0] == "user" {
		return r.profileMeta(ctx, parts[1])
	}

	if len(parts) == 2 && parts[0] == "game-board" {
		if _, err := uuid.Parse(parts[1]); err == nil {
			return r.postMeta(ctx, parts[1])
		}
		return r.gameBoardCornerMeta(parts[1])
	}

	if len(parts) == 3 && parts[0] == "gallery" && parts[1] == "art" {
		if _, err := uuid.Parse(parts[2]); err == nil {
			return r.artMeta(ctx, parts[2])
		}
	}

	if len(parts) == 3 && parts[0] == "gallery" && parts[1] == "view" {
		if _, err := uuid.Parse(parts[2]); err == nil {
			return r.galleryMeta(ctx, parts[2])
		}
	}

	if len(parts) == 1 && parts[0] == "mysteries" {
		return &Meta{
			Title:       "Mysteries - Umineko City of Books",
			Description: "Browse and solve fan-created mysteries inspired by Umineko no Naku Koro ni.",
			URL:         r.baseURL + "/mysteries",
		}
	}

	if len(parts) == 2 && parts[0] == "mystery" {
		if _, err := uuid.Parse(parts[1]); err == nil {
			return r.mysteryMeta(ctx, parts[1])
		}
	}

	if len(parts) == 1 && parts[0] == "ships" {
		return &Meta{
			Title:       "Ships - Umineko City of Books",
			Description: "Declare your favourite Umineko and Higurashi pairings. Vote on crackships and debate the merits of your OTPs.",
			URL:         r.baseURL + "/ships",
		}
	}

	if len(parts) == 2 && parts[0] == "ships" {
		if _, err := uuid.Parse(parts[1]); err == nil {
			return r.shipMeta(ctx, parts[1])
		}
	}

	if len(parts) == 1 && parts[0] == "announcements" {
		return &Meta{
			Title:       "Announcements - Umineko City of Books",
			Description: "Latest announcements from the Umineko City of Books moderation team.",
			URL:         r.baseURL + "/announcements",
		}
	}

	if len(parts) == 2 && parts[0] == "announcements" {
		if _, err := uuid.Parse(parts[1]); err == nil {
			return r.announcementMeta(ctx, parts[1])
		}
	}

	if len(parts) == 1 && parts[0] == "fanfiction" {
		return &Meta{
			Title:       "Fanfiction - Umineko City of Books",
			Description: "Browse and share fan-created stories inspired by When They Cry.",
			URL:         r.baseURL + "/fanfiction",
		}
	}
	if len(parts) >= 2 && parts[0] == "fanfiction" {
		if _, err := uuid.Parse(parts[1]); err == nil {
			return r.fanficMeta(ctx, parts[1])
		}
	}

	if len(parts) == 1 && parts[0] == "suggestions" {
		return &Meta{
			Title:       "Site Improvements - Umineko City of Books",
			Description: "Suggest improvements, report issues, and share ideas for the site.",
			URL:         r.baseURL + "/suggestions",
		}
	}

	if len(parts) == 1 && parts[0] == "gallery" {
		return &Meta{
			Title:       "Gallery - Umineko City of Books",
			Description: "Browse fan art galleries from the Umineko community.",
			URL:         r.baseURL + "/gallery",
		}
	}

	if len(parts) == 1 && parts[0] == "journals" {
		return &Meta{
			Title:       "Reading Journals - Umineko City of Books",
			Description: "Live-blog your read-throughs of Ryukishi07's works. Post reactions, theories, and predictions as you go.",
			URL:         r.baseURL + "/journals",
		}
	}

	if len(parts) == 2 && parts[0] == "journals" {
		if _, err := uuid.Parse(parts[1]); err == nil {
			return r.journalMeta(ctx, parts[1])
		}
	}

	if len(parts) == 1 && parts[0] == "rooms" {
		return &Meta{
			Title:       "Chat Rooms - Umineko City of Books",
			Description: "Live group chats for roleplay, book clubs, episode reactions, and more.",
			URL:         r.baseURL + "/rooms",
		}
	}

	if len(parts) == 2 && parts[0] == "rooms" {
		if _, err := uuid.Parse(parts[1]); err == nil {
			return r.roomMeta(ctx, parts[1])
		}
	}

	if len(parts) == 1 && parts[0] == "secrets" {
		return &Meta{
			Title:       "Secrets - Umineko City of Books",
			Description: "Quiet things scattered across the site. Open hunts, live progress leaderboards, and the people who spoke the answer first.",
			URL:         r.baseURL + "/secrets",
		}
	}

	if len(parts) == 2 && parts[0] == "secrets" {
		return r.secretMeta(parts[1])
	}

	if len(parts) == 2 && parts[0] == "gallery" {
		corner := parts[1]
		name := strings.ToUpper(corner[:1]) + corner[1:]
		return &Meta{
			Title:       name + " Gallery - Umineko City of Books",
			Description: fmt.Sprintf("Browse %s fan art from the Umineko community.", corner),
			URL:         fmt.Sprintf("%s/gallery/%s", r.baseURL, corner),
		}
	}

	if len(parts) == 1 && parts[0] == "games" {
		return &Meta{
			Title:       "Games - Umineko City of Books",
			Description: "Play multiplayer games with other players. Chess first, with more to come.",
			URL:         r.baseURL + "/games",
		}
	}

	if len(parts) == 2 && parts[0] == "games" && parts[1] == "live" {
		return &Meta{
			Title:       "Live Games - Umineko City of Books",
			Description: "Watch chess matches in progress. Spectate live and chat with other viewers.",
			URL:         r.baseURL + "/games/live",
		}
	}

	if len(parts) == 2 && parts[0] == "games" && parts[1] == "past" {
		return &Meta{
			Title:       "Past Games - Umineko City of Books",
			Description: "Every finished match on the site. Review final positions, move histories and per-game stats.",
			URL:         r.baseURL + "/games/past",
		}
	}

	if len(parts) == 2 && parts[0] == "games" {
		game := parts[1]
		name := strings.ToUpper(game[:1]) + game[1:]
		return &Meta{
			Title:       name + " - Umineko City of Books",
			Description: "Play " + name + " with other players. See the scoreboard, start a new game, or spectate live matches.",
			URL:         fmt.Sprintf("%s/games/%s", r.baseURL, game),
		}
	}

	if len(parts) == 3 && parts[0] == "games" && parts[2] == "scoreboard" {
		return &Meta{
			Title:       strings.ToUpper(parts[1][:1]) + parts[1][1:] + " Scoreboard - Umineko City of Books",
			Description: fmt.Sprintf("See the top %s players across the community.", parts[1]),
			URL:         fmt.Sprintf("%s/games/%s/scoreboard", r.baseURL, parts[1]),
		}
	}

	if len(parts) == 3 && parts[0] == "games" && parts[1] == "chess" && parts[2] == "new" {
		return &Meta{
			Title:       "New Chess Game - Umineko City of Books",
			Description: "Invite another player to a game of chess.",
			URL:         r.baseURL + "/games/chess/new",
		}
	}

	if len(parts) == 3 && parts[0] == "games" && parts[1] == "chess" {
		if _, err := uuid.Parse(parts[2]); err == nil {
			return &Meta{
				Title:       "Chess Game - Umineko City of Books",
				Description: "A chess match between two players.",
				URL:         fmt.Sprintf("%s/games/chess/%s", r.baseURL, parts[2]),
			}
		}
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
		Image:       u.BannerURL,
		URL:         fmt.Sprintf("%s/user/%s", r.baseURL, username),
	}
}

func (r *Resolver) gameBoardCornerMeta(corner string) *Meta {
	titles := map[string]string{
		"umineko":   "Umineko",
		"higurashi": "Higurashi",
		"ciconia":   "Ciconia",
		"higanbana": "Higanbana",
		"roseguns":  "Rose Guns Days",
	}
	name, ok := titles[corner]
	if !ok {
		return nil
	}
	return &Meta{
		Title:       name + " Game Board - Umineko City of Books",
		Description: fmt.Sprintf("Discuss %s with fellow players on the game board.", name),
		URL:         fmt.Sprintf("%s/game-board/%s", r.baseURL, corner),
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

func (r *Resolver) artMeta(ctx context.Context, idStr string) *Meta {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil
	}

	art, err := r.artRepo.GetByID(ctx, id, uuid.Nil)
	if err != nil || art == nil {
		return nil
	}

	desc := art.Description
	if desc == "" {
		desc = fmt.Sprintf("Art by %s on Umineko City of Books", art.AuthorDisplayName)
	}
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}

	return &Meta{
		Title:       fmt.Sprintf("%s - by %s", art.Title, art.AuthorDisplayName),
		Description: desc,
		Image:       art.ImageURL,
		URL:         fmt.Sprintf("%s/gallery/art/%s", r.baseURL, idStr),
	}
}

func (r *Resolver) galleryMeta(ctx context.Context, idStr string) *Meta {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil
	}

	gallery, err := r.artRepo.GetGalleryByID(ctx, id)
	if err != nil || gallery == nil {
		return nil
	}

	desc := gallery.Description
	if desc == "" {
		desc = fmt.Sprintf("%s's art gallery on Umineko City of Books", gallery.AuthorDisplayName)
	}
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}

	meta := &Meta{
		Title:       fmt.Sprintf("%s - %s's Gallery", gallery.Name, gallery.AuthorDisplayName),
		Description: desc,
		URL:         fmt.Sprintf("%s/gallery/view/%s", r.baseURL, idStr),
	}

	if gallery.CoverImageURL != "" {
		meta.Image = gallery.CoverImageURL
	} else {
		previews, _ := r.artRepo.GetGalleryPreviewImages(ctx, id, 1)
		if len(previews) > 0 {
			meta.Image = previews[0].ImageURL
		}
	}

	return meta
}

func (r *Resolver) mysteryMeta(ctx context.Context, idStr string) *Meta {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil
	}

	mystery, err := r.mysteryRepo.GetByID(ctx, id)
	if err != nil || mystery == nil {
		return nil
	}

	desc := mystery.Body
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}

	status := "Open"
	if mystery.Solved {
		status = "Solved"
	}

	title := fmt.Sprintf("%s (%s) - Mystery by %s", mystery.Title, status, mystery.AuthorDisplayName)

	return &Meta{
		Title:       title,
		Description: desc,
		URL:         fmt.Sprintf("%s/mystery/%s", r.baseURL, idStr),
	}
}

func (r *Resolver) announcementMeta(ctx context.Context, idStr string) *Meta {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil
	}

	ann, err := r.announcementRepo.GetByID(ctx, id)
	if err != nil || ann == nil {
		return nil
	}

	desc := ann.Body
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}

	title := fmt.Sprintf("%s - Announcement by %s", ann.Title, ann.AuthorDisplayName)

	return &Meta{
		Title:       title,
		Description: desc,
		URL:         fmt.Sprintf("%s/announcements/%s", r.baseURL, idStr),
	}
}

func (r *Resolver) shipMeta(ctx context.Context, idStr string) *Meta {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil
	}

	ship, err := r.shipRepo.GetByID(ctx, id, uuid.Nil)
	if err != nil || ship == nil {
		return nil
	}

	characters, _ := r.shipRepo.GetCharacters(ctx, id)
	charNames := make([]string, len(characters))
	for i, c := range characters {
		charNames[i] = c.CharacterName
	}
	pairing := strings.Join(charNames, " \u00D7 ")

	desc := ship.Description
	if desc == "" {
		desc = fmt.Sprintf("A ship by %s featuring %s", ship.AuthorDisplayName, pairing)
	}
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}

	title := fmt.Sprintf("%s - %s", ship.Title, pairing)

	meta := &Meta{
		Title:       title,
		Description: desc,
		URL:         fmt.Sprintf("%s/ships/%s", r.baseURL, idStr),
	}
	if ship.ImageURL != "" {
		meta.Image = ship.ImageURL
	} else {
		meta.Image = ship.AuthorAvatarURL
	}
	return meta
}

func (r *Resolver) fanficMeta(ctx context.Context, idStr string) *Meta {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil
	}

	fanfic, err := r.fanficRepo.GetByID(ctx, id, uuid.Nil)
	if err != nil || fanfic == nil {
		return nil
	}

	desc := fanfic.Summary
	if desc == "" {
		desc = fmt.Sprintf("A fanfic by %s", fanfic.AuthorDisplayName)
	}
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}

	meta := &Meta{
		Title:       fanfic.Title + " - Fanfiction",
		Description: desc,
		URL:         fmt.Sprintf("%s/fanfiction/%s", r.baseURL, idStr),
	}
	if fanfic.CoverImageURL != "" {
		meta.Image = fanfic.CoverImageURL
	}
	return meta
}

func (r *Resolver) journalMeta(ctx context.Context, idStr string) *Meta {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil
	}

	journal, err := r.journalRepo.GetByID(ctx, id, uuid.Nil)
	if err != nil || journal == nil {
		return nil
	}

	desc := journal.Body
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}

	title := fmt.Sprintf("%s - %s's Reading Journal", journal.Title, journal.Author.DisplayName)

	return &Meta{
		Title:       title,
		Description: desc,
		URL:         fmt.Sprintf("%s/journals/%s", r.baseURL, idStr),
	}
}

func (r *Resolver) secretMeta(id string) *Meta {
	spec, ok := secrets.Lookup(id)
	if !ok || spec.Title == "" {
		return nil
	}
	desc := spec.Description
	if desc == "" {
		desc = "A hidden hunt on Umineko City of Books."
	}
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}
	return &Meta{
		Title:       fmt.Sprintf("%s - Umineko City of Books", spec.Title),
		Description: desc,
		URL:         fmt.Sprintf("%s/secrets/%s", r.baseURL, id),
	}
}

func (r *Resolver) roomMeta(ctx context.Context, idStr string) *Meta {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil
	}

	room, err := r.chatRepo.GetRoomByID(ctx, id, uuid.Nil)
	if err != nil || room == nil {
		return nil
	}

	desc := room.Description
	if desc == "" {
		desc = fmt.Sprintf("A chat room with %d members on Umineko City of Books", room.MemberCount)
	}
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}

	return &Meta{
		Title:       room.Name + " - Chat Room",
		Description: desc,
		URL:         fmt.Sprintf("%s/rooms/%s", r.baseURL, idStr),
	}
}

func (r *Resolver) inject(meta Meta) string {
	html := r.baseHTML
	html = replaceMetaContent(html, "property", "og:title", defaultTitle, escapeAttr(meta.Title))
	html = replaceMetaContent(html, "name", "twitter:title", defaultTitle, escapeAttr(meta.Title))
	html = replaceMetaContent(html, "name", "twitter:description", defaultDescription, escapeAttr(meta.Description))
	html = replaceMetaContent(html, "property", "og:description", defaultDescription, escapeAttr(meta.Description))
	html = replaceMetaContent(html, "name", "description", defaultDescription, escapeAttr(meta.Description))
	html = replaceTitleTag(html, defaultTitle, escapeAttr(meta.Title))

	if meta.URL != "" {
		html = replaceMetaContent(html, "property", "og:url", r.baseURL+"/", meta.URL)
		html = replaceCanonical(html, r.baseURL+"/", meta.URL)
	}

	if meta.Image != "" {
		img := r.absoluteURL(meta.Image)
		defaultImage := r.baseURL + defaultImagePath
		html = replaceMetaContent(html, "property", "og:image", defaultImage, img)
		html = replaceMetaContent(html, "name", "twitter:image", defaultImage, img)
		html = stripMetaTag(html, "property", "og:image:width")
		html = stripMetaTag(html, "property", "og:image:height")
	}

	return html
}

func stripMetaTag(html, attrName, attrValue string) string {
	prefix := `<meta ` + attrName + `="` + attrValue + `" content="`
	idx := strings.Index(html, prefix)
	if idx < 0 {
		return html
	}
	end := strings.Index(html[idx:], `>`)
	if end < 0 {
		return html
	}
	return html[:idx] + html[idx+end+1:]
}

func replaceMetaContent(html, attrName, attrValue, oldContent, newContent string) string {
	old := attrName + `="` + attrValue + `" content="` + oldContent + `"`
	repl := attrName + `="` + attrValue + `" content="` + newContent + `"`
	return strings.Replace(html, old, repl, 1)
}

func replaceCanonical(html, oldHref, newHref string) string {
	old := `<link rel="canonical" href="` + oldHref + `">`
	repl := `<link rel="canonical" href="` + newHref + `">`
	return strings.Replace(html, old, repl, 1)
}

func replaceTitleTag(html, oldTitle, newTitle string) string {
	old := `<title>` + oldTitle + `</title>`
	repl := `<title>` + newTitle + `</title>`
	return strings.Replace(html, old, repl, 1)
}

func (r *Resolver) absoluteURL(u string) string {
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	return r.baseURL + u
}

func escapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
