package og

import (
	"context"
	"strings"
	"testing"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const testBaseHTML = `<head>
<title>When They Cry City of Books</title>
<meta name="description" content="A social platform for fans of Umineko, Higurashi, and the wider When They Cry series. Post theories, solve mysteries, share fan art, chronicle read-throughs, ship pairings, write fanfiction, and chat in live rooms.">
<meta property="og:title" content="When They Cry City of Books">
<meta property="og:site_name" content="When They Cry City of Books">
<meta property="og:description" content="A social platform for fans of Umineko, Higurashi, and the wider When They Cry series. Post theories, solve mysteries, share fan art, chronicle read-throughs, ship pairings, write fanfiction, and chat in live rooms.">
<meta property="og:url" content="https://example.com/">
<meta property="og:image" content="https://example.com/Featherine.jpg">
<meta property="og:image:type" content="image/jpeg">
<meta property="og:image:width" content="2734">
<meta property="og:image:height" content="1533">
<meta name="twitter:title" content="When They Cry City of Books">
<meta name="twitter:description" content="A social platform for fans of Umineko, Higurashi, and the wider When They Cry series. Post theories, solve mysteries, share fan art, chronicle read-throughs, ship pairings, write fanfiction, and chat in live rooms.">
<meta name="twitter:image" content="https://example.com/Featherine.jpg">
<link rel="canonical" href="https://example.com/">
</head>`

func newTestResolver(t *testing.T, ogDefaultImage string) *Resolver {
	ss := settings.NewMockService(t)
	ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return(ogDefaultImage)
	ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("")
	return &Resolver{settingsSvc: ss, baseHTML: testBaseHTML, baseURL: "https://example.com"}
}

func TestResolver_Resolve_DefaultImage(t *testing.T) {
	tests := []struct {
		name           string
		ogDefaultImage string
		path           string
		wantImage      string
		wantSizeTags   bool
	}{
		{name: "builtin image when unset", ogDefaultImage: "", path: "/mysteries", wantImage: "https://example.com/Featherine.jpg", wantSizeTags: true},
		{name: "custom image on meta page", ogDefaultImage: "/uploads/branding/og_default_1.jpg", path: "/mysteries", wantImage: "https://example.com/uploads/branding/og_default_1.jpg", wantSizeTags: false},
		{name: "custom image on unknown page", ogDefaultImage: "/uploads/branding/og_default_1.jpg", path: "/some/unknown/page", wantImage: "https://example.com/uploads/branding/og_default_1.jpg", wantSizeTags: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			r := newTestResolver(t, tc.ogDefaultImage)

			// when
			html := r.Resolve(context.Background(), tc.path, "")

			// then
			assert.Contains(t, html, `property="og:image" content="`+tc.wantImage+`"`)
			assert.Contains(t, html, `name="twitter:image" content="`+tc.wantImage+`"`)
			assert.Equal(t, tc.wantSizeTags, strings.Contains(html, "og:image:width"))
		})
	}
}

func TestResolver_Resolve_SiteName(t *testing.T) {
	// given
	ss := settings.NewMockService(t)
	ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("Custom Site")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("A custom description")
	r := &Resolver{settingsSvc: ss, baseHTML: testBaseHTML, baseURL: "https://example.com"}

	// when
	html := r.Resolve(context.Background(), "/", "")

	// then
	assert.Contains(t, html, `<title>Custom Site</title>`)
	assert.Contains(t, html, `property="og:title" content="Custom Site"`)
	assert.Contains(t, html, `property="og:site_name" content="Custom Site"`)
	assert.Contains(t, html, `property="og:description" content="A custom description"`)
}

