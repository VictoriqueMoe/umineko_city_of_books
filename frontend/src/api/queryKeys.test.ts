import { describe, expect, it } from "vitest";
import { queryKeys, type RoomsListParams } from "./queryKeys";

function startsWith(key: readonly unknown[], prefix: readonly unknown[]): boolean {
    return prefix.length <= key.length && JSON.stringify(key.slice(0, prefix.length)) === JSON.stringify(prefix);
}

describe("queryKeys entity namespaces", () => {
    it("namespaces every content entity under its own root key", () => {
        // then
        expect(queryKeys.theory.all).toEqual(["theory"]);
        expect(queryKeys.post.all).toEqual(["post"]);
        expect(queryKeys.art.all).toEqual(["art"]);
        expect(queryKeys.ship.all).toEqual(["ship"]);
        expect(queryKeys.oc.all).toEqual(["oc"]);
        expect(queryKeys.journal.all).toEqual(["journal"]);
        expect(queryKeys.fanfic.all).toEqual(["fanfic"]);
        expect(queryKeys.mystery.all).toEqual(["mystery"]);
        expect(queryKeys.gameRoom.all).toEqual(["gameRoom"]);
        expect(queryKeys.notifications.all).toEqual(["notifications"]);
    });

    it("gives every root key a distinct value so invalidation cannot bleed across entities", () => {
        // given
        const roots = [
            queryKeys.theory.all,
            queryKeys.post.all,
            queryKeys.art.all,
            queryKeys.ship.all,
            queryKeys.oc.all,
            queryKeys.journal.all,
            queryKeys.fanfic.all,
            queryKeys.mystery.all,
            queryKeys.gameRoom.all,
            queryKeys.notifications.all,
        ];

        // when
        const unique = new Set(roots.map(root => root.join("|")));

        // then
        expect(unique.size).toBe(roots.length);
    });

    it("starts every detail key with the entity root so a root invalidation reaches it", () => {
        // then
        expect(queryKeys.theory.detail("t1")[0]).toBe(queryKeys.theory.all[0]);
        expect(queryKeys.post.detail("p1")[0]).toBe(queryKeys.post.all[0]);
        expect(queryKeys.art.detail("a1")[0]).toBe(queryKeys.art.all[0]);
        expect(queryKeys.oc.detail("o1")[0]).toBe(queryKeys.oc.all[0]);
    });
});

describe("queryKeys detail keys", () => {
    it("builds a stable tuple of entity, detail and id", () => {
        // then
        expect(queryKeys.theory.detail("t1")).toEqual(["theory", "detail", "t1"]);
        expect(queryKeys.post.detail("p1")).toEqual(["post", "detail", "p1"]);
        expect(queryKeys.mystery.detail("m1")).toEqual(["mystery", "detail", "m1"]);
        expect(queryKeys.gameRoom.detail("g1")).toEqual(["gameRoom", "detail", "g1"]);
    });

    it("returns an equal key for the same id on every call", () => {
        // then
        expect(queryKeys.theory.detail("t1")).toEqual(queryKeys.theory.detail("t1"));
    });

    it("returns different keys for different ids", () => {
        // then
        expect(queryKeys.theory.detail("t1")).not.toEqual(queryKeys.theory.detail("t2"));
    });

    it("never collides across entities that share an id", () => {
        // then
        expect(queryKeys.theory.detail("same")).not.toEqual(queryKeys.post.detail("same"));
        expect(queryKeys.art.detail("same")).not.toEqual(queryKeys.ship.detail("same"));
    });

    it("keeps an empty id in the tuple rather than collapsing to the list key", () => {
        // then
        expect(queryKeys.theory.detail("")).toEqual(["theory", "detail", ""]);
        expect(queryKeys.theory.detail("")).not.toEqual(queryKeys.theory.all);
    });
});

