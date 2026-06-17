package og

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/secrets"
	"umineko_city_of_books/internal/settings"

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
		ocRepo           repository.OCRepository
		fanficRepo       repository.FanficRepository
		announcementRepo repository.AnnouncementRepository
		journalRepo      repository.JournalRepository
		chatRepo         repository.ChatRepository
		liveStreamRepo   repository.LiveStreamRepository
		settingsSvc      settings.Service
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
	defaultTitle       = "When They Cry City of Books"
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
	ocRepo repository.OCRepository,
	fanficRepo repository.FanficRepository,
	announcementRepo repository.AnnouncementRepository,
	journalRepo repository.JournalRepository,
	chatRepo repository.ChatRepository,
	liveStreamRepo repository.LiveStreamRepository,
	settingsSvc settings.Service,
	baseHTML, baseURL string,
) *Resolver {
	return &Resolver{
		theoryRepo:       theoryRepo,
		userRepo:         userRepo,
		postRepo:         postRepo,
		artRepo:          artRepo,
		mysteryRepo:      mysteryRepo,
		shipRepo:         shipRepo,
		ocRepo:           ocRepo,
		fanficRepo:       fanficRepo,
		announcementRepo: announcementRepo,
		journalRepo:      journalRepo,
		chatRepo:         chatRepo,
		liveStreamRepo:   liveStreamRepo,
		settingsSvc:      settingsSvc,
		baseHTML:         strings.ReplaceAll(baseHTML, baseURLPlaceholder, baseURL),
		baseURL:          baseURL,
	}
}

func (r *Resolver) Resolve(ctx context.Context, path string) string {
	html, defaultImage := r.withDefaultImage(ctx)
	meta := r.metaForPath(ctx, path)
	if meta == nil {
		return html
	}

	return r.inject(ctx, html, *meta, defaultImage)
}

func (r *Resolver) withDefaultImage(ctx context.Context) (string, string) {
	builtin := r.baseURL + defaultImagePath
	custom := r.settingsSvc.Get(ctx, config.SettingOGDefaultImage)
	if custom == "" {
		return r.baseHTML, builtin
	}

	img := r.absoluteURL(custom)
	html := replaceMetaContent(r.baseHTML, "property", "og:image", builtin, img)
	html = replaceMetaContent(html, "name", "twitter:image", builtin, img)
	html = stripMetaTag(html, "property", "og:image:width")
	html = stripMetaTag(html, "property", "og:image:height")
	return html, img
}

