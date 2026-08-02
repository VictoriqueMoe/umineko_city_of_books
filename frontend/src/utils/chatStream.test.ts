import type { Dispatch, SetStateAction } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { markChatRoomRead } from "../api/endpoints";
import type { ChatMessage, ChatRoomMember, PostMedia, WSMessage } from "../types/api";
import {
    applyChatMemberUpdate,
    applyChatMessageEdited,
    applyChatMessagePinned,
    applyChatMessageUnpinned,
    applyLocalMemberChange,
    applyReactionAdded,
    applyReactionRemoved,
    applySharedChatWSBranch,
    handleIncomingChatMessage,
    maybePlayChatMessageSound,
    type ChatMemberUpdatedPayload,
    type ChatReactionPayload,
    type MaybePlayChatSoundOpts,
    type SharedWSBranchOptions,
} from "./chatStream";
import { playMessageSound, playRemoteAudio } from "./sound";

vi.mock("../api/endpoints", () => ({
    markChatRoomRead: vi.fn(() => Promise.resolve()),
}));

vi.mock("./sound", () => ({
    playMessageSound: vi.fn(),
    playRemoteAudio: vi.fn(),
}));

interface StateBox<T> {
    set: Dispatch<SetStateAction<T>>;
    value: () => T;
}

function stateBox<T>(initial: T): StateBox<T> {
    let current = initial;
    const set: Dispatch<SetStateAction<T>> = update => {
        current = typeof update === "function" ? (update as (prev: T) => T)(current) : update;
    };

    return { set, value: () => current };
}

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return {
        id: "m1",
        room_id: "room-1",
        sender: { id: "u1", username: "beatrice", display_name: "Beatrice" },
        body: "hello",
        is_system: false,
        created_at: "2026-01-01T00:00:00Z",
        pinned: false,
        reactions: [],
        ...overrides,
    };
}

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

function makeSoundOpts(overrides: Partial<MaybePlayChatSoundOpts> = {}): MaybePlayChatSoundOpts {
    return {
        senderId: "u2",
        currentUserId: "u1",
        roomMuted: false,
        enabled: true,
        ...overrides,
    };
}

function makeMedia(id: number): PostMedia {
    return { id, media_url: `/uploads/${id}.png`, media_type: "image", sort_order: 0 };
}

function setDocumentState(visibility: DocumentVisibilityState, focused: boolean): void {
    Object.defineProperty(document, "visibilityState", { configurable: true, get: () => visibility });
    Object.defineProperty(document, "hasFocus", { configurable: true, value: () => focused });
}

beforeEach(() => {
    setDocumentState("visible", true);
});

describe("handleIncomingChatMessage", () => {
    it("ignores a message addressed to another room", () => {
        // given
        const messages = stateBox<ChatMessage[]>([]);
        const scrollToBottom = vi.fn();

        // when
        const handled = handleIncomingChatMessage(
            makeMessage({ id: "m2", room_id: "other-room" }),
            "room-1",
            messages.set,
            scrollToBottom,
        );

        // then
        expect(handled).toBe(false);
        expect(messages.value()).toEqual([]);
        expect(scrollToBottom).not.toHaveBeenCalled();
        expect(markChatRoomRead).not.toHaveBeenCalled();
    });

    it("appends a message for the active room and scrolls to it", () => {
        // given
        const existing = makeMessage({ id: "m1" });
        const messages = stateBox([existing]);
        const scrollToBottom = vi.fn();

        // when
        const handled = handleIncomingChatMessage(makeMessage({ id: "m2" }), "room-1", messages.set, scrollToBottom);

        // then
        expect(handled).toBe(true);
        expect(messages.value().map(m => m.id)).toEqual(["m1", "m2"]);
        expect(scrollToBottom).toHaveBeenCalledOnce();
    });

    it("does not append a message it already holds", () => {
        // given
        const messages = stateBox([makeMessage({ id: "m1" })]);
        const before = messages.value();

        // when
        const handled = handleIncomingChatMessage(
            makeMessage({ id: "m1", body: "resent" }),
            "room-1",
            messages.set,
            () => {},
        );

        // then
        expect(handled).toBe(true);
        expect(messages.value()).toBe(before);
    });

    it("marks the room read when the tab is visible and focused", () => {
        // given
        setDocumentState("visible", true);
        const messages = stateBox<ChatMessage[]>([]);

        // when
        handleIncomingChatMessage(makeMessage({ id: "m2" }), "room-1", messages.set, () => {});

        // then
        expect(markChatRoomRead).toHaveBeenCalledWith("room-1");
    });

    it("does not mark the room read when the window has lost focus", () => {
        // given
        setDocumentState("visible", false);
        const messages = stateBox<ChatMessage[]>([]);

        // when
        handleIncomingChatMessage(makeMessage({ id: "m2" }), "room-1", messages.set, () => {});

        // then
        expect(markChatRoomRead).not.toHaveBeenCalled();
    });

    it("does not mark the room read when the tab is hidden", () => {
        // given
        setDocumentState("hidden", true);
        const messages = stateBox<ChatMessage[]>([]);

        // when
        handleIncomingChatMessage(makeMessage({ id: "m2" }), "room-1", messages.set, () => {});

        // then
        expect(markChatRoomRead).not.toHaveBeenCalled();
    });
});

