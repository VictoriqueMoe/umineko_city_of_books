package mention

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
)

type (
	Kind    string
	Level   string
	KeyMode string

	Target struct {
		Kind  Kind
		Level Level
		Key   KeyMode

		Path   string
		Anchor string
	}

	entry struct {
		Target
		Comment bool
	}

	catalogue struct {
		kinds        []Kind
		commentKinds []Kind
		targets      []Target
		byKind       map[Kind]Target
		commentSet   map[Kind]struct{}
	}

	Reference struct {
		Kind        Kind
		EntityID    uuid.UUID
		EntityKey   string
		EntryNumber int
		ChildID     uuid.UUID
	}
)

const (
	LevelEntity Level = "entity"
	LevelChild  Level = "child"

	KeyUUID  KeyMode = ""
	KeySlug  KeyMode = "slug"
	KeyEntry KeyMode = "entry"
)

const (
	KindPost                Kind = "post"
	KindPostComment         Kind = "post_comment"
	KindArt                 Kind = "art"
	KindArtComment          Kind = "art_comment"
	KindTheory              Kind = "theory"
	KindTheoryResponse      Kind = "theory_response"
	KindShip                Kind = "ship"
	KindShipComment         Kind = "ship_comment"
	KindOC                  Kind = "oc"
	KindOCComment           Kind = "oc_comment"
	KindMystery             Kind = "mystery"
	KindMysteryAttempt      Kind = "mystery_attempt"
	KindMysteryComment      Kind = "mystery_comment"
	KindFanfic              Kind = "fanfic"
	KindFanficComment       Kind = "fanfic_comment"
	KindAnnouncement        Kind = "announcement"
	KindAnnouncementComment Kind = "announcement_comment"
	KindJournal             Kind = "journal"
	KindJournalComment      Kind = "journal_comment"
	KindJournalEntry        Kind = "journal_entry"
	KindJournalEntryComment Kind = "journal_entry_comment"
	KindSecretComment       Kind = "secret_comment"
)

var (
	ErrUnknownKind      = errors.New("mention: kind is not registered")
	ErrInvalidReference = errors.New("mention: reference is incomplete for its kind")
)

var registry = []entry{
	{Kind: KindPost, Level: LevelEntity, Path: "/game-board"},
	{Kind: KindPostComment, Level: LevelChild, Path: "/game-board", Anchor: "comment", Comment: true},
	{Kind: KindArt, Level: LevelEntity, Path: "/gallery/art"},
	{Kind: KindArtComment, Level: LevelChild, Path: "/gallery/art", Anchor: "comment", Comment: true},
	{Kind: KindTheory, Level: LevelEntity, Path: "/theory"},
	{Kind: KindTheoryResponse, Level: LevelChild, Path: "/theory", Anchor: "response"},
	{Kind: KindShip, Level: LevelEntity, Path: "/ships"},
	{Kind: KindShipComment, Level: LevelChild, Path: "/ships", Anchor: "comment", Comment: true},
	{Kind: KindOC, Level: LevelEntity, Path: "/oc"},
	{Kind: KindOCComment, Level: LevelChild, Path: "/oc", Anchor: "comment", Comment: true},
	{Kind: KindMystery, Level: LevelEntity, Path: "/mystery"},
	{Kind: KindMysteryAttempt, Level: LevelChild, Path: "/mystery", Anchor: "attempt"},
	{Kind: KindMysteryComment, Level: LevelChild, Path: "/mystery", Anchor: "comment", Comment: true},
	{Kind: KindFanfic, Level: LevelEntity, Path: "/fanfiction"},
	{Kind: KindFanficComment, Level: LevelChild, Path: "/fanfiction", Anchor: "comment", Comment: true},
	{Kind: KindAnnouncement, Level: LevelEntity, Path: "/announcements"},
	{Kind: KindAnnouncementComment, Level: LevelChild, Path: "/announcements", Anchor: "comment", Comment: true},
	{Kind: KindJournal, Level: LevelEntity, Path: "/journals"},
	{Kind: KindJournalComment, Level: LevelChild, Path: "/journals", Anchor: "comment", Comment: true},
	{Kind: KindJournalEntry, Level: LevelEntity, Key: KeyEntry, Path: "/journals"},
	{Kind: KindJournalEntryComment, Level: LevelChild, Key: KeyEntry, Path: "/journals", Anchor: "comment", Comment: true},
	{Kind: KindSecretComment, Level: LevelChild, Key: KeySlug, Path: "/secrets", Anchor: "comment", Comment: true},
}

