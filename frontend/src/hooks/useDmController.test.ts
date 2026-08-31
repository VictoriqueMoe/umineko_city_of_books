import type { QueryClient } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { useLocation, useNavigate } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { queryKeys } from "../api/queryKeys";
import type * as BusModule from "../api/realtime/bus";
import type * as OutboundModule from "../api/realtime/outbound";
import { makeChatMessage, makeChatRoom, makeDmRoom, makePublicUser, makeUser } from "../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../test-utils/render";
import { makeWSHarness, type RealtimeTestEvent, type RealtimeTestNames, type WSHarness } from "../test-utils/ws";
import type { ChatMessage, ChatRoom, User, UserProfile } from "../types/api";
import type * as ChatMutations from "./mutations/chat";
import type * as ChatQueries from "./queries/chat";
import { useDmController } from "./useDmController";

const mocks = vi.hoisted(() => ({
    getUserRooms: vi.fn(),
    fetchResolveDMRoom: vi.fn(),
    fetchRoomMessages: vi.fn(),
    fetchRoomMessagesBefore: vi.fn(),
    fetchMutualFollowers: vi.fn(),
    fetchSearchUsers: vi.fn(),
    deleteChatRoom: vi.fn(),
    setChatRoomMuted: vi.fn(),
    markChatRoomRead: vi.fn(),
    deleteChatMessage: vi.fn(),
    editChatMessage: vi.fn(),
    addChatMessageReaction: vi.fn(),
    removeChatMessageReaction: vi.fn(),
    useVoiceChat: vi.fn(),
    playMessageSound: vi.fn(),
    playRemoteAudio: vi.fn(),
}));

const holder = vi.hoisted(() => ({ ws: null as unknown as WSHarness }));

vi.mock("../api/realtime/pipeline", () => ({
    ensureRealtimePipeline: () => {},
    getRealtimeEpoch: () => holder.ws.getEpoch(),
    subscribeRealtimeEpoch: (listener: () => void) => holder.ws.subscribeEpoch(listener),
}));

vi.mock("../api/realtime/bus", async importOriginal => {
    const actual = await importOriginal<typeof BusModule>();

    return {
        ...actual,
        subscribe: (names: RealtimeTestNames, handler: BusModule.RealtimeEventHandler) =>
            holder.ws.subscribe(names, handler),
    };
});

vi.mock("../api/realtime/outbound", async importOriginal => {
    const actual = await importOriginal<typeof OutboundModule>();

    return { ...actual, sendRealtime: (command: OutboundModule.RealtimeCommand) => holder.ws.sendRealtime(command) };
});

vi.mock("../api/endpoints/chat", async importOriginal => {
    const actual = await importOriginal<Record<string, unknown>>();

    return {
        ...actual,
        getUserRooms: mocks.getUserRooms,
        deleteChatRoom: mocks.deleteChatRoom,
        setChatRoomMuted: mocks.setChatRoomMuted,
    };
});

vi.mock("./queries/chat", async importOriginal => {
    const actual = await importOriginal<typeof ChatQueries>();

    return {
        ...actual,
        fetchResolveDMRoom: mocks.fetchResolveDMRoom,
        fetchRoomMessages: mocks.fetchRoomMessages,
        fetchRoomMessagesBefore: mocks.fetchRoomMessagesBefore,
    };
});

vi.mock("./queries/user", () => ({
    fetchMutualFollowers: mocks.fetchMutualFollowers,
    fetchSearchUsers: mocks.fetchSearchUsers,
}));

vi.mock("./mutations/chat", async importOriginal => {
    const actual = await importOriginal<typeof ChatMutations>();

    return {
        ...actual,
        useMarkChatRoomRead: () => ({ mutate: mocks.markChatRoomRead, mutateAsync: mocks.markChatRoomRead }),
        useDeleteChatMessage: () => ({ mutateAsync: mocks.deleteChatMessage }),
        useEditChatMessage: () => ({ mutateAsync: mocks.editChatMessage }),
        useAddChatMessageReaction: () => ({ mutateAsync: mocks.addChatMessageReaction }),
        useRemoveChatMessageReaction: () => ({ mutateAsync: mocks.removeChatMessageReaction }),
    };
});