describe("maybePlayChatMessageSound", () => {
    it("stays silent when message sounds are switched off", () => {
        // given
        setDocumentState("hidden", false);

        // when
        maybePlayChatMessageSound(makeSoundOpts({ enabled: false }));

        // then
        expect(playMessageSound).not.toHaveBeenCalled();
    });

    it("stays silent when the room is muted", () => {
        // given
        setDocumentState("hidden", false);

        // when
        maybePlayChatMessageSound(makeSoundOpts({ roomMuted: true }));

        // then
        expect(playMessageSound).not.toHaveBeenCalled();
    });

    it("stays silent for your own message", () => {
        // given
        setDocumentState("hidden", false);

        // when
        maybePlayChatMessageSound(makeSoundOpts({ senderId: "u1", currentUserId: "u1" }));

        // then
        expect(playMessageSound).not.toHaveBeenCalled();
    });

    it("stays silent while the tab is in view", () => {
        // given
        setDocumentState("visible", true);

        // when
        maybePlayChatMessageSound(makeSoundOpts());

        // then
        expect(playMessageSound).not.toHaveBeenCalled();
    });

    it("plays when a hidden tab receives somebody else's message", () => {
        // given
        setDocumentState("hidden", false);

        // when
        maybePlayChatMessageSound(makeSoundOpts());

        // then
        expect(playMessageSound).toHaveBeenCalledOnce();
    });
});

describe("applyLocalMemberChange", () => {
    it("swaps in the changed member and leaves the rest of the roster alone", () => {
        // given
        const other = makeMember({ user: { id: "u2", username: "ange", display_name: "Ange" } });
        const members = stateBox([makeMember(), other]);
        const messages = stateBox<ChatMessage[]>([]);

        // when
        applyLocalMemberChange(makeMember({ nickname: "Golden Witch" }), members.set, messages.set);

        // then
        expect(members.value()[0].nickname).toBe("Golden Witch");
        expect(members.value()[1]).toBe(other);
    });

    it("restamps that member's messages with the new nickname and avatar", () => {
        // given
        const messages = stateBox([makeMessage()]);
        const members = stateBox([makeMember()]);

        // when
        applyLocalMemberChange(
            makeMember({ nickname: "Golden Witch", member_avatar_url: "/uploads/beato.png" }),
            members.set,
            messages.set,
        );

        // then
        expect(messages.value()[0].sender_nickname).toBe("Golden Witch");
        expect(messages.value()[0].sender_member_avatar_url).toBe("/uploads/beato.png");
    });

    it("clears the stamped nickname and avatar when the member has neither", () => {
        // given
        const messages = stateBox([
            makeMessage({ sender_nickname: "Golden Witch", sender_member_avatar_url: "/uploads/beato.png" }),
        ]);
        const members = stateBox([makeMember()]);

        // when
        applyLocalMemberChange(makeMember(), members.set, messages.set);

        // then
        expect(messages.value()[0].sender_nickname).toBeUndefined();
        expect(messages.value()[0].sender_member_avatar_url).toBeUndefined();
    });

    it("leaves messages from other senders untouched", () => {
        // given
        const foreign = makeMessage({ id: "m2", sender: { id: "u2", username: "ange", display_name: "Ange" } });
        const messages = stateBox([foreign]);
        const members = stateBox([makeMember()]);

        // when
        applyLocalMemberChange(makeMember({ nickname: "Golden Witch" }), members.set, messages.set);

        // then
        expect(messages.value()[0]).toBe(foreign);
    });
});