var registered = indexRegistry(registry)

func indexRegistry(entries []entry) catalogue {
	out := catalogue{
		kinds:      make([]Kind, 0, len(entries)),
		targets:    make([]Target, 0, len(entries)),
		byKind:     make(map[Kind]Target, len(entries)),
		commentSet: make(map[Kind]struct{}),
	}

	for _, e := range entries {
		if _, dup := out.byKind[e.Kind]; dup {
			panic("mention: duplicate registry entry for kind " + string(e.Kind))
		}

		if e.Level != LevelEntity && e.Level != LevelChild {
			panic("mention: kind " + string(e.Kind) + " declares no level")
		}

		if (e.Level == LevelChild) != (e.Anchor != "") {
			panic("mention: kind " + string(e.Kind) + " has an anchor that contradicts its level")
		}

		if e.Path == "" {
			panic("mention: kind " + string(e.Kind) + " declares no path")
		}

		if e.Comment && e.Level != LevelChild {
			panic("mention: comment kind " + string(e.Kind) + " is not a child of anything")
		}

		out.kinds = append(out.kinds, e.Kind)
		out.targets = append(out.targets, e.Target)
		out.byKind[e.Kind] = e.Target

		if e.Comment {
			out.commentKinds = append(out.commentKinds, e.Kind)
			out.commentSet[e.Kind] = struct{}{}
		}
	}

	return out
}

func Kinds() []Kind {
	return registered.kinds
}

func CommentKinds() []Kind {
	return registered.commentKinds
}

func IsCommentKind(k Kind) bool {
	_, ok := registered.commentSet[k]
	return ok
}

func Targets() []Target {
	return registered.targets
}

func TargetFor(k Kind) (Target, bool) {
	t, ok := registered.byKind[k]
	return t, ok
}

func (t Target) ReferenceType(ref Reference) string {
	base := string(t.Kind)

	switch t.Key {
	case KeySlug:
		base += ":" + ref.EntityKey
	case KeyEntry:
		base += ":" + strconv.Itoa(ref.EntryNumber)
	}

	if t.Level == LevelChild {
		base += ":" + ref.ChildID.String()
	}

	return base
}

func (t Target) LinkURL(ref Reference) string {
	link := t.Path + "/" + ref.EntityID.String()
	if t.Key == KeySlug {
		link = t.Path + "/" + ref.EntityKey
	}

	if t.Key == KeyEntry {
		link += "/entry/" + strconv.Itoa(ref.EntryNumber)
	}

	if t.Level == LevelChild {
		link += "#" + t.Anchor + "-" + ref.ChildID.String()
	}

	return link
}

func (t Target) ReferenceID(ref Reference) uuid.UUID {
	if t.Key == KeySlug {
		return uuid.Nil
	}

	return ref.EntityID
}

func (ref Reference) Resolve() (Target, error) {
	t, ok := registered.byKind[ref.Kind]
	if !ok {
		return Target{}, fmt.Errorf("%w: %q", ErrUnknownKind, ref.Kind)
	}

	if t.Key == KeySlug && ref.EntityKey == "" {
		return Target{}, fmt.Errorf("%w: %q needs an entity key", ErrInvalidReference, ref.Kind)
	}

	if t.Key != KeySlug && ref.EntityID == uuid.Nil {
		return Target{}, fmt.Errorf("%w: %q needs an entity id", ErrInvalidReference, ref.Kind)
	}

	if t.Level == LevelChild && ref.ChildID == uuid.Nil {
		return Target{}, fmt.Errorf("%w: %q needs a child id", ErrInvalidReference, ref.Kind)
	}

	return t, nil
}