vi.mock("./useVoiceChat", () => ({
    useVoiceChat: (roomId: string, initialParticipants?: string[]) => {
        mocks.useVoiceChat(roomId, initialParticipants);

        return { status: "idle", room: null, participantIds: [], join: () => {}, leave: () => {} };
    },
}));

vi.mock("../platform/sound", () => ({
    playMessageSound: mocks.playMessageSound,
    playRemoteAudio: mocks.playRemoteAudio,
}));

const viewer = makeUser({ id: "u1", username: "beatrice", display_name: "Beatrice" });

function makeMember(overrides: Partial<User> = {}): User {
    return { id: "u2", username: "battler", display_name: "Battler", ...overrides };
}

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeDmRoom({ members: [makePublicUser(), makeMember()], ...overrides });
}

function makeGroupRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeChatRoom({ id: "room-group", name: "Rokkenjima", ...overrides });
}

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return makeChatMessage({
        sender: makeMember(),
        body: "without love it cannot be seen",
        created_at: "2026-08-02T10:00:00Z",
        ...overrides,
    });
}

interface HarnessOptions {
    user?: UserProfile | null;
    route?: string;
    path?: string;
    queryClient?: QueryClient;
}

async function settle(): Promise<void> {
    await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
    });
}

function renderDm(options: HarnessOptions = {}) {
    const queryClient = options.queryClient ?? createTestQueryClient();
    const wrapper = providerWrapper({
        user: options.user === undefined ? viewer : options.user,
        route: options.route ?? "/chat/room-1",
        path: options.path ?? "/chat/:roomId?",
        queryClient,
    });
    const rendered = renderHook(() => useDmController(), { wrapper });

    async function emit(event: RealtimeTestEvent): Promise<void> {
        holder.ws.emit(event);
        await settle();
    }

    return {
        ...rendered,
        queryClient,
        emit,
        sendRealtime: holder.ws.sendRealtime,
        subscribe: holder.ws.subscribe,
        unsubscribe: holder.ws.unsubscribe,
    };
}

async function renderLoadedDm(options: HarnessOptions = {}) {
    const harness = renderDm(options);
    await waitFor(() => {
        expect(harness.result.current.loading).toBe(false);
    });

    return harness;
}

async function renderNavigableDm(route = "/chat/room-1") {
    const wrapper = providerWrapper({ user: viewer, route, path: "/chat/:roomId?" });
    const rendered = renderHook(
        () => ({ dm: useDmController(), navigate: useNavigate(), pathname: useLocation().pathname }),
        { wrapper },
    );

    await waitFor(() => {
        expect(rendered.result.current.dm.loading).toBe(false);
    });

    return rendered;
}

function roomCommands(): OutboundModule.RealtimeCommand[] {
    return holder.ws.sendRealtime.mock.calls
        .map(call => call[0])
        .filter(command => command.type === "join_room" || command.type === "leave_room");
}

beforeEach(() => {
    holder.ws = makeWSHarness();
    mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom()] });
    mocks.fetchRoomMessages.mockResolvedValue({ messages: [], total: 0 });
    mocks.fetchRoomMessagesBefore.mockResolvedValue({ messages: [], total: 0 });
    mocks.fetchResolveDMRoom.mockResolvedValue({ room: null, recipient: makeMember() });
    mocks.fetchMutualFollowers.mockResolvedValue([]);
    mocks.fetchSearchUsers.mockResolvedValue([]);
    mocks.markChatRoomRead.mockResolvedValue(undefined);
    mocks.deleteChatRoom.mockResolvedValue(undefined);
    mocks.setChatRoomMuted.mockResolvedValue({ muted: true });
    mocks.addChatMessageReaction.mockResolvedValue(undefined);
    mocks.removeChatMessageReaction.mockResolvedValue(undefined);
});

