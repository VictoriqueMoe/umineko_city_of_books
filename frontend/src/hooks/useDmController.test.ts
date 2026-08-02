import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { WSMessageHandler } from "../context/notificationContextValue";
import { makeUser } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import type { ChatMessage, ChatRoom, User, UserProfile, WSMessage } from "../types/api";
import { getRoomAvatarUser, getRoomDisplayName, renderSeenLabel, useDmController } from "./useDmController";

const mocks = vi.hoisted(() => ({
    fetchUserRooms: vi.fn(),
    fetchResolveDMRoom: vi.fn(),
    fetchRoomMessages: vi.fn(),
    fetchRoomMessagesBefore: vi.fn(),
    fetchMutualFollowers: vi.fn(),
    fetchSearchUsers: vi.fn(),
    deleteChatRoom: vi.fn(),
    markChatRoomRead: vi.fn(),
    deleteChatMessage: vi.fn(),
    editChatMessage: vi.fn(),
    playMessageSound: vi.fn(),
    playRemoteAudio: vi.fn(),
}));

vi.mock("../api/queries/chat", () => ({
    fetchUserRooms: mocks.fetchUserRooms,
    fetchResolveDMRoom: mocks.fetchResolveDMRoom,
    fetchRoomMessages: mocks.fetchRoomMessages,
    fetchRoomMessagesBefore: mocks.fetchRoomMessagesBefore,
}));

vi.mock("../api/queries/misc", () => ({
    fetchMutualFollowers: mocks.fetchMutualFollowers,
    fetchSearchUsers: mocks.fetchSearchUsers,
}));

vi.mock("../api/mutations/chat", () => ({
    useDeleteChatRoom: () => ({ mutateAsync: mocks.deleteChatRoom }),
    useMarkChatRoomRead: () => ({ mutate: mocks.markChatRoomRead, mutateAsync: mocks.markChatRoomRead }),
    useDeleteChatMessage: () => ({ mutateAsync: mocks.deleteChatMessage }),
    useEditChatMessage: () => ({ mutateAsync: mocks.editChatMessage }),
}));

vi.mock("../components/chat/Voice/useVoiceChat", () => ({
    useVoiceChat: () => ({ status: "idle", room: null, participantIds: [], join: () => {}, leave: () => {} }),
}));

vi.mock("../utils/sound", () => ({
    playMessageSound: mocks.playMessageSound,
    playRemoteAudio: mocks.playRemoteAudio,
}));

const viewer = makeUser({ id: "u1", username: "beatrice", display_name: "Beatrice" });

function makeMember(overrides: Partial<User> = {}): User {
    return { id: "u2", username: "battler", display_name: "Battler", ...overrides };
}

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return {
        id: "room-1",
        name: "",
        description: "",
        type: "dm",
        is_public: false,
        is_rp: false,
        is_system: false,
        tags: [],
        viewer_muted: false,
        viewer_ghost: false,
        is_member: true,
        member_count: 2,
        hot_score: 0,
        members: [{ id: "u1", username: "beatrice", display_name: "Beatrice" }, makeMember()],
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return {
        id: "m1",
        room_id: "room-1",
        sender: makeMember(),
        body: "without love it cannot be seen",
        is_system: false,
        created_at: "2026-08-02T10:00:00Z",
        pinned: false,
        reactions: [],
        ...overrides,
    };
}

interface HarnessOptions {
    user?: UserProfile | null;
    route?: string;
    path?: string;
}

function renderDm(options: HarnessOptions = {}) {
    const handlers: WSMessageHandler[] = [];
    const unsubscribe = vi.fn();
    const addWSListener = vi.fn((handler: WSMessageHandler) => {
        handlers.push(handler);
        return unsubscribe;
    });
    const sendWSMessage = vi.fn();
    const wrapper = providerWrapper({
        user: options.user === undefined ? viewer : options.user,
        route: options.route ?? "/chat/room-1",
        path: options.path ?? "/chat/:roomId",
        notification: { addWSListener, sendWSMessage },
    });
    const rendered = renderHook(() => useDmController(), { wrapper });

    function emit(msg: WSMessage): void {
        act(() => {
            for (const handler of handlers.slice()) {
                handler(msg);
            }
        });
    }

    return { ...rendered, emit, sendWSMessage, addWSListener, unsubscribe };
}