func TestResolver_Resolve_WatchParty(t *testing.T) {
	roomID := uuid.New()
	partyID := uuid.New()
	otherRoomID := uuid.New()

	tests := []struct {
		name      string
		session   *model.ChatWatchPartySessionRow
		wantTitle string
	}{
		{
			name:      "party title wins over room name",
			session:   &model.ChatWatchPartySessionRow{ID: partyID, RoomID: roomID, Title: "Umineko Episode 4", Status: "active"},
			wantTitle: "Umineko Episode 4 - Watch Party in Rokkenjima",
		},
		{
			name:      "untitled party falls back to a generic label",
			session:   &model.ChatWatchPartySessionRow{ID: partyID, RoomID: roomID, Title: "", Status: "active"},
			wantTitle: "Watch Party in Rokkenjima",
		},
		{
			name:      "party belonging to another room is ignored",
			session:   &model.ChatWatchPartySessionRow{ID: partyID, RoomID: otherRoomID, Title: "Somewhere Else", Status: "active"},
			wantTitle: "Rokkenjima - Chat Room",
		},
		{
			name:      "missing party falls back to the room",
			session:   nil,
			wantTitle: "Rokkenjima - Chat Room",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			ss := settings.NewMockService(t)
			ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return("")
			ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("")
			ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("")

			chatRepo := repository.NewMockChatRepository(t)
			chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: uuid.Nil}).
				Return(&model.ChatRoomRow{ID: roomID, Name: "Rokkenjima", Description: "A room", Type: dto.RoomTypeGroup, IsPublic: true}, nil)

			partyRepo := repository.NewMockChatWatchPartyRepository(t)
			partyRepo.EXPECT().GetByID(mock.Anything, partyID).Return(tc.session, nil)

			r := &Resolver{
				settingsSvc:    ss,
				chatRepo:       chatRepo,
				watchPartyRepo: partyRepo,
				baseHTML:       testBaseHTML,
				baseURL:        "https://example.com",
			}

			// when
			html := r.Resolve(context.Background(), "/rooms/"+roomID.String(), partyID.String())

			// then
			assert.Contains(t, html, `property="og:title" content="`+tc.wantTitle+`"`)
		})
	}
}

func TestResolver_Resolve_IgnoresNonUUIDPartyParam(t *testing.T) {
	// given
	roomID := uuid.New()

	ss := settings.NewMockService(t)
	ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("")

	chatRepo := repository.NewMockChatRepository(t)
	chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: uuid.Nil}).
		Return(&model.ChatRoomRow{ID: roomID, Name: "Rokkenjima", Type: dto.RoomTypeGroup, IsPublic: true}, nil)

	r := &Resolver{
		settingsSvc: ss,
		chatRepo:    chatRepo,
		baseHTML:    testBaseHTML,
		baseURL:     "https://example.com",
	}

	// when
	html := r.Resolve(context.Background(), "/rooms/"+roomID.String(), "not-a-uuid")

	// then
	assert.Contains(t, html, `property="og:title" content="Rokkenjima - Chat Room"`)
}

func TestResolver_Resolve_HidesRoomsThatAreNotPubliclyListed(t *testing.T) {
	roomID := uuid.New()

	tests := []struct {
		name   string
		room   *model.ChatRoomRow
		secret string
	}{
		{name: "private group room", room: &model.ChatRoomRow{ID: roomID, Name: "Rokkenjima Conspiracy", Description: "Plotting", Type: dto.RoomTypeGroup, IsPublic: false}, secret: "Rokkenjima Conspiracy"},
		{name: "direct message", room: &model.ChatRoomRow{ID: roomID, Name: "Battler and Beatrice", Description: "Private", Type: dto.RoomTypeDM, IsPublic: false}, secret: "Battler and Beatrice"},
		{name: "system room", room: &model.ChatRoomRow{ID: roomID, Name: "Moderator Log", Type: dto.RoomTypeGroup, IsPublic: true, IsSystem: true}, secret: "Moderator Log"},
		{name: "room does not exist", room: nil, secret: "should-not-appear"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			ss := settings.NewMockService(t)
			ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return("")
			ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("")
			ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("")

			chatRepo := repository.NewMockChatRepository(t)
			chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: uuid.Nil}).Return(tc.room, nil)

			r := &Resolver{settingsSvc: ss, chatRepo: chatRepo, baseHTML: testBaseHTML, baseURL: "https://example.com"}

			// when
			html := r.Resolve(context.Background(), "/rooms/"+roomID.String(), "")

			// then
			assert.NotContains(t, html, tc.secret)
			assert.Contains(t, html, `property="og:title" content="When They Cry City of Books"`)
		})
	}
}

func TestResolver_Resolve_HidesWatchPartyInPrivateRoom(t *testing.T) {
	// given
	roomID := uuid.New()
	partyID := uuid.New()

	ss := settings.NewMockService(t)
	ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("")

	chatRepo := repository.NewMockChatRepository(t)
	chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: uuid.Nil}).
		Return(&model.ChatRoomRow{ID: roomID, Name: "Rokkenjima Conspiracy", Type: dto.RoomTypeGroup, IsPublic: false}, nil)

	partyRepo := repository.NewMockChatWatchPartyRepository(t)
	partyRepo.EXPECT().GetByID(mock.Anything, partyID).
		Return(&model.ChatWatchPartySessionRow{ID: partyID, RoomID: roomID, Title: "Secret Screening", Status: "active"}, nil)

	r := &Resolver{settingsSvc: ss, chatRepo: chatRepo, watchPartyRepo: partyRepo, baseHTML: testBaseHTML, baseURL: "https://example.com"}

	// when
	html := r.Resolve(context.Background(), "/rooms/"+roomID.String(), partyID.String())

	// then
	assert.NotContains(t, html, "Secret Screening")
	assert.NotContains(t, html, "Rokkenjima Conspiracy")
	assert.Contains(t, html, `property="og:title" content="When They Cry City of Books"`)
}

