import { describe, expect, it } from "vitest";
import type { ChatRoomMember } from "../types/api";
import { effectiveMemberUser, memberModPermissions, type MemberModContext } from "./chatMembers";

function makeMember(overrides: Partial<ChatRoomMember> = {}): ChatRoomMember {
    return {
        user: { id: "u1", username: "beatrice", display_name: "Beatrice" },
        role: "member",
        joined_at: "2026-01-01T00:00:00Z",
        nickname: "",
        member_avatar_url: "",
        nickname_locked: false,
        ...overrides,
    };
}

function makeContext(overrides: Partial<MemberModContext> = {}): MemberModContext {
    return {
        selfId: "self",
        isSystem: false,
        isSiteMod: false,
        canModerateRoom: false,
        ...overrides,
    };
}

describe("effectiveMemberUser", () => {
    it("prefers the room nickname over every other name", () => {
        // given
        const member = makeMember({ nickname: "Golden Witch" });

        // when
        const user = effectiveMemberUser(member);

        // then
        expect(user.display_name).toBe("Golden Witch");
    });

    it("falls back to the account display name when the nickname is only whitespace", () => {
        // given
        const member = makeMember({ nickname: "   " });

        // when
        const user = effectiveMemberUser(member);

        // then
        expect(user.display_name).toBe("Beatrice");
    });

    it("falls back to the username when there is no nickname and no display name", () => {
        // given
        const member = makeMember({ nickname: "", user: { id: "u1", username: "beatrice", display_name: "  " } });

        // when
        const user = effectiveMemberUser(member);

        // then
        expect(user.display_name).toBe("beatrice");
    });

    it("prefers the per-room avatar over the account avatar", () => {
        // given
        const member = makeMember({
            member_avatar_url: "/uploads/room.png",
            user: { id: "u1", username: "beatrice", display_name: "Beatrice", avatar_url: "/uploads/account.png" },
        });

        // when
        const user = effectiveMemberUser(member);

        // then
        expect(user.avatar_url).toBe("/uploads/room.png");
    });

    it("falls back to the account avatar when the per-room avatar is blank", () => {
        // given
        const member = makeMember({
            member_avatar_url: "   ",
            user: { id: "u1", username: "beatrice", display_name: "Beatrice", avatar_url: "/uploads/account.png" },
        });

        // when
        const user = effectiveMemberUser(member);

        // then
        expect(user.avatar_url).toBe("/uploads/account.png");
    });

    it("carries the rest of the account through untouched", () => {
        // given
        const member = makeMember({
            nickname: "Golden Witch",
            user: { id: "u1", username: "beatrice", display_name: "Beatrice", role: "moderator", banned: true },
        });

        // when
        const user = effectiveMemberUser(member);

        // then
        expect(user.id).toBe("u1");
        expect(user.username).toBe("beatrice");
        expect(user.role).toBe("moderator");
        expect(user.banned).toBe(true);
    });
});