describe("useDmController conversation list", () => {
    it("keeps only the direct messages the viewer belongs to", async () => {
        // given
        mocks.getUserRooms.mockResolvedValue({
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
        expect(mocks.getUserRooms).not.toHaveBeenCalled();
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

    it("describes the conversation on screen as a pair, with its own capabilities", async () => {
        // given
        const options: HarnessOptions = { route: "/chat/room-1" };

        // when
        const { result } = await renderLoadedDm(options);

        // then
        expect(result.current.capabilities.kind).toBe("pair");
        expect(result.current.capabilities.readReceipts).toBe("pairwise");
        expect(result.current.capabilities.canEditSettings).toBe(false);
    });

    it("shows the conversation list when the url names no room", async () => {
        // given
        const options: HarnessOptions = { route: "/chat" };

        // when
        const { result } = await renderLoadedDm(options);

        // then
        expect(result.current.activeRoomId).toBeNull();
        expect(result.current.mobileView).toBe("list");
    });
});

describe("useDmController conversation list cache", () => {
    it("keeps the page waiting only until the conversations arrive", async () => {
        // given
        let release: (rooms: { rooms: ChatRoom[] }) => void = () => {};
        mocks.getUserRooms.mockReturnValue(
            new Promise(resolve => {
                release = resolve;
            }),
        );
        const { result } = renderDm();

        // when
        const whileFetching = result.current.loading;
        await act(async () => {
            release({ rooms: [makeRoom()] });
        });

        // then
        expect(whileFetching).toBe(true);
        expect(result.current.loading).toBe(false);
    });

    it("stops waiting when the conversation list cannot be fetched", async () => {
        // given
        mocks.getUserRooms.mockRejectedValue(new Error("offline"));

        // when
        const { result } = renderDm();

        // then
        await waitFor(() => {
            expect(result.current.loading).toBe(false);
        });
        expect(result.current.rooms).toEqual([]);
    });

    it("shows the conversations a cache refresh brought in, having asked for none of them itself", async () => {
        // given
        const { result, queryClient } = await renderLoadedDm();
        mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom(), makeRoom({ id: "room-5" })] });

        // when
        await act(async () => {
            await queryClient.invalidateQueries({ queryKey: queryKeys.chat.userRooms() });
        });
        await settle();

        // then
        expect(result.current.rooms.map(r => r.id)).toEqual(["room-1", "room-5"]);
        expect(mocks.getUserRooms).toHaveBeenCalledTimes(2);
    });

    it("never asks for the conversation list again when a message lands in a group room", async () => {
        // given
        mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom(), makeGroupRoom()] });
        const { result, emit } = await renderLoadedDm();
        mocks.getUserRooms.mockClear();

        // when
        await emit({ type: "chat_message", data: makeMessage({ id: "m9", room_id: "room-group" }) });

        // then
        expect(mocks.getUserRooms).not.toHaveBeenCalled();
        expect(result.current.rooms.map(r => r.id)).toEqual(["room-1"]);
    });

    it("leaves the conversation the viewer is reading at the top while group rooms are busy", async () => {
        // given
        mocks.getUserRooms.mockResolvedValue({
            rooms: [makeRoom(), makeRoom({ id: "room-2" }), makeGroupRoom()],
        });
        const { result, emit } = await renderLoadedDm();

        // when
        await emit({ type: "chat_message", data: makeMessage({ id: "m9", room_id: "room-group" }) });

        // then
        expect(result.current.rooms.map(r => r.id)).toEqual(["room-1", "room-2"]);
        expect(result.current.rooms[0].unread).toBeFalsy();
    });
});

describe("useDmController socket wiring", () => {
    it("joins the active room over the socket and leaves when the view closes", async () => {
        // given
        const { sendRealtime, unmount } = await renderLoadedDm();

        // when
        const joined = sendRealtime.mock.calls.map(call => call[0]);
        unmount();

        // then
        expect(joined).toContainEqual({ type: "join_room", data: { room_id: "room-1" } });
        expect(sendRealtime).toHaveBeenLastCalledWith({ type: "leave_room", data: { room_id: "room-1" } });
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
        const { unmount, unsubscribe, subscribe } = await renderLoadedDm();

        // when
        unmount();

        // then
        expect(subscribe).toHaveBeenCalled();
        expect(unsubscribe).toHaveBeenCalled();
    });

    it("announces that the viewer is typing in the active conversation", async () => {
        // given
        const { result, sendRealtime } = await renderLoadedDm();

        // when
        act(() => {
            result.current.notifyTyping();
        });

        // then
        expect(sendRealtime).toHaveBeenCalledWith({ type: "typing", data: { room_id: "room-1" } });
    });

    it("says nothing about typing while no conversation is open", async () => {
        // given
        const { result, sendRealtime } = await renderLoadedDm({ route: "/chat" });
        sendRealtime.mockClear();

        // when
        act(() => {
            result.current.notifyTyping();
        });

        // then
        expect(sendRealtime).not.toHaveBeenCalled();
    });
});