describe("queryKeys feed keys", () => {
    it("defaults to an empty parameter object when no filters are given", () => {
        // then
        expect(queryKeys.theory.feed()).toEqual(["theory", "feed", {}]);
        expect(queryKeys.post.feed()).toEqual(["post", "feed", {}]);
        expect(queryKeys.gameRoom.list()).toEqual(["gameRoom", "list", {}]);
    });

    it("carries the filters as the last element of the key", () => {
        // given
        const params = { limit: 20, sort: "newest" };

        // when
        const key = queryKeys.art.feed(params);

        // then
        expect(key).toEqual(["art", "feed", { limit: 20, sort: "newest" }]);
    });

    it("returns different keys for different filters", () => {
        // then
        expect(queryKeys.art.feed({ page: 1 })).not.toEqual(queryKeys.art.feed({ page: 2 }));
        expect(queryKeys.art.feed({ page: 1 })).not.toEqual(queryKeys.art.feed());
    });

    it("returns an equal key for equal filters written as separate objects", () => {
        // then
        expect(queryKeys.art.feed({ page: 1 })).toEqual(queryKeys.art.feed({ page: 1 }));
    });

    it("separates a feed from a detail of the same entity", () => {
        // then
        expect(queryKeys.journal.feed()).not.toEqual(queryKeys.journal.detail("feed"));
    });
});

describe("queryKeys.oc user scoped keys", () => {
    it("keys the list and the summaries of a user separately", () => {
        // then
        expect(queryKeys.oc.userList("u1")).toEqual(["oc", "userList", "u1"]);
        expect(queryKeys.oc.userSummaries("u1")).toEqual(["oc", "userSummaries", "u1"]);
        expect(queryKeys.oc.userList("u1")).not.toEqual(queryKeys.oc.userSummaries("u1"));
    });

    it("returns different keys for different users", () => {
        // then
        expect(queryKeys.oc.userList("u1")).not.toEqual(queryKeys.oc.userList("u2"));
    });
});

describe("queryKeys.journal.entry", () => {
    it("builds the entry key from the journal, the entry marker and the entry number", () => {
        // then
        expect(queryKeys.journal.entry("j1", 4)).toEqual(["journal", "detail", "j1", "entry", 4]);
    });

    it("nests an entry under its journal so a detail invalidation reaches it", () => {
        // then
        expect(startsWith(queryKeys.journal.entry("j1", 4), queryKeys.journal.detail("j1"))).toBe(true);
    });

    it("keeps the entries of one journal apart from each other and from other journals", () => {
        // then
        expect(queryKeys.journal.entry("j1", 4)).not.toEqual(queryKeys.journal.entry("j1", 5));
        expect(queryKeys.journal.entry("j1", 4)).not.toEqual(queryKeys.journal.entry("j2", 4));
    });
});

describe("queryKeys.user", () => {
    it("builds every owner scoped list as owner, kind and paging", () => {
        // given
        const paging = { limit: 20, offset: 40 };

        // then
        expect(queryKeys.user.posts("u1", paging)).toEqual(["user", "u1", "posts", paging]);
        expect(queryKeys.user.art("u1", paging)).toEqual(["user", "u1", "art", paging]);
        expect(queryKeys.user.galleries("u1", paging)).toEqual(["user", "u1", "galleries", paging]);
        expect(queryKeys.user.ships("u1", paging)).toEqual(["user", "u1", "ships", paging]);
        expect(queryKeys.user.mysteries("u1", paging)).toEqual(["user", "u1", "mysteries", paging]);
        expect(queryKeys.user.fanfics("u1", paging)).toEqual(["user", "u1", "fanfics", paging]);
        expect(queryKeys.user.fanficFavourites("u1", paging)).toEqual(["user", "u1", "fanfic-favourites", paging]);
        expect(queryKeys.user.journals("u1", paging)).toEqual(["user", "u1", "journals", paging]);
        expect(queryKeys.user.followedJournals("u1", paging)).toEqual(["user", "u1", "followed-journals", paging]);
        expect(queryKeys.user.activity("beatrice", paging)).toEqual(["user", "beatrice", "activity", paging]);
    });

    it("keeps the paging slot as an empty object when a list has no paging of its own", () => {
        // then
        expect(queryKeys.user.galleries("u1")).toEqual(["user", "u1", "galleries", {}]);
        expect(queryKeys.user.activity("beatrice")).toEqual(["user", "beatrice", "activity", {}]);
    });

    it("keeps a favourited list apart from an authored list of the same kind", () => {
        // then
        expect(queryKeys.user.fanficFavourites("u1")).not.toEqual(queryKeys.user.fanfics("u1"));
        expect(queryKeys.user.followedJournals("u1")).not.toEqual(queryKeys.user.journals("u1"));
    });

    it("keeps every kind of the same owner apart", () => {
        // given
        const kinds = [
            queryKeys.user.posts("u1"),
            queryKeys.user.art("u1"),
            queryKeys.user.galleries("u1"),
            queryKeys.user.ships("u1"),
            queryKeys.user.mysteries("u1"),
            queryKeys.user.fanfics("u1"),
            queryKeys.user.fanficFavourites("u1"),
            queryKeys.user.journals("u1"),
            queryKeys.user.followedJournals("u1"),
            queryKeys.user.activity("u1"),
        ];

        // when
        const unique = new Set(kinds.map(kind => JSON.stringify(kind)));

        // then
        expect(unique.size).toBe(kinds.length);
    });

    it("keeps the same kind of different owners apart", () => {
        // then
        expect(queryKeys.user.posts("u1")).not.toEqual(queryKeys.user.posts("u2"));
    });

    it("never lets the singular user root and the plural users root reach each other", () => {
        // given
        const owned = [queryKeys.user.posts("u1"), queryKeys.user.galleries("u1"), queryKeys.user.activity("u1")];
        const directory = [queryKeys.users.byId("u1"), queryKeys.users.followers("u1", {}), queryKeys.users.mutuals()];

        // when
        const crossings = owned.flatMap(one =>
            directory.map(other => startsWith(one, other) || startsWith(other, one)),
        );

        // then
        expect(queryKeys.user.posts("u1")[0]).not.toBe(queryKeys.users.byId("u1")[0]);
        expect(crossings).not.toContain(true);
    });
});

