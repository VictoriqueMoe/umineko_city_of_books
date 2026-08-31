import { describe, expect, it } from "vitest";
import { makeChatMessage, makeRoomMember } from "../../test-utils/fixtures";
import type { ChatMessage, PostMedia } from "../../types/api";
import {
    applyChatMemberUpdateToMembers,
    applyChatMemberUpdateToMessages,
    applyChatMessageDeleted,
    applyChatMessageEdited,
    applyChatMessagePinned,
    applyChatMessageUnpinned,
    applyLocalMemberChangeToMembers,
    applyLocalMemberChangeToMessages,
    applyReaction,
    applySiteRoleChangeToMembers,
    applySiteRoleChangeToMessages,
    type ChatMemberUpdatedPayload,
    type ChatReactionPayload,
} from "./messagePatches";

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return makeChatMessage({ body: "hello", ...overrides });
}

function makeMemberUpdate(overrides: Partial<ChatMemberUpdatedPayload> = {}): ChatMemberUpdatedPayload {
    return {
        room_id: "room-1",
        user_id: "u1",
        nickname: "",
        display_name: "Beatrice",
        username: "beatrice",
        member_avatar_url: "",
        nickname_locked: false,
        timeout_until: "",
        timeout_set_by_staff: false,
        ...overrides,
    };
}

function makeReaction(overrides: Partial<ChatReactionPayload> = {}): ChatReactionPayload {
    return {
        room_id: "room-1",
        message_id: "m1",
        emoji: "🌹",
        user_id: "u2",
        display_name: "Ange",
        ...overrides,
    };
}

function makeMedia(id: number): PostMedia {
    return { id, media_url: `/uploads/${id}.png`, media_type: "image", sort_order: 0 };
}

describe("applyLocalMemberChangeToMembers", () => {
    it("swaps in the changed member and leaves the rest of the roster alone", () => {
        // given
        const other = makeRoomMember({ user: { id: "u2", username: "ange", display_name: "Ange" } });
        const current = [makeRoomMember(), other];
        const updated = makeRoomMember({ nickname: "Golden Witch" });

        // when
        const next = applyLocalMemberChangeToMembers(current, updated);

        // then
        expect(next[0].nickname).toBe("Golden Witch");
        expect(next[1]).toBe(other);
    });

    it("leaves a member with a different id alone", () => {
        // given
        const other = makeRoomMember({ user: { id: "u2", username: "ange", display_name: "Ange" } });

        // when
        const next = applyLocalMemberChangeToMembers([other], makeRoomMember({ nickname: "Golden Witch" }));

        // then
        expect(next[0]).toBe(other);
    });
});

describe("applyLocalMemberChangeToMessages", () => {
    it("restamps that member's messages with the new nickname and avatar", () => {
        // given
        const current = [makeMessage()];

        // when
        const next = applyLocalMemberChangeToMessages(
            current,
            makeRoomMember({ nickname: "Golden Witch", member_avatar_url: "/uploads/beato.png" }),
        );

        // then
        expect(next[0].sender_nickname).toBe("Golden Witch");
        expect(next[0].sender_member_avatar_url).toBe("/uploads/beato.png");
    });

    it("clears the stamped nickname and avatar when the member has neither", () => {
        // given
        const current = [
            makeMessage({ sender_nickname: "Golden Witch", sender_member_avatar_url: "/uploads/beato.png" }),
        ];

        // when
        const next = applyLocalMemberChangeToMessages(current, makeRoomMember());

        // then
        expect(next[0].sender_nickname).toBeUndefined();
        expect(next[0].sender_member_avatar_url).toBeUndefined();
    });

    it("leaves messages from other senders untouched", () => {
        // given
        const foreign = makeMessage({ id: "m2", sender: { id: "u2", username: "ange", display_name: "Ange" } });

        // when
        const next = applyLocalMemberChangeToMessages([foreign], makeRoomMember({ nickname: "Golden Witch" }));

        // then
        expect(next[0]).toBe(foreign);
    });
});

