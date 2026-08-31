package mention

import (
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testEntityID = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	testChildID  = uuid.MustParse("55555555-5555-5555-5555-555555555555")
)

const (
	testEntityKey   = "gohda-recipe"
	testEntryNumber = 7
)

func TestTargetReferenceContract(t *testing.T) {
	entity := Reference{EntityID: testEntityID, EntityKey: testEntityKey, EntryNumber: testEntryNumber}
	child := Reference{EntityID: testEntityID, EntityKey: testEntityKey, EntryNumber: testEntryNumber, ChildID: testChildID}

	tests := []struct {
		name    string
		ref     Reference
		wantID  uuid.UUID
		refType string
		link    string
	}{
		{
			name:    "post",
			ref:     entity,
			wantID:  testEntityID,
			refType: "post",
			link:    "/game-board/44444444-4444-4444-4444-444444444444",
		},
		{
			name:    "post_comment",
			ref:     child,
			wantID:  testEntityID,
			refType: "post_comment:55555555-5555-5555-5555-555555555555",
			link:    "/game-board/44444444-4444-4444-4444-444444444444#comment-55555555-5555-5555-5555-555555555555",
		},
		{
			name:    "art",
			ref:     entity,
			wantID:  testEntityID,
			refType: "art",
			link:    "/gallery/art/44444444-4444-4444-4444-444444444444",
		},
		{
			name:    "art_comment",
			ref:     child,
			wantID:  testEntityID,
			refType: "art_comment:55555555-5555-5555-5555-555555555555",
			link:    "/gallery/art/44444444-4444-4444-4444-444444444444#comment-55555555-5555-5555-5555-555555555555",
		},
		{
			name:    "theory",
			ref:     entity,
			wantID:  testEntityID,
			refType: "theory",
			link:    "/theory/44444444-4444-4444-4444-444444444444",
		},
		{
			name:    "theory_response",
			ref:     child,
			wantID:  testEntityID,
			refType: "theory_response:55555555-5555-5555-5555-555555555555",
			link:    "/theory/44444444-4444-4444-4444-444444444444#response-55555555-5555-5555-5555-555555555555",
		},
		{
			name:    "ship",
			ref:     entity,
			wantID:  testEntityID,
			refType: "ship",
			link:    "/ships/44444444-4444-4444-4444-444444444444",
		},
		{
			name:    "ship_comment",
			ref:     child,
			wantID:  testEntityID,
			refType: "ship_comment:55555555-5555-5555-5555-555555555555",
			link:    "/ships/44444444-4444-4444-4444-444444444444#comment-55555555-5555-5555-5555-555555555555",
		},
		{
			name:    "oc",
			ref:     entity,
			wantID:  testEntityID,
			refType: "oc",
			link:    "/oc/44444444-4444-4444-4444-444444444444",
		},
		{
			name:    "oc_comment",
			ref:     child,
			wantID:  testEntityID,
			refType: "oc_comment:55555555-5555-5555-5555-555555555555",
			link:    "/oc/44444444-4444-4444-4444-444444444444#comment-55555555-5555-5555-5555-555555555555",
		},
		{
			name:    "mystery",
			ref:     entity,
			wantID:  testEntityID,
			refType: "mystery",
			link:    "/mystery/44444444-4444-4444-4444-444444444444",
		},
		{
			name:    "mystery_attempt",
			ref:     child,
			wantID:  testEntityID,
			refType: "mystery_attempt:55555555-5555-5555-5555-555555555555",
			link:    "/mystery/44444444-4444-4444-4444-444444444444#attempt-55555555-5555-5555-5555-555555555555",
		},
		{
			name:    "mystery_comment",
			ref:     child,
			wantID:  testEntityID,
			refType: "mystery_comment:55555555-5555-5555-5555-555555555555",
			link:    "/mystery/44444444-4444-4444-4444-444444444444#comment-55555555-5555-5555-5555-555555555555",
		},
		{
			name:    "fanfic",
			ref:     entity,
			wantID:  testEntityID,
			refType: "fanfic",
			link:    "/fanfiction/44444444-4444-4444-4444-444444444444",
		},
		{
			name:    "fanfic_comment",
			ref:     child,
			wantID:  testEntityID,
			refType: "fanfic_comment:55555555-5555-5555-5555-555555555555",
			link:    "/fanfiction/44444444-4444-4444-4444-444444444444#comment-55555555-5555-5555-5555-555555555555",
		},
		{
			name:    "announcement",
			ref:     entity,
			wantID:  testEntityID,
			refType: "announcement",
			link:    "/announcements/44444444-4444-4444-4444-444444444444",
		},
		{
			name:    "announcement_comment",
			ref:     child,
			wantID:  testEntityID,
			refType: "announcement_comment:55555555-5555-5555-5555-555555555555",
			link:    "/announcements/44444444-4444-4444-4444-444444444444#comment-55555555-5555-5555-5555-555555555555",
		},
		{
			name:    "journal",
			ref:     entity,
			wantID:  testEntityID,
			refType: "journal",
			link:    "/journals/44444444-4444-4444-4444-444444444444",
		},
		{
			name:    "journal_comment",
			ref:     child,
			wantID:  testEntityID,
			refType: "journal_comment:55555555-5555-5555-5555-555555555555",
			link:    "/journals/44444444-4444-4444-4444-444444444444#comment-55555555-5555-5555-5555-555555555555",
		},
		{
			name:    "journal_entry",
			ref:     entity,
			wantID:  testEntityID,
			refType: "journal_entry:7",
			link:    "/journals/44444444-4444-4444-4444-444444444444/entry/7",
		},
		{
			name:    "journal_entry_comment",
			ref:     child,
			wantID:  testEntityID,
			refType: "journal_entry_comment:7:55555555-5555-5555-5555-555555555555",
			link:    "/journals/44444444-4444-4444-4444-444444444444/entry/7#comment-55555555-5555-5555-5555-555555555555",
		},
		{
			name:    "secret_comment",
			ref:     child,
			wantID:  uuid.Nil,
			refType: "secret_comment:gohda-recipe:55555555-5555-5555-5555-555555555555",
			link:    "/secrets/gohda-recipe#comment-55555555-5555-5555-5555-555555555555",
		},
	}

	covered := make(map[Kind]bool, len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given a kind resolved from the registry
			kind := Kind(tt.name)
			covered[kind] = true

			target, ok := TargetFor(kind)
			require.True(t, ok)

			ref := tt.ref
			ref.Kind = kind

			// when
			gotType := target.ReferenceType(ref)
			gotLink := target.LinkURL(ref)
			gotID := target.ReferenceID(ref)

			// then
			assert.Equal(t, tt.refType, gotType)
			assert.Equal(t, tt.link, gotLink)
			assert.Equal(t, tt.wantID, gotID)
		})
	}

	// then every registered kind is pinned by the table above
	for _, kind := range Kinds() {
		assert.True(t, covered[kind], "kind %q has no golden reference case", kind)
	}
}

