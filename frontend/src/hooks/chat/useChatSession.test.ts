import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeChatMessage, makeUser } from "../../test-utils/fixtures";
import { emitRealtimeEvent } from "../../test-utils/ws";
import type { ChatMessage, ChatMessageListResponse } from "../../types/api";
import { useChatSession, type UseChatSessionOptions } from "./useChatSession";

const mocks = vi.hoisted(() => ({
    fetchRoomMessages: vi.fn(),
    fetchRoomMessagesBefore: vi.fn(),
    markReadDebounced: vi.fn(),
    deleteMessage: vi.fn(),
    editMessage: vi.fn(),
    playMessageSound: vi.fn(),
    playRemoteAudio: vi.fn(),
}));

vi.mock("../queries/chat", () => ({
    fetchRoomMessages: mocks.fetchRoomMessages,
    fetchRoomMessagesBefore: mocks.fetchRoomMessagesBefore,
}));

vi.mock("../mutations/chat", () => ({
    useMarkChatRoomRead: () => ({ mutate: vi.fn(), mutateAsync: vi.fn() }),
    useDeleteChatMessage: () => ({ mutateAsync: mocks.deleteMessage }),
    useEditChatMessage: () => ({ mutateAsync: mocks.editMessage }),
}));

vi.mock("./useDebouncedMarkChatRoomRead", () => ({
    useDebouncedMarkChatRoomRead: () => mocks.markReadDebounced,
}));

vi.mock("../../platform/sound", () => ({
    playMessageSound: mocks.playMessageSound,
    playRemoteAudio: mocks.playRemoteAudio,
}));

const viewer = makeUser({ id: "viewer-1", username: "battler", display_name: "Battler" });

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return makeChatMessage({
        sender: { id: "sender-1", username: "beatrice", display_name: "Beatrice" },
        created_at: "2026-08-01T10:05:00Z",
        ...overrides,
    });
}

function historyOf(messages: ChatMessage[], total = messages.length): ChatMessageListResponse {
    return { messages, total, limit: 50, offset: 0 };
}

function setDocumentState(visibility: DocumentVisibilityState, focused: boolean): void {
    Object.defineProperty(document, "visibilityState", { configurable: true, get: () => visibility });
    Object.defineProperty(document, "hasFocus", { configurable: true, value: () => focused });
}

async function openSession(overrides: Partial<UseChatSessionOptions> = {}) {
    const initialProps: UseChatSessionOptions = { roomId: "room-1", user: viewer, ...overrides };
    const rendered = renderHook((props: UseChatSessionOptions) => useChatSession(props), { initialProps });

    await waitFor(() => {
        expect(rendered.result.current.status).not.toBe("idle");
    });
    await act(async () => {});

    return rendered;
}

beforeEach(() => {
    setDocumentState("visible", true);
    mocks.fetchRoomMessages.mockResolvedValue(historyOf([]));
    mocks.fetchRoomMessagesBefore.mockResolvedValue(historyOf([]));
});