describe("applyChatMemberUpdateToMembers", () => {
    it("writes the broadcast nickname, avatar and lock onto the matching member", () => {
        // given
        const current = [makeRoomMember()];

        // when
        const next = applyChatMemberUpdateToMembers(
            current,
            makeMemberUpdate({
                nickname: "Golden Witch",
                member_avatar_url: "/uploads/beato.png",
                nickname_locked: true,
            }),
        );

        // then
        expect(next[0].nickname).toBe("Golden Witch");
        expect(next[0].member_avatar_url).toBe("/uploads/beato.png");
        expect(next[0].nickname_locked).toBe(true);
    });

    it("turns an empty timeout into no timeout at all", () => {
        // given
        const current = [makeRoomMember({ timeout_until: "2026-01-01T01:00:00Z", timeout_set_by_staff: true })];

        // when
        const next = applyChatMemberUpdateToMembers(current, makeMemberUpdate());

        // then
        expect(next[0].timeout_until).toBeUndefined();
        expect(next[0].timeout_set_by_staff).toBe(false);
    });

    it("keeps a live timeout and who set it", () => {
        // given
        const current = [makeRoomMember()];

        // when
        const next = applyChatMemberUpdateToMembers(
            current,
            makeMemberUpdate({ timeout_until: "2026-01-01T01:00:00Z", timeout_set_by_staff: true }),
        );

        // then
        expect(next[0].timeout_until).toBe("2026-01-01T01:00:00Z");
        expect(next[0].timeout_set_by_staff).toBe(true);
    });

    it("leaves members with a different id alone", () => {
        // given
        const other = makeRoomMember({ user: { id: "u2", username: "ange", display_name: "Ange" } });

        // when
        const next = applyChatMemberUpdateToMembers([other], makeMemberUpdate({ nickname: "Golden Witch" }));

        // then
        expect(next[0]).toBe(other);
    });
});

describe("applyChatMemberUpdateToMessages", () => {
    it("restamps the sender fields on that member's messages", () => {
        // given
        const current = [makeMessage()];

        // when
        const next = applyChatMemberUpdateToMessages(
            current,
            makeMemberUpdate({ nickname: "Golden Witch", member_avatar_url: "/uploads/beato.png" }),
        );

        // then
        expect(next[0].sender_nickname).toBe("Golden Witch");
        expect(next[0].sender_member_avatar_url).toBe("/uploads/beato.png");
    });

    it("renames the reply preview with the new nickname", () => {
        // given
        const current = [
            makeMessage({
                id: "m2",
                sender: { id: "u2", username: "ange", display_name: "Ange" },
                reply_to: { id: "m1", sender_id: "u1", sender_name: "Beatrice", body_preview: "hello" },
            }),
        ];

        // when
        const next = applyChatMemberUpdateToMessages(current, makeMemberUpdate({ nickname: "Golden Witch" }));

        // then
        expect(next[0].reply_to?.sender_name).toBe("Golden Witch");
        expect(next[0].sender_nickname).toBeUndefined();
    });

    it("falls back to the display name in the reply preview when there is no nickname", () => {
        // given
        const current = [
            makeMessage({
                id: "m2",
                sender: { id: "u2", username: "ange", display_name: "Ange" },
                reply_to: { id: "m1", sender_id: "u1", sender_name: "stale", body_preview: "hello" },
            }),
        ];

        // when
        const next = applyChatMemberUpdateToMessages(current, makeMemberUpdate({ nickname: "   " }));

        // then
        expect(next[0].reply_to?.sender_name).toBe("Beatrice");
    });

    it("falls back to the username in the reply preview when neither name is set", () => {
        // given
        const current = [
            makeMessage({
                id: "m2",
                sender: { id: "u2", username: "ange", display_name: "Ange" },
                reply_to: { id: "m1", sender_id: "u1", sender_name: "stale", body_preview: "hello" },
            }),
        ];

        // when
        const next = applyChatMemberUpdateToMessages(current, makeMemberUpdate({ nickname: "", display_name: "  " }));

        // then
        expect(next[0].reply_to?.sender_name).toBe("beatrice");
    });

    it("leaves messages that neither sent nor replied to the member untouched", () => {
        // given
        const foreign = makeMessage({ id: "m2", sender: { id: "u2", username: "ange", display_name: "Ange" } });

        // when
        const next = applyChatMemberUpdateToMessages([foreign], makeMemberUpdate({ nickname: "Golden Witch" }));

        // then
        expect(next[0]).toBe(foreign);
    });
});

