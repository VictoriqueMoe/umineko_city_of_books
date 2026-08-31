import { describe, expect, it } from "vitest";
import { makePublicUser, makeRoomMember } from "../../test-utils/fixtures";
import type { ChatRoomMember, SiteRole } from "../../types/api";
import { ROLE_GROUPS } from "../permissions";
import {
    groupMembers,
    memberOnlineWeight,
    memberRankLabel,
    memberRankWeight,
    memberSortName,
    mergePresence,
    sortMembers,
    typingNames,
    type PresenceMap,
} from "./memberRoster";

function member(id: string, role: SiteRole | undefined, roomRole: string, name: string): ChatRoomMember {
    return makeRoomMember({
        user: makePublicUser({ id, username: name.toLowerCase(), display_name: name, role }),
        role: roomRole,
    });
}

describe("mergePresence", () => {
    it("seeds the map from the presence the server sent with the membership", () => {
        // given
        const members = [
            makeRoomMember({ user: makePublicUser({ id: "u1" }), presence: "idle" }),
            makeRoomMember({ user: makePublicUser({ id: "u2" }), presence: "active" }),
        ];

        // when
        const merged = mergePresence(members, {});

        // then
        expect(merged).toEqual({ u1: "idle", u2: "active" });
    });

    it("ignores a member the server reported with no presence at all", () => {
        // given
        const members = [
            makeRoomMember({ user: makePublicUser({ id: "u1" }), presence: "" }),
            makeRoomMember({ user: makePublicUser({ id: "u2" }) }),
        ];

        // when
        const merged = mergePresence(members, {});

        // then
        expect(merged).toEqual({});
    });

    it("lets a live update win over the seeded presence", () => {
        // given
        const members = [makeRoomMember({ user: makePublicUser({ id: "u1" }), presence: "idle" })];

        // when
        const merged = mergePresence(members, { u1: "active" });

        // then
        expect(merged).toEqual({ u1: "active" });
    });

    it("keeps a live entry for somebody the membership does not list", () => {
        // given
        const members = [makeRoomMember({ user: makePublicUser({ id: "u1" }), presence: "idle" })];

        // when
        const merged = mergePresence(members, { ghost: "active" });

        // then
        expect(merged).toEqual({ u1: "idle", ghost: "active" });
    });

    it("seeds the presence map from the membership the server sent", () => {
        // given
        const members = [
            makeRoomMember({
                user: makePublicUser({ id: "u2", username: "battler", display_name: "Battler" }),
                presence: "idle",
            }),
        ];

        // when
        const merged = mergePresence(members, {});

        // then
        expect(merged).toEqual({ u2: "idle" });
    });
});

describe("memberOnlineWeight", () => {
    it("weighs somebody who is around ahead of somebody who is not", () => {
        // given
        const presence: PresenceMap = { u1: "active", u2: "idle" };

        // when
        const weights = [
            memberOnlineWeight(presence, "u1"),
            memberOnlineWeight(presence, "u2"),
            memberOnlineWeight(presence, "ghost"),
        ];

        // then
        expect(weights).toEqual([0, 0, 1]);
    });

    it("weighs a member who is nowhere to be seen below one who is present", () => {
        // given
        const members = [
            makeRoomMember({
                user: makePublicUser({ id: "u2", username: "battler", display_name: "Battler" }),
                presence: "active",
            }),
        ];

        // when
        const presence = mergePresence(members, {});

        // then
        expect(memberOnlineWeight(presence, "u2")).toBe(0);
        expect(memberOnlineWeight(presence, "ghost")).toBe(1);
    });
});