describe("applyChatMemberUpdate", () => {
    it("writes the broadcast nickname, avatar and lock onto the matching member", () => {
        // given
        const members = stateBox([makeMember()]);
        const messages = stateBox<ChatMessage[]>([]);

        // when
        applyChatMemberUpdate(
            makeMemberUpdate({
                nickname: "Golden Witch",
                member_avatar_url: "/uploads/beato.png",
                nickname_locked: true,
            }),
            members.set,
            messages.set,
        );

        // then
        expect(members.value()[0].nickname).toBe("Golden Witch");
        expect(members.value()[0].member_avatar_url).toBe("/uploads/beato.png");
        expect(members.value()[0].nickname_locked).toBe(true);
    });

    it("turns an empty timeout into no timeout at all", () => {
        // given
        const members = stateBox([makeMember({ timeout_until: "2026-01-01T01:00:00Z", timeout_set_by_staff: true })]);
        const messages = stateBox<ChatMessage[]>([]);

        // when
        applyChatMemberUpdate(makeMemberUpdate(), members.set, messages.set);

        // then
        expect(members.value()[0].timeout_until).toBeUndefined();
        expect(members.value()[0].timeout_set_by_staff).toBe(false);
    });

    it("keeps a live timeout and who set it", () => {
        // given
        const members = stateBox([makeMember()]);
        const messages = stateBox<ChatMessage[]>([]);

        // when
        applyChatMemberUpdate(
            makeMemberUpdate({ timeout_until: "2026-01-01T01:00:00Z", timeout_set_by_staff: true }),
            members.set,
            messages.set,
        );

        // then
        expect(members.value()[0].timeout_until).toBe("2026-01-01T01:00:00Z");
        expect(members.value()[0].timeout_set_by_staff).toBe(true);
    });

    it("leaves members with a different id alone", () => {
        // given
        const other = makeMember({ user: { id: "u2", username: "ange", display_name: "Ange" } });
        const members = stateBox([other]);
        const messages = stateBox<ChatMessage[]>([]);

        // when
        applyChatMemberUpdate(makeMemberUpdate({ nickname: "Golden Witch" }), members.set, messages.set);

        // then
        expect(members.value()[0]).toBe(other);
    });

    it("restamps the sender fields on that member's messages", () => {
        // given
        const members = stateBox([makeMember()]);
        const messages = stateBox([makeMessage()]);

        // when
        applyChatMemberUpdate(
            makeMemberUpdate({ nickname: "Golden Witch", member_avatar_url: "/uploads/beato.png" }),
            members.set,
            messages.set,
        );

        // then
        expect(messages.value()[0].sender_nickname).toBe("Golden Witch");
        expect(messages.value()[0].sender_member_avatar_url).toBe("/uploads/beato.png");
    });

    it("renames the reply preview with the new nickname", () => {
        // given
        const members = stateBox([makeMember()]);
        const messages = stateBox([
            makeMessage({
                id: "m2",
                sender: { id: "u2", username: "ange", display_name: "Ange" },
                reply_to: { id: "m1", sender_id: "u1", sender_name: "Beatrice", body_preview: "hello" },
            }),
        ]);

        // when
        applyChatMemberUpdate(makeMemberUpdate({ nickname: "Golden Witch" }), members.set, messages.set);

        // then
        expect(messages.value()[0].reply_to?.sender_name).toBe("Golden Witch");
        expect(messages.value()[0].sender_nickname).toBeUndefined();
    });

    it("falls back to the display name in the reply preview when there is no nickname", () => {
        // given
        const members = stateBox([makeMember()]);
        const messages = stateBox([
            makeMessage({
                id: "m2",
                sender: { id: "u2", username: "ange", display_name: "Ange" },
                reply_to: { id: "m1", sender_id: "u1", sender_name: "stale", body_preview: "hello" },
            }),
        ]);

        // when
        applyChatMemberUpdate(makeMemberUpdate({ nickname: "   " }), members.set, messages.set);

        // then
        expect(messages.value()[0].reply_to?.sender_name).toBe("Beatrice");
    });

    it("falls back to the username in the reply preview when neither name is set", () => {
        // given
        const members = stateBox([makeMember()]);
        const messages = stateBox([
            makeMessage({
                id: "m2",
                sender: { id: "u2", username: "ange", display_name: "Ange" },
                reply_to: { id: "m1", sender_id: "u1", sender_name: "stale", body_preview: "hello" },
            }),
        ]);

        // when
        applyChatMemberUpdate(makeMemberUpdate({ nickname: "", display_name: "  " }), members.set, messages.set);

        // then
        expect(messages.value()[0].reply_to?.sender_name).toBe("beatrice");
    });

    it("leaves messages that neither sent nor replied to the member untouched", () => {
        // given
        const foreign = makeMessage({ id: "m2", sender: { id: "u2", username: "ange", display_name: "Ange" } });
        const members = stateBox([makeMember()]);
        const messages = stateBox([foreign]);

        // when
        applyChatMemberUpdate(makeMemberUpdate({ nickname: "Golden Witch" }), members.set, messages.set);

        // then
        expect(messages.value()[0]).toBe(foreign);
    });
});

