package og

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/corner"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/reserved"
	"umineko_city_of_books/internal/secrets"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/text"

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
		watchPartyRepo   repository.ChatWatchPartyRepository
		liveStreamRepo   repository.LiveStreamRepository
		settingsSvc      settings.Service
		cache            *cache.Manager
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
	statusLive         = "live"
	maxDescRunes       = 200
	maxDescClipRunes   = 197
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
	watchPartyRepo repository.ChatWatchPartyRepository,
	liveStreamRepo repository.LiveStreamRepository,
	settingsSvc settings.Service,
	cacheMgr *cache.Manager,
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
		watchPartyRepo:   watchPartyRepo,
		liveStreamRepo:   liveStreamRepo,
		settingsSvc:      settingsSvc,
		cache:            cacheMgr,
		baseHTML:         strings.ReplaceAll(baseHTML, baseURLPlaceholder, baseURL),
		baseURL:          baseURL,
	}
}

func (r *Resolver) Resolve(ctx context.Context, path, partyID string) string {
	if _, err := uuid.Parse(partyID); err != nil {
		partyID = ""
	}

	html, defaultImage := r.withDefaultImage(ctx)
	meta := r.resolveMeta(ctx, path, partyID)
	if meta == nil {
		return html
	}

	return r.inject(ctx, html, *meta, defaultImage)
}

func (r *Resolver) resolveMeta(ctx context.Context, path, partyID string) *Meta {
	load := func(ctx context.Context) (*Meta, error) {
		return r.metaForPath(ctx, path, partyID)
	}

	meta, err := r.cache.Load(ctx, cache.OGMeta.Key(metaKeyParts(path, partyID)...), load)
	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("path", path).Msg("failed to build page metadata, serving the default")

		return nil
	}

	return meta
}

func metaKeyParts(path, partyID string) []string {
	parts := []string{canonicalMetaPath(path)}
	if partyID != "" {
		parts = append(parts, partyID)
	}

	return parts
}

func canonicalMetaPath(path string) string {
	trimmed := strings.TrimRight(path, "/")
	if trimmed == "" {
		return "/"
	}

	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if isStreamerLivePath(parts) {
		return "/" + strings.ToLower(parts[0]) + "/live"
	}

	return trimmed
}