async function renderLoadedDm(options: HarnessOptions = {}) {
    const harness = renderDm(options);
    await waitFor(() => {
        expect(harness.result.current.loading).toBe(false);
    });

    return harness;
}

beforeEach(() => {
    mocks.fetchUserRooms.mockResolvedValue({ rooms: [makeRoom()] });
    mocks.fetchRoomMessages.mockResolvedValue({ messages: [], total: 0 });
    mocks.fetchRoomMessagesBefore.mockResolvedValue({ messages: [], total: 0 });
    mocks.fetchResolveDMRoom.mockResolvedValue({ room: null, recipient: makeMember() });
    mocks.fetchMutualFollowers.mockResolvedValue([]);
    mocks.fetchSearchUsers.mockResolvedValue([]);
    mocks.markChatRoomRead.mockResolvedValue(undefined);
    mocks.deleteChatRoom.mockResolvedValue(undefined);
});

describe("getRoomDisplayName", () => {
    it("uses the name a group chat was given", () => {
        // given
        const room = makeRoom({ type: "group", name: "Rokkenjima" });

        // when
        const name = getRoomDisplayName(room, viewer);

        // then
        expect(name).toBe("Rokkenjima");
    });

    it("falls back to a generic label for an unnamed group chat", () => {
        // given
        const room = makeRoom({ type: "group", name: "" });

        // when
        const name = getRoomDisplayName(room, viewer);

        // then
        expect(name).toBe("Group Chat");
    });

    it("names a direct message after the other person", () => {
        // given
        const room = makeRoom();

        // when
        const name = getRoomDisplayName(room, viewer);

        // then
        expect(name).toBe("Battler");
    });

    it("falls back to a generic label when the viewer is the only member left", () => {
        // given
        const room = makeRoom({ members: [{ id: "u1", username: "beatrice", display_name: "Beatrice" }] });

        // when
        const name = getRoomDisplayName(room, viewer);

        // then
        expect(name).toBe("Direct Message");
    });
});

describe("getRoomAvatarUser", () => {
    it("picks the other person in a direct message", () => {
        // given
        const room = makeRoom();

        // when
        const avatarUser = getRoomAvatarUser(room, viewer);

        // then
        expect(avatarUser?.id).toBe("u2");
    });

    it("has nobody to show for a group chat", () => {
        // given
        const room = makeRoom({ type: "group", name: "Rokkenjima" });

        // when
        const avatarUser = getRoomAvatarUser(room, viewer);

        // then
        expect(avatarUser).toBeNull();
    });

    it("has nobody to show when the viewer is alone in the conversation", () => {
        // given
        const room = makeRoom({ members: [{ id: "u1", username: "beatrice", display_name: "Beatrice" }] });

        // when
        const avatarUser = getRoomAvatarUser(room, viewer);

        // then
        expect(avatarUser).toBeNull();
    });
});