describe("memberRankWeight", () => {
    it("puts the site owner above everybody, including a room host", () => {
        // given
        const owner = member("u1", "super_admin", "member", "Beatrice");
        const host = member("u2", undefined, "host", "Battler");

        // when
        const weights = [memberRankWeight(owner), memberRankWeight(host)];

        // then
        expect(weights).toEqual([0, 1]);
    });

    it("ranks a room host above the site roles below owner", () => {
        // given
        const host = member("u1", undefined, "host", "Battler");
        const admin = member("u2", "admin", "member", "Lambda");
        const moderator = member("u3", "moderator", "member", "Bern");
        const plain = member("u4", undefined, "member", "Ange");

        // when
        const weights = [
            memberRankWeight(host),
            memberRankWeight(admin),
            memberRankWeight(moderator),
            memberRankWeight(plain),
        ];

        // then
        expect(weights).toEqual([1, 2, 3, 4]);
    });

    it("keeps every site role in the order the role hierarchy declares, offset by the host slot", () => {
        // given
        const byRole = ROLE_GROUPS.map(group => member("u1", group.role, "member", "X"));

        // when
        const weights = byRole.map(m => memberRankWeight(m));

        // then
        expect(weights).toEqual([0, ...ROLE_GROUPS.slice(1).map((_, i) => i + 2)]);
        expect(weights).toEqual([...weights].sort((a, b) => a - b));
    });

    it("lets the owner keep the top slot even when they also host the room", () => {
        // given
        const ownerHost = member("u1", "super_admin", "host", "Beatrice");

        // when
        const weight = memberRankWeight(ownerHost);

        // then
        expect(weight).toBe(0);
    });

    it("promotes an admin who hosts the room into the host slot", () => {
        // given
        const adminHost = member("u1", "admin", "host", "Lambda");

        // when
        const weight = memberRankWeight(adminHost);

        // then
        expect(weight).toBe(1);
    });
});

describe("memberSortName", () => {
    it("prefers the room nickname, folded to lower case", () => {
        // given
        const m = makeRoomMember({ nickname: "Golden Witch" });

        // when
        const name = memberSortName(m);

        // then
        expect(name).toBe("golden witch");
    });

    it("falls back to the account display name when the nickname is only whitespace", () => {
        // given
        const m = makeRoomMember({ nickname: "   " });

        // when
        const name = memberSortName(m);

        // then
        expect(name).toBe("beatrice");
    });

    it("falls back to the username when there is neither nickname nor display name", () => {
        // given
        const m = makeRoomMember({ nickname: "", user: makePublicUser({ display_name: "  ", username: "Beato" }) });

        // when
        const name = memberSortName(m);

        // then
        expect(name).toBe("beato");
    });
});

describe("memberRankLabel", () => {
    it("labels each site role with its own group heading", () => {
        // given
        const roles: SiteRole[] = ["super_admin", "admin", "moderator"];

        // when
        const labels = roles.map(role => memberRankLabel(member("u1", role, "member", "X")));

        // then
        expect(labels).toEqual(["Reality Author", "Voyager Witches", "Witches"]);
    });

    it("labels a room host as a host", () => {
        // given
        const host = member("u1", undefined, "host", "Battler");

        // when
        const label = memberRankLabel(host);

        // then
        expect(label).toBe("Host");
    });

    it("keeps the owner under their own heading even when they host the room", () => {
        // given
        const ownerHost = member("u1", "super_admin", "host", "Beatrice");

        // when
        const label = memberRankLabel(ownerHost);

        // then
        expect(label).toBe("Reality Author");
    });

    it("labels everybody else as a plain member", () => {
        // given
        const plain = member("u1", undefined, "member", "Ange");

        // when
        const label = memberRankLabel(plain);

        // then
        expect(label).toBe("Members");
    });
});

