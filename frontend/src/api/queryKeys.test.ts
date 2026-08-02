import { describe, expect, it } from "vitest";
import { queryKeys } from "./queryKeys";

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