describe("renderSeenLabel", () => {
    const readAt = "2026-08-02T10:30:00Z";
    const time = new Date(readAt).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });

    it("says nothing when the room has not loaded", () => {
        // given
        const messages = [makeMessage()];

        // when
        const label = renderSeenLabel(messages[0], 0, messages, undefined, "u1", {});

        // then
        expect(label).toBeNull();
    });

    it("only labels the newest message the viewer sent", () => {
        // given
        const messages = [
            makeMessage({ id: "m1", sender: { id: "u1", username: "beatrice", display_name: "Beatrice" } }),
            makeMessage({ id: "m2", sender: { id: "u1", username: "beatrice", display_name: "Beatrice" } }),
        ];
        const receipts = { "room-1": { u2: readAt } };

        // when
        const label = renderSeenLabel(messages[0], 0, messages, makeRoom(), "u1", receipts);

        // then
        expect(label).toBeNull();
    });

    it("says nothing while nobody has read anything in the room", () => {
        // given
        const messages = [makeMessage()];

        // when
        const label = renderSeenLabel(messages[0], 0, messages, makeRoom(), "u1", { "room-2": { u2: readAt } });

        // then
        expect(label).toBeNull();
    });

    it("ignores a receipt from before the message was sent", () => {
        // given
        const messages = [makeMessage({ created_at: "2026-08-02T11:00:00Z" })];
        const receipts = { "room-1": { u2: readAt } };

        // when
        const label = renderSeenLabel(messages[0], 0, messages, makeRoom(), "u1", receipts);

        // then
        expect(label).toBeNull();
    });

    it("reports the time a direct message was read without naming the reader", () => {
        // given
        const messages = [makeMessage()];
        const receipts = { "room-1": { u2: readAt } };

        // when
        const label = renderSeenLabel(messages[0], 0, messages, makeRoom(), "u1", receipts);

        // then
        expect(label).toBe(`seen ${time}`);
    });

    it("names the most recent reader in a group chat", () => {
        // given
        const room = makeRoom({
            type: "group",
            name: "Rokkenjima",
            members: [
                { id: "u1", username: "beatrice", display_name: "Beatrice" },
                makeMember(),
                makeMember({ id: "u3", username: "ange", display_name: "Ange" }),
            ],
        });
        const messages = [makeMessage()];
        const receipts = { "room-1": { u2: "2026-08-02T10:15:00Z", u3: readAt } };

        // when
        const label = renderSeenLabel(messages[0], 0, messages, room, "u1", receipts);

        // then
        expect(label).toBe(`seen by Ange ${time}`);
    });

    it("never counts the viewer's own read receipt", () => {
        // given
        const messages = [makeMessage()];
        const receipts = { "room-1": { u1: readAt } };

        // when
        const label = renderSeenLabel(messages[0], 0, messages, makeRoom(), "u1", receipts);

        // then
        expect(label).toBeNull();
    });
});

describe("useDmController conversation list", () => {
    it("keeps only the direct messages the viewer belongs to", async () => {
        // given
        mocks.fetchUserRooms.mockResolvedValue({
            rooms: [makeRoom(), makeRoom({ id: "room-2", type: "group", name: "Rokkenjima" })],
        });

        // when
        const { result } = await renderLoadedDm();

        // then
        expect(result.current.rooms.map(r => r.id)).toEqual(["room-1"]);
    });

    it("never asks for conversations while nobody is signed in", () => {
        // given
        const options: HarnessOptions = { user: null };

        // when
        const { result } = renderDm(options);

        // then
        expect(mocks.fetchUserRooms).not.toHaveBeenCalled();
        expect(result.current.loading).toBe(true);
    });

    it("opens the conversation named in the url", async () => {
        // given
        const options: HarnessOptions = { route: "/chat/room-1", path: "/chat/:roomId" };

        // when
        const { result } = await renderLoadedDm(options);

        // then
        expect(result.current.activeRoomId).toBe("room-1");
        expect(result.current.activeRoom?.id).toBe("room-1");
        expect(result.current.mobileView).toBe("room");
    });

    it("shows the conversation list when the url names no room", async () => {
        // given
        const options: HarnessOptions = { route: "/chat", path: "/chat" };

        // when
        const { result } = await renderLoadedDm(options);

        // then
        expect(result.current.activeRoomId).toBeNull();
        expect(result.current.mobileView).toBe("list");
    });
});

describe("useDmController socket wiring", () => {
    it("joins the active room over the socket and leaves when the view closes", async () => {
        // given
        const { sendWSMessage, unmount } = await renderLoadedDm();

        // when
        const joined = sendWSMessage.mock.calls.map(call => call[0]);
        unmount();

        // then
        expect(joined).toContainEqual({ type: "join_room", data: { room_id: "room-1" } });
        expect(sendWSMessage).toHaveBeenLastCalledWith({ type: "leave_room", data: { room_id: "room-1" } });
    });

    it("marks the active conversation read as soon as it opens", async () => {
        // given
        const options: HarnessOptions = {};

        // when
        await renderLoadedDm(options);

        // then
        expect(mocks.markChatRoomRead).toHaveBeenCalledWith("room-1");
    });

    it("marks the active conversation read again when the window regains focus", async () => {
        // given
        await renderLoadedDm();
        mocks.markChatRoomRead.mockClear();

        // when
        act(() => {
            window.dispatchEvent(new Event("focus"));
        });

        // then
        expect(mocks.markChatRoomRead).toHaveBeenCalledWith("room-1");
    });

    it("stops listening to the socket once the view goes away", async () => {
        // given
        const { unmount, unsubscribe, addWSListener } = await renderLoadedDm();

        // when
        unmount();

        // then
        expect(addWSListener).toHaveBeenCalled();
        expect(unsubscribe).toHaveBeenCalled();
    });

    it("announces that the viewer is typing in the active conversation", async () => {
        // given
        const { result, sendWSMessage } = await renderLoadedDm();

        // when
        act(() => {
            result.current.notifyTyping();
        });

        // then
        expect(sendWSMessage).toHaveBeenCalledWith({ type: "typing", data: { room_id: "room-1" } });
    });
});