describe("useChatSession live messages", () => {
    it("appends a message for its own room and asks for a debounced read", async () => {
        // given
        const { result } = await openSession();

        // when
        emitRealtimeEvent({ type: "chat_message", data: makeMessage({ id: "m-live" }) });

        // then
        expect(result.current.messages.map(m => m.id)).toEqual(["m-live"]);
        expect(mocks.markReadDebounced).toHaveBeenCalledExactlyOnceWith("room-1");
    });

    it("appends a message for the active room and scrolls to it", async () => {
        // given
        const { result } = await openSession();
        const container = document.createElement("div");
        Object.defineProperty(container, "scrollTo", { configurable: true, writable: true, value: vi.fn() });
        act(() => {
            result.current.scroll.containerRef(container);
        });

        // when
        emitRealtimeEvent({ type: "chat_message", data: makeMessage({ id: "m-live" }) });

        // then
        expect(result.current.messages.map(m => m.id)).toEqual(["m-live"]);
        await waitFor(() => {
            expect(container.scrollTo).toHaveBeenCalled();
        });
    });

    it("leaves a message for another room alone", async () => {
        // given
        const { result } = await openSession();

        // when
        emitRealtimeEvent({ type: "chat_message", data: makeMessage({ id: "m-elsewhere", room_id: "room-2" }) });

        // then
        expect(result.current.messages).toEqual([]);
        expect(mocks.markReadDebounced).not.toHaveBeenCalled();
    });

    it("does not append a message it already holds", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue(historyOf([makeMessage({ id: "m1" })]));
        const { result } = await openSession();

        // when
        emitRealtimeEvent({ type: "chat_message", data: makeMessage({ id: "m1", body: "resent" }) });

        // then
        expect(result.current.messages).toHaveLength(1);
        expect(result.current.messages[0].body).toBe("the golden truth");
    });

    it("clears the sender from the typing set as soon as their message lands", async () => {
        // given a room where someone is mid-sentence
        const { result } = await openSession();
        emitRealtimeEvent({ type: "typing", data: { room_id: "room-1", user_id: "sender-1" } });
        expect(result.current.typing.userIds).toEqual(["sender-1"]);

        // when they send it
        emitRealtimeEvent({ type: "chat_message", data: makeMessage({ id: "m-live" }) });

        // then the indicator goes immediately, rather than waiting out the five second expiry
        expect(result.current.typing.userIds).toEqual([]);
    });

    it("keeps a typing user listed while somebody else posts", async () => {
        // given
        const { result } = await openSession();
        emitRealtimeEvent({ type: "typing", data: { room_id: "room-1", user_id: "sender-1" } });

        // when
        emitRealtimeEvent({
            type: "chat_message",
            data: makeMessage({ id: "m-live", sender: { id: "sender-2", username: "lambda", display_name: "Lambda" } }),
        });

        // then
        expect(result.current.typing.userIds).toEqual(["sender-1"]);
    });

    it("reports typing from the active room", async () => {
        // given
        const { result } = await openSession();

        // when
        emitRealtimeEvent({ type: "typing", data: { room_id: "room-1", user_id: "sender-1" } });

        // then
        expect(result.current.typing.userIds).toEqual(["sender-1"]);
    });

    it("ignores typing from another room", async () => {
        // given
        const { result } = await openSession();

        // when
        emitRealtimeEvent({ type: "typing", data: { room_id: "room-2", user_id: "sender-1" } });

        // then
        expect(result.current.typing.userIds).toEqual([]);
    });
});