func isStreamerLivePath(parts []string) bool {
	return len(parts) == 2 && strings.EqualFold(parts[1], "live") && !reserved.IsPathSegment(parts[0])
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

func (r *Resolver) metaForPath(ctx context.Context, path, partyID string) (*Meta, error) {

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
		}, nil
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
		}, nil
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
		}, nil
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
		}, nil
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
		}, nil
	}

	if len(parts) == 1 && parts[0] == "rules" {
		return &Meta{
			Title:       formatSiteName("Rules"),
			Description: fmt.Sprintf("Community rules and posting guidelines for the %s.", siteName),
			URL:         r.baseURL + "/rules",
		}, nil
	}

	if len(parts) == 1 && parts[0] == "about" {
		return &Meta{
			Title:       formatSiteName("About"),
			Description: fmt.Sprintf("Who made %s, why, and how.", siteName),
			URL:         r.baseURL + "/about",
			Image:       "/about-creator.jpg",
		}, nil
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
		}, nil
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
		}, nil
	}

	if len(parts) == 1 && parts[0] == "gallery" {
		return &Meta{
			Title:       formatSiteName("Gallery"),
			Description: "Browse fan art galleries from the Umineko community.",
			URL:         r.baseURL + "/gallery",
		}, nil
	}

	if len(parts) == 1 && parts[0] == "journals" {
		return &Meta{
			Title:       formatSiteName("Reading Journals"),
			Description: "Live-blog your read-throughs of Ryukishi07's works. Post reactions, theories, and predictions as you go.",
			URL:         r.baseURL + "/journals",
		}, nil
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
		}, nil
	}

	if len(parts) == 2 && parts[0] == "rooms" {
		if _, err := uuid.Parse(parts[1]); err == nil {
			if partyID != "" {
				meta, err := r.watchPartyMeta(ctx, parts[1], partyID)
				if err != nil {
					return nil, err
				}

				if meta != nil {
					return meta, nil
				}
			}

			return r.roomMeta(ctx, parts[1])
		}
	}

	if len(parts) == 1 && parts[0] == "secrets" {
		return &Meta{
			Title:       formatSiteName("Secrets"),
			Description: "Quiet things scattered across the site. Open hunts, live progress leaderboards, and the people who spoke the answer first.",
			URL:         r.baseURL + "/secrets",
		}, nil
	}

	if len(parts) == 2 && parts[0] == "secrets" {
		return r.secretMeta(ctx, parts[1])
	}

	if len(parts) == 1 && parts[0] == "live" {
		return &Meta{
			Title:       formatSiteName("Live Streams"),
			Description: "Watch live streams from the community. Reading parties, playthroughs, and real-time discussion about When They Cry.",
			URL:         r.baseURL + "/live",
		}, nil
	}

	if len(parts) == 2 && parts[0] == "gallery" {
		gallery, ok := corner.GalleryAtPath("/gallery/" + parts[1])
		if !ok {
			return nil, nil
		}

		return &Meta{
			Title:       formatSiteName(gallery.Name() + " Gallery"),
			Description: fmt.Sprintf("Browse %s fan art from the Umineko community.", gallery),
			URL:         r.baseURL + gallery.Path(),
		}, nil
	}

	if len(parts) == 1 && parts[0] == "games" {
		return &Meta{
			Title:       formatSiteName("Games"),
			Description: "Play multiplayer games with other players. Chess first, with more to come.",
			URL:         r.baseURL + "/games",
		}, nil
	}

	if len(parts) == 1 && parts[0] == "search" {
		return &Meta{
			Title:       formatSiteName("Search"),
			Description: "Search posts, comments, theories, mysteries, art, fanfiction, journals, ships, and users across the site.",
			URL:         r.baseURL + "/search",
		}, nil
	}

	if len(parts) == 2 && parts[0] == "games" && parts[1] == "live" {
		return &Meta{
			Title:       formatSiteName("Live Games"),
			Description: "Watch chess matches in progress. Spectate live and chat with other viewers.",
			URL:         r.baseURL + "/games/live",
		}, nil
	}

	if len(parts) == 2 && parts[0] == "games" && parts[1] == "past" {
		return &Meta{
			Title:       formatSiteName("Past Games"),
			Description: "Every finished match on the site. Review final positions, move histories and per-game stats.",
			URL:         r.baseURL + "/games/past",
		}, nil
	}

	if len(parts) == 2 && parts[0] == "games" {
		game := parts[1]
		name := strings.ToUpper(game[:1]) + game[1:]
		return &Meta{
			Title:       formatSiteName(name),
			Description: "Play " + name + " with other players. See the scoreboard, start a new game, or spectate live matches.",
			URL:         fmt.Sprintf("%s/games/%s", r.baseURL, game),
		}, nil
	}

	if len(parts) == 3 && parts[0] == "games" && parts[1] != "" && parts[2] == "scoreboard" {
		return &Meta{
			Title:       formatSiteName(strings.ToUpper(parts[1][:1]) + parts[1][1:] + " Scoreboard"),
			Description: fmt.Sprintf("See the top %s players across the community.", parts[1]),
			URL:         fmt.Sprintf("%s/games/%s/scoreboard", r.baseURL, parts[1]),
		}, nil
	}

	if len(parts) == 3 && parts[0] == "games" && parts[1] != "" && parts[2] == "new" {
		name := strings.ToUpper(parts[1][:1]) + parts[1][1:]
		return &Meta{
			Title:       formatSiteName("New " + name + " Game"),
			Description: "Invite another player to a game of " + parts[1] + ".",
			URL:         r.baseURL + "/games/" + parts[1] + "/new",
		}, nil
	}

	if len(parts) == 3 && parts[0] == "games" && parts[1] != "" {
		if _, err := uuid.Parse(parts[2]); err == nil {
			name := strings.ToUpper(parts[1][:1]) + parts[1][1:]
			return &Meta{
				Title:       formatSiteName(name + " Game"),
				Description: "A " + parts[1] + " match between two players.",
				URL:         fmt.Sprintf("%s/games/%s/%s", r.baseURL, parts[1], parts[2]),
			}, nil
		}
	}

	if isStreamerLivePath(parts) {
		return r.liveStreamMetaByUsername(ctx, parts[0])
	}

	return nil, nil
}