describe("useDmController incoming messages", () => {
    it("shows a message that arrives for the conversation on screen", async () => {
        // given
        const { result, emit } = await renderLoadedDm();

        // when
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // then
        expect(result.current.messages.map(m => m.id)).toEqual(["m1"]);
    });

    it("never shows the viewer's own message twice when the echo arrives", async () => {
        // given
        const { result, emit } = await renderLoadedDm();
        const own = makeMessage({ id: "m1", sender: { id: "u1", username: "beatrice", display_name: "Beatrice" } });
        act(() => {
            result.current.handleSentMessage(own);
        });

        // when
        emit({ type: "chat_message", data: own });

        // then
        expect(result.current.messages).toHaveLength(1);
    });

    it("leaves the open conversation alone when a message lands in another one", async () => {
        // given
        mocks.fetchUserRooms.mockResolvedValue({ rooms: [makeRoom(), makeRoom({ id: "room-2" })] });
        const { result, emit } = await renderLoadedDm();

        // when
        emit({ type: "chat_message", data: makeMessage({ id: "m9", room_id: "room-2" }) });

        // then
        expect(result.current.messages).toEqual([]);
    });

    it("badges another conversation as unread and floats it to the top", async () => {
        // given
        mocks.fetchUserRooms.mockResolvedValue({ rooms: [makeRoom(), makeRoom({ id: "room-2" })] });
        const { result, emit } = await renderLoadedDm();

        // when
        emit({
            type: "chat_message",
            data: makeMessage({ id: "m9", room_id: "room-2", created_at: "2026-08-02T12:00:00Z" }),
        });

        // then
        expect(result.current.rooms.map(r => r.id)).toEqual(["room-2", "room-1"]);
        expect(result.current.rooms[0].unread).toBe(true);
        expect(result.current.rooms[0].last_message_at).toBe("2026-08-02T12:00:00Z");
    });

    it("never badges a conversation for the viewer's own message", async () => {
        // given
        mocks.fetchUserRooms.mockResolvedValue({ rooms: [makeRoom(), makeRoom({ id: "room-2" })] });
        const { result, emit } = await renderLoadedDm();

        // when
        emit({
            type: "chat_message",
            data: makeMessage({
                id: "m9",
                room_id: "room-2",
                sender: { id: "u1", username: "beatrice", display_name: "Beatrice" },
            }),
        });

        // then
        expect(result.current.rooms[0].unread).toBe(false);
    });

    it("never badges the conversation the viewer is reading", async () => {
        // given
        const { result, emit } = await renderLoadedDm();

        // when
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // then
        expect(result.current.rooms[0].unread).toBe(false);
    });

    it("reloads the conversation list when a message arrives for a room it does not know", async () => {
        // given
        const { emit } = await renderLoadedDm();
        mocks.fetchUserRooms.mockClear();

        // when
        emit({ type: "chat_message", data: makeMessage({ id: "m9", room_id: "room-unknown" }) });

        // then
        await waitFor(() => {
            expect(mocks.fetchUserRooms).toHaveBeenCalled();
        });
    });

    it("drops a message the server says was deleted", async () => {
        // given
        const { result, emit } = await renderLoadedDm();
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // when
        emit({ type: "chat_message_deleted", data: { room_id: "room-1", message_id: "m1" } });

        // then
        expect(result.current.messages).toEqual([]);
    });

    it("applies an edit that arrives over the socket", async () => {
        // given
        const { result, emit } = await renderLoadedDm();
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // when
        emit({
            type: "chat_message_edited",
            data: makeMessage({ id: "m1", body: "the red truth", edited_at: "2026-08-02T10:05:00Z" }),
        });

        // then
        expect(result.current.messages[0].body).toBe("the red truth");
        expect(result.current.messages[0].edited_at).toBe("2026-08-02T10:05:00Z");
    });

    it("ignores an edit meant for another conversation", async () => {
        // given
        const { result, emit } = await renderLoadedDm();
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // when
        emit({ type: "chat_message_edited", data: makeMessage({ id: "m1", room_id: "room-2", body: "elsewhere" }) });

        // then
        expect(result.current.messages[0].body).toBe("without love it cannot be seen");
    });

    it("plays a sound for someone else's message while the tab is in the background", async () => {
        // given
        Object.defineProperty(document, "visibilityState", { configurable: true, get: () => "hidden" });
        const { emit } = await renderLoadedDm();

        // when
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // then
        expect(mocks.playMessageSound).toHaveBeenCalled();
        Reflect.deleteProperty(document, "visibilityState");
    });

    it("stays silent for the viewer's own message", async () => {
        // given
        Object.defineProperty(document, "visibilityState", { configurable: true, get: () => "hidden" });
        const { emit } = await renderLoadedDm();

        // when
        emit({
            type: "chat_message",
            data: makeMessage({ id: "m1", sender: { id: "u1", username: "beatrice", display_name: "Beatrice" } }),
        });

        // then
        expect(mocks.playMessageSound).not.toHaveBeenCalled();
        Reflect.deleteProperty(document, "visibilityState");
    });
});

