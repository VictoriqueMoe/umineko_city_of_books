import { describe, expect, it } from "vitest";
import { userIdentityPatch } from "./identityPatch";

describe("userIdentityPatch", () => {
    it("ignores a role change that names nobody", () => {
        // given
        const change = { type: "role_changed", data: { role: "moderator" } } as const;

        // when
        const update = userIdentityPatch(change);

        // then
        expect(update).toBeNull();
    });

    it("ignores a change whose user id is the empty string", () => {
        // given
        const change = { type: "ban_changed", data: { user_id: "", banned: true } } as const;

        // when
        const update = userIdentityPatch(change);

        // then
        expect(update).toBeNull();
    });

    it("carries the new role", () => {
        // given
        const change = { type: "role_changed", data: { user_id: "user-1", role: "moderator" } } as const;

        // when
        const update = userIdentityPatch(change);

        // then
        expect(update).toEqual({ userId: "user-1", patch: { role: "moderator" } });
    });

    it("turns a cleared role into the empty string rather than dropping the field", () => {
        // given
        const change = { type: "role_changed", data: { user_id: "user-1", role: "" } } as const;

        // when
        const update = userIdentityPatch(change);

        // then
        expect(update).toEqual({ userId: "user-1", patch: { role: "" } });
    });

    it("falls back to the empty role when the event carries none", () => {
        // given
        const change = { type: "role_changed", data: { user_id: "user-1" } } as const;

        // when
        const update = userIdentityPatch(change);

        // then
        expect(update).toEqual({ userId: "user-1", patch: { role: "" } });
    });

    it("defaults a missing ban reason to the empty string", () => {
        // given
        const change = { type: "ban_changed", data: { user_id: "user-1", banned: true } } as const;

        // when
        const update = userIdentityPatch(change);

        // then
        expect(update).toEqual({ userId: "user-1", patch: { banned: true, ban_reason: "" } });
    });

    it("carries an unban with its reason cleared", () => {
        // given
        const change = { type: "ban_changed", data: { user_id: "user-1", banned: false, ban_reason: "" } } as const;

        // when
        const update = userIdentityPatch(change);

        // then
        expect(update).toEqual({ userId: "user-1", patch: { banned: false, ban_reason: "" } });
    });

    it("carries a lock with its reason", () => {
        // given
        const change = {
            type: "lock_changed",
            data: { user_id: "user-1", locked: true, lock_reason: "too much magic" },
        } as const;

        // when
        const update = userIdentityPatch(change);

        // then
        expect(update).toEqual({ userId: "user-1", patch: { locked: true, lock_reason: "too much magic" } });
    });

    it("defaults a missing lock reason to the empty string", () => {
        // given
        const change = { type: "lock_changed", data: { user_id: "user-1", locked: false } } as const;

        // when
        const update = userIdentityPatch(change);

        // then
        expect(update).toEqual({ userId: "user-1", patch: { locked: false, lock_reason: "" } });
    });

    it("patches only the profile fields the event carried", () => {
        // given
        const change = { type: "profile_changed", data: { user_id: "user-1", display_name: "Beato" } } as const;

        // when
        const update = userIdentityPatch(change);

        // then
        expect(update).toEqual({ userId: "user-1", patch: { display_name: "Beato" } });
    });

    it("carries an emptied avatar, which is a value and not an absent field", () => {
        // given
        const change = { type: "profile_changed", data: { user_id: "user-1", avatar_url: "" } } as const;

        // when
        const update = userIdentityPatch(change);

        // then
        expect(update).toEqual({ userId: "user-1", patch: { avatar_url: "" } });
    });

    it("names the user with an empty patch when a profile change carried no fields", () => {
        // given
        const change = { type: "profile_changed", data: { user_id: "user-1" } } as const;

        // when
        const update = userIdentityPatch(change);

        // then
        expect(update).toEqual({ userId: "user-1", patch: {} });
    });

    it("keeps the four events apart rather than flattening them into one shape", () => {
        // given
        const userId = "user-1";

        // when
        const role = userIdentityPatch({ type: "role_changed", data: { user_id: userId, role: "admin" } });
        const ban = userIdentityPatch({ type: "ban_changed", data: { user_id: userId, banned: true } });
        const lock = userIdentityPatch({ type: "lock_changed", data: { user_id: userId, locked: true } });
        const profile = userIdentityPatch({ type: "profile_changed", data: { user_id: userId, avatar_url: "a.png" } });

        // then
        expect(Object.keys(role?.patch ?? {})).toEqual(["role"]);
        expect(Object.keys(ban?.patch ?? {})).toEqual(["banned", "ban_reason"]);
        expect(Object.keys(lock?.patch ?? {})).toEqual(["locked", "lock_reason"]);
        expect(Object.keys(profile?.patch ?? {})).toEqual(["avatar_url"]);
    });
});