func (r *Resolver) metaForPath(ctx context.Context, path string) *Meta {

	parts := strings.Split(strings.Trim(path, "/"), "/")
	siteName, siteDescription := r.getSiteMeta(ctx)

	formatSiteName := func(pageName string) string {
		return fmt.Sprintf("%s - %s", pageName, siteName)
	}

	if len(parts) == 1 && (parts[0] == "" || parts[0] == "welcome") {
		url := r.baseURL + "/"
		if parts[0] == "welcome" {
			url = r.baseURL + "/welcome"
		}

		return &Meta{
			Title:       siteName,
			Description: siteDescription,
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
		return r.gameBoardCornerMeta(ctx, parts[1])
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
			Title:       formatSiteName("Mysteries"),
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
			Title:       formatSiteName("Ships"),
			Description: "Declare your favourite Umineko and Higurashi pairings. Vote on crackships and debate the merits of your OTPs.",
			URL:         r.baseURL + "/ships",
		}
	}

	if len(parts) == 2 && parts[0] == "ships" {
		if _, err := uuid.Parse(parts[1]); err == nil {
			return r.shipMeta(ctx, parts[1])
		}
	}

	if len(parts) == 1 && parts[0] == "oc" {
		return &Meta{
			Title:       formatSiteName("Original Characters"),
			Description: "Browse Original Characters created by the community. Tag them as Umineko, Higurashi, Ciconia, or a custom series.",
			URL:         r.baseURL + "/oc",
		}
	}

	if len(parts) == 2 && parts[0] == "oc" {
		if _, err := uuid.Parse(parts[1]); err == nil {
			return r.ocMeta(ctx, parts[1])
		}
	}

	if len(parts) == 1 && parts[0] == "announcements" {
		return &Meta{
			Title:       formatSiteName("Announcements"),
			Description: fmt.Sprintf("Latest announcements from the %s moderation team.", siteName),
			URL:         r.baseURL + "/announcements",
		}
	}

	if len(parts) == 1 && parts[0] == "rules" {
		return &Meta{
			Title:       formatSiteName("Rules"),
			Description: fmt.Sprintf("Community rules and posting guidelines for the %s.", siteName),
			URL:         r.baseURL + "/rules",
		}
	}

	if len(parts) == 2 && parts[0] == "announcements" {
		if _, err := uuid.Parse(parts[1]); err == nil {
			return r.announcementMeta(ctx, parts[1])
		}
	}

	if len(parts) == 1 && parts[0] == "fanfiction" {
		return &Meta{
			Title:       formatSiteName("Fanfiction"),
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
			Title:       formatSiteName("Site Improvements"),
			Description: "Suggest improvements, report issues, and share ideas for the site.",
			URL:         r.baseURL + "/suggestions",
		}
	}

	if len(parts) == 1 && parts[0] == "gallery" {
		return &Meta{
			Title:       formatSiteName("Gallery"),
			Description: "Browse fan art galleries from the Umineko community.",
			URL:         r.baseURL + "/gallery",
		}
	}

	if len(parts) == 1 && parts[0] == "journals" {
		return &Meta{
			Title:       formatSiteName("Reading Journals"),
			Description: "Live-blog your read-throughs of Ryukishi07's works. Post reactions, theories, and predictions as you go.",
			URL:         r.baseURL + "/journals",
		}
	}

	if len(parts) == 2 && parts[0] == "journals" {
		if _, err := uuid.Parse(parts[1]); err == nil {
			return r.journalMeta(ctx, parts[1])
		}
	}

	if len(parts) == 4 && parts[0] == "journals" && parts[2] == "entry" {
		if _, err := uuid.Parse(parts[1]); err == nil {
			return r.journalEntryMeta(ctx, parts[1], parts[3])
		}
	}

	if len(parts) == 1 && parts[0] == "rooms" {
		return &Meta{
			Title:       formatSiteName("Chat Rooms"),
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
			Title:       formatSiteName("Secrets"),
			Description: "Quiet things scattered across the site. Open hunts, live progress leaderboards, and the people who spoke the answer first.",
			URL:         r.baseURL + "/secrets",
		}
	}

	if len(parts) == 2 && parts[0] == "secrets" {
		return r.secretMeta(ctx, parts[1])
	}

	if len(parts) == 1 && parts[0] == "live" {
		return &Meta{
			Title:       formatSiteName("Live Streams"),
			Description: "Watch live streams from the community. Reading parties, playthroughs, and real-time discussion about When They Cry.",
			URL:         r.baseURL + "/live",
		}
	}

	if len(parts) == 2 && parts[0] == "live" {
		if _, err := uuid.Parse(parts[1]); err == nil {
			return r.liveStreamMeta(ctx, parts[1])
		}
	}

	if len(parts) == 2 && parts[0] == "gallery" {
		corner := parts[1]
		name := strings.ToUpper(corner[:1]) + corner[1:]
		return &Meta{
			Title:       formatSiteName(name + " Gallery"),
			Description: fmt.Sprintf("Browse %s fan art from the Umineko community.", corner),
			URL:         fmt.Sprintf("%s/gallery/%s", r.baseURL, corner),
		}
	}

	if len(parts) == 1 && parts[0] == "games" {
		return &Meta{
			Title:       formatSiteName("Games"),
			Description: "Play multiplayer games with other players. Chess first, with more to come.",
			URL:         r.baseURL + "/games",
		}
	}

	if len(parts) == 1 && parts[0] == "search" {
		return &Meta{
			Title:       formatSiteName("Search"),
			Description: "Search posts, comments, theories, mysteries, art, fanfiction, journals, ships, and users across the site.",
			URL:         r.baseURL + "/search",
		}
	}

	if len(parts) == 2 && parts[0] == "games" && parts[1] == "live" {
		return &Meta{
			Title:       formatSiteName("Live Games"),
			Description: "Watch chess matches in progress. Spectate live and chat with other viewers.",
			URL:         r.baseURL + "/games/live",
		}
	}

	if len(parts) == 2 && parts[0] == "games" && parts[1] == "past" {
		return &Meta{
			Title:       formatSiteName("Past Games"),
			Description: "Every finished match on the site. Review final positions, move histories and per-game stats.",
			URL:         r.baseURL + "/games/past",
		}
	}

	if len(parts) == 2 && parts[0] == "games" {
		game := parts[1]
		name := strings.ToUpper(game[:1]) + game[1:]
		return &Meta{
			Title:       formatSiteName(name),
			Description: "Play " + name + " with other players. See the scoreboard, start a new game, or spectate live matches.",
			URL:         fmt.Sprintf("%s/games/%s", r.baseURL, game),
		}
	}

	if len(parts) == 3 && parts[0] == "games" && parts[2] == "scoreboard" {
		return &Meta{
			Title:       formatSiteName(strings.ToUpper(parts[1][:1]) + parts[1][1:] + " Scoreboard"),
			Description: fmt.Sprintf("See the top %s players across the community.", parts[1]),
			URL:         fmt.Sprintf("%s/games/%s/scoreboard", r.baseURL, parts[1]),
		}
	}

	if len(parts) == 3 && parts[0] == "games" && parts[2] == "new" {
		name := strings.ToUpper(parts[1][:1]) + parts[1][1:]
		return &Meta{
			Title:       formatSiteName("New " + name + " Game"),
			Description: "Invite another player to a game of " + parts[1] + ".",
			URL:         r.baseURL + "/games/" + parts[1] + "/new",
		}
	}

	if len(parts) == 3 && parts[0] == "games" {
		if _, err := uuid.Parse(parts[2]); err == nil {
			name := strings.ToUpper(parts[1][:1]) + parts[1][1:]
			return &Meta{
				Title:       formatSiteName(name + " Game"),
				Description: "A " + parts[1] + " match between two players.",
				URL:         fmt.Sprintf("%s/games/%s/%s", r.baseURL, parts[1], parts[2]),
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
		siteName, _ := r.getSiteMeta(ctx)
		desc = fmt.Sprintf("%s's profile on %s", u.DisplayLabel(), siteName)
	}
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}

	return &Meta{
		Title:       fmt.Sprintf("%s (@%s)", u.DisplayLabel(), u.Username),
		Description: desc,
		Image:       u.BannerURL,
		URL:         fmt.Sprintf("%s/user/%s", r.baseURL, username),
	}
}

func (r *Resolver) gameBoardCornerMeta(ctx context.Context, corner string) *Meta {
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

	siteName, _ := r.getSiteMeta(ctx)
	return &Meta{
		Title:       fmt.Sprintf("%s Game Board - %s", name, siteName),
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
		siteName, _ := r.getSiteMeta(ctx)
		desc = fmt.Sprintf("Art by %s on %s", art.AuthorDisplayName, siteName)
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
		siteName, _ := r.getSiteMeta(ctx)
		desc = fmt.Sprintf("%s's art gallery on %s", gallery.AuthorDisplayName, siteName)
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

func (r *Resolver) ocMeta(ctx context.Context, idStr string) *Meta {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil
	}

	row, err := r.ocRepo.GetByID(ctx, id, uuid.Nil)
	if err != nil || row == nil {
		return nil
	}

	desc := row.Description
	if desc == "" {
		seriesLabel := row.Series
		if row.Series == "custom" && row.CustomSeriesName != "" {
			seriesLabel = row.CustomSeriesName
		}
		desc = fmt.Sprintf("%s, an OC by %s (%s)", row.Name, row.AuthorDisplayName, seriesLabel)
	}
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}

	meta := &Meta{
		Title:       fmt.Sprintf("%s - OC by %s", row.Name, row.AuthorDisplayName),
		Description: desc,
		URL:         fmt.Sprintf("%s/oc/%s", r.baseURL, idStr),
	}
	if row.ImageURL != "" {
		meta.Image = row.ImageURL
	} else {
		meta.Image = row.AuthorAvatarURL
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

	desc := journal.LatestEntryExcerpt
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}
	if desc == "" {
		desc = fmt.Sprintf("%s's Reading Journal", journal.Author.DisplayName)
	}

	title := fmt.Sprintf("%s - %s's Reading Journal", journal.Title, journal.Author.DisplayName)

	return &Meta{
		Title:       title,
		Description: desc,
		URL:         fmt.Sprintf("%s/journals/%s", r.baseURL, idStr),
	}
}

func (r *Resolver) journalEntryMeta(ctx context.Context, journalIDStr, numberStr string) *Meta {
	journalID, err := uuid.Parse(journalIDStr)
	if err != nil {
		return nil
	}
	number, err := strconv.Atoi(numberStr)
	if err != nil || number < 1 {
		return nil
	}

	journal, err := r.journalRepo.GetByID(ctx, journalID, uuid.Nil)
	if err != nil || journal == nil {
		return nil
	}

	entry, err := r.journalRepo.GetEntry(ctx, journalID, number)
	if err != nil || entry == nil {
		return nil
	}

	desc := entry.Body
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}
	if desc == "" {
		desc = fmt.Sprintf("Entry %d in %s's Reading Journal", number, journal.Author.DisplayName)
	}

	entryLabel := fmt.Sprintf("Entry %d", number)
	if entry.Title != nil && *entry.Title != "" {
		entryLabel = fmt.Sprintf("Entry %d: %s", number, *entry.Title)
	}

	title := fmt.Sprintf("%s - %s - %s's Reading Journal", entryLabel, journal.Title, journal.Author.DisplayName)

	return &Meta{
		Title:       title,
		Description: desc,
		URL:         fmt.Sprintf("%s/journals/%s/entry/%d", r.baseURL, journalIDStr, number),
	}
}