describe("applyChatMessagePinned", () => {
    it("pins only the target message and records who pinned it", () => {
        // given
        const other = makeMessage({ id: "m2" });
        const messages = stateBox([makeMessage({ id: "m1" }), other]);

        // when
        applyChatMessagePinned(
            { room_id: "room-1", message_id: "m1", pinned_at: "2026-01-02T00:00:00Z", pinned_by: "u9" },
            messages.set,
        );

        // then
        expect(messages.value()[0].pinned).toBe(true);
        expect(messages.value()[0].pinned_at).toBe("2026-01-02T00:00:00Z");
        expect(messages.value()[0].pinned_by).toBe("u9");
        expect(messages.value()[1]).toBe(other);
    });
});

describe("applyChatMessageUnpinned", () => {
    it("unpins the message and forgets the pin metadata", () => {
        // given
        const messages = stateBox([
            makeMessage({ id: "m1", pinned: true, pinned_at: "2026-01-02T00:00:00Z", pinned_by: "u9" }),
        ]);

        // when
        applyChatMessageUnpinned({ room_id: "room-1", message_id: "m1" }, messages.set);

        // then
        expect(messages.value()[0].pinned).toBe(false);
        expect(messages.value()[0].pinned_at).toBeUndefined();
        expect(messages.value()[0].pinned_by).toBeUndefined();
    });

    it("ignores an unpin for a message it does not hold", () => {
        // given
        const pinned = makeMessage({ id: "m1", pinned: true });
        const messages = stateBox([pinned]);

        // when
        applyChatMessageUnpinned({ room_id: "room-1", message_id: "m404" }, messages.set);

        // then
        expect(messages.value()[0]).toBe(pinned);
    });
});

describe("applyChatMessageEdited", () => {
    it("replaces the body and records when it was edited", () => {
        // given
        const messages = stateBox([makeMessage({ id: "m1", body: "hello" })]);

        // when
        applyChatMessageEdited(
            makeMessage({ id: "m1", body: "goodbye", edited_at: "2026-01-03T00:00:00Z" }),
            messages.set,
        );

        // then
        expect(messages.value()[0].body).toBe("goodbye");
        expect(messages.value()[0].edited_at).toBe("2026-01-03T00:00:00Z");
    });

    it("keeps the existing media when the edit carries none", () => {
        // given
        const media = [makeMedia(1)];
        const messages = stateBox([makeMessage({ id: "m1", media })]);

        // when
        applyChatMessageEdited(makeMessage({ id: "m1", body: "goodbye" }), messages.set);

        // then
        expect(messages.value()[0].media).toBe(media);
    });

    it("replaces the media when the edit carries some", () => {
        // given
        const messages = stateBox([makeMessage({ id: "m1", media: [makeMedia(1)] })]);

        // when
        applyChatMessageEdited(makeMessage({ id: "m1", media: [makeMedia(2)] }), messages.set);

        // then
        expect(messages.value()[0].media?.map(m => m.id)).toEqual([2]);
    });

    it("clears the media when the edit carries an empty list", () => {
        // given
        const messages = stateBox([makeMessage({ id: "m1", media: [makeMedia(1)] })]);

        // when
        applyChatMessageEdited(makeMessage({ id: "m1", media: [] }), messages.set);

        // then
        expect(messages.value()[0].media).toEqual([]);
    });

    it("ignores an edit for a message it does not hold", () => {
        // given
        const original = makeMessage({ id: "m1", body: "hello" });
        const messages = stateBox([original]);

        // when
        applyChatMessageEdited(makeMessage({ id: "m404", body: "goodbye" }), messages.set);

        // then
        expect(messages.value()[0]).toBe(original);
    });
});