describe("memberModPermissions", () => {
    it("gives an ordinary member no powers at all", () => {
        // given
        const ctx = makeContext();

        // when
        const perms = memberModPermissions(makeMember(), ctx);

        // then
        expect(perms).toEqual({
            isSelf: false,
            timeoutIsActive: false,
            canKick: false,
            canEditNickname: false,
            canTimeout: false,
            canClearTimeout: false,
            canActOnMember: false,
        });
    });

    it("lets a room moderator kick and time out a plain member", () => {
        // given
        const ctx = makeContext({ canModerateRoom: true });

        // when
        const perms = memberModPermissions(makeMember(), ctx);

        // then
        expect(perms.canKick).toBe(true);
        expect(perms.canTimeout).toBe(true);
        expect(perms.canActOnMember).toBe(true);
    });

    it("flags the viewer's own row and refuses to act on it", () => {
        // given
        const ctx = makeContext({ selfId: "u1", canModerateRoom: true, isSiteMod: true });

        // when
        const perms = memberModPermissions(makeMember(), ctx);

        // then
        expect(perms.isSelf).toBe(true);
        expect(perms.canKick).toBe(false);
        expect(perms.canTimeout).toBe(false);
        expect(perms.canEditNickname).toBe(false);
    });

    it("refuses to let a moderator lift their own timeout", () => {
        // given
        const ctx = makeContext({ selfId: "u1", canModerateRoom: true });

        // when
        const perms = memberModPermissions(
            makeMember({ timeout_until: "2026-01-01T01:00:00Z", timeout_set_by_staff: false }),
            ctx,
        );

        // then
        expect(perms.isSelf).toBe(true);
        expect(perms.timeoutIsActive).toBe(true);
        expect(perms.canClearTimeout).toBe(false);
        expect(perms.canActOnMember).toBe(false);
    });

    it("refuses every action inside a system room", () => {
        // given
        const ctx = makeContext({ canModerateRoom: true, isSiteMod: true, isSystem: true });

        // when
        const perms = memberModPermissions(makeMember({ timeout_until: "2026-01-01T01:00:00Z" }), ctx);

        // then
        expect(perms.canKick).toBe(false);
        expect(perms.canTimeout).toBe(false);
        expect(perms.canEditNickname).toBe(false);
        expect(perms.canClearTimeout).toBe(false);
        expect(perms.canActOnMember).toBe(false);
    });

    it("shields site staff from being kicked or timed out by anyone", () => {
        // given
        const target = makeMember({ user: { id: "u1", username: "ronove", display_name: "Ronove", role: "admin" } });
        const ctx = makeContext({ canModerateRoom: true, isSiteMod: true });

        // when
        const perms = memberModPermissions(target, ctx);

        // then
        expect(perms.canKick).toBe(false);
        expect(perms.canTimeout).toBe(false);
        expect(perms.canEditNickname).toBe(false);
    });

    it("shields the room host from being kicked", () => {
        // given
        const ctx = makeContext({ canModerateRoom: true });

        // when
        const perms = memberModPermissions(makeMember({ role: "host" }), ctx);

        // then
        expect(perms.canKick).toBe(false);
    });

    it("stops a room moderator timing out the host but lets a site moderator do it", () => {
        // given
        const host = makeMember({ role: "host" });

        // when
        const roomMod = memberModPermissions(host, makeContext({ canModerateRoom: true }));
        const siteMod = memberModPermissions(host, makeContext({ canModerateRoom: true, isSiteMod: true }));

        // then
        expect(roomMod.canTimeout).toBe(false);
        expect(siteMod.canTimeout).toBe(true);
    });

    it("only lets site moderators rename other people", () => {
        // given
        const target = makeMember();

        // when
        const roomMod = memberModPermissions(target, makeContext({ canModerateRoom: true }));
        const siteMod = memberModPermissions(target, makeContext({ isSiteMod: true }));

        // then
        expect(roomMod.canEditNickname).toBe(false);
        expect(siteMod.canEditNickname).toBe(true);
    });

    it("reports an active timeout and offers to clear it", () => {
        // given
        const timedOut = makeMember({ timeout_until: "2026-01-01T01:00:00Z" });

        // when
        const perms = memberModPermissions(timedOut, makeContext({ canModerateRoom: true }));

        // then
        expect(perms.timeoutIsActive).toBe(true);
        expect(perms.canClearTimeout).toBe(true);
    });

    it("offers nothing to clear when there is no timeout running", () => {
        // given
        const member = makeMember();

        // when
        const perms = memberModPermissions(member, makeContext({ canModerateRoom: true }));

        // then
        expect(perms.timeoutIsActive).toBe(false);
        expect(perms.canClearTimeout).toBe(false);
    });

    it("stops a room moderator undoing a timeout set by site staff", () => {
        // given
        const timedOut = makeMember({ timeout_until: "2026-01-01T01:00:00Z", timeout_set_by_staff: true });

        // when
        const perms = memberModPermissions(timedOut, makeContext({ canModerateRoom: true }));

        // then
        expect(perms.canClearTimeout).toBe(false);
        expect(perms.canTimeout).toBe(false);
    });

    it("lets a site moderator overrule a timeout set by site staff", () => {
        // given
        const timedOut = makeMember({ timeout_until: "2026-01-01T01:00:00Z", timeout_set_by_staff: true });

        // when
        const perms = memberModPermissions(timedOut, makeContext({ canModerateRoom: true, isSiteMod: true }));

        // then
        expect(perms.canClearTimeout).toBe(true);
        expect(perms.canTimeout).toBe(true);
    });

    it("lets a room moderator retime somebody they timed out themselves", () => {
        // given
        const timedOut = makeMember({ timeout_until: "2026-01-01T01:00:00Z", timeout_set_by_staff: false });

        // when
        const perms = memberModPermissions(timedOut, makeContext({ canModerateRoom: true }));

        // then
        expect(perms.canTimeout).toBe(true);
        expect(perms.canClearTimeout).toBe(true);
    });

    it("still offers a menu when renaming is the only thing left to do", () => {
        // given
        const ctx = makeContext({ isSiteMod: true, canModerateRoom: false });

        // when
        const perms = memberModPermissions(makeMember(), ctx);

        // then
        expect(perms.canKick).toBe(false);
        expect(perms.canTimeout).toBe(false);
        expect(perms.canEditNickname).toBe(true);
        expect(perms.canActOnMember).toBe(true);
    });
});