func (r *Resolver) theoryMeta(ctx context.Context, idStr string) (*Meta, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, nil
	}

	theory, err := r.theoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if theory == nil {
		return nil, nil
	}

	desc := theory.Body
	desc = truncateDesc(desc)

	status := "Open"
	switch theory.Status {
	case dto.TheoryStatusContested:
		status = "Contested"
	case dto.TheoryStatusRefuted:
		status = "Refuted"
	}

	title := fmt.Sprintf("%s (%s) - %s's Blue Truth", theory.Title, status, theory.Author.DisplayName)

	return &Meta{
		Title:       title,
		Description: desc,
		URL:         fmt.Sprintf("%s/theory/%s", r.baseURL, idStr),
	}, nil
}

func (r *Resolver) profileMeta(ctx context.Context, username string) (*Meta, error) {
	u, _, err := r.userRepo.GetProfileByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	if u == nil {
		return nil, nil
	}

	desc := u.Bio
	if desc == "" {
		siteName, _ := r.getSiteMeta(ctx)
		desc = fmt.Sprintf("%s's profile on %s", u.DisplayLabel(), siteName)
	}
	desc = truncateDesc(desc)

	return &Meta{
		Title:       fmt.Sprintf("%s (@%s)", u.DisplayLabel(), u.Username),
		Description: desc,
		Image:       u.BannerURL,
		URL:         fmt.Sprintf("%s/user/%s", r.baseURL, username),
	}, nil
}

func (r *Resolver) gameBoardCornerMeta(ctx context.Context, slug string) (*Meta, error) {
	board, ok := corner.GameBoardAtPath("/game-board/" + slug)
	if !ok {
		return nil, nil
	}

	siteName, _ := r.getSiteMeta(ctx)
	return &Meta{
		Title:       fmt.Sprintf("%s Game Board - %s", board.Name(), siteName),
		Description: fmt.Sprintf("Discuss %s with fellow players on the game board.", board.Name()),
		URL:         r.baseURL + board.Path(),
	}, nil
}

func (r *Resolver) postMeta(ctx context.Context, idStr string) (*Meta, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, nil
	}

	post, err := r.postRepo.GetByID(ctx, spec.PostLookup{ID: id, ViewerID: uuid.Nil})
	if err != nil {
		return nil, err
	}

	if post == nil {
		return nil, nil
	}

	desc := post.Body
	desc = truncateDesc(desc)

	title := fmt.Sprintf("%s on Game Board", post.AuthorDisplayName)

	meta := &Meta{
		Title:       title,
		Description: desc,
		URL:         fmt.Sprintf("%s/game-board/%s", r.baseURL, idStr),
	}

	media, err := r.postRepo.GetMedia(ctx, id)
	if err != nil {
		return nil, err
	}

	if len(media) > 0 {
		first := media[0]
		if first.IsSpoiler {
			return meta, nil
		}

		if first.MediaType == "video" && first.ThumbnailURL != "" {
			meta.Image = first.ThumbnailURL
		} else if first.MediaType == "image" {
			meta.Image = first.MediaURL
		}
	}

	return meta, nil
}