describe("applyReactionAdded", () => {
    it("creates a group with a single reaction when the emoji is new", () => {
        // given
        const messages = stateBox([makeMessage({ id: "m1", reactions: [] })]);

        // when
        applyReactionAdded(makeReaction(), "u1", messages.set);

        // then
        expect(messages.value()[0].reactions).toEqual([
            { emoji: "🌹", count: 1, viewer_reacted: false, display_names: ["Ange"] },
        ]);
    });

    it("marks a new group as viewer reacted when you are the one reacting", () => {
        // given
        const messages = stateBox([makeMessage({ id: "m1", reactions: [] })]);

        // when
        applyReactionAdded(makeReaction({ user_id: "u1", display_name: "Beatrice" }), "u1", messages.set);

        // then
        expect(messages.value()[0].reactions[0].viewer_reacted).toBe(true);
    });

    it("omits the display name from a new group when the payload has none", () => {
        // given
        const messages = stateBox([makeMessage({ id: "m1", reactions: [] })]);

        // when
        applyReactionAdded(makeReaction({ display_name: "" }), "u1", messages.set);

        // then
        expect(messages.value()[0].reactions[0].display_names).toEqual([]);
    });

    it("increments an existing group and records the new display name", () => {
        // given
        const messages = stateBox([
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🌹", count: 1, viewer_reacted: true, display_names: ["Beatrice"] }],
            }),
        ]);

        // when
        applyReactionAdded(makeReaction(), "u1", messages.set);

        // then
        expect(messages.value()[0].reactions[0].count).toBe(2);
        expect(messages.value()[0].reactions[0].display_names).toEqual(["Beatrice", "Ange"]);
    });

    it("keeps your own flag intact when somebody else reacts", () => {
        // given
        const messages = stateBox([
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🌹", count: 1, viewer_reacted: true, display_names: ["Beatrice"] }],
            }),
        ]);

        // when
        applyReactionAdded(makeReaction(), "u1", messages.set);

        // then
        expect(messages.value()[0].reactions[0].viewer_reacted).toBe(true);
    });

    it("does not record the same display name twice", () => {
        // given
        const messages = stateBox([
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🌹", count: 1, viewer_reacted: false, display_names: ["Ange"] }],
            }),
        ]);

        // when
        applyReactionAdded(makeReaction(), "u1", messages.set);

        // then
        expect(messages.value()[0].reactions[0].display_names).toEqual(["Ange"]);
        expect(messages.value()[0].reactions[0].count).toBe(2);
    });

    it("trusts the authoritative count from the server over the local increment", () => {
        // given
        const messages = stateBox([
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🌹", count: 1, viewer_reacted: false, display_names: [] }],
            }),
        ]);

        // when
        applyReactionAdded(makeReaction({ count: 7 }), "u1", messages.set);

        // then
        expect(messages.value()[0].reactions[0].count).toBe(7);
    });

    it("ignores a brand new emoji whose authoritative count is zero", () => {
        // given
        const messages = stateBox([makeMessage({ id: "m1", reactions: [] })]);

        // when
        applyReactionAdded(makeReaction({ count: 0 }), "u1", messages.set);

        // then
        expect(messages.value()[0].reactions).toEqual([]);
    });

    it("only touches the message the reaction belongs to", () => {
        // given
        const other = makeMessage({ id: "m2" });
        const messages = stateBox([makeMessage({ id: "m1" }), other]);

        // when
        applyReactionAdded(makeReaction(), "u1", messages.set);

        // then
        expect(messages.value()[1]).toBe(other);
    });
});