describe("useChatSession message patches", () => {
    it("applies an edit broadcast for its own room", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue(historyOf([makeMessage({ id: "m1", body: "hello" })]));
        const { result } = await openSession();

        // when
        emitRealtimeEvent({ type: "chat_message_edited", data: makeMessage({ id: "m1", body: "goodbye" }) });

        // then
        expect(result.current.messages[0].body).toBe("goodbye");
    });

    it("ignores an edit broadcast for another room", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue(historyOf([makeMessage({ id: "m1", body: "hello" })]));
        const { result } = await openSession();

        // when
        emitRealtimeEvent({
            type: "chat_message_edited",
            data: makeMessage({ id: "m1", room_id: "room-2", body: "goodbye" }),
        });

        // then
        expect(result.current.messages[0].body).toBe("hello");
    });

    it("removes a message deleted in its own room", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue(
            historyOf([makeMessage({ id: "m1" }), makeMessage({ id: "m2", created_at: "2026-08-01T10:06:00Z" })]),
        );
        const { result } = await openSession();

        // when
        emitRealtimeEvent({ type: "chat_message_deleted", data: { room_id: "room-1", message_id: "m1" } });

        // then
        expect(result.current.messages.map(m => m.id)).toEqual(["m2"]);
    });

    it("keeps a message deleted in another room", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue(historyOf([makeMessage({ id: "m1" })]));
        const { result } = await openSession();

        // when
        emitRealtimeEvent({ type: "chat_message_deleted", data: { room_id: "room-2", message_id: "m1" } });

        // then
        expect(result.current.messages.map(m => m.id)).toEqual(["m1"]);
    });

    it("declines a message type it does not own", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue(historyOf([makeMessage({ id: "m1" })]));
        const { result } = await openSession();

        // when
        emitRealtimeEvent({
            type: "chat_read_receipt",
            data: { room_id: "room-1", user_id: "sender-1", read_at: "2026-08-01T10:06:00Z" },
        });

        // then
        expect(result.current.messages.map(m => m.id)).toEqual(["m1"]);
        expect(result.current.typing.userIds).toEqual([]);
    });

    it("marks a message pinned and unpinned in its own room", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue(historyOf([makeMessage({ id: "m1" })]));
        const { result } = await openSession();

        // when
        emitRealtimeEvent({
            type: "chat_message_pinned",
            data: { room_id: "room-1", message_id: "m1", pinned_at: "2026-08-01T11:00:00Z", pinned_by: "viewer-1" },
        });

        // then
        expect(result.current.messages[0].pinned).toBe(true);

        // when
        emitRealtimeEvent({ type: "chat_message_unpinned", data: { room_id: "room-1", message_id: "m1" } });

        // then
        expect(result.current.messages[0].pinned).toBe(false);
    });

    it("folds a reaction onto the message it was left on", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue(historyOf([makeMessage({ id: "m1" })]));
        const { result } = await openSession();

        // when
        emitRealtimeEvent({
            type: "chat_reaction_added",
            data: {
                room_id: "room-1",
                message_id: "m1",
                emoji: "\u{1F339}",
                user_id: "viewer-1",
                display_name: "Battler",
                count: 1,
            },
        });

        // then
        expect(result.current.messages[0].reactions).toEqual([
            { emoji: "\u{1F339}", count: 1, viewer_reacted: true, display_names: ["Battler"] },
        ]);

        // when
        emitRealtimeEvent({
            type: "chat_reaction_removed",
            data: {
                room_id: "room-1",
                message_id: "m1",
                emoji: "\u{1F339}",
                user_id: "viewer-1",
                display_name: "Battler",
                count: 0,
            },
        });

        // then
        expect(result.current.messages[0].reactions).toEqual([]);
    });

    it("adds a reaction that arrives over the socket", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue(historyOf([makeMessage({ id: "m1" })]));
        const { result } = await openSession();

        // when
        emitRealtimeEvent({
            type: "chat_reaction_added",
            data: {
                room_id: "room-1",
                message_id: "m1",
                emoji: "\u{1F339}",
                user_id: "sender-1",
                display_name: "Beatrice",
            },
        });

        // then
        expect(result.current.messages[0].reactions).toEqual([
            { emoji: "\u{1F339}", count: 1, viewer_reacted: false, display_names: ["Beatrice"] },
        ]);
    });

    it("ignores a reaction aimed at another room", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue(historyOf([makeMessage({ id: "m1" })]));
        const { result } = await openSession();

        // when
        emitRealtimeEvent({
            type: "chat_reaction_added",
            data: {
                room_id: "room-2",
                message_id: "m1",
                emoji: "\u{1F339}",
                user_id: "sender-1",
                display_name: "Beatrice",
            },
        });

        // then
        expect(result.current.messages[0].reactions).toEqual([]);
    });

    it("restamps a sender's nickname across the messages they wrote", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue(historyOf([makeMessage({ id: "m1" })]));
        const { result } = await openSession();

        // when
        emitRealtimeEvent({
            type: "chat_member_updated",
            data: {
                room_id: "room-1",
                user_id: "sender-1",
                nickname: "The Golden Witch",
                display_name: "Beatrice",
                username: "beatrice",
                member_avatar_url: "",
                nickname_locked: false,
                timeout_until: "",
                timeout_set_by_staff: false,
            },
        });

        // then
        expect(result.current.messages[0].sender_nickname).toBe("The Golden Witch");
    });

    it("restamps a sender's site role across the messages they wrote", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue(historyOf([makeMessage({ id: "m1" })]));
        const { result } = await openSession();

        // when
        emitRealtimeEvent({ type: "role_changed", data: { user_id: "sender-1", role: "moderator" } });

        // then
        expect(result.current.messages[0].sender.role).toBe("moderator");
    });

    it("leaves the transcript alone for a role change with no user", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue(historyOf([makeMessage({ id: "m1" })]));
        const { result } = await openSession();
        const before = result.current.messages;

        // when
        emitRealtimeEvent({ type: "role_changed", data: { user_id: "", role: "moderator" } });

        // then
        expect(result.current.messages).toBe(before);
    });
});