func (r *Resolver) artMeta(ctx context.Context, idStr string) (*Meta, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, nil
	}

	art, err := r.artRepo.GetByID(ctx, spec.ArtLookup{ID: id, ViewerID: uuid.Nil})
	if err != nil {
		return nil, err
	}

	if art == nil {
		return nil, nil
	}

	desc := art.Description
	if desc == "" {
		siteName, _ := r.getSiteMeta(ctx)
		desc = fmt.Sprintf("Art by %s on %s", art.AuthorDisplayName, siteName)
	}
	desc = truncateDesc(desc)

	return &Meta{
		Title:       fmt.Sprintf("%s - by %s", art.Title, art.AuthorDisplayName),
		Description: desc,
		Image:       art.ImageURL,
		URL:         fmt.Sprintf("%s/gallery/art/%s", r.baseURL, idStr),
	}, nil
}

func (r *Resolver) galleryMeta(ctx context.Context, idStr string) (*Meta, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, nil
	}

	gallery, err := r.artRepo.GetGalleryByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if gallery == nil {
		return nil, nil
	}

	desc := gallery.Description
	if desc == "" {
		siteName, _ := r.getSiteMeta(ctx)
		desc = fmt.Sprintf("%s's art gallery on %s", gallery.AuthorDisplayName, siteName)
	}
	desc = truncateDesc(desc)

	meta := &Meta{
		Title:       fmt.Sprintf("%s - %s's Gallery", gallery.Name, gallery.AuthorDisplayName),
		Description: desc,
		URL:         fmt.Sprintf("%s/gallery/view/%s", r.baseURL, idStr),
	}

	if gallery.CoverImageURL != "" {
		meta.Image = gallery.CoverImageURL
	} else {
		previews, err := r.artRepo.GetGalleryPreviewImages(ctx, spec.GalleryPreviewFilter{GalleryID: id, Limit: 1})
		if err != nil {
			return nil, err
		}

		if len(previews) > 0 {
			meta.Image = previews[0].ImageURL
		}
	}

	return meta, nil
}

func (r *Resolver) mysteryMeta(ctx context.Context, idStr string) (*Meta, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, nil
	}

	mystery, err := r.mysteryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if mystery == nil {
		return nil, nil
	}

	desc := mystery.Body
	desc = truncateDesc(desc)

	status := "Open"
	if mystery.Solved {
		status = "Solved"
	}

	title := fmt.Sprintf("%s (%s) - Mystery by %s", mystery.Title, status, mystery.AuthorDisplayName)

	return &Meta{
		Title:       title,
		Description: desc,
		URL:         fmt.Sprintf("%s/mystery/%s", r.baseURL, idStr),
	}, nil
}

func (r *Resolver) announcementMeta(ctx context.Context, idStr string) (*Meta, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, nil
	}

	ann, err := r.announcementRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if ann == nil {
		return nil, nil
	}

	desc := ann.Body
	desc = truncateDesc(desc)

	title := fmt.Sprintf("%s - Announcement by %s", ann.Title, ann.AuthorDisplayName)

	return &Meta{
		Title:       title,
		Description: desc,
		URL:         fmt.Sprintf("%s/announcements/%s", r.baseURL, idStr),
	}, nil
}

func (r *Resolver) shipMeta(ctx context.Context, idStr string) (*Meta, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, nil
	}

	ship, err := r.shipRepo.GetByID(ctx, spec.ShipLookup{ID: id, ViewerID: uuid.Nil})
	if err != nil {
		return nil, err
	}

	if ship == nil {
		return nil, nil
	}

	characters, err := r.shipRepo.GetCharacters(ctx, id)
	if err != nil {
		return nil, err
	}

	charNames := make([]string, len(characters))
	for i, c := range characters {
		charNames[i] = c.CharacterName
	}
	pairing := strings.Join(charNames, " \u00D7 ")

	desc := ship.Description
	if desc == "" {
		desc = fmt.Sprintf("A ship by %s featuring %s", ship.AuthorDisplayName, pairing)
	}
	desc = truncateDesc(desc)

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
	return meta, nil
}