describe("useDmController url navigation", () => {
    interface NavigationCase {
        name: string;
        to: string;
        expectedRoomId: string | null;
        expectedView: "list" | "room";
        expectedCommands: OutboundModule.RealtimeCommand[];
    }

    const navigationCases: NavigationCase[] = [
        {
            name: "another conversation",
            to: "/chat/room-2",
            expectedRoomId: "room-2",
            expectedView: "room",
            expectedCommands: [
                { type: "join_room", data: { room_id: "room-1" } },
                { type: "leave_room", data: { room_id: "room-1" } },
                { type: "join_room", data: { room_id: "room-2" } },
            ],
        },
        {
            name: "the conversation list",
            to: "/chat",
            expectedRoomId: null,
            expectedView: "list",
            expectedCommands: [
                { type: "join_room", data: { room_id: "room-1" } },
                { type: "leave_room", data: { room_id: "room-1" } },
            ],
        },
    ];

    it.each(navigationCases)(
        "re-points the open conversation and the socket when the url moves to $name",
        async ({ to, expectedRoomId, expectedView, expectedCommands }) => {
            // given
            const { result } = await renderNavigableDm();

            // when
            act(() => {
                result.current.navigate(to);
            });

            // then
            expect(roomCommands()).toEqual(expectedCommands);
            expect(result.current.dm.activeRoomId).toBe(expectedRoomId);
            expect(result.current.dm.mobileView).toBe(expectedView);
        },
    );

    it("drops the reply target belonging to the conversation the url left behind", async () => {
        // given
        const { result } = await renderNavigableDm();
        act(() => {
            result.current.dm.setReplyingTo({
                id: "m1",
                senderName: "Battler",
                bodyPreview: "without love it cannot be seen",
            });
        });

        // when
        act(() => {
            result.current.navigate("/chat/room-2");
        });

        // then
        expect(result.current.dm.replyingTo).toBeNull();
    });

    it("leaves the conversation the viewer opened in the history, so back returns to the list", async () => {
        // given
        const { result } = await renderNavigableDm("/chat");

        // when
        act(() => {
            result.current.dm.handleRoomSelect("room-1");
        });
        const afterSelect = result.current.pathname;
        act(() => {
            result.current.navigate(-1);
        });

        // then
        expect(afterSelect).toBe("/chat/room-1");
        expect(result.current.pathname).toBe("/chat");
        expect(result.current.dm.activeRoomId).toBeNull();
    });

    it("keeps the mobile back button out of the history, so back returns to the conversation before it", async () => {
        // given
        const { result } = await renderNavigableDm("/chat");
        act(() => {
            result.current.dm.handleRoomSelect("room-1");
        });
        act(() => {
            result.current.dm.handleRoomSelect("room-2");
        });

        // when
        act(() => {
            result.current.dm.handleMobileBack();
        });
        const afterBack = result.current.pathname;
        act(() => {
            result.current.navigate(-1);
        });

        // then
        expect(afterBack).toBe("/chat");
        expect(result.current.pathname).toBe("/chat/room-1");
        expect(result.current.dm.activeRoomId).toBe("room-1");
    });

    it("keeps the conversation view mounted while the url changes under it", async () => {
        // given
        const { result } = await renderNavigableDm();
        act(() => {
            result.current.dm.showToast("the golden land");
        });

        // when
        act(() => {
            result.current.navigate("/chat/room-2");
        });

        // then
        expect(result.current.dm.toast).toBe("the golden land");
        expect(mocks.getUserRooms).toHaveBeenCalledTimes(1);
    });
});

