package og

import (
	"context"
	"strings"
	"testing"

	"umineko_city_of_books/internal/cache"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntityPath(t *testing.T) {
	tests := []struct {
		name string
		kind Kind
		id   string
		want string
	}{
		{name: "theory", kind: KindTheory, id: "abc", want: "/theory/abc"},
		{name: "post lives under game-board", kind: KindPost, id: "abc", want: "/game-board/abc"},
		{name: "art", kind: KindArt, id: "abc", want: "/gallery/art/abc"},
		{name: "gallery", kind: KindGallery, id: "abc", want: "/gallery/view/abc"},
		{name: "mystery", kind: KindMystery, id: "abc", want: "/mystery/abc"},
		{name: "ship", kind: KindShip, id: "abc", want: "/ships/abc"},
		{name: "oc", kind: KindOC, id: "abc", want: "/oc/abc"},
		{name: "announcement", kind: KindAnnouncement, id: "abc", want: "/announcements/abc"},
		{name: "fanfic", kind: KindFanfic, id: "abc", want: "/fanfiction/abc"},
		{name: "journal", kind: KindJournal, id: "abc", want: "/journals/abc"},
		{name: "room", kind: KindRoom, id: "abc", want: "/rooms/abc"},
		{name: "live stream keys on username", kind: KindLiveStream, id: "kyle", want: "/kyle/live"},
		{name: "user keys on username", kind: KindUser, id: "kyle", want: "/user/kyle"},
		{name: "unknown kind yields no path", kind: Kind("nonsense"), id: "abc", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a content kind and an id

			// when the page path is built
			got := entityPath(tc.kind, tc.id)

			// then it matches the route the SPA serves that entity on
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEntityPathIsRoutedByMetaForPath(t *testing.T) {
	entityKinds := []Kind{
		KindTheory, KindPost, KindArt, KindGallery, KindMystery, KindShip, KindOC,
		KindAnnouncement, KindFanfic, KindJournal, KindRoom, KindLiveStream, KindUser,
	}

	for _, kind := range entityKinds {
		t.Run(string(kind), func(t *testing.T) {
			// given the path this kind's cache entry is keyed on
			path := entityPath(kind, uuid.New().String())
			require.NotEmpty(t, path)

			// when it is split the way metaForPath splits an incoming request
			parts := strings.Split(strings.Trim(path, "/"), "/")

			// then it lands on an entity branch, never the bare listing page
			assert.GreaterOrEqual(t, len(parts), 2, "path %q must carry an id segment", path)
		})
	}
}

func TestCanonicalMetaPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "plain path is untouched", path: "/theory/abc", want: "/theory/abc"},
		{name: "trailing slash is trimmed", path: "/theory/abc/", want: "/theory/abc"},
		{name: "repeated trailing slashes are trimmed", path: "/theory/abc///", want: "/theory/abc"},
		{name: "root stays root", path: "/", want: "/"},
		{name: "empty becomes root", path: "", want: "/"},
		{name: "casing collapses onto one key", path: "/Featherine/live", want: "/featherine/live"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a request path

			// when it is canonicalised for the cache key
			got := canonicalMetaPath(tc.path)

			// then variants of the same page collapse onto one key
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResolverClearMetaCache(t *testing.T) {
	t.Run("deletes the key the resolver would read", func(t *testing.T) {
		// given a cached meta entry for a journal
		ctx := context.Background()
		mgr := cache.New()
		r := &Resolver{cache: mgr}
		id := uuid.New().String()
		key := cache.OGMeta.Key(entityPath(KindJournal, id))
		require.NoError(t, mgr.Set(ctx, key, &Meta{Title: "doomed"}, cache.OGMeta.TTL))

		// when the journal is cleared
		require.NoError(t, r.ClearMetaCache(ctx, KindJournal, id))

		// then the entry is gone rather than waiting out its ttl
		_, err := mgr.Get[*Meta](ctx, key)
		assert.Error(t, err)
	})

	t.Run("a trailing slash request shares the cleared key", func(t *testing.T) {
		// given an entry cached by a request that carried a trailing slash
		ctx := context.Background()
		mgr := cache.New()
		r := &Resolver{cache: mgr}
		id := uuid.New().String()
		key := cache.OGMeta.Key(metaKeyParts(entityPath(KindTheory, id)+"/", "")...)
		require.NoError(t, mgr.Set(ctx, key, &Meta{Title: "doomed"}, cache.OGMeta.TTL))

		// when the theory is cleared
		require.NoError(t, r.ClearMetaCache(ctx, KindTheory, id))

		// then that entry is gone too, because both spellings canonicalise the same
		_, err := mgr.Get[*Meta](ctx, key)
		assert.Error(t, err)
	})

	t.Run("unknown kind is a no-op", func(t *testing.T) {
		// given a resolver
		ctx := context.Background()
		r := &Resolver{cache: cache.New()}

		// when a kind with no page is cleared
		err := r.ClearMetaCache(ctx, Kind("nonsense"), uuid.New().String())

		// then it neither errors nor deletes a wrong key
		assert.NoError(t, err)
	})

	t.Run("nil resolver is a no-op", func(t *testing.T) {
		// given no resolver wired in, as in a unit test
		var r *Resolver

		// when a clear is attempted
		err := r.ClearMetaCache(context.Background(), KindJournal, uuid.New().String())

		// then the caller is not forced to nil check
		assert.NoError(t, err)
	})
}