describe("queryKeys.chat", () => {
    it("builds the room key from the room id", () => {
        // then
        expect(queryKeys.chat.room("r1")).toEqual(["chat", "room", "r1"]);
    });

    it("nests members and pinned messages under the room key so the room invalidates both", () => {
        // given
        const room = queryKeys.chat.room("r1");

        // when
        const members = queryKeys.chat.roomMembers("r1");
        const pinned = queryKeys.chat.pinned("r1");

        // then
        expect(members).toEqual(["chat", "room", "r1", "members"]);
        expect(pinned).toEqual(["chat", "room", "r1", "pinned"]);
        expect(members.slice(0, room.length)).toEqual([...room]);
        expect(pinned.slice(0, room.length)).toEqual([...room]);
    });

    it("keeps different rooms apart", () => {
        // then
        expect(queryKeys.chat.roomMembers("r1")).not.toEqual(queryKeys.chat.roomMembers("r2"));
        expect(queryKeys.chat.pinned("r1")).not.toEqual(queryKeys.chat.roomMembers("r1"));
    });
});

describe("queryKeys.chat rooms list", () => {
    const filters: RoomsListParams = {
        search: "tea",
        rpOnly: true,
        tagFilter: "horror",
        includeArchived: false,
        pages: 2,
    };

    it("names each scope under the shared rooms list root when asked for no filters", () => {
        // then
        expect(queryKeys.chat.roomsList()).toEqual(["chat", "rooms-list"]);
        expect(queryKeys.chat.roomsListHosted()).toEqual(["chat", "rooms-list", "hosted"]);
        expect(queryKeys.chat.roomsListJoined()).toEqual(["chat", "rooms-list", "joined"]);
        expect(queryKeys.chat.roomsListDiscover()).toEqual(["chat", "rooms-list", "discover"]);
    });

    it("appends the filter tuple after the scope", () => {
        // then
        expect(queryKeys.chat.roomsListHosted(filters)).toEqual([
            "chat",
            "rooms-list",
            "hosted",
            "tea",
            true,
            "horror",
            false,
            2,
        ]);
    });

    it("keeps every filtered key under the scope its own builder invalidates", () => {
        // then
        expect(startsWith(queryKeys.chat.roomsListHosted(filters), queryKeys.chat.roomsListHosted())).toBe(true);
        expect(startsWith(queryKeys.chat.roomsListJoined(filters), queryKeys.chat.roomsListJoined())).toBe(true);
        expect(startsWith(queryKeys.chat.roomsListDiscover(filters), queryKeys.chat.roomsListDiscover())).toBe(true);
    });

    it("keeps every filtered key under the shared rooms list root", () => {
        // then
        expect(startsWith(queryKeys.chat.roomsListHosted(filters), queryKeys.chat.roomsList())).toBe(true);
        expect(startsWith(queryKeys.chat.roomsListJoined(filters), queryKeys.chat.roomsList())).toBe(true);
        expect(startsWith(queryKeys.chat.roomsListDiscover(filters), queryKeys.chat.roomsList())).toBe(true);
    });

    it("never lets one scope reach another", () => {
        // then
        expect(startsWith(queryKeys.chat.roomsListHosted(filters), queryKeys.chat.roomsListJoined())).toBe(false);
        expect(startsWith(queryKeys.chat.roomsListJoined(filters), queryKeys.chat.roomsListDiscover())).toBe(false);
    });

    it("gives a different page window and a different filter their own keys", () => {
        // then
        expect(queryKeys.chat.roomsListHosted(filters)).not.toEqual(
            queryKeys.chat.roomsListHosted({ ...filters, pages: 3 }),
        );
        expect(queryKeys.chat.roomsListHosted(filters)).not.toEqual(
            queryKeys.chat.roomsListHosted({ ...filters, search: "cake" }),
        );
    });

    it("never lets the rooms list keys collide with the user rooms entry", () => {
        // then
        expect(startsWith(queryKeys.chat.roomsList(), queryKeys.chat.userRooms())).toBe(false);
        expect(startsWith(queryKeys.chat.userRooms(), queryKeys.chat.roomsList())).toBe(false);
    });
});