describe("applyReactionRemoved", () => {
    it("decrements the count and drops the display name", () => {
        // given
        const messages = stateBox([
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🌹", count: 2, viewer_reacted: false, display_names: ["Beatrice", "Ange"] }],
            }),
        ]);

        // when
        applyReactionRemoved(makeReaction(), "u1", messages.set);

        // then
        expect(messages.value()[0].reactions[0].count).toBe(1);
        expect(messages.value()[0].reactions[0].display_names).toEqual(["Beatrice"]);
    });

    it("removes only one copy of a duplicated display name", () => {
        // given
        const messages = stateBox([
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🌹", count: 3, viewer_reacted: false, display_names: ["Ange", "Ange"] }],
            }),
        ]);

        // when
        applyReactionRemoved(makeReaction(), "u1", messages.set);

        // then
        expect(messages.value()[0].reactions[0].display_names).toEqual(["Ange"]);
    });

    it("drops the whole group when the last reaction is taken away", () => {
        // given
        const messages = stateBox([
            makeMessage({
                id: "m1",
                reactions: [
                    { emoji: "🌹", count: 1, viewer_reacted: true, display_names: ["Beatrice"] },
                    { emoji: "🍰", count: 1, viewer_reacted: false, display_names: ["Ange"] },
                ],
            }),
        ]);

        // when
        applyReactionRemoved(makeReaction({ user_id: "u1", display_name: "Beatrice" }), "u1", messages.set);

        // then
        expect(messages.value()[0].reactions.map(r => r.emoji)).toEqual(["🍰"]);
    });

    it("clears your own flag when you take your reaction back", () => {
        // given
        const messages = stateBox([
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🌹", count: 2, viewer_reacted: true, display_names: ["Beatrice", "Ange"] }],
            }),
        ]);

        // when
        applyReactionRemoved(makeReaction({ user_id: "u1", display_name: "Beatrice" }), "u1", messages.set);

        // then
        expect(messages.value()[0].reactions[0].viewer_reacted).toBe(false);
    });

    it("does not invent the reaction it was asked to remove when the message has none", () => {
        // given
        const messages = stateBox([makeMessage({ id: "m1", reactions: [] })]);

        // when
        applyReactionRemoved(makeReaction(), "u1", messages.set);

        // then
        expect(messages.value()[0].reactions).toEqual([]);
    });

    it("does not invent a reaction from an authoritative count when the emoji is unknown", () => {
        // given
        const messages = stateBox([
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🍰", count: 1, viewer_reacted: false, display_names: ["Ange"] }],
            }),
        ]);

        // when
        applyReactionRemoved(makeReaction({ count: 3 }), "u1", messages.set);

        // then
        expect(messages.value()[0].reactions).toEqual([
            { emoji: "🍰", count: 1, viewer_reacted: false, display_names: ["Ange"] },
        ]);
    });

    it("drops the group when the authoritative count says nobody is left", () => {
        // given
        const messages = stateBox([
            makeMessage({
                id: "m1",
                reactions: [{ emoji: "🌹", count: 5, viewer_reacted: false, display_names: ["Ange"] }],
            }),
        ]);

        // when
        applyReactionRemoved(makeReaction({ count: 0 }), "u1", messages.set);

        // then
        expect(messages.value()[0].reactions).toEqual([]);
    });
});