describe("useDmController incoming messages", () => {
    it("shows a message that arrives for the conversation on screen", async () => {
        // given
        const { result, emit } = await renderLoadedDm();

        // when
        await emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

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
        await emit({ type: "chat_message", data: own });

        // then
        expect(result.current.messages).toHaveLength(1);
    });

    it("leaves the open conversation alone when a message lands in another one", async () => {
        // given
        mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom(), makeRoom({ id: "room-2" })] });
        const { result, emit } = await renderLoadedDm();

        // when
        await emit({ type: "chat_message", data: makeMessage({ id: "m9", room_id: "room-2" }) });

        // then
        expect(result.current.messages).toEqual([]);
    });

    it("badges another conversation as unread and floats it to the top", async () => {
        // given
        mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom(), makeRoom({ id: "room-2" })] });
        const { result, emit } = await renderLoadedDm();

        // when
        await emit({
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
        mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom(), makeRoom({ id: "room-2" })] });
        const { result, emit } = await renderLoadedDm();

        // when
        await emit({
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
        await emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // then
        expect(result.current.rooms[0].unread).toBe(false);
    });

    it("reloads the conversation list when a message arrives for a room it does not know", async () => {
        // given
        const { emit } = await renderLoadedDm();
        mocks.getUserRooms.mockClear();

        // when
        await emit({ type: "chat_message", data: makeMessage({ id: "m9", room_id: "room-unknown" }) });

        // then
        await waitFor(() => {
            expect(mocks.getUserRooms).toHaveBeenCalled();
        });
    });

    it("asks once, not once per message, for a room the server keeps leaving off the list", async () => {
        // given
        const { emit } = await renderLoadedDm();
        mocks.getUserRooms.mockClear();

        // when
        await emit({ type: "chat_message", data: makeMessage({ id: "m1", room_id: "room-stream" }) });
        await emit({ type: "chat_message", data: makeMessage({ id: "m2", room_id: "room-stream" }) });
        await emit({ type: "chat_message", data: makeMessage({ id: "m3", room_id: "room-stream" }) });

        // then
        expect(mocks.getUserRooms).toHaveBeenCalledTimes(1);
    });

    it("asks again for a conversation that came onto the roster and then fell off it", async () => {
        // given
        const { emit, queryClient } = await renderLoadedDm();
        mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom(), makeRoom({ id: "room-9" })] });
        await emit({ type: "chat_message", data: makeMessage({ id: "m1", room_id: "room-9" }) });
        await waitFor(() => {
            expect(mocks.getUserRooms).toHaveBeenCalledTimes(2);
        });
        mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom()] });
        await act(async () => {
            await queryClient.invalidateQueries({ queryKey: queryKeys.chat.userRooms() });
        });
        await settle();

        // when
        await emit({ type: "chat_message", data: makeMessage({ id: "m2", room_id: "room-9" }) });

        // then
        await waitFor(() => {
            expect(mocks.getUserRooms).toHaveBeenCalledTimes(4);
        });
    });

    it("drops a message the server says was deleted", async () => {
        // given
        const { result, emit } = await renderLoadedDm();
        await emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // when
        await emit({ type: "chat_message_deleted", data: { room_id: "room-1", message_id: "m1" } });

        // then
        expect(result.current.messages).toEqual([]);
    });

    it("applies an edit that arrives over the socket", async () => {
        // given
        const { result, emit } = await renderLoadedDm();
        await emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // when
        await emit({
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
        await emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // when
        await emit({
            type: "chat_message_edited",
            data: makeMessage({ id: "m1", room_id: "room-2", body: "elsewhere" }),
        });

        // then
        expect(result.current.messages[0].body).toBe("without love it cannot be seen");
    });

    it("plays a sound for someone else's message while the tab is in the background", async () => {
        // given
        Object.defineProperty(document, "visibilityState", { configurable: true, get: () => "hidden" });
        const { emit } = await renderLoadedDm();

        // when
        await emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // then
        expect(mocks.playMessageSound).toHaveBeenCalled();
        Reflect.deleteProperty(document, "visibilityState");
    });

    it("stays silent for the viewer's own message", async () => {
        // given
        Object.defineProperty(document, "visibilityState", { configurable: true, get: () => "hidden" });
        const { emit } = await renderLoadedDm();

        // when
        await emit({
            type: "chat_message",
            data: makeMessage({ id: "m1", sender: { id: "u1", username: "beatrice", display_name: "Beatrice" } }),
        });

        // then
        expect(mocks.playMessageSound).not.toHaveBeenCalled();
        Reflect.deleteProperty(document, "visibilityState");
    });
});