describe("queryKeys.streams", () => {
    it("keys the directory, one stream, the owner view and the ingress credentials apart", () => {
        // then
        expect(queryKeys.streams.live()).toEqual(["streams", "live"]);
        expect(queryKeys.streams.byUsername("beato")).toEqual(["streams", "by-username", "beato"]);
        expect(queryKeys.streams.mine()).toEqual(["streams", "mine"]);
        expect(queryKeys.streams.credentials()).toEqual(["streams", "credentials"]);
    });

    it("never lets the owner view or the credentials collide with a stream whose id spells them", () => {
        // then
        expect(queryKeys.streams.mine()).not.toEqual(queryKeys.streams.byUsername("mine"));
        expect(queryKeys.streams.credentials()).not.toEqual(queryKeys.streams.byUsername("credentials"));
    });

    it("keeps every stream key apart", () => {
        // given
        const keys = [
            queryKeys.streams.live(),
            queryKeys.streams.byUsername("beato"),
            queryKeys.streams.mine(),
            queryKeys.streams.credentials(),
        ];

        // when
        const unique = new Set(keys.map(key => JSON.stringify(key)));

        // then
        expect(unique.size).toBe(keys.length);
    });
});

describe("queryKeys.watchParty", () => {
    it("scopes the session list by the room it belongs to", () => {
        // then
        expect(queryKeys.watchParty.list("r1")).toEqual(["watch-party", "list", "r1"]);
        expect(queryKeys.watchParty.list("r1")).not.toEqual(queryKeys.watchParty.list("r2"));
    });

    it("nests the list under the watch party root so a root invalidation reaches it", () => {
        // then
        expect(startsWith(queryKeys.watchParty.list("r1"), queryKeys.watchParty.all)).toBe(true);
    });

    it("collapses a missing room to one key rather than two", () => {
        // then
        expect(queryKeys.watchParty.list(null)).toEqual(["watch-party", "list", null]);
        expect(queryKeys.watchParty.list(undefined)).toEqual(queryKeys.watchParty.list(null));
    });

    it("never reaches the chat room keys of the same room", () => {
        // then
        expect(startsWith(queryKeys.watchParty.list("r1"), queryKeys.chat.room("r1"))).toBe(false);
        expect(startsWith(queryKeys.chat.room("r1"), queryKeys.watchParty.list("r1"))).toBe(false);
    });
});

describe("queryKeys.overlay", () => {
    it("keys the overlay connection under the overlay root", () => {
        // then
        expect(queryKeys.overlay.connection()).toEqual(["overlay", "connection"]);
        expect(startsWith(queryKeys.overlay.connection(), queryKeys.overlay.all)).toBe(true);
    });
});