describe("applySiteRoleChangeToMembers", () => {
    it("restamps the promoted member and leaves the others by reference", () => {
        // given
        const other = makeRoomMember({ user: { id: "u2", username: "battler", display_name: "Battler" } });
        const current = [makeRoomMember(), other];

        // when
        const next = applySiteRoleChangeToMembers(current, { user_id: "u1", role: "moderator" });

        // then
        expect(next[0].user.role).toBe("moderator");
        expect(next[1]).toBe(other);
    });

    it("reads an empty role as no role at all", () => {
        // given
        const current = [
            makeRoomMember({ user: { id: "u1", username: "beatrice", display_name: "B", role: "admin" } }),
        ];

        // when
        const next = applySiteRoleChangeToMembers(current, { user_id: "u1", role: "" });

        // then
        expect(next[0].user.role).toBeUndefined();
    });
});

describe("applySiteRoleChangeToMessages", () => {
    it("restamps every message that person sent", () => {
        // given
        const foreign = makeMessage({ id: "m2", sender: { id: "u2", username: "battler", display_name: "Battler" } });
        const current = [makeMessage({ id: "m1" }), foreign, makeMessage({ id: "m3" })];

        // when
        const next = applySiteRoleChangeToMessages(current, { user_id: "u1", role: "moderator" });

        // then
        expect(next[0].sender.role).toBe("moderator");
        expect(next[2].sender.role).toBe("moderator");
        expect(next[1]).toBe(foreign);
    });

    it("reads an empty role as no role at all", () => {
        // given
        const current = [makeMessage({ sender: { id: "u1", username: "beatrice", display_name: "B", role: "admin" } })];

        // when
        const next = applySiteRoleChangeToMessages(current, { user_id: "u1", role: "" });

        // then
        expect(next[0].sender.role).toBeUndefined();
    });
});

describe("applyChatMessagePinned", () => {
    it("pins only the target message and records who pinned it", () => {
        // given
        const other = makeMessage({ id: "m2" });
        const current = [makeMessage({ id: "m1" }), other];

        // when
        const next = applyChatMessagePinned(current, {
            room_id: "room-1",
            message_id: "m1",
            pinned_at: "2026-01-01T02:00:00Z",
            pinned_by: "u2",
        });

        // then
        expect(next[0].pinned).toBe(true);
        expect(next[0].pinned_at).toBe("2026-01-01T02:00:00Z");
        expect(next[0].pinned_by).toBe("u2");
        expect(next[1]).toBe(other);
    });

    it("leaves a message it does not hold alone", () => {
        // given
        const other = makeMessage({ id: "m2" });

        // when
        const next = applyChatMessagePinned([other], {
            room_id: "room-1",
            message_id: "m1",
            pinned_at: "2026-01-01T02:00:00Z",
            pinned_by: "u2",
        });

        // then
        expect(next[0]).toBe(other);
    });
});

describe("applyChatMessageUnpinned", () => {
    it("unpins the message and forgets the pin metadata", () => {
        // given
        const current = [makeMessage({ id: "m1", pinned: true, pinned_at: "2026-01-01T02:00:00Z", pinned_by: "u2" })];

        // when
        const next = applyChatMessageUnpinned(current, { room_id: "room-1", message_id: "m1" });

        // then
        expect(next[0].pinned).toBe(false);
        expect(next[0].pinned_at).toBeUndefined();
        expect(next[0].pinned_by).toBeUndefined();
    });

    it("ignores an unpin for a message it does not hold", () => {
        // given
        const other = makeMessage({ id: "m1", pinned: true });

        // when
        const next = applyChatMessageUnpinned([other], { room_id: "room-1", message_id: "m404" });

        // then
        expect(next[0]).toBe(other);
    });
});

describe("applyChatMessageDeleted", () => {
    it("drops the deleted message", () => {
        // given
        const current = [makeMessage({ id: "m1" }), makeMessage({ id: "m2" })];

        // when
        const next = applyChatMessageDeleted(current, { room_id: "room-1", message_id: "m1" });

        // then
        expect(next.map(m => m.id)).toEqual(["m2"]);
    });

    it("keeps the list whole when the message is not there", () => {
        // given
        const current = [makeMessage({ id: "m1" })];

        // when
        const next = applyChatMessageDeleted(current, { room_id: "room-1", message_id: "m404" });

        // then
        expect(next.map(m => m.id)).toEqual(["m1"]);
    });

    it("drops a message the server says was deleted", () => {
        // given
        const current = [makeMessage({ id: "m1" })];

        // when
        const next = applyChatMessageDeleted(current, { room_id: "room-1", message_id: "m1" });

        // then
        expect(next).toEqual([]);
    });
});