func TestResolver_OGImageURL(t *testing.T) {
	r := &Resolver{baseURL: "https://example.com"}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "webp upload rewritten to jpeg endpoint", in: "https://example.com/uploads/posts/abc.webp", want: "https://example.com/og-image/posts/abc.jpg"},
		{name: "uppercase extension rewritten", in: "https://example.com/uploads/posts/abc.WEBP", want: "https://example.com/og-image/posts/abc.jpg"},
		{name: "non webp upload untouched", in: "https://example.com/uploads/posts/abc.gif", want: "https://example.com/uploads/posts/abc.gif"},
		{name: "external url untouched", in: "https://media.giphy.com/abc.webp", want: "https://media.giphy.com/abc.webp"},
		{name: "default image untouched", in: "https://example.com/Featherine.jpg", want: "https://example.com/Featherine.jpg"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := r.ogImageURL(tc.in)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResolver_Resolve_HidesDraftFanfic(t *testing.T) {
	// given
	fanficID := uuid.New()

	ss := settings.NewMockService(t)
	ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("")

	fanficRepo := repository.NewMockFanficRepository(t)
	fanficRepo.EXPECT().GetByID(mock.Anything, spec.FanficLookup{ID: fanficID, ViewerID: uuid.Nil}).
		Return(&model.FanficRow{ID: fanficID, Title: "Unfinished Golden Witch", Summary: "Secret draft summary", CoverImageURL: "https://example.com/uploads/fanfics/secret.webp", Status: "draft"}, nil)

	r := &Resolver{settingsSvc: ss, fanficRepo: fanficRepo, baseHTML: testBaseHTML, baseURL: "https://example.com"}

	// when
	html := r.Resolve(context.Background(), "/fanfiction/"+fanficID.String(), "")

	// then
	assert.NotContains(t, html, "Unfinished Golden Witch")
	assert.NotContains(t, html, "Secret draft summary")
	assert.NotContains(t, html, "secret.webp")
	assert.Contains(t, html, `property="og:title" content="When They Cry City of Books"`)
}

func TestResolver_Resolve_HidesDraftJournalEntry(t *testing.T) {
	// given
	journalID := uuid.New()
	draftTitle := "Working notes on Episode 8"

	ss := settings.NewMockService(t)
	ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("")

	journalRepo := repository.NewMockJournalRepository(t)
	journalRepo.EXPECT().GetByID(mock.Anything, spec.JournalLookup{ID: journalID, ViewerID: uuid.Nil}).
		Return(&dto.JournalResponse{ID: journalID, Title: "Umineko Read-through", Author: dto.UserResponse{DisplayName: "Battler"}}, nil)
	journalRepo.EXPECT().GetEntry(mock.Anything, spec.JournalEntryLookup{JournalID: journalID, EntryNumber: 6}).
		Return(&model.JournalEntryRow{JournalID: journalID, EntryNumber: 6, Title: &draftTitle, Body: "Secret unpublished body", IsDraft: true}, nil)

	r := &Resolver{settingsSvc: ss, journalRepo: journalRepo, baseHTML: testBaseHTML, baseURL: "https://example.com"}

	// when
	html := r.Resolve(context.Background(), "/journals/"+journalID.String()+"/entry/6", "")

	// then
	assert.NotContains(t, html, draftTitle)
	assert.NotContains(t, html, "Secret unpublished body")
	assert.Contains(t, html, `property="og:title" content="When They Cry City of Books"`)
}

func TestResolver_Resolve_LiveStreamByUsername(t *testing.T) {
	streamID := uuid.New()
	row := &model.LiveStreamRow{
		ID:          streamID,
		Title:       "Ciconia blind run",
		Status:      "live",
		Username:    "Featherine",
		DisplayName: "Featherine",
	}

	tests := []struct {
		name      string
		path      string
		row       *model.LiveStreamRow
		wantTitle string
		wantURL   string
	}{
		{
			name:      "stable username url resolves the live stream",
			path:      "/Featherine/live",
			row:       row,
			wantTitle: "Ciconia blind run - Featherine's live stream",
			wantURL:   "https://example.com/Featherine/live",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a request for a streamer's stable live URL
			ss := settings.NewMockService(t)
			ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return("")
			ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("")
			ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("")

			streamRepo := repository.NewMockLiveStreamRepository(t)
			streamRepo.EXPECT().GetActiveByUsername(mock.Anything, "Featherine").Return(tc.row, nil)

			r := &Resolver{settingsSvc: ss, liveStreamRepo: streamRepo, cache: cache.New(), baseHTML: testBaseHTML, baseURL: "https://example.com"}

			// when the crawler is served
			html := r.Resolve(context.Background(), tc.path, "")

			// then the card advertises the stable url
			assert.Contains(t, html, `property="og:title" content="`+tc.wantTitle+`"`)
			assert.Contains(t, html, `property="og:url" content="`+tc.wantURL+`"`)
		})
	}
}