describe("useDmController message failures", () => {
    it("tells the viewer why their edit was refused", async () => {
        // given
        mocks.editChatMessage.mockRejectedValue(new Error("that message is too old to edit"));
        const { result } = await renderLoadedDm();

        // when
        await act(async () => {
            await result.current.handleEditMessage(makeMessage({ id: "m1" }), "rewritten").catch(() => {});
        });

        // then
        expect(result.current.toast).toBe("that message is too old to edit");
    });

    it("tells the viewer why their delete was refused", async () => {
        // given
        mocks.deleteChatMessage.mockRejectedValue(new Error("the witch forbids it"));
        const { result } = await renderLoadedDm();

        // when
        await act(async () => {
            await result.current.handleDeleteMessage(makeMessage({ id: "m1" }));
        });

        // then
        expect(result.current.toast).toBe("the witch forbids it");
    });
});

describe("useDmController read receipts", () => {
    it("records who has read how far", async () => {
        // given
        const { result, emit } = await renderLoadedDm();

        // when
        await emit({
            type: "chat_read_receipt",
            data: { room_id: "room-1", user_id: "u2", read_at: "2026-08-02T10:30:00Z" },
        });

        // then
        expect(result.current.readReceipts["room-1"].u2).toBe("2026-08-02T10:30:00Z");
    });

    it("never records a receipt for a room that is not a pair", async () => {
        // given
        mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom(), makeGroupRoom()] });
        const { result, emit } = await renderLoadedDm({ route: "/chat/room-group" });

        // when
        await emit({
            type: "chat_read_receipt",
            data: { room_id: "room-group", user_id: "u2", read_at: "2026-08-02T10:30:00Z" },
        });

        // then
        expect(result.current.capabilities.kind).toBe("group");
        expect(result.current.readReceipts["room-group"]).toBeUndefined();
    });

    it("keeps a receipt for a pair that lands while the viewer is on the conversation list", async () => {
        // given
        const { result, emit } = await renderLoadedDm({ route: "/chat" });

        // when
        await emit({
            type: "chat_read_receipt",
            data: { room_id: "room-1", user_id: "u2", read_at: "2026-08-02T10:30:00Z" },
        });

        // then
        expect(result.current.readReceipts["room-1"].u2).toBe("2026-08-02T10:30:00Z");
    });

    it("keeps a receipt for a pair that lands while another kind of room is open", async () => {
        // given
        mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom(), makeGroupRoom()] });
        const { result, emit } = await renderLoadedDm({ route: "/chat/room-group" });

        // when
        await emit({
            type: "chat_read_receipt",
            data: { room_id: "room-1", user_id: "u2", read_at: "2026-08-02T10:30:00Z" },
        });

        // then
        expect(result.current.readReceipts["room-1"].u2).toBe("2026-08-02T10:30:00Z");
    });

    it("ignores a receipt older than the one it already holds", async () => {
        // given
        const { result, emit } = await renderLoadedDm();
        await emit({
            type: "chat_read_receipt",
            data: { room_id: "room-1", user_id: "u2", read_at: "2026-08-02T10:30:00Z" },
        });

        // when
        await emit({
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
        await emit({ type: "typing", data: { room_id: "room-1", user_id: "u2" } });

        // then
        expect(result.current.typingNames).toEqual(["Battler"]);
    });

    it("never says the viewer is typing to themselves", async () => {
        // given
        const { result, emit } = await renderLoadedDm();

        // when
        await emit({ type: "typing", data: { room_id: "room-1", user_id: "u1" } });

        // then
        expect(result.current.typingNames).toEqual([]);
    });

    it("falls back to a placeholder for somebody the room does not list", async () => {
        // given
        const { result, emit } = await renderLoadedDm();

        // when
        await emit({ type: "typing", data: { room_id: "room-1", user_id: "ghost" } });

        // then
        expect(result.current.typingNames).toEqual(["Someone"]);
    });

    it("ignores typing from a conversation the viewer is not reading", async () => {
        // given
        const { result, emit } = await renderLoadedDm();

        // when
        await emit({ type: "typing", data: { room_id: "room-2", user_id: "u2" } });

        // then
        expect(result.current.typingNames).toEqual([]);
    });

    it("clears the typing indicator once that person's message lands", async () => {
        // given
        const { result, emit } = await renderLoadedDm();
        await emit({ type: "typing", data: { room_id: "room-1", user_id: "u2" } });

        // when
        await emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // then
        expect(result.current.typingNames).toEqual([]);
    });
});