describe("applyChatMessageEdited", () => {
    it("applies an edit that arrives over the socket", () => {
        // given
        const current = [makeMessage({ id: "m1" })];

        // when
        const next = applyChatMessageEdited(
            current,
            makeMessage({ id: "m1", body: "the red truth", edited_at: "2026-08-02T10:05:00Z" }),
        );

        // then
        expect(next[0].body).toBe("the red truth");
    });

    it("replaces the body and records when it was edited", () => {
        // given
        const current = [makeMessage({ id: "m1", body: "hello" })];

        // when
        const next = applyChatMessageEdited(
            current,
            makeMessage({ id: "m1", body: "goodbye", edited_at: "2026-01-01T03:00:00Z" }),
        );

        // then
        expect(next[0].body).toBe("goodbye");
        expect(next[0].edited_at).toBe("2026-01-01T03:00:00Z");
    });

    it("keeps the existing media when the edit carries none", () => {
        // given
        const current = [makeMessage({ id: "m1", media: [makeMedia(1)] })];

        // when
        const next = applyChatMessageEdited(current, makeMessage({ id: "m1", body: "goodbye" }));

        // then
        expect(next[0].media).toEqual([makeMedia(1)]);
    });

    it("replaces the media when the edit carries some", () => {
        // given
        const current = [makeMessage({ id: "m1", media: [makeMedia(1)] })];

        // when
        const next = applyChatMessageEdited(current, makeMessage({ id: "m1", media: [makeMedia(2)] }));

        // then
        expect(next[0].media).toEqual([makeMedia(2)]);
    });

    it("clears the media when the edit carries an empty list", () => {
        // given
        const current = [makeMessage({ id: "m1", media: [makeMedia(1)] })];

        // when
        const next = applyChatMessageEdited(current, makeMessage({ id: "m1", media: [] }));

        // then
        expect(next[0].media).toEqual([]);
    });

    it("ignores an edit for a message it does not hold", () => {
        // given
        const original = makeMessage({ id: "m1", body: "hello" });

        // when
        const next = applyChatMessageEdited([original], makeMessage({ id: "m404", body: "goodbye" }));

        // then
        expect(next[0]).toBe(original);
    });
});