func TestResolver_Resolve_OfflineStreamerStillGetsTheirOwnCard(t *testing.T) {
	tests := []struct {
		name      string
		user      *model.User
		wantTitle string
		wantImage string
	}{
		{
			name:      "an offline streamer is named, with their avatar converted for crawlers",
			user:      &model.User{Username: "Featherine", DisplayName: "Featherine", AvatarURL: "/uploads/avatars/f.webp"},
			wantTitle: "Featherine is not live right now",
			wantImage: "https://example.com/og-image/avatars/f.jpg",
		},
		{
			name:      "an offline streamer with no avatar keeps the site default image",
			user:      &model.User{Username: "Featherine", DisplayName: "Featherine"},
			wantTitle: "Featherine is not live right now",
			wantImage: "https://example.com/Featherine.jpg",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a streamer who is not broadcasting
			ss := settings.NewMockService(t)
			ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return("")
			ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("")
			ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("")

			streamRepo := repository.NewMockLiveStreamRepository(t)
			streamRepo.EXPECT().GetActiveByUsername(mock.Anything, "Featherine").Return(nil, nil)

			userRepo := repository.NewMockUserRepository(t)
			userRepo.EXPECT().GetProfileByUsername(mock.Anything, "Featherine").Return(tc.user, nil, nil)

			r := &Resolver{settingsSvc: ss, liveStreamRepo: streamRepo, userRepo: userRepo, cache: cache.New(), baseHTML: testBaseHTML, baseURL: "https://example.com"}

			// when a crawler unfurls the stable url
			html := r.Resolve(context.Background(), "/Featherine/live", "")

			// then the card is about the streamer, not the site
			assert.Contains(t, html, `property="og:title" content="`+tc.wantTitle+`"`)
			assert.Contains(t, html, `property="og:url" content="https://example.com/Featherine/live"`)
			assert.Contains(t, html, `property="og:image" content="`+tc.wantImage+`"`)
		})
	}
}

func TestResolver_Resolve_UnknownStreamerFallsBackToTheSiteCard(t *testing.T) {
	// given a name that belongs to nobody
	ss := settings.NewMockService(t)
	ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("")

	streamRepo := repository.NewMockLiveStreamRepository(t)
	streamRepo.EXPECT().GetActiveByUsername(mock.Anything, "nobody").Return(nil, nil)

	userRepo := repository.NewMockUserRepository(t)
	userRepo.EXPECT().GetProfileByUsername(mock.Anything, "nobody").Return(nil, nil, nil)

	r := &Resolver{settingsSvc: ss, liveStreamRepo: streamRepo, userRepo: userRepo, cache: cache.New(), baseHTML: testBaseHTML, baseURL: "https://example.com"}

	// when
	html := r.Resolve(context.Background(), "/nobody/live", "")

	// then there is nothing to say, so the generic card stands
	assert.Contains(t, html, `property="og:title" content="When They Cry City of Books"`)
}