func (r *Resolver) secretMeta(ctx context.Context, id string) *Meta {
	spec, ok := secrets.Lookup(id)
	if !ok || spec.Title == "" {
		return nil
	}

	siteName, _ := r.getSiteMeta(ctx)
	desc := spec.Description
	if desc == "" {
		desc = fmt.Sprintf("A hidden hunt on %s.", siteName)
	}
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}
	return &Meta{
		Title:       fmt.Sprintf("%s - %s", spec.Title, siteName),
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
		siteName, _ := r.getSiteMeta(ctx)
		desc = fmt.Sprintf("A chat room with %d members on %s", room.MemberCount, siteName)
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

func (r *Resolver) liveStreamMeta(ctx context.Context, idStr string) *Meta {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil
	}

	stream, err := r.liveStreamRepo.GetByID(ctx, id)
	if err != nil || stream == nil || stream.Status != "live" {
		return nil
	}

	name := stream.DisplayName
	if name == "" {
		name = stream.Username
	}

	desc := stream.Title
	if desc == "" {
		siteName, _ := r.getSiteMeta(ctx)
		desc = fmt.Sprintf("A live stream by %s on %s", name, siteName)
	}

	runes := []rune(desc)
	if len(runes) > 200 {
		desc = string(runes[:197]) + "..."
	}

	meta := &Meta{
		Title:       fmt.Sprintf("%s - %s's live stream", stream.Title, name),
		Description: desc,
		URL:         fmt.Sprintf("%s/live/%s", r.baseURL, idStr),
	}
	if stream.ThumbnailURL != "" {
		meta.Image = r.absoluteURL(stream.ThumbnailURL)
	} else if stream.AvatarURL != "" {
		meta.Image = r.absoluteURL(stream.AvatarURL)
	}

	return meta
}

