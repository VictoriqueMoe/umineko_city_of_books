package og

import (
	"context"
	"strings"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
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
		session   *repository.ChatWatchPartySessionRow
		wantTitle string
	}{
		{
			name:      "party title wins over room name",
			session:   &repository.ChatWatchPartySessionRow{ID: partyID, RoomID: roomID, Title: "Umineko Episode 4", Status: "active"},
			wantTitle: "Umineko Episode 4 - Watch Party in Rokkenjima",
		},
		{
			name:      "untitled party falls back to a generic label",
			session:   &repository.ChatWatchPartySessionRow{ID: partyID, RoomID: roomID, Title: "", Status: "active"},
			wantTitle: "Watch Party in Rokkenjima",
		},
		{
			name:      "party belonging to another room is ignored",
			session:   &repository.ChatWatchPartySessionRow{ID: partyID, RoomID: otherRoomID, Title: "Somewhere Else", Status: "active"},
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
			chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, uuid.Nil).
				Return(&repository.ChatRoomRow{ID: roomID, Name: "Rokkenjima", Description: "A room", Type: dto.RoomTypeGroup, IsPublic: true}, nil)

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
	chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, uuid.Nil).
		Return(&repository.ChatRoomRow{ID: roomID, Name: "Rokkenjima", Type: dto.RoomTypeGroup, IsPublic: true}, nil)

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
		room   *repository.ChatRoomRow
		secret string
	}{
		{name: "private group room", room: &repository.ChatRoomRow{ID: roomID, Name: "Rokkenjima Conspiracy", Description: "Plotting", Type: dto.RoomTypeGroup, IsPublic: false}, secret: "Rokkenjima Conspiracy"},
		{name: "direct message", room: &repository.ChatRoomRow{ID: roomID, Name: "Battler and Beatrice", Description: "Private", Type: dto.RoomTypeDM, IsPublic: false}, secret: "Battler and Beatrice"},
		{name: "system room", room: &repository.ChatRoomRow{ID: roomID, Name: "Moderator Log", Type: dto.RoomTypeGroup, IsPublic: true, IsSystem: true}, secret: "Moderator Log"},
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
			chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, uuid.Nil).Return(tc.room, nil)

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
	chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, uuid.Nil).
		Return(&repository.ChatRoomRow{ID: roomID, Name: "Rokkenjima Conspiracy", Type: dto.RoomTypeGroup, IsPublic: false}, nil)

	partyRepo := repository.NewMockChatWatchPartyRepository(t)
	partyRepo.EXPECT().GetByID(mock.Anything, partyID).
		Return(&repository.ChatWatchPartySessionRow{ID: partyID, RoomID: roomID, Title: "Secret Screening", Status: "active"}, nil)

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