func TestResolver_Resolve_RetiredStreamURLHasNoCard(t *testing.T) {
	// given a link shared under the retired /live/{uuid} address
	streamID := uuid.New()

	ss := settings.NewMockService(t)
	ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("")

	streamRepo := repository.NewMockLiveStreamRepository(t)

	r := &Resolver{settingsSvc: ss, liveStreamRepo: streamRepo, cache: cache.New(), baseHTML: testBaseHTML, baseURL: "https://example.com"}

	// when a crawler unfurls it
	html := r.Resolve(context.Background(), "/live/"+streamID.String(), "")

	// then no stream is looked up and the generic site card is served
	assert.Contains(t, html, `property="og:title" content="When They Cry City of Books"`)
	streamRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestResolver_Resolve_ReservedSegmentsAreNotStreamers(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantTitle string
	}{
		{name: "games live keeps its own card", path: "/games/live", wantTitle: "Live Games - When They Cry City of Books"},
		{name: "the live directory keeps its own card", path: "/live", wantTitle: "Live Streams - When They Cry City of Books"},
		{name: "a static mount is never a streamer", path: "/uploads/live", wantTitle: "When They Cry City of Books"},
		{name: "a protected page is never a streamer", path: "/settings/live", wantTitle: "When They Cry City of Books"},
		{name: "casing does not get a reserved name past the guard", path: "/Games/LIVE", wantTitle: "When They Cry City of Books"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a reserved first segment followed by "live"
			ss := settings.NewMockService(t)
			ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return("")
			ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("")
			ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("")

			streamRepo := repository.NewMockLiveStreamRepository(t)

			r := &Resolver{settingsSvc: ss, liveStreamRepo: streamRepo, cache: cache.New(), baseHTML: testBaseHTML, baseURL: "https://example.com"}

			// when it is resolved
			html := r.Resolve(context.Background(), tc.path, "")

			// then no streamer lookup happens and the site's own meta stands
			assert.Contains(t, html, `property="og:title" content="`+tc.wantTitle+`"`)
			streamRepo.AssertNotCalled(t, "GetActiveByUsername", mock.Anything, mock.Anything)
		})
	}
}

func TestResolver_Resolve_LiveStreamCasingSharesOneCard(t *testing.T) {
	// given a live streamer whose stable url is linked in mixed case
	ss := settings.NewMockService(t)
	ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("")

	streamRepo := repository.NewMockLiveStreamRepository(t)
	streamRepo.EXPECT().GetActiveByUsername(mock.Anything, mock.Anything).
		Return(&model.LiveStreamRow{Title: "Ciconia blind run", Status: "live", Username: "Featherine", DisplayName: "Featherine"}, nil)

	r := &Resolver{settingsSvc: ss, liveStreamRepo: streamRepo, cache: cache.New(), baseHTML: testBaseHTML, baseURL: "https://example.com"}

	// when an oddly cased variant is crawled first, then the real link
	first := r.Resolve(context.Background(), "/FEATHERINE/LIVE", "")
	second := r.Resolve(context.Background(), "/featherine/live", "")

	// then neither poisons the other: both render the same card
	assert.Contains(t, first, `property="og:url" content="https://example.com/Featherine/live"`)
	assert.Contains(t, second, `property="og:url" content="https://example.com/Featherine/live"`)
}

func TestResolver_Resolve_CasingDoesNotPoisonOtherEntities(t *testing.T) {
	// given a page whose path segment is case-sensitive
	ss := settings.NewMockService(t)
	ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("")

	r := &Resolver{settingsSvc: ss, cache: cache.New(), baseHTML: testBaseHTML, baseURL: "https://example.com"}

	// when a mis-cased variant is crawled before the real one
	r.Resolve(context.Background(), "/GAMES/past", "")
	html := r.Resolve(context.Background(), "/games/past", "")

	// then the real page still gets its own card
	assert.Contains(t, html, `property="og:title" content="Past Games - When They Cry City of Books"`)
}

func TestResolver_Resolve_PostSpoilerImageStaysOffTheCard(t *testing.T) {
	postID := uuid.New()

	tests := []struct {
		name      string
		media     []model.PostMediaRow
		wantImage bool
	}{
		{
			name:      "an ordinary attachment becomes the card image",
			media:     []model.PostMediaRow{{MediaURL: "/uploads/posts/one.png", MediaType: "image"}},
			wantImage: true,
		},
		{
			name:      "a spoilered attachment is withheld",
			media:     []model.PostMediaRow{{MediaURL: "/uploads/posts/ending.png", MediaType: "image", IsSpoiler: true}},
			wantImage: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a post whose first attachment may or may not be a spoiler
			ss := settings.NewMockService(t)
			ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return("")
			ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("")
			ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("")

			postRepo := repository.NewMockPostRepository(t)
			postRepo.EXPECT().GetByID(mock.Anything, spec.PostLookup{ID: postID, ViewerID: uuid.Nil}).
				Return(&model.PostRow{ID: postID, Body: "look at this"}, nil)
			postRepo.EXPECT().GetMedia(mock.Anything, postID).Return(tc.media, nil)

			r := &Resolver{settingsSvc: ss, postRepo: postRepo, cache: cache.New(), baseHTML: testBaseHTML, baseURL: "https://example.com"}

			// when a crawler unfurls the post
			html := r.Resolve(context.Background(), "/game-board/"+postID.String(), "")

			// then a spoilered image never reaches the card
			if tc.wantImage {
				assert.Contains(t, html, "/uploads/posts/one.png")
				return
			}

			assert.NotContains(t, html, "ending.png")
		})
	}
}