func (r *Resolver) getSiteMeta(ctx context.Context) (siteName string, siteDescription string) {
	siteName = r.settingsSvc.Get(ctx, config.SettingSiteName)
	if siteName == "" {
		siteName = defaultTitle
	}

	siteDescription = r.settingsSvc.Get(ctx, config.SettingSiteDescription)
	if siteDescription == "" {
		siteDescription = defaultDescription
	}

	return
}

func (r *Resolver) inject(ctx context.Context, html string, meta Meta, defaultImage string) string {
	siteName, _ := r.getSiteMeta(ctx)

	html = replaceMetaContent(html, "property", "og:title", defaultTitle, escapeAttr(meta.Title))
	html = replaceMetaContent(html, "name", "twitter:title", defaultTitle, escapeAttr(meta.Title))
	html = replaceMetaContent(html, "name", "twitter:description", defaultDescription, escapeAttr(meta.Description))
	html = replaceMetaContent(html, "property", "og:description", defaultDescription, escapeAttr(meta.Description))
	html = replaceMetaContent(html, "name", "description", defaultDescription, escapeAttr(meta.Description))
	html = replaceMetaContent(html, "property", "og:site_name", defaultTitle, escapeAttr(siteName))
	html = replaceTitleTag(html, defaultTitle, escapeAttr(meta.Title))

	if meta.URL != "" {
		html = replaceMetaContent(html, "property", "og:url", r.baseURL+"/", meta.URL)
		html = replaceCanonical(html, r.baseURL+"/", meta.URL)
	}

	if meta.Image != "" {
		img := r.ogImageURL(r.absoluteURL(meta.Image))
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

func (r *Resolver) ogImageURL(img string) string {
	prefix := r.baseURL + "/uploads/"
	if !strings.HasPrefix(img, prefix) || !strings.HasSuffix(strings.ToLower(img), ".webp") {
		return img
	}

	rel := strings.TrimPrefix(img, prefix)
	return r.baseURL + "/og-image/" + rel[:len(rel)-len(".webp")] + ".jpg"
}

func escapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