describe("useDmController selecting conversations", () => {
    it("clears the unread badge for the conversation the viewer opens", async () => {
        // given
        mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom({ unread: true })] });
        const { result } = await renderLoadedDm();

        // when
        act(() => {
            result.current.handleRoomSelect("room-1");
        });
        await settle();

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
        await settle();

        // then
        expect(mocks.fetchResolveDMRoom).toHaveBeenCalledWith("u2");
        expect(result.current.activeRoomId).toBe("room-9");
        expect(result.current.rooms.map(r => r.id)).toContain("room-9");
    });

    it("holds the person as a draft when there is no conversation yet", async () => {
        // given
        const { result } = await renderLoadedDm({ route: "/chat" });
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

    it("clears the conversation on screen when the viewer starts a fresh draft", async () => {
        // given
        const { result, emit } = await renderLoadedDm();
        await emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });
        mocks.fetchResolveDMRoom.mockResolvedValue({ room: null, recipient: makeMember({ id: "u5" }) });

        // when
        await act(async () => {
            await result.current.handleSelectUser(makeMember({ id: "u5" }));
        });
        await settle();

        // then
        expect(result.current.draftRecipient?.id).toBe("u5");
        expect(result.current.activeRoomId).toBeNull();
        expect(result.current.messages).toEqual([]);
    });

    it("reports why opening a conversation failed", async () => {
        // given
        const { result } = await renderLoadedDm({ route: "/chat" });
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
        await settle();

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
        mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom({ id: "room-2" }), makeRoom()] });
        const { result } = await renderLoadedDm();

        // when
        act(() => {
            result.current.handleSentMessage(makeMessage({ id: "m1", created_at: "2026-08-02T12:00:00Z" }));
        });
        await settle();

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

    it("mutes the conversation and reflects it on the room before the roster catches up", async () => {
        // given
        const { result } = await renderLoadedDm();
        mocks.getUserRooms.mockReturnValue(new Promise(() => {}));

        // when
        await act(async () => {
            await result.current.handleToggleMute();
        });
        await settle();

        // then
        expect(mocks.setChatRoomMuted).toHaveBeenCalledWith("room-1", true);
        expect(result.current.rooms[0].viewer_muted).toBe(true);
        expect(result.current.toast).toBe("Notifications muted");
    });

    it("keeps the muted conversation once the refreshed roster confirms it, with nobody asking for it", async () => {
        // given
        const { result } = await renderLoadedDm();
        mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom({ viewer_muted: true })] });

        // when
        await act(async () => {
            await result.current.handleToggleMute();
        });

        // then
        await waitFor(() => {
            expect(mocks.getUserRooms).toHaveBeenCalledTimes(2);
        });
        expect(result.current.rooms[0].viewer_muted).toBe(true);
    });

    it("leaves the conversation unmuted when the server refuses", async () => {
        // given
        mocks.setChatRoomMuted.mockRejectedValue(new Error("nope"));
        const { result } = await renderLoadedDm();

        // when
        await act(async () => {
            await result.current.handleToggleMute();
        });

        // then
        expect(result.current.rooms[0].viewer_muted).toBeFalsy();
        expect(result.current.toast).toBe("nope");
        expect(result.current.mutePending).toBe(false);
    });

    it("removes the conversation and returns to the list before the roster catches up", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const { result } = await renderLoadedDm();
        mocks.getUserRooms.mockReturnValue(new Promise(() => {}));

        // when
        await act(async () => {
            await result.current.handleDeleteChat();
        });
        await settle();

        // then
        expect(mocks.deleteChatRoom).toHaveBeenCalledWith("room-1");
        expect(result.current.rooms).toEqual([]);
        expect(result.current.activeRoomId).toBeNull();
        confirm.mockRestore();
    });

    it("keeps the deleted conversation off the roster once it refreshes itself", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const { result } = await renderLoadedDm();
        mocks.getUserRooms.mockResolvedValue({ rooms: [] });

        // when
        await act(async () => {
            await result.current.handleDeleteChat();
        });

        // then
        await waitFor(() => {
            expect(mocks.getUserRooms).toHaveBeenCalledTimes(2);
        });
        expect(result.current.rooms).toEqual([]);
        confirm.mockRestore();
    });

    it("keeps the conversation and says why when the server refuses to delete it", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        mocks.deleteChatRoom.mockRejectedValue(new Error("the witch forbids it"));
        const { result } = await renderLoadedDm();

        // when
        await act(async () => {
            await result.current.handleDeleteChat();
        });

        // then
        expect(result.current.rooms).toHaveLength(1);
        expect(result.current.activeRoomId).toBe("room-1");
        expect(result.current.toast).toBe("the witch forbids it");
        confirm.mockRestore();
    });
});