describe("sortMembers", () => {
    it("orders by rank before anything else", () => {
        // given
        const members = [
            member("u4", undefined, "member", "Ange"),
            member("u3", "admin", "member", "Lambda"),
            member("u2", undefined, "host", "Battler"),
            member("u1", "super_admin", "member", "Beatrice"),
        ];

        // when
        const sorted = sortMembers(members, {});

        // then
        expect(sorted.map(m => m.user.id)).toEqual(["u1", "u2", "u3", "u4"]);
    });

    it("sorts people who are around ahead of people who are not, within a rank", () => {
        // given
        const members = [member("u2", undefined, "member", "Battler"), member("u3", undefined, "member", "Ange")];

        // when
        const sorted = sortMembers(members, { u3: "active" });

        // then
        expect(sorted.map(m => m.user.id)).toEqual(["u3", "u2"]);
    });

    it("falls back to the sort name when rank and presence tie", () => {
        // given
        const members = [member("u1", undefined, "member", "Ronove"), member("u2", undefined, "member", "Ange")];

        // when
        const sorted = sortMembers(members, {});

        // then
        expect(sorted.map(m => m.user.id)).toEqual(["u2", "u1"]);
    });

    it("sorts by the room nickname rather than the account name", () => {
        // given
        const members = [
            makeRoomMember({
                user: makePublicUser({ id: "u1", username: "ange", display_name: "Ange" }),
                nickname: "Zepar",
            }),
            makeRoomMember({ user: makePublicUser({ id: "u2", username: "ronove", display_name: "Ronove" }) }),
        ];

        // when
        const sorted = sortMembers(members, {});

        // then
        expect(sorted.map(m => m.user.id)).toEqual(["u2", "u1"]);
    });

    it("sorts people who are around ahead of people who are not", () => {
        // given
        const members = [
            member("u2", undefined, "member", "Battler"),
            makeRoomMember({
                user: makePublicUser({ id: "u3", username: "ange", display_name: "Ange" }),
                presence: "active",
            }),
        ];

        // when
        const sorted = sortMembers(members, mergePresence(members, {}));

        // then
        expect(sorted.map(m => m.user.id)).toEqual(["u3", "u2"]);
    });

    it("leaves the array it was given untouched", () => {
        // given
        const members = [member("u2", undefined, "member", "Ronove"), member("u1", "super_admin", "member", "Beato")];

        // when
        sortMembers(members, {});

        // then
        expect(members.map(m => m.user.id)).toEqual(["u2", "u1"]);
    });
});

describe("groupMembers", () => {
    it("collapses a run of members sharing a heading into one group", () => {
        // given
        const sorted = sortMembers(
            [
                member("u4", undefined, "member", "Ange"),
                member("u3", "admin", "member", "Lambda"),
                member("u2", undefined, "host", "Battler"),
                member("u1", "super_admin", "member", "Beatrice"),
                member("u5", undefined, "member", "Ronove"),
            ],
            {},
        );

        // when
        const groups = groupMembers(sorted, new Set());

        // then
        expect(groups.map(g => g.label)).toEqual(["Reality Author", "Host", "Voyager Witches", "Members"]);
        expect(groups[3].members.map(m => m.user.id)).toEqual(["u4", "u5"]);
    });

    it("groups members by rank with staff at the top", () => {
        // given
        const members = [
            member("u4", undefined, "member", "Ange"),
            member("u3", "admin", "member", "Lambda"),
            member("u2", undefined, "host", "Battler"),
            member("u1", "super_admin", "member", "Beatrice"),
        ];

        // when
        const groups = groupMembers(sortMembers(members, mergePresence(members, {})), new Set());

        // then
        expect(groups.map(g => g.label)).toEqual(["Reality Author", "Host", "Voyager Witches", "Members"]);
    });

    it("lists the people on the voice call in their own group", () => {
        // given
        const sorted = sortMembers(
            [member("u2", undefined, "member", "Battler"), member("u3", undefined, "member", "Ange")],
            {},
        );

        // when
        const groups = groupMembers(sorted, new Set(["u3"]));

        // then
        expect(groups[0].label).toBe("In Voice");
        expect(groups[0].members.map(m => m.user.id)).toEqual(["u3"]);
    });

    it("still lists somebody on the voice call under their own rank as well", () => {
        // given
        const sorted = sortMembers([member("u3", undefined, "member", "Ange")], {});

        // when
        const groups = groupMembers(sorted, new Set(["u3"]));

        // then
        expect(groups.map(g => g.label)).toEqual(["In Voice", "Members"]);
        expect(groups[1].members.map(m => m.user.id)).toEqual(["u3"]);
    });

    it("adds no voice group when nobody on the call is in the room", () => {
        // given
        const sorted = sortMembers([member("u3", undefined, "member", "Ange")], {});

        // when
        const groups = groupMembers(sorted, new Set(["someone-else"]));

        // then
        expect(groups.map(g => g.label)).toEqual(["Members"]);
    });

    it("returns nothing for an empty roster", () => {
        // given
        const sorted: ChatRoomMember[] = [];

        // when
        const groups = groupMembers(sorted, new Set(["u3"]));

        // then
        expect(groups).toEqual([]);
    });
});