func (r *Resolver) ocMeta(ctx context.Context, idStr string) (*Meta, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, nil
	}

	row, err := r.ocRepo.GetByID(ctx, spec.OCByID{ID: id, ViewerID: uuid.Nil})
	if err != nil {
		return nil, err
	}

	if row == nil {
		return nil, nil
	}

	desc := row.Description
	if desc == "" {
		seriesLabel := row.Series
		if row.Series == "custom" && row.CustomSeriesName != "" {
			seriesLabel = row.CustomSeriesName
		}
		desc = fmt.Sprintf("%s, an OC by %s (%s)", row.Name, row.AuthorDisplayName, seriesLabel)
	}
	desc = truncateDesc(desc)

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
	return meta, nil
}

func (r *Resolver) fanficMeta(ctx context.Context, idStr string) (*Meta, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, nil
	}

	fanfic, err := r.fanficRepo.GetByID(ctx, spec.FanficLookup{ID: id, ViewerID: uuid.Nil})
	if err != nil {
		return nil, err
	}

	if fanfic == nil || fanfic.Status == "draft" {
		return nil, nil
	}

	desc := fanfic.Summary
	if desc == "" {
		desc = fmt.Sprintf("A fanfic by %s", fanfic.AuthorDisplayName)
	}
	desc = truncateDesc(desc)

	meta := &Meta{
		Title:       fanfic.Title + " - Fanfiction",
		Description: desc,
		URL:         fmt.Sprintf("%s/fanfiction/%s", r.baseURL, idStr),
	}
	if fanfic.CoverImageURL != "" {
		meta.Image = fanfic.CoverImageURL
	}
	return meta, nil
}

func (r *Resolver) journalMeta(ctx context.Context, idStr string) (*Meta, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, nil
	}

	journal, err := r.journalRepo.GetByID(ctx, spec.JournalLookup{ID: id, ViewerID: uuid.Nil})
	if err != nil {
		return nil, err
	}

	if journal == nil {
		return nil, nil
	}

	desc := journal.LatestEntryExcerpt
	desc = truncateDesc(desc)
	if desc == "" {
		desc = fmt.Sprintf("%s's Reading Journal", journal.Author.DisplayName)
	}

	title := fmt.Sprintf("%s - %s's Reading Journal", journal.Title, journal.Author.DisplayName)

	return &Meta{
		Title:       title,
		Description: desc,
		URL:         fmt.Sprintf("%s/journals/%s", r.baseURL, idStr),
	}, nil
}

func (r *Resolver) journalEntryMeta(ctx context.Context, journalIDStr, numberStr string) (*Meta, error) {
	journalID, err := uuid.Parse(journalIDStr)
	if err != nil {
		return nil, nil
	}
	number, err := strconv.Atoi(numberStr)
	if err != nil || number < 1 {
		return nil, nil
	}

	journal, err := r.journalRepo.GetByID(ctx, spec.JournalLookup{ID: journalID, ViewerID: uuid.Nil})
	if err != nil {
		return nil, err
	}

	if journal == nil {
		return nil, nil
	}

	entry, err := r.journalRepo.GetEntry(ctx, spec.JournalEntryLookup{JournalID: journalID, EntryNumber: number})
	if err != nil {
		return nil, err
	}

	if entry == nil || entry.IsDraft {
		return nil, nil
	}

	desc := entry.Body
	desc = truncateDesc(desc)
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
	}, nil
}

func (r *Resolver) secretMeta(ctx context.Context, id string) (*Meta, error) {
	secret, ok := secrets.Lookup(id)
	if !ok || secret.Title == "" {
		return nil, nil
	}

	siteName, _ := r.getSiteMeta(ctx)
	desc := secret.Description
	if desc == "" {
		desc = fmt.Sprintf("A hidden hunt on %s.", siteName)
	}
	desc = truncateDesc(desc)
	return &Meta{
		Title:       fmt.Sprintf("%s - %s", secret.Title, siteName),
		Description: desc,
		URL:         fmt.Sprintf("%s/secrets/%s", r.baseURL, id),
	}, nil
}