describe("useChatSession audio", () => {
    it("plays remote audio for its own room at the requested volume", async () => {
        // given
        await openSession();

        // when
        emitRealtimeEvent({ type: "chat_audio", data: { room_id: "room-1", url: "/uploads/a.mp3", volume: 0.2 } });

        // then
        expect(mocks.playRemoteAudio).toHaveBeenCalledExactlyOnceWith("/uploads/a.mp3", 0.2);
    });

    it("plays remote audio at half volume when none is given", async () => {
        // given
        await openSession();

        // when
        emitRealtimeEvent({ type: "chat_audio", data: { room_id: "room-1", url: "/uploads/a.mp3" } });

        // then
        expect(mocks.playRemoteAudio).toHaveBeenCalledExactlyOnceWith("/uploads/a.mp3", 0.5);
    });

    it("ignores remote audio for another room", async () => {
        // given
        await openSession();

        // when
        emitRealtimeEvent({ type: "chat_audio", data: { room_id: "room-2", url: "/uploads/a.mp3", volume: 0.2 } });

        // then
        expect(mocks.playRemoteAudio).not.toHaveBeenCalled();
    });

    it("plays nothing for an audio event with no url", async () => {
        // given
        await openSession();

        // when
        emitRealtimeEvent({ type: "chat_audio", data: { room_id: "room-1", volume: 0.2 } });

        // then
        expect(mocks.playRemoteAudio).not.toHaveBeenCalled();
    });

    it("stays silent by default when somebody else posts to a hidden tab", async () => {
        // given a caller that asked for no message sound
        setDocumentState("hidden", false);
        await openSession();

        // when
        emitRealtimeEvent({ type: "chat_message", data: makeMessage({ id: "m-live" }) });

        // then
        expect(mocks.playMessageSound).not.toHaveBeenCalled();
    });

    it("plays a message sound for somebody else's message on a hidden tab", async () => {
        // given
        setDocumentState("hidden", false);
        await openSession({ sound: { enabled: true, muted: false } });

        // when
        emitRealtimeEvent({ type: "chat_message", data: makeMessage({ id: "m-live" }) });

        // then
        expect(mocks.playMessageSound).toHaveBeenCalledOnce();
    });

    it("stays silent when message sounds are switched off", async () => {
        // given
        setDocumentState("hidden", false);
        await openSession({ sound: { enabled: false, muted: false } });

        // when
        emitRealtimeEvent({ type: "chat_message", data: makeMessage({ id: "m-live" }) });

        // then
        expect(mocks.playMessageSound).not.toHaveBeenCalled();
    });

    it("stays silent while the room is muted", async () => {
        // given
        setDocumentState("hidden", false);
        await openSession({ sound: { enabled: true, muted: true } });

        // when
        emitRealtimeEvent({ type: "chat_message", data: makeMessage({ id: "m-live" }) });

        // then
        expect(mocks.playMessageSound).not.toHaveBeenCalled();
    });

    it("stays silent for the viewer's own message", async () => {
        // given
        setDocumentState("hidden", false);
        await openSession({ sound: { enabled: true, muted: false } });

        // when
        emitRealtimeEvent({
            type: "chat_message",
            data: makeMessage({
                id: "m-live",
                sender: { id: "viewer-1", username: "battler", display_name: "Battler" },
            }),
        });

        // then
        expect(mocks.playMessageSound).not.toHaveBeenCalled();
    });

    it("stays silent while the tab is in view", async () => {
        // given
        setDocumentState("visible", true);
        await openSession({ sound: { enabled: true, muted: false } });

        // when
        emitRealtimeEvent({ type: "chat_message", data: makeMessage({ id: "m-live" }) });

        // then
        expect(mocks.playMessageSound).not.toHaveBeenCalled();
    });
});