describe("useDmController read receipts", () => {
    it("records who has read how far", async () => {
        // given
        const { result, emit } = await renderLoadedDm();

        // when
        emit({
            type: "chat_read_receipt",
            data: { room_id: "room-1", user_id: "u2", read_at: "2026-08-02T10:30:00Z" },
        });

        // then
        expect(result.current.readReceipts["room-1"].u2).toBe("2026-08-02T10:30:00Z");
    });

    it("ignores a receipt older than the one it already holds", async () => {
        // given
        const { result, emit } = await renderLoadedDm();
        emit({
            type: "chat_read_receipt",
            data: { room_id: "room-1", user_id: "u2", read_at: "2026-08-02T10:30:00Z" },
        });

        // when
        emit({
            type: "chat_read_receipt",
            data: { room_id: "room-1", user_id: "u2", read_at: "2026-08-02T09:00:00Z" },
        });

        // then
        expect(result.current.readReceipts["room-1"].u2).toBe("2026-08-02T10:30:00Z");
    });
});

describe("useDmController typing", () => {
    it("names the person typing in the active conversation", async () => {
        // given
        const { result, emit } = await renderLoadedDm();

        // when
        emit({ type: "typing", data: { room_id: "room-1", user_id: "u2" } });

        // then
        expect(result.current.typingNames).toEqual(["Battler"]);
    });

    it("never says the viewer is typing to themselves", async () => {
        // given
        const { result, emit } = await renderLoadedDm();

        // when
        emit({ type: "typing", data: { room_id: "room-1", user_id: "u1" } });

        // then
        expect(result.current.typingNames).toEqual([]);
    });

    it("falls back to a placeholder for somebody the room does not list", async () => {
        // given
        const { result, emit } = await renderLoadedDm();

        // when
        emit({ type: "typing", data: { room_id: "room-1", user_id: "ghost" } });

        // then
        expect(result.current.typingNames).toEqual(["Someone"]);
    });

    it("ignores typing from a conversation the viewer is not reading", async () => {
        // given
        const { result, emit } = await renderLoadedDm();

        // when
        emit({ type: "typing", data: { room_id: "room-2", user_id: "u2" } });

        // then
        expect(result.current.typingNames).toEqual([]);
    });

    it("clears the typing indicator once that person's message lands", async () => {
        // given
        const { result, emit } = await renderLoadedDm();
        emit({ type: "typing", data: { room_id: "room-1", user_id: "u2" } });

        // when
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // then
        expect(result.current.typingNames).toEqual([]);
    });
});