func (r *Resolver) roomMeta(ctx context.Context, idStr string) (*Meta, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, nil
	}

	room, err := r.chatRepo.GetRoomByID(ctx, spec.ChatRoomViewer{RoomID: id, ViewerID: uuid.Nil})
	if err != nil {
		return nil, err
	}

	if !room.PubliclyVisible() {
		return nil, nil
	}

	desc := room.Description
	if desc == "" {
		siteName, _ := r.getSiteMeta(ctx)
		desc = fmt.Sprintf("A chat room with %d members on %s", room.MemberCount, siteName)
	}
	desc = truncateDesc(desc)

	return &Meta{
		Title:       room.Name + " - Chat Room",
		Description: desc,
		URL:         fmt.Sprintf("%s/rooms/%s", r.baseURL, idStr),
	}, nil
}

func (r *Resolver) watchPartyMeta(ctx context.Context, roomIDStr, partyIDStr string) (*Meta, error) {
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		return nil, nil
	}

	partyID, err := uuid.Parse(partyIDStr)
	if err != nil {
		return nil, nil
	}

	session, err := r.watchPartyRepo.GetByID(ctx, partyID)
	if err != nil {
		return nil, err
	}

	if session == nil || session.RoomID != roomID {
		return nil, nil
	}

	room, err := r.chatRepo.GetRoomByID(ctx, spec.ChatRoomViewer{RoomID: roomID, ViewerID: uuid.Nil})
	if err != nil {
		return nil, err
	}

	if !room.PubliclyVisible() {
		return nil, nil
	}

	siteName, _ := r.getSiteMeta(ctx)

	title := fmt.Sprintf("Watch Party in %s", room.Name)
	if session.Title != "" {
		title = fmt.Sprintf("%s - Watch Party in %s", session.Title, room.Name)
	}

	desc := fmt.Sprintf("A watch party in %s on %s", room.Name, siteName)
	if session.Status != "active" {
		desc = fmt.Sprintf("A finished watch party in %s on %s", room.Name, siteName)
	}

	desc = truncateDesc(desc)

	return &Meta{
		Title:       title,
		Description: desc,
		URL:         fmt.Sprintf("%s/rooms/%s?party=%s", r.baseURL, roomIDStr, partyIDStr),
	}, nil
}

func (r *Resolver) liveStreamMetaByUsername(ctx context.Context, username string) (*Meta, error) {
	stream, err := r.liveStreamRepo.GetActiveByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	if meta := r.liveStreamMeta(ctx, stream); meta != nil {
		return meta, nil
	}

	return r.offlineStreamMeta(ctx, username)
}

func (r *Resolver) offlineStreamMeta(ctx context.Context, username string) (*Meta, error) {
	u, _, err := r.userRepo.GetProfileByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	if u == nil {
		return nil, nil
	}

	siteName, _ := r.getSiteMeta(ctx)
	name := u.DisplayLabel()

	meta := &Meta{
		Title:       fmt.Sprintf("%s is not live right now", name),
		Description: truncateDesc(fmt.Sprintf("%s's stream page on %s. This address stays the same, so open it whenever you want to see if they are on air.", name, siteName)),
		URL:         fmt.Sprintf("%s/%s/live", r.baseURL, u.Username),
	}
	if u.AvatarURL != "" {
		meta.Image = r.absoluteURL(u.AvatarURL)
	}

	return meta, nil
}

func (r *Resolver) liveStreamMeta(ctx context.Context, stream *model.LiveStreamRow) *Meta {
	if stream == nil || stream.Status != statusLive {
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

	desc = truncateDesc(desc)

	meta := &Meta{
		Title:       fmt.Sprintf("%s - %s's live stream", stream.Title, name),
		Description: desc,
		URL:         fmt.Sprintf("%s/%s/live", r.baseURL, stream.Username),
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

func truncateDesc(desc string) string {
	if utf8.RuneCountInString(desc) <= maxDescRunes {
		return desc
	}

	return text.ClampRunes(desc, maxDescClipRunes) + "..."
}

func escapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