func TestRegistryIsComplete(t *testing.T) {
	// given the declared kinds

	// when each is looked up

	// then every one resolves to exactly one target
	assert.Len(t, Targets(), len(Kinds()))

	for _, kind := range Kinds() {
		target, ok := TargetFor(kind)
		assert.True(t, ok, "kind %q has no registry entry", kind)
		assert.Equal(t, kind, target.Kind)
	}
}

func TestRegistryDerivesEverySet(t *testing.T) {
	// given the one table each kind is declared in
	declared := make(map[Kind]entry, len(registry))
	for _, e := range registry {
		declared[e.Kind] = e
	}

	// when the derived sets are read back

	// then they hold exactly what the table declares, with nothing maintained by hand
	require.Len(t, declared, len(Kinds()))

	for _, kind := range Kinds() {
		e, ok := declared[kind]
		require.True(t, ok, "kind %q is not declared in the registry", kind)

		target, ok := TargetFor(kind)
		require.True(t, ok, "kind %q has no target", kind)
		assert.Equal(t, e.Target, target)
		assert.Equal(t, e.Comment, IsCommentKind(kind), "kind %q disagrees with its comment flag", kind)
	}

	for _, kind := range CommentKinds() {
		assert.True(t, IsCommentKind(kind))
		assert.Contains(t, Kinds(), kind)
	}

	var wantComments int
	for _, e := range registry {
		if e.Comment {
			wantComments++
		}
	}

	assert.Len(t, CommentKinds(), wantComments)
}

func TestRegistryRejectsAnIncoherentEntry(t *testing.T) {
	tests := []struct {
		name  string
		entry entry
	}{
		{
			name:  "a kind with no level",
			entry: entry{Target: Target{Kind: Kind("widget"), Path: "/widgets"}},
		},
		{
			name:  "a child kind with no anchor",
			entry: entry{Target: Target{Kind: Kind("widget_comment"), Level: LevelChild, Path: "/widgets"}},
		},
		{
			name:  "an entity kind with an anchor",
			entry: entry{Target: Target{Kind: Kind("widget"), Level: LevelEntity, Path: "/widgets", Anchor: "comment"}},
		},
		{
			name:  "a kind with no path",
			entry: entry{Target: Target{Kind: Kind("widget"), Level: LevelEntity}},
		},
		{
			name:  "a comment kind that is not a child",
			entry: entry{Target: Target{Kind: Kind("widget"), Level: LevelEntity, Path: "/widgets"}, Comment: true},
		},
		{
			name:  "a duplicate kind",
			entry: entry{Target: Target{Kind: KindPost, Level: LevelEntity, Path: "/game-board"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given the registry with one extra entry

			// when it is indexed

			// then indexing refuses it rather than silently dropping the kind
			assert.Panics(t, func() {
				indexRegistry(append(slices.Clone(registry), tt.entry))
			})
		})
	}
}

func TestReferenceResolve(t *testing.T) {
	tests := []struct {
		name    string
		ref     Reference
		wantErr error
	}{
		{
			name:    "an unregistered kind is rejected",
			ref:     Reference{Kind: Kind("ship_wheel"), EntityID: testEntityID},
			wantErr: ErrUnknownKind,
		},
		{
			name:    "the zero kind is rejected",
			ref:     Reference{EntityID: testEntityID},
			wantErr: ErrUnknownKind,
		},
		{
			name:    "a uuid-keyed kind needs an entity id",
			ref:     Reference{Kind: KindArtComment, ChildID: testChildID},
			wantErr: ErrInvalidReference,
		},
		{
			name:    "a slug-keyed kind needs an entity key",
			ref:     Reference{Kind: KindSecretComment, ChildID: testChildID},
			wantErr: ErrInvalidReference,
		},
		{
			name:    "a child kind needs a child id",
			ref:     Reference{Kind: KindArtComment, EntityID: testEntityID},
			wantErr: ErrInvalidReference,
		},
		{
			name: "a complete child reference resolves",
			ref:  Reference{Kind: KindArtComment, EntityID: testEntityID, ChildID: testChildID},
		},
		{
			name: "a complete slug reference resolves",
			ref:  Reference{Kind: KindSecretComment, EntityKey: testEntityKey, ChildID: testChildID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given the reference under test

			// when
			target, err := tt.ref.Resolve()

			// then
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.ref.Kind, target.Kind)
		})
	}
}
