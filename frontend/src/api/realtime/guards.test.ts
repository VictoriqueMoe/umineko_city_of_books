import { describe, expect, it } from "vitest";
import {
    REALTIME_EVENT_GUARDS,
    hasRealtimeGuard,
    isBanChangedPayload,
    isChatUnreadBumpedPayload,
    isLiveGamesCountPayload,
    isLockChangedPayload,
    isProfileChangedPayload,
    isRoleChangedPayload,
    passesRealtimeGuard,
} from "./guards";
import type { RealtimeEvent } from "./events";

describe("isRoleChangedPayload", () => {
    it("accepts what the server sends, including the empty role that means removal", () => {
        // then
        expect(isRoleChangedPayload({ user_id: "u1", role: "moderator" })).toBe(true);
        expect(isRoleChangedPayload({ user_id: "u1", role: "" })).toBe(true);
    });

    it("accepts a role the client does not know yet, because a guard stricter than the server is an outage", () => {
        // then
        expect(isRoleChangedPayload({ user_id: "u1", role: "archivist" })).toBe(true);
    });

    it("rejects a non-string role, which would otherwise be written into every cached user", () => {
        // then
        expect(isRoleChangedPayload({ user_id: "u1", role: { name: "admin" } })).toBe(false);
        expect(isRoleChangedPayload({ user_id: "u1", role: 7 })).toBe(false);
        expect(isRoleChangedPayload({ user_id: "u1", role: null })).toBe(false);
        expect(isRoleChangedPayload({ user_id: "u1" })).toBe(false);
    });

    it("rejects a missing or unusable user id, because the patch has no target", () => {
        // then
        expect(isRoleChangedPayload({ role: "admin" })).toBe(false);
        expect(isRoleChangedPayload({ user_id: "", role: "admin" })).toBe(false);
        expect(isRoleChangedPayload({ user_id: 42, role: "admin" })).toBe(false);
    });

    it("rejects a payload that is not an object", () => {
        // then
        expect(isRoleChangedPayload(null)).toBe(false);
        expect(isRoleChangedPayload("role_changed")).toBe(false);
        expect(isRoleChangedPayload([{ user_id: "u1", role: "admin" }])).toBe(false);
    });
});

describe("isBanChangedPayload", () => {
    it("accepts both directions of what the server sends", () => {
        // then
        expect(isBanChangedPayload({ user_id: "u1", banned: true, ban_reason: "spam" })).toBe(true);
        expect(isBanChangedPayload({ user_id: "u1", banned: false, ban_reason: "" })).toBe(true);
    });

    it("rejects an absent banned flag, which the provider would coerce into a silent unban", () => {
        // then
        expect(isBanChangedPayload({ user_id: "u1", ban_reason: "spam" })).toBe(false);
    });

    it("rejects a non-boolean banned flag, which the provider would coerce into a ban", () => {
        // then
        expect(isBanChangedPayload({ user_id: "u1", banned: "yes", ban_reason: "spam" })).toBe(false);
        expect(isBanChangedPayload({ user_id: "u1", banned: 1, ban_reason: "spam" })).toBe(false);
    });

    it("rejects a non-string ban reason, which is rendered as text", () => {
        // then
        expect(isBanChangedPayload({ user_id: "u1", banned: true, ban_reason: null })).toBe(false);
    });

    it("accepts an absent ban reason, because the consumer has always coalesced it to an empty string", () => {
        // then
        expect(isBanChangedPayload({ user_id: "u1", banned: true })).toBe(true);
    });

    it("rejects a missing user id", () => {
        // then
        expect(isBanChangedPayload({ banned: true, ban_reason: "spam" })).toBe(false);
    });
});

describe("isLockChangedPayload", () => {
    it("accepts both directions of what the server sends", () => {
        // then
        expect(isLockChangedPayload({ user_id: "u1", locked: true, lock_reason: "appeal pending" })).toBe(true);
        expect(isLockChangedPayload({ user_id: "u1", locked: false, lock_reason: "" })).toBe(true);
    });

    it("rejects an absent locked flag, which the provider would coerce into a silent unlock", () => {
        // then
        expect(isLockChangedPayload({ user_id: "u1", lock_reason: "appeal pending" })).toBe(false);
    });

    it("rejects a non-boolean locked flag, which the provider would coerce into a lock", () => {
        // then
        expect(isLockChangedPayload({ user_id: "u1", locked: "true", lock_reason: "" })).toBe(false);
    });

    it("rejects a non-string lock reason, which is rendered as text", () => {
        // then
        expect(isLockChangedPayload({ user_id: "u1", locked: true, lock_reason: 3 })).toBe(false);
    });

    it("accepts an absent lock reason, because the consumer has always coalesced it to an empty string", () => {
        // then
        expect(isLockChangedPayload({ user_id: "u1", locked: true })).toBe(true);
    });

    it("rejects a missing user id", () => {
        // then
        expect(isLockChangedPayload({ locked: true, lock_reason: "" })).toBe(false);
    });
});