describe("queryKeys.quotes", () => {
    it("keys a single quote by the series, the identifier and the language", () => {
        // then
        expect(queryKeys.quotes.byAudioId("umineko", "ep2_0042", "jp")).toEqual([
            "quotes",
            "audio",
            "umineko",
            "ep2_0042",
            "jp",
        ]);
        expect(queryKeys.quotes.byIndex("higurashi", 42, "en")).toEqual(["quotes", "index", "higurashi", 42, "en"]);
    });

    it("records the absence of a language rather than leaving a hole in the key", () => {
        // then
        expect(queryKeys.quotes.byAudioId("umineko", "ep2_0042")).toEqual([
            "quotes",
            "audio",
            "umineko",
            "ep2_0042",
            null,
        ]);
        expect(queryKeys.quotes.byIndex("umineko", 42)).toEqual(["quotes", "index", "umineko", 42, null]);
    });

    it("keeps the same identifier in different series and languages apart", () => {
        // given
        const keys = [
            queryKeys.quotes.byAudioId("umineko", "ep2_0042", "en"),
            queryKeys.quotes.byAudioId("umineko", "ep2_0042", "jp"),
            queryKeys.quotes.byAudioId("higurashi", "ep2_0042", "en"),
            queryKeys.quotes.byIndex("umineko", 42, "en"),
            queryKeys.quotes.byIndex("umineko", 43, "en"),
        ];

        // when
        const unique = new Set(keys.map(key => JSON.stringify(key)));

        // then
        expect(unique.size).toBe(keys.length);
    });

    it("keeps single quote lookups out of the search and browse caches", () => {
        // then
        expect(startsWith(queryKeys.quotes.byAudioId("umineko", "ep2_0042"), ["quotes", "search"])).toBe(false);
        expect(startsWith(queryKeys.quotes.byIndex("umineko", 42), ["quotes", "browse"])).toBe(false);
    });
});

describe("queryKeys.profile", () => {
    it("keys a profile by username", () => {
        // then
        expect(queryKeys.profile.byUsername("beatrice")).toEqual(["profile", "username", "beatrice"]);
    });

    it("keys the blocked list by user id", () => {
        // then
        expect(queryKeys.profile.blockedUsers("u1")).toEqual(["profile", "u1", "blocked"]);
    });

    it("treats usernames that differ only by case as different keys", () => {
        // then
        expect(queryKeys.profile.byUsername("beatrice")).not.toEqual(queryKeys.profile.byUsername("Beatrice"));
    });
});

describe("queryKeys.notifications", () => {
    it("keys the list and the unread count under the notifications root", () => {
        // then
        expect(queryKeys.notifications.list()).toEqual(["notifications", "list", {}]);
        expect(queryKeys.notifications.unreadCount()).toEqual(["notifications", "unread-count"]);
    });

    it("returns different list keys for different parameters", () => {
        // then
        expect(queryKeys.notifications.list({ unread: true })).not.toEqual(
            queryKeys.notifications.list({ unread: false }),
        );
    });
});

describe("queryKeys.admin", () => {
    it("builds stable static admin keys", () => {
        // then
        expect(queryKeys.admin.announcements()).toEqual(["admin", "announcements"]);
        expect(queryKeys.admin.invites()).toEqual(["admin", "invites"]);
        expect(queryKeys.admin.bannedGifs()).toEqual(["admin", "banned-gifs"]);
        expect(queryKeys.admin.vanityRoles()).toEqual(["admin", "vanity-roles"]);
    });

    it("builds parameterised admin keys with the filters last", () => {
        // then
        expect(queryKeys.admin.users({ page: 2 })).toEqual(["admin", "users", { page: 2 }]);
        expect(queryKeys.admin.reports()).toEqual(["admin", "reports", {}]);
        expect(queryKeys.admin.auditLog({ action: "ban" })).toEqual(["admin", "audit-log", { action: "ban" }]);
    });

    it("nests the paged invite list under the invites key so an invites invalidation reaches it", () => {
        // then
        expect(queryKeys.admin.invitesList(20, 40)).toEqual(["admin", "invites", 20, 40]);
        expect(startsWith(queryKeys.admin.invitesList(20, 40), queryKeys.admin.invites())).toBe(true);
        expect(queryKeys.admin.invitesList(20, 40)).not.toEqual(queryKeys.admin.invitesList(20, 0));
    });

    it("scopes banned words by their scope", () => {
        // then
        expect(queryKeys.admin.bannedWords("chat")).toEqual(["admin", "banned-words", "chat"]);
        expect(queryKeys.admin.bannedWords("chat")).not.toEqual(queryKeys.admin.bannedWords("username"));
    });

    it("keeps every admin section apart", () => {
        // given
        const sections = [
            queryKeys.admin.announcements(),
            queryKeys.admin.invites(),
            queryKeys.admin.bannedGifs(),
            queryKeys.admin.vanityRoles(),
            queryKeys.admin.users(),
            queryKeys.admin.reports(),
            queryKeys.admin.auditLog(),
        ];

        // when
        const unique = new Set(sections.map(section => JSON.stringify(section)));

        // then
        expect(unique.size).toBe(sections.length);
    });
});