describe("typingNames over a room roster", () => {
    const members = [
        makeRoomMember({ user: makePublicUser({ id: "u1", username: "beatrice", display_name: "Beatrice" }) }),
        makeRoomMember({ user: makePublicUser({ id: "u2", username: "battler", display_name: "Battler" }) }),
    ];

    it("names the person typing in this room", () => {
        // when
        const names = typingNames(["u2"], members, "u1");

        // then
        expect(names).toEqual(["Battler"]);
    });

    it("prefers the nickname the room gave somebody", () => {
        // given
        const nicknamed = [
            makeRoomMember({ user: makePublicUser({ id: "u2", username: "battler" }), nickname: "Battler-kun" }),
        ];

        // when
        const names = typingNames(["u2"], nicknamed, "u1");

        // then
        expect(names).toEqual(["Battler-kun"]);
    });

    it("ignores a nickname that is only whitespace", () => {
        // given
        const nicknamed = [
            makeRoomMember({
                user: makePublicUser({ id: "u2", username: "battler", display_name: "Battler" }),
                nickname: "   ",
            }),
        ];

        // when
        const names = typingNames(["u2"], nicknamed, "u1");

        // then
        expect(names).toEqual(["Battler"]);
    });

    it("falls back to the username when there is no display name", () => {
        // given
        const nameless = [
            makeRoomMember({ user: makePublicUser({ id: "u2", username: "battler", display_name: "" }) }),
        ];

        // when
        const names = typingNames(["u2"], nameless, "u1");

        // then
        expect(names).toEqual(["battler"]);
    });

    it("falls back to a placeholder for somebody the room does not list", () => {
        // when
        const names = typingNames(["ghost"], members, "u1");

        // then
        expect(names).toEqual(["Someone"]);
    });

    it("never says the viewer is typing to themselves", () => {
        // when
        const names = typingNames(["u1", "u2"], members, "u1");
        const alone = typingNames(["u1"], members, "u1");

        // then
        expect(names).toEqual(["Battler"]);
        expect(alone).toEqual([]);
    });

    it("keeps everybody when there is no signed-in viewer to exclude", () => {
        // when
        const names = typingNames(["u1", "u2"], members, undefined);

        // then
        expect(names).toEqual(["Beatrice", "Battler"]);
    });

    it("keeps the order the typing ids arrived in", () => {
        // when
        const names = typingNames(["u2", "ghost"], members, "u1");

        // then
        expect(names).toEqual(["Battler", "Someone"]);
    });
});

describe("typingNames over a direct message roster", () => {
    const users = [
        makePublicUser({ id: "u1" }),
        makePublicUser({ id: "u2", username: "battler", display_name: "Battler" }),
    ];

    it("names the person typing", () => {
        // when
        const names = typingNames(["u2"], users, "u1");

        // then
        expect(names).toEqual(["Battler"]);
    });

    it("falls back to the username when there is no display name", () => {
        // given
        const nameless = [makePublicUser({ id: "u2", username: "battler", display_name: "" })];

        // when
        const names = typingNames(["u2"], nameless, "u1");

        // then
        expect(names).toEqual(["battler"]);
    });

    it("falls back to a placeholder when no conversation is open", () => {
        // when
        const names = typingNames(["u2"], undefined, "u1");

        // then
        expect(names).toEqual(["Someone"]);
    });
});