describe("isProfileChangedPayload", () => {
    it("accepts each of the two shapes the server actually sends, one field at a time", () => {
        // then
        expect(isProfileChangedPayload({ user_id: "u1", display_name: "Beatrice" })).toBe(true);
        expect(isProfileChangedPayload({ user_id: "u1", avatar_url: "/uploads/a.png" })).toBe(true);
    });

    it("accepts both fields together and neither field at all", () => {
        // then
        expect(isProfileChangedPayload({ user_id: "u1", display_name: "Beatrice", avatar_url: "/a.png" })).toBe(true);
        expect(isProfileChangedPayload({ user_id: "u1" })).toBe(true);
    });

    it("rejects a non-string display name or avatar url", () => {
        // then
        expect(isProfileChangedPayload({ user_id: "u1", display_name: 42 })).toBe(false);
        expect(isProfileChangedPayload({ user_id: "u1", avatar_url: null })).toBe(false);
    });

    it("rejects a missing user id, because the patch has no target", () => {
        // then
        expect(isProfileChangedPayload({ display_name: "Beatrice" })).toBe(false);
        expect(isProfileChangedPayload({ user_id: "", display_name: "Beatrice" })).toBe(false);
    });
});

describe("isChatUnreadBumpedPayload", () => {
    it("accepts the count the server sends", () => {
        // then
        expect(isChatUnreadBumpedPayload({ room_id: "r1", total: 0 })).toBe(true);
        expect(isChatUnreadBumpedPayload({ room_id: "r1", total: 12 })).toBe(true);
    });

    it("rejects a total that is not a finite number, because it lands in the unread badge", () => {
        // then
        expect(isChatUnreadBumpedPayload({ room_id: "r1", total: "12" })).toBe(false);
        expect(isChatUnreadBumpedPayload({ room_id: "r1", total: null })).toBe(false);
        expect(isChatUnreadBumpedPayload({ room_id: "r1" })).toBe(false);
        expect(isChatUnreadBumpedPayload({ room_id: "r1", total: Number.POSITIVE_INFINITY })).toBe(false);
        expect(isChatUnreadBumpedPayload({ room_id: "r1", total: Number.NaN })).toBe(false);
    });

    it("does not demand a room id, because nothing reads it yet", () => {
        // then
        expect(isChatUnreadBumpedPayload({ total: 3 })).toBe(true);
    });
});

describe("isLiveGamesCountPayload", () => {
    it("accepts the count the server sends", () => {
        // then
        expect(isLiveGamesCountPayload({ count: 0 })).toBe(true);
        expect(isLiveGamesCountPayload({ count: 5 })).toBe(true);
    });

    it("rejects a count that is not a finite number, because it lands in the live games badge", () => {
        // then
        expect(isLiveGamesCountPayload({ count: "5" })).toBe(false);
        expect(isLiveGamesCountPayload({})).toBe(false);
        expect(isLiveGamesCountPayload({ count: Number.POSITIVE_INFINITY })).toBe(false);
        expect(isLiveGamesCountPayload(null)).toBe(false);
    });
});

describe("the guard table is opt-in", () => {
    it("registers exactly the six events whose bad payload would corrupt user visible state", () => {
        // then
        expect(Object.keys(REALTIME_EVENT_GUARDS).sort()).toEqual([
            "ban_changed",
            "chat_unread_bumped",
            "live_games_count",
            "lock_changed",
            "profile_changed",
            "role_changed",
        ]);
    });

    it("reports which names carry a guard", () => {
        // then
        expect(hasRealtimeGuard("role_changed")).toBe(true);
        expect(hasRealtimeGuard("chat_message")).toBe(false);
    });

    it("passes an unguarded event through untouched, which is exactly today's trust level", () => {
        // given
        const event = { type: "chat_message", data: { nonsense: true } } as unknown as RealtimeEvent;

        // then
        expect(passesRealtimeGuard(event)).toBe(true);
    });

    it("applies the guard when the event has one", () => {
        // given
        const good = { type: "live_games_count", data: { count: 4 } } as RealtimeEvent;
        const bad = { type: "live_games_count", data: { count: "4" } } as unknown as RealtimeEvent;

        // then
        expect(passesRealtimeGuard(good)).toBe(true);
        expect(passesRealtimeGuard(bad)).toBe(false);
    });
});
