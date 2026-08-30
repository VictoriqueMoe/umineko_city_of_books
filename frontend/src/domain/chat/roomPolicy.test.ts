import { describe, expect, it } from "vitest";
import type { ChatRoom, SiteRole } from "../../types/api";
import { isTimeoutActive, roomViewerPolicy } from "./roomPolicy";

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return {
        id: "room-1",
        name: "Rokkenjima",
        description: "",
        type: "group",
        is_public: true,
        is_rp: false,
        is_system: false,
        tags: [],
        viewer_muted: false,
        viewer_ghost: false,
        is_member: true,
        member_count: 3,
        hot_score: 0,
        members: [],
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

describe("roomViewerPolicy", () => {
    it("makes the room host a moderator of the room", () => {
        // given
        const room = makeRoom({ viewer_role: "host" });

        // when
        const policy = roomViewerPolicy(room, { role: undefined });

        // then
        expect(policy).toEqual({ isHost: true, isSystem: false, isSiteMod: false, canModerateRoom: true });
    });

    it("makes an ordinary member no kind of moderator", () => {
        // given
        const room = makeRoom({ viewer_role: "member" });

        // when
        const policy = roomViewerPolicy(room, { role: undefined });

        // then
        expect(policy).toEqual({ isHost: false, isSystem: false, isSiteMod: false, canModerateRoom: false });
    });

    const staffRoles: SiteRole[] = ["super_admin", "admin", "moderator"];

    for (const role of staffRoles) {
        it(`makes a ${role} a moderator of a room they only joined`, () => {
            // given
            const room = makeRoom({ viewer_role: "member" });

            // when
            const policy = roomViewerPolicy(room, { role });

            // then
            expect(policy).toEqual({ isHost: false, isSystem: false, isSiteMod: true, canModerateRoom: true });
        });
    }

    it("reports a system room as one, without withdrawing moderation", () => {
        // given
        const room = makeRoom({ is_system: true, viewer_role: "host" });

        // when
        const policy = roomViewerPolicy(room, { role: "moderator" });

        // then
        expect(policy).toEqual({ isHost: true, isSystem: true, isSiteMod: true, canModerateRoom: true });
    });

    it("treats a room with no viewer_role as one the viewer does not host", () => {
        // given
        const room = makeRoom({ viewer_role: undefined });

        // when
        const policy = roomViewerPolicy(room, { role: "admin" });

        // then
        expect(policy.isHost).toBe(false);
        expect(policy.canModerateRoom).toBe(true);
    });

    it("still reports the site role when the room has not loaded", () => {
        // given
        const room = null;

        // when
        const policy = roomViewerPolicy(room, { role: "admin" });

        // then
        expect(policy).toEqual({ isHost: false, isSystem: false, isSiteMod: true, canModerateRoom: true });
    });

    it("denies everything when there is no viewer", () => {
        // given
        const room = makeRoom({ viewer_role: "member" });

        // when
        const policy = roomViewerPolicy(room, null);

        // then
        expect(policy).toEqual({ isHost: false, isSystem: false, isSiteMod: false, canModerateRoom: false });
    });
});

describe("isTimeoutActive", () => {
    const now = Date.parse("2026-08-28T12:00:00Z");

    it("is active while the timeout is still in the future", () => {
        // given
        const until = "2026-08-28T12:00:01Z";

        // when
        const active = isTimeoutActive(until, now);

        // then
        expect(active).toBe(true);
    });

    it("is over once the instant is reached, not a tick later", () => {
        // given
        const until = "2026-08-28T12:00:00Z";

        // when
        const active = isTimeoutActive(until, now);

        // then
        expect(active).toBe(false);
    });

    it("is over when the timeout is in the past", () => {
        // given
        const until = "2026-08-28T11:59:59Z";

        // when
        const active = isTimeoutActive(until, now);

        // then
        expect(active).toBe(false);
    });

    it("reads a timezone-less server timestamp as UTC rather than local time", () => {
        // given
        const until = "2026-08-28 12:00:01";

        // when
        const active = isTimeoutActive(until, now);

        // then
        expect(active).toBe(true);
    });

    const absentValues: (string | null | undefined)[] = [undefined, null, "", "   "];

    for (const until of absentValues) {
        it(`is inactive for ${JSON.stringify(until)}`, () => {
            // given
            const timeoutUntil = until;

            // when
            const active = isTimeoutActive(timeoutUntil, now);

            // then
            expect(active).toBe(false);
        });
    }

    it("is inactive for an unparseable timestamp rather than throwing", () => {
        // given
        const until = "not a date";

        // when
        const active = isTimeoutActive(until, now);

        // then
        expect(active).toBe(false);
    });

    it("takes now as a parameter, so the same timeout expires as the clock is advanced", () => {
        // given
        const until = "2026-08-28T12:30:00Z";

        // when
        const before = isTimeoutActive(until, now);
        const after = isTimeoutActive(until, now + 31 * 60 * 1000);

        // then
        expect(before).toBe(true);
        expect(after).toBe(false);
    });
});