describe("applySharedChatWSBranch", () => {
    function branchOpts(overrides: Partial<SharedWSBranchOptions> = {}): SharedWSBranchOptions {
        return {
            activeRoomId: "room-1",
            setMessages: () => {},
            noteTyping: () => {},
            ...overrides,
        };
    }

    it("declines a message type it does not own", () => {
        // given
        const msg: WSMessage = { type: "notification", data: {} };

        // when
        const handled = applySharedChatWSBranch(msg, branchOpts());

        // then
        expect(handled).toBe(false);
    });

    it("deletes a message broadcast for the active room", () => {
        // given
        const messages = stateBox([makeMessage({ id: "m1" }), makeMessage({ id: "m2" })]);
        const msg: WSMessage = { type: "chat_message_deleted", data: { room_id: "room-1", message_id: "m1" } };

        // when
        const handled = applySharedChatWSBranch(msg, branchOpts({ setMessages: messages.set }));

        // then
        expect(handled).toBe(true);
        expect(messages.value().map(m => m.id)).toEqual(["m2"]);
    });

    it("claims a delete for another room without touching the messages", () => {
        // given
        const messages = stateBox([makeMessage({ id: "m1" })]);
        const before = messages.value();
        const msg: WSMessage = { type: "chat_message_deleted", data: { room_id: "other-room", message_id: "m1" } };

        // when
        const handled = applySharedChatWSBranch(msg, branchOpts({ setMessages: messages.set }));

        // then
        expect(handled).toBe(true);
        expect(messages.value()).toBe(before);
    });

    it("applies an edit broadcast for the active room", () => {
        // given
        const messages = stateBox([makeMessage({ id: "m1", body: "hello" })]);
        const msg: WSMessage = { type: "chat_message_edited", data: makeMessage({ id: "m1", body: "goodbye" }) };

        // when
        const handled = applySharedChatWSBranch(msg, branchOpts({ setMessages: messages.set }));

        // then
        expect(handled).toBe(true);
        expect(messages.value()[0].body).toBe("goodbye");
    });

    it("ignores an edit broadcast for another room", () => {
        // given
        const messages = stateBox([makeMessage({ id: "m1", body: "hello" })]);
        const msg: WSMessage = {
            type: "chat_message_edited",
            data: makeMessage({ id: "m1", room_id: "other-room", body: "goodbye" }),
        };

        // when
        applySharedChatWSBranch(msg, branchOpts({ setMessages: messages.set }));

        // then
        expect(messages.value()[0].body).toBe("hello");
    });

    it("reports typing from the active room", () => {
        // given
        const noteTyping = vi.fn();
        const msg: WSMessage = { type: "typing", data: { room_id: "room-1", user_id: "u2" } };

        // when
        const handled = applySharedChatWSBranch(msg, branchOpts({ noteTyping }));

        // then
        expect(handled).toBe(true);
        expect(noteTyping).toHaveBeenCalledWith("u2");
    });

    it("does not report typing from another room", () => {
        // given
        const noteTyping = vi.fn();
        const msg: WSMessage = { type: "typing", data: { room_id: "other-room", user_id: "u2" } };

        // when
        applySharedChatWSBranch(msg, branchOpts({ noteTyping }));

        // then
        expect(noteTyping).not.toHaveBeenCalled();
    });

    it("plays remote audio at the requested volume", () => {
        // given
        const msg: WSMessage = { type: "chat_audio", data: { room_id: "room-1", url: "/uploads/a.mp3", volume: 0.2 } };

        // when
        const handled = applySharedChatWSBranch(msg, branchOpts());

        // then
        expect(handled).toBe(true);
        expect(playRemoteAudio).toHaveBeenCalledWith("/uploads/a.mp3", 0.2);
    });

    it("plays remote audio at half volume when none is given", () => {
        // given
        const msg: WSMessage = { type: "chat_audio", data: { room_id: "room-1", url: "/uploads/a.mp3" } };

        // when
        applySharedChatWSBranch(msg, branchOpts());

        // then
        expect(playRemoteAudio).toHaveBeenCalledWith("/uploads/a.mp3", 0.5);
    });

    it("plays nothing for an audio event with no url", () => {
        // given
        const msg: WSMessage = { type: "chat_audio", data: { room_id: "room-1", url: "" } };

        // when
        const handled = applySharedChatWSBranch(msg, branchOpts());

        // then
        expect(handled).toBe(true);
        expect(playRemoteAudio).not.toHaveBeenCalled();
    });

    it("plays nothing for an audio event from another room", () => {
        // given
        const msg: WSMessage = { type: "chat_audio", data: { room_id: "other-room", url: "/uploads/a.mp3" } };

        // when
        applySharedChatWSBranch(msg, branchOpts());

        // then
        expect(playRemoteAudio).not.toHaveBeenCalled();
    });
});