describe("applyReaction", () => {
    it("creates a group with a single reaction when the emoji is new", () => {
        // given
        const current = [makeMessage({ id: "m1", reactions: [] })];

        // when
        const next = applyReaction(current, makeReaction(), "u1", 1);

        // then
        expect(next[0].reactions).toEqual([{ emoji: "🌹", count: 1, viewer_reacted: false, display_names: ["Ange"] }]);
    });

    it("marks a new group as viewer reacted when you are the one reacting", () => {
        // given
        const current = [makeMessage({ id: "m1", reactions: [] })];

        // when
        const next = applyReaction(current, makeReaction({ user_id: "u1", display_name: "Beatrice" }), "u1", 1);

        // then
        expect(next[0].reactions[0].viewer_reacted).toBe(true);
    });

    it("omits the display name from a new group when the payload has none", () => {
        // given
        const current = [makeMessage({ id: "m1", reactions: [] })];

        // when
        const next = applyReaction(current, makeReaction({ display_name: "" }), "u1", 1);

        // then
        expect(next[0].reactions[0].display_names).toEqual([]);
    });

    it("increments an existing group and records the new display name", () => {
        // given
        const current = [
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🌹", count: 1, viewer_reacted: true, display_names: ["Beatrice"] }],
            }),
        ];

        // when
        const next = applyReaction(current, makeReaction(), "u1", 1);

        // then
        expect(next[0].reactions[0].count).toBe(2);
        expect(next[0].reactions[0].display_names).toEqual(["Beatrice", "Ange"]);
    });

    it("keeps your own flag intact when somebody else reacts", () => {
        // given
        const current = [
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🌹", count: 1, viewer_reacted: true, display_names: ["Beatrice"] }],
            }),
        ];

        // when
        const next = applyReaction(current, makeReaction(), "u1", 1);

        // then
        expect(next[0].reactions[0].viewer_reacted).toBe(true);
    });

    it("does not record the same display name twice", () => {
        // given
        const current = [
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🌹", count: 1, viewer_reacted: false, display_names: ["Ange"] }],
            }),
        ];

        // when
        const next = applyReaction(current, makeReaction(), "u1", 1);

        // then
        expect(next[0].reactions[0].display_names).toEqual(["Ange"]);
        expect(next[0].reactions[0].count).toBe(2);
    });

    it("trusts the authoritative count from the server over the local increment", () => {
        // given
        const current = [
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🌹", count: 1, viewer_reacted: false, display_names: [] }],
            }),
        ];

        // when
        const next = applyReaction(current, makeReaction({ count: 7 }), "u1", 1);

        // then
        expect(next[0].reactions[0].count).toBe(7);
    });

    it("ignores a brand new emoji whose authoritative count is zero", () => {
        // given
        const current = [makeMessage({ id: "m1", reactions: [] })];

        // when
        const next = applyReaction(current, makeReaction({ count: 0 }), "u1", 1);

        // then
        expect(next[0].reactions).toEqual([]);
    });

    it("only touches the message the reaction belongs to", () => {
        // given
        const other = makeMessage({ id: "m2" });

        // when
        const next = applyReaction([makeMessage({ id: "m1" }), other], makeReaction(), "u1", 1);

        // then
        expect(next[1]).toBe(other);
    });

    it("decrements the count and drops the display name", () => {
        // given
        const current = [
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🌹", count: 2, viewer_reacted: false, display_names: ["Beatrice", "Ange"] }],
            }),
        ];

        // when
        const next = applyReaction(current, makeReaction(), "u1", -1);

        // then
        expect(next[0].reactions[0].count).toBe(1);
        expect(next[0].reactions[0].display_names).toEqual(["Beatrice"]);
    });

    it("removes only one copy of a duplicated display name", () => {
        // given
        const current = [
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🌹", count: 3, viewer_reacted: false, display_names: ["Ange", "Ange"] }],
            }),
        ];

        // when
        const next = applyReaction(current, makeReaction(), "u1", -1);

        // then
        expect(next[0].reactions[0].display_names).toEqual(["Ange"]);
    });

    it("drops the whole group when the last reaction is taken away", () => {
        // given
        const current = [
            makeMessage({
                id: "m1",
                reactions: [
                    { emoji: "🌹", count: 1, viewer_reacted: true, display_names: ["Beatrice"] },
                    { emoji: "🍰", count: 1, viewer_reacted: false, display_names: ["Ange"] },
                ],
            }),
        ];

        // when
        const next = applyReaction(current, makeReaction({ user_id: "u1", display_name: "Beatrice" }), "u1", -1);

        // then
        expect(next[0].reactions.map(r => r.emoji)).toEqual(["🍰"]);
    });

    it("clears your own flag when you take your reaction back", () => {
        // given
        const current = [
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🌹", count: 2, viewer_reacted: true, display_names: ["Beatrice", "Ange"] }],
            }),
        ];

        // when
        const next = applyReaction(current, makeReaction({ user_id: "u1", display_name: "Beatrice" }), "u1", -1);

        // then
        expect(next[0].reactions[0].viewer_reacted).toBe(false);
    });

    it("does not invent the reaction it was asked to remove when the message has none", () => {
        // given
        const current = [makeMessage({ id: "m1", reactions: [] })];

        // when
        const next = applyReaction(current, makeReaction(), "u1", -1);

        // then
        expect(next[0].reactions).toEqual([]);
    });

    it("does not invent a reaction from an authoritative count when the emoji is unknown", () => {
        // given
        const current = [
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🍰", count: 1, viewer_reacted: false, display_names: ["Ange"] }],
            }),
        ];

        // when
        const next = applyReaction(current, makeReaction({ count: 3 }), "u1", -1);

        // then
        expect(next[0].reactions).toEqual([{ emoji: "🍰", count: 1, viewer_reacted: false, display_names: ["Ange"] }]);
    });

    it("drops the group when the authoritative count says nobody is left", () => {
        // given
        const current = [
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🌹", count: 5, viewer_reacted: false, display_names: ["Ange"] }],
            }),
        ];

        // when
        const next = applyReaction(current, makeReaction({ count: 0 }), "u1", -1);

        // then
        expect(next[0].reactions).toEqual([]);
    });
});