describe("useDmController selecting conversations", () => {
    it("clears the unread badge for the conversation the viewer opens", async () => {
        // given
        mocks.fetchUserRooms.mockResolvedValue({ rooms: [makeRoom({ unread: true })] });
        const { result } = await renderLoadedDm();

        // when
        act(() => {
            result.current.handleRoomSelect("room-1");
        });

        // then
        expect(result.current.rooms[0].unread).toBe(false);
        expect(result.current.activeRoomId).toBe("room-1");
    });

    it("drops back to the conversation list when the viewer goes back", async () => {
        // given
        const { result } = await renderLoadedDm();

        // when
        act(() => {
            result.current.handleMobileBack();
        });

        // then
        expect(result.current.activeRoomId).toBeNull();
        expect(result.current.draftRecipient).toBeNull();
    });

    it("opens the conversation the viewer already had with the person they picked", async () => {
        // given
        const { result } = await renderLoadedDm({ route: "/chat", path: "/chat/:roomId?" });
        mocks.fetchResolveDMRoom.mockResolvedValue({ room: makeRoom({ id: "room-9" }), recipient: makeMember() });

        // when
        await act(async () => {
            await result.current.handleSelectUser(makeMember());
        });

        // then
        expect(mocks.fetchResolveDMRoom).toHaveBeenCalledWith("u2");
        expect(result.current.activeRoomId).toBe("room-9");
        expect(result.current.rooms.map(r => r.id)).toContain("room-9");
    });

    it("holds the person as a draft when there is no conversation yet", async () => {
        // given
        const { result } = await renderLoadedDm({ route: "/chat", path: "/chat" });
        mocks.fetchResolveDMRoom.mockResolvedValue({ room: null, recipient: makeMember({ id: "u5" }) });

        // when
        await act(async () => {
            await result.current.handleSelectUser(makeMember({ id: "u5" }));
        });

        // then
        expect(result.current.draftRecipient?.id).toBe("u5");
        expect(result.current.activeRoomId).toBeNull();
        expect(result.current.mobileView).toBe("room");
    });

    it("reports why opening a conversation failed", async () => {
        // given
        const { result } = await renderLoadedDm({ route: "/chat", path: "/chat" });
        mocks.fetchResolveDMRoom.mockRejectedValue(new Error("that user blocked you"));

        // when
        await act(async () => {
            await result.current.handleSelectUser(makeMember());
        });

        // then
        expect(result.current.dmError).toBe("that user blocked you");
        expect(result.current.dmCreating).toBe(false);
    });
});

describe("useDmController sending", () => {
    it("opens the brand new conversation the first message created", async () => {
        // given
        const { result } = await renderLoadedDm({ route: "/chat", path: "/chat/:roomId?" });
        const room = makeRoom({ id: "room-9" });
        const message = makeMessage({ id: "m1", room_id: "room-9" });

        // when
        act(() => {
            result.current.handleSentMessage(message, room);
        });

        // then
        expect(result.current.activeRoomId).toBe("room-9");
        expect(result.current.rooms.map(r => r.id)).toContain("room-9");
        expect(result.current.draftRecipient).toBeNull();
    });

    it("shows the first message of the brand new conversation before the server answers", async () => {
        // given
        const { result } = await renderLoadedDm({ route: "/chat", path: "/chat/:roomId?" });
        const room = makeRoom({ id: "room-9" });
        const message = makeMessage({ id: "m1", room_id: "room-9" });
        mocks.fetchRoomMessages.mockReturnValue(new Promise(() => {}));

        // when
        act(() => {
            result.current.handleSentMessage(message, room);
        });

        // then
        expect(result.current.messages.map(m => m.id)).toEqual(["m1"]);
    });

    it("floats the conversation the viewer replied to back to the top", async () => {
        // given
        mocks.fetchUserRooms.mockResolvedValue({ rooms: [makeRoom({ id: "room-2" }), makeRoom()] });
        const { result } = await renderLoadedDm();

        // when
        act(() => {
            result.current.handleSentMessage(makeMessage({ id: "m1", created_at: "2026-08-02T12:00:00Z" }));
        });

        // then
        expect(result.current.rooms.map(r => r.id)).toEqual(["room-1", "room-2"]);
        expect(result.current.rooms[0].last_message_at).toBe("2026-08-02T12:00:00Z");
        expect(result.current.messages.map(m => m.id)).toEqual(["m1"]);
    });
});