describe("useChatSession ephemeral rooms", () => {
    it("stays idle until a room resolves", () => {
        // given a stream chat panel that has not joined yet

        // when
        const { result } = renderHook(() => useChatSession({ roomId: undefined, user: viewer }));

        // then
        expect(result.current.status).toBe("idle");
        expect(mocks.fetchRoomMessages).not.toHaveBeenCalled();
    });

    it("goes live the moment a room resolves", async () => {
        // given
        const { result } = await openSession();

        // then
        expect(result.current.status).toBe("live");
    });

    it("ends the session when its own room is deleted underneath it", async () => {
        // given a watch party room the viewer is still pointed at
        const { result } = await openSession();

        // when the host ends the party and the room cascade deletes
        emitRealtimeEvent({ type: "chat_room_deleted", data: { room_id: "room-1" } });

        // then
        expect(result.current.status).toBe("ended");
    });

    it("stays live when a different room is deleted", async () => {
        // given
        const { result } = await openSession();

        // when
        emitRealtimeEvent({ type: "chat_room_deleted", data: { room_id: "room-2" } });

        // then
        expect(result.current.status).toBe("live");
    });

    it("keeps the transcript of a room that was deleted underneath it", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue(historyOf([makeMessage({ id: "m1" })]));
        const { result } = await openSession();

        // when
        emitRealtimeEvent({ type: "chat_room_deleted", data: { room_id: "room-1" } });

        // then the viewer can still read what was said, but nothing more arrives
        expect(result.current.messages.map(m => m.id)).toEqual(["m1"]);
    });

    it("takes no further messages once the session has ended", async () => {
        // given a stream whose room has gone
        const { result } = await openSession();
        emitRealtimeEvent({ type: "chat_room_deleted", data: { room_id: "room-1" } });

        // when a late broadcast arrives for that room
        emitRealtimeEvent({ type: "chat_message", data: makeMessage({ id: "m-late" }) });

        // then
        expect(result.current.messages).toEqual([]);
        expect(mocks.markReadDebounced).not.toHaveBeenCalled();
    });

    it("stops offering older pages once the session has ended", async () => {
        // given a room with more history behind it
        mocks.fetchRoomMessages.mockResolvedValue(historyOf([makeMessage({ id: "m1" })], 10));
        const { result } = await openSession();
        expect(result.current.history.hasMore).toBe(true);

        // when
        emitRealtimeEvent({ type: "chat_room_deleted", data: { room_id: "room-1" } });

        // then the panel stops inviting a page request against a room that no longer exists
        expect(result.current.history.hasMore).toBe(false);

        // when
        act(() => {
            result.current.scroll.onScroll();
        });

        // then
        expect(mocks.fetchRoomMessagesBefore).not.toHaveBeenCalled();
    });

    it("ends the session when the caller stops resolving a room", async () => {
        // given a live stream chat
        const { result, rerender } = await openSession();

        // when the stream goes offline and the panel withdraws the room
        rerender({ roomId: undefined, user: viewer });

        // then
        expect(result.current.status).toBe("ended");
    });

    it("releases the transcript when the caller stops resolving a room", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue(historyOf([makeMessage({ id: "m1" })]));
        const { result, rerender } = await openSession();

        // when
        rerender({ roomId: undefined, user: viewer });

        // then the transcript belongs to the room, and there is no room
        expect(result.current.messages).toEqual([]);
    });

    it("issues no read and no fetch for a room the caller has withdrawn", async () => {
        // given
        const { rerender } = await openSession();
        mocks.fetchRoomMessages.mockClear();

        // when
        rerender({ roomId: undefined, user: viewer });
        emitRealtimeEvent({ type: "chat_message", data: makeMessage({ id: "m-late" }) });
        await act(async () => {});

        // then
        expect(mocks.fetchRoomMessages).not.toHaveBeenCalled();
        expect(mocks.markReadDebounced).not.toHaveBeenCalled();
    });

    it("starts a fresh live session when the next room resolves", async () => {
        // given a stream that ended
        const { result, rerender } = await openSession();
        emitRealtimeEvent({ type: "chat_room_deleted", data: { room_id: "room-1" } });
        expect(result.current.status).toBe("ended");

        // when the viewer opens another room
        mocks.fetchRoomMessages.mockResolvedValue(historyOf([makeMessage({ id: "m9", room_id: "room-2" })]));
        rerender({ roomId: "room-2", user: viewer });
        await act(async () => {});

        // then
        expect(result.current.status).toBe("live");
        expect(result.current.messages.map(m => m.id)).toEqual(["m9"]);
    });

    it("comes back to life when the same room resolves again", async () => {
        // given a stream that went offline
        const { result, rerender } = await openSession();
        rerender({ roomId: undefined, user: viewer });
        expect(result.current.status).toBe("ended");

        // when the broadcaster comes back on the same stream id
        rerender({ roomId: "room-1", user: viewer });
        await act(async () => {});

        // then
        expect(result.current.status).toBe("live");
    });
});

describe("useChatSession editing", () => {
    it("holds the message the viewer is editing", async () => {
        // given
        const { result } = await openSession();

        // when
        act(() => {
            result.current.editing.start(makeMessage({ id: "m1" }));
        });

        // then
        expect(result.current.editing.messageId).toBe("m1");

        // when
        act(() => {
            result.current.editing.cancel();
        });

        // then
        expect(result.current.editing.messageId).toBeNull();
    });

    it("keeps the whole backlog loaded while a message is being edited", async () => {
        // given a windowed panel holding two messages
        mocks.fetchRoomMessages.mockResolvedValue(
            historyOf([makeMessage({ id: "m1" }), makeMessage({ id: "m2", created_at: "2026-08-01T10:06:00Z" })]),
        );
        const { result } = await openSession({ maxMessages: 1 });

        // when the viewer starts editing and a new message lands
        act(() => {
            result.current.editing.start(makeMessage({ id: "m1" }));
        });
        emitRealtimeEvent({
            type: "chat_message",
            data: makeMessage({ id: "m3", created_at: "2026-08-01T10:07:00Z" }),
        });

        // then the window is not trimmed out from under the editor
        expect(result.current.messages.map(m => m.id)).toEqual(["m1", "m2", "m3"]);
    });
});