describe("useDmController reactions", () => {
    it("adds a reaction the viewer has not given yet", async () => {
        // given
        const message = makeMessage({ id: "m1", reactions: [] });
        const { result } = await renderLoadedDm();

        // when
        await act(async () => {
            await result.current.handleReactionToggle(message, "❤");
        });

        // then
        expect(mocks.addChatMessageReaction).toHaveBeenCalledWith({ messageId: "m1", emoji: "❤" });
        expect(mocks.removeChatMessageReaction).not.toHaveBeenCalled();
    });

    it("takes back the reaction the viewer already gave", async () => {
        // given
        const message = makeMessage({
            id: "m1",
            reactions: [{ emoji: "❤", count: 1, viewer_reacted: true, display_names: ["Beatrice"] }],
        });
        const { result } = await renderLoadedDm();

        // when
        await act(async () => {
            await result.current.handleReactionToggle(message, "❤");
        });

        // then
        expect(mocks.removeChatMessageReaction).toHaveBeenCalledWith({ messageId: "m1", emoji: "❤" });
        expect(mocks.addChatMessageReaction).not.toHaveBeenCalled();
    });

    it("adds to a reaction somebody else started", async () => {
        // given
        const message = makeMessage({
            id: "m1",
            reactions: [{ emoji: "❤", count: 1, viewer_reacted: false, display_names: ["Battler"] }],
        });
        const { result } = await renderLoadedDm();

        // when
        await act(async () => {
            await result.current.handleReactionToggle(message, "❤");
        });

        // then
        expect(mocks.addChatMessageReaction).toHaveBeenCalledWith({ messageId: "m1", emoji: "❤" });
    });

    it("tells the viewer why their reaction was refused", async () => {
        // given
        mocks.addChatMessageReaction.mockRejectedValue(new Error("that emoji is forbidden"));
        const { result } = await renderLoadedDm();

        // when
        await act(async () => {
            await result.current.handleReactionToggle(makeMessage({ id: "m1" }), "❤");
        });

        // then
        expect(result.current.toast).toBe("that emoji is forbidden");
    });
});

describe("useDmController room presence", () => {
    it("seeds the voice call with whoever the room already had in it", async () => {
        // given
        mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom({ voice_participants: ["u2"] })] });

        // when
        await renderLoadedDm();

        // then
        expect(mocks.useVoiceChat).toHaveBeenLastCalledWith("room-1", ["u2"]);
    });

    it("reports the viewer as watching the conversation they opened", async () => {
        // given
        const { sendRealtime } = await renderLoadedDm();

        // when
        const sent = sendRealtime.mock.calls.map(call => call[0]);

        // then
        expect(sent).toContainEqual({ type: "viewer_state", data: { room_id: "room-1", state: "active" } });
    });

    it("reports nothing while no conversation is open", async () => {
        // given
        const { sendRealtime } = await renderLoadedDm({ route: "/chat" });

        // when
        const sent = sendRealtime.mock.calls.map(call => call[0]);

        // then
        expect(sent.filter(command => command.type === "viewer_state")).toEqual([]);
    });

    it("titles the tab with the person on the other end", async () => {
        // given
        const options: HarnessOptions = {};

        // when
        await renderLoadedDm(options);

        // then
        expect(document.title).toContain("Battler");
    });

    it("falls back to a plain title while no conversation is open", async () => {
        // given
        const options: HarnessOptions = { route: "/chat" };

        // when
        await renderLoadedDm(options);

        // then
        expect(document.title).not.toContain("Battler");
        expect(document.title).toContain("Chat");
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