describe("useDmController deleting a conversation", () => {
    it("leaves the conversation alone when the viewer backs out of the confirmation", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const { result } = await renderLoadedDm();

        // when
        await act(async () => {
            await result.current.handleDeleteChat();
        });

        // then
        expect(mocks.deleteChatRoom).not.toHaveBeenCalled();
        expect(result.current.rooms).toHaveLength(1);
        confirm.mockRestore();
    });

    it("removes the conversation and returns to the list once confirmed", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const { result } = await renderLoadedDm();

        // when
        await act(async () => {
            await result.current.handleDeleteChat();
        });

        // then
        expect(mocks.deleteChatRoom).toHaveBeenCalledWith("room-1");
        expect(result.current.rooms).toEqual([]);
        expect(result.current.activeRoomId).toBeNull();
        confirm.mockRestore();
    });

    it("keeps the conversation when the server refuses to delete it", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        mocks.deleteChatRoom.mockRejectedValue(new Error("nope"));
        const { result } = await renderLoadedDm();

        // when
        await act(async () => {
            await result.current.handleDeleteChat();
        });

        // then
        expect(result.current.rooms).toHaveLength(1);
        expect(result.current.activeRoomId).toBe("room-1");
        confirm.mockRestore();
    });
});

describe("useDmController finding people", () => {
    it("waits for the viewer to stop typing before searching", async () => {
        // given
        vi.useFakeTimers();
        const harness = renderDm();
        await act(async () => {
            await vi.advanceTimersByTimeAsync(0);
        });

        // when
        act(() => {
            harness.result.current.setDmSearch("bat");
        });
        act(() => {
            vi.advanceTimersByTime(150);
        });
        const beforeDebounce = mocks.fetchSearchUsers.mock.calls.length;
        await act(async () => {
            await vi.advanceTimersByTimeAsync(100);
        });

        // then
        expect(beforeDebounce).toBe(0);
        expect(mocks.fetchSearchUsers).toHaveBeenCalledWith("bat");
    });

    it("shows the people the search found", async () => {
        // given
        vi.useFakeTimers();
        mocks.fetchSearchUsers.mockResolvedValue([makeMember({ id: "u7", display_name: "Ange" })]);
        const harness = renderDm();
        await act(async () => {
            await vi.advanceTimersByTimeAsync(0);
        });

        // when
        act(() => {
            harness.result.current.setDmSearch("ang");
        });
        await act(async () => {
            await vi.advanceTimersByTimeAsync(250);
        });

        // then
        expect(harness.result.current.dmResults.map(u => u.id)).toEqual(["u7"]);
    });

    it("clears the results when the search box is emptied", async () => {
        // given
        vi.useFakeTimers();
        mocks.fetchSearchUsers.mockResolvedValue([makeMember({ id: "u7" })]);
        const harness = renderDm();
        await act(async () => {
            await vi.advanceTimersByTimeAsync(0);
        });
        act(() => {
            harness.result.current.setDmSearch("ang");
        });
        await act(async () => {
            await vi.advanceTimersByTimeAsync(250);
        });

        // when
        act(() => {
            harness.result.current.setDmSearch("   ");
        });
        await act(async () => {
            await vi.advanceTimersByTimeAsync(10);
        });

        // then
        expect(harness.result.current.dmResults).toEqual([]);
        expect(mocks.fetchSearchUsers).toHaveBeenCalledTimes(1);
    });

    it("suggests mutual followers when the new conversation panel opens", async () => {
        // given
        mocks.fetchMutualFollowers.mockResolvedValue([makeMember({ id: "u8", display_name: "Maria" })]);
        const { result } = await renderLoadedDm();

        // when
        act(() => {
            result.current.setShowNewDm(true);
        });

        // then
        await waitFor(() => {
            expect(result.current.dmMutuals.map(u => u.id)).toEqual(["u8"]);
        });
    });

    it("shows nobody when the mutual follower lookup fails", async () => {
        // given
        mocks.fetchMutualFollowers.mockRejectedValue(new Error("offline"));
        const { result } = await renderLoadedDm();

        // when
        act(() => {
            result.current.setShowNewDm(true);
        });

        // then
        await waitFor(() => {
            expect(result.current.dmMutuals).toEqual([]);
        });
    });
});
