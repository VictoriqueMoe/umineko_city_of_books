import { act, renderHook, waitFor } from "@testing-library/react";
import { useLocation } from "react-router";
import { beforeEach, describe, expect, it, onTestFinished, vi, type Mock } from "vitest";
import type * as BusModule from "../api/realtime/bus";
import type * as OutboundModule from "../api/realtime/outbound";
import { queryKeys } from "../api/queryKeys";
import { makeChatMessage, makeChatRoom, makeDmRoom, makeRoomMember, makeUser } from "../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../test-utils/render";
import { makeWSHarness, type RealtimeTestEvent, type RealtimeTestNames, type WSHarness } from "../test-utils/ws";
import type { ChatMessage, ChatRoom, ChatRoomMember, UserProfile } from "../types/api";
import { useRoomController } from "./useRoomController";

const mocks = vi.hoisted(() => ({
    useUserRooms: vi.fn(),
    useChatRoomMembers: vi.fn(),
    fetchRoomMessages: vi.fn(),
    markRead: vi.fn(),
    joinRoom: vi.fn(),
    leaveRoom: vi.fn(),
    deleteRoom: vi.fn(),
    setMuted: vi.fn(),
    pin: vi.fn(),
    unpin: vi.fn(),
    addReaction: vi.fn(),
    removeReaction: vi.fn(),
    watchPartyRefresh: vi.fn(),
    playMessageSound: vi.fn(),
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

vi.mock("./queries/chat", async importOriginal => {
    const actual = await importOriginal<Record<string, unknown>>();

    return {
        ...actual,
        useUserRooms: mocks.useUserRooms,
        useChatRoomMembers: mocks.useChatRoomMembers,
        fetchRoomMessages: mocks.fetchRoomMessages,
    };
});

vi.mock("./mutations/chat", async importOriginal => {
    const actual = await importOriginal<Record<string, unknown>>();

    return {
        ...actual,
        useMarkChatRoomRead: () => ({ mutate: mocks.markRead }),
        useJoinChatRoom: () => ({ mutateAsync: mocks.joinRoom }),
        useLeaveChatRoom: () => ({ mutateAsync: mocks.leaveRoom }),
        useDeleteChatRoom: () => ({ mutateAsync: mocks.deleteRoom }),
        useSetChatRoomMuted: () => ({ mutateAsync: mocks.setMuted }),
        usePinChatMessage: () => ({ mutateAsync: mocks.pin }),
        useUnpinChatMessage: () => ({ mutateAsync: mocks.unpin }),
        useAddChatMessageReaction: () => ({ mutateAsync: mocks.addReaction }),
        useRemoveChatMessageReaction: () => ({ mutateAsync: mocks.removeReaction }),
    };
});

vi.mock("./useWatchParty", () => {
    const watchParty = { loaded: true, sessions: [], join: () => Promise.resolve(), refresh: mocks.watchPartyRefresh };

    return { useWatchParty: () => watchParty };
});

vi.mock("./useVoiceChat", () => {
    const voice = { participantIds: [] };

    return { useVoiceChat: () => voice };
});

vi.mock("./usePresenceReporter", () => ({ usePresenceReporter: () => {} }));

vi.mock("../platform/sound", () => ({ playMessageSound: mocks.playMessageSound, playAudio: () => {} }));

const beatrice = { id: "u1", username: "beatrice", display_name: "Beatrice" };

const viewer = makeUser(beatrice);

const battler = { id: "u2", username: "battler", display_name: "Battler" };

const seventhTwilight = {
    name: "Purgatory",
    description: "the seventh twilight",
    tags: ["beato", "seventh-twilight"],
    is_public: false,
    is_rp: true,
};

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeChatRoom({ name: "Golden Land", description: "a place for tea", ...overrides });
}

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return makeChatMessage({
        sender: battler,
        body: "without love it cannot be seen",
        created_at: "2026-08-02T10:00:00Z",
        ...overrides,
    });
}

interface RoomHarnessOptions {
    user?: UserProfile | null;
    rooms?: ChatRoom[];
    roomsLoading?: boolean;
    members?: ChatRoomMember[];
    path?: string;
}

function renderRoom(options: RoomHarnessOptions = {}) {
    mocks.useUserRooms.mockReturnValue({
        rooms: options.rooms ?? [makeRoom()],
        loading: options.roomsLoading ?? false,
        refresh: () => {},
    });
    mocks.useChatRoomMembers.mockReturnValue({
        members: options.members ?? [makeRoomMember({ user: battler })],
        loading: false,
        refresh: () => {},
    });

    const queryClient = createTestQueryClient();
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const wrapper = providerWrapper({
        user: options.user === undefined ? viewer : options.user,
        route: "/rooms/room-1",
        path: options.path ?? "/rooms/:roomId",
        queryClient,
    });
    const rendered = renderHook(() => ({ ...useRoomController(), pathname: useLocation().pathname }), { wrapper });

    return { ...rendered, ...holder.ws, invalidateQueries };
}

async function renderLoadedRoom(options: RoomHarnessOptions = {}) {
    const harness = renderRoom(options);

    await waitFor(() => {
        expect(mocks.fetchRoomMessages).toHaveBeenCalled();
    });
    await act(async () => {
        await Promise.resolve();
    });

    return harness;
}

function answerConfirm(answer: boolean): void {
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(answer);

    onTestFinished(() => {
        confirm.mockRestore();
    });
}

beforeEach(() => {
    vi.resetAllMocks();
    holder.ws = makeWSHarness();
    mocks.fetchRoomMessages.mockResolvedValue({ messages: [], total: 0 });
});

describe("useRoomController room loading", () => {
    it("shows the room the url points at, described as a group with its own capabilities", async () => {
        // given
        const options: RoomHarnessOptions = { rooms: [makeRoom(), makeRoom({ id: "room-2", name: "Purgatory" })] };

        // when
        const { result } = await renderLoadedRoom(options);

        // then
        expect(result.current.room.id).toBe("room-1");
        expect(result.current.room.data?.name).toBe("Golden Land");
        expect(result.current.capabilities.kind).toBe("group");
        expect(result.current.capabilities.readReceipts).toBe("none");
        expect(result.current.capabilities.archivesWhenStale).toBe(true);
    });

    it("reports itself as loading while the viewer's rooms are still on the way", () => {
        // given
        const options: RoomHarnessOptions = { rooms: [], roomsLoading: true };

        // when
        const { result } = renderRoom(options);

        // then
        expect(result.current.room.loading).toBe(true);
        expect(result.current.room.data).toBeNull();
    });

    it.each([
        {
            name: "sends a dm reached through the rooms url on to the dm page",
            room: makeDmRoom(),
            pathname: "/chat/room-1",
        },
        { name: "leaves a group room on the rooms url", room: makeRoom(), pathname: "/rooms/room-1" },
    ])("$name", async ({ room, pathname }) => {
        // given
        const options: RoomHarnessOptions = { rooms: [room], path: "/:section/:roomId" };

        // when
        const { result } = renderRoom(options);
        await waitFor(() => {
            expect(result.current.room.data).not.toBeNull();
        });

        // then
        await waitFor(() => {
            expect(result.current.pathname).toBe(pathname);
        });
    });

    it("has no room to show when the viewer does not belong to it", async () => {
        // given
        const options: RoomHarnessOptions = { rooms: [makeRoom({ id: "room-2" })] };

        // when
        const { result } = renderRoom(options);

        // then
        expect(result.current.room.data).toBeNull();
        await waitFor(() => {
            expect(mocks.fetchRoomMessages).not.toHaveBeenCalled();
        });
    });

    it("marks the room read once for a given last message, and again when the window regains focus", async () => {
        // given
        const room = makeRoom({ last_message_at: "2026-08-02T10:00:00Z" });
        const { rerender } = await renderLoadedRoom({ rooms: [room] });

        // when
        mocks.useUserRooms.mockReturnValue({ rooms: [], loading: false, refresh: () => {} });
        act(() => {
            rerender();
        });
        mocks.useUserRooms.mockReturnValue({ rooms: [room], loading: false, refresh: () => {} });
        act(() => {
            rerender();
        });
        const markedBeforeFocus = mocks.markRead.mock.calls.length;
        act(() => {
            window.dispatchEvent(new Event("focus"));
        });

        // then
        expect(markedBeforeFocus).toBe(1);
        expect(mocks.markRead.mock.calls).toEqual([["room-1"], ["room-1"]]);
    });

    it("joins the room over the socket, and leaves it and stops listening when the view closes", async () => {
        // given
        const { sendRealtime, unsubscribe, unmount } = await renderLoadedRoom();

        // when
        const sent = sendRealtime.mock.calls.map(call => call[0]);
        unmount();

        // then
        expect(sent).toContainEqual({ type: "join_room", data: { room_id: "room-1" } });
        expect(sendRealtime).toHaveBeenLastCalledWith({ type: "leave_room", data: { room_id: "room-1" } });
        expect(unsubscribe).toHaveBeenCalled();
    });

    it("ignores everything on the socket while nobody is signed in", () => {
        // given
        const { result, emit } = renderRoom({ user: null });

        // when
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // then
        expect(result.current.session.messages).toEqual([]);
    });

    it("refetches the backlog after the socket reconnects", async () => {
        // given
        const { reconnect } = await renderLoadedRoom();
        mocks.fetchRoomMessages.mockClear();

        // when
        reconnect();

        // then
        await waitFor(() => {
            expect(mocks.fetchRoomMessages).toHaveBeenCalledWith("room-1", 50);
        });
    });
});

describe("useRoomController timeouts", () => {
    it.each([
        {
            name: "treats a timeout that has not expired as still in force and refuses to reopen the viewer's last message",
            timeoutUntil: "2099-01-01T00:00:00Z",
            timedOut: true,
            editing: null,
        },
        {
            name: "treats an expired timeout as over",
            timeoutUntil: "2020-01-01T00:00:00Z",
            timedOut: false,
            editing: "m1",
        },
        {
            name: "reopens the viewer's last message once no timeout is in force",
            timeoutUntil: undefined,
            timedOut: false,
            editing: "m1",
        },
    ])("$name", async ({ timeoutUntil, timedOut, editing }) => {
        // given
        const members = [makeRoomMember({ user: beatrice, timeout_until: timeoutUntil })];
        const { result, emit } = await renderLoadedRoom({ members });
        emit({ type: "chat_message", data: makeMessage({ id: "m1", sender: beatrice }) });

        // when
        act(() => {
            result.current.session.editLast();
        });

        // then
        expect(result.current.room.viewerTimedOut).toBe(timedOut);
        expect(result.current.room.viewerTimeoutUntil).toBe(timeoutUntil);
        expect(result.current.session.editingMessageId).toBe(editing);
    });
});

describe("useRoomController incoming messages", () => {
    it("never shows the viewer's own message twice when the echo arrives", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();
        const own = makeMessage({ id: "m1", sender: beatrice });
        act(() => {
            result.current.session.onSent(own);
        });

        // when
        emit({ type: "chat_message", data: own });

        // then
        expect(result.current.session.messages).toHaveLength(1);
    });

    it.each([
        {
            name: "plays a sound for somebody else's message while the tab is in the background",
            muted: false,
            plays: 1,
        },
        { name: "stays silent while the room is muted", muted: true, plays: 0 },
    ])("$name", async ({ muted, plays }) => {
        // given
        Object.defineProperty(document, "visibilityState", { configurable: true, get: () => "hidden" });
        onTestFinished(() => {
            Reflect.deleteProperty(document, "visibilityState");
        });
        const { emit } = await renderLoadedRoom({ rooms: [makeRoom({ viewer_muted: muted })] });

        // when
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // then
        expect(mocks.playMessageSound).toHaveBeenCalledTimes(plays);
    });
});

describe("useRoomController reactions and pins", () => {
    it.each<{ name: string; pinned: boolean; event: RealtimeTestEvent }>([
        {
            name: "pins a message and stales the pinned panel's query",
            pinned: false,
            event: {
                type: "chat_message_pinned",
                data: { room_id: "room-1", message_id: "m1", pinned_at: "2026-08-02T11:00:00Z", pinned_by: "u2" },
            },
        },
        {
            name: "unpins a message and stales the pinned panel's query",
            pinned: true,
            event: { type: "chat_message_unpinned", data: { room_id: "room-1", message_id: "m1" } },
        },
    ])("$name", async ({ pinned, event }) => {
        // given
        const { result, emit, invalidateQueries } = await renderLoadedRoom();
        emit({ type: "chat_message", data: makeMessage({ id: "m1", pinned }) });
        invalidateQueries.mockClear();

        // when
        emit(event);

        // then
        expect(result.current.session.messages[0].pinned).toBe(!pinned);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.chat.pinned("room-1") });
    });

    it.each([
        {
            name: "asks the server to add a reaction the viewer has not left yet",
            reactions: [],
            called: mocks.addReaction,
            idle: mocks.removeReaction,
        },
        {
            name: "takes back a reaction the viewer already left",
            reactions: [{ emoji: "🌹", count: 1, viewer_reacted: true, display_names: ["Beatrice"] }],
            called: mocks.removeReaction,
            idle: mocks.addReaction,
        },
    ])("$name", async ({ reactions, called, idle }) => {
        // given
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.session.toggleReaction(makeMessage({ id: "m1", reactions }), "🌹");
        });

        // then
        expect(called).toHaveBeenCalledWith({ messageId: "m1", emoji: "🌹" });
        expect(idle).not.toHaveBeenCalled();
    });

    it("reports why a reaction could not be saved", async () => {
        // given
        mocks.addReaction.mockRejectedValue(new Error("you are timed out"));
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.session.toggleReaction(makeMessage(), "🌹");
        });

        // then
        expect(result.current.toast.message).toBe("you are timed out");
    });

    it.each([
        { name: "pins a message the viewer chose to pin", pinned: false, called: mocks.pin, idle: mocks.unpin },
        { name: "unpins a message that was already pinned", pinned: true, called: mocks.unpin, idle: mocks.pin },
    ])("$name", async ({ pinned, called, idle }) => {
        // given
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.session.togglePin(makeMessage({ id: "m1", pinned }));
        });

        // then
        expect(called).toHaveBeenCalledWith("m1");
        expect(idle).not.toHaveBeenCalled();
    });
});

describe("useRoomController membership events", () => {
    it.each<{ name: string; room: ChatRoom; event: RealtimeTestEvent; expected: Partial<ChatRoom> }>([
        {
            name: "counts a new arrival on the room straight away",
            room: makeRoom({ member_count: 2 }),
            event: {
                type: "chat_member_joined",
                data: { room_id: "room-1", user: { id: "u3", username: "ange", display_name: "Ange" } },
            },
            expected: { member_count: 3 },
        },
        {
            name: "takes somebody who left off the room count straight away",
            room: makeRoom({ member_count: 2 }),
            event: { type: "chat_member_left", data: { room_id: "room-1", user_id: "u2" } },
            expected: { member_count: 1 },
        },
        {
            name: "patches the room in place when its settings are edited",
            room: makeRoom({ tags: ["beato"], viewer_muted: true, member_count: 7 }),
            event: { type: "chat_room_updated", data: { room_id: "room-1", ...seventhTwilight } },
            expected: {
                ...seventhTwilight,
                tags: ["beato", "seventh-twilight"],
                viewer_muted: true,
                member_count: 7,
            },
        },
        {
            name: "ignores an edit to another room",
            room: makeRoom(),
            event: { type: "chat_room_updated", data: { room_id: "room-2", ...seventhTwilight } },
            expected: { name: "Golden Land", description: "a place for tea", tags: [], is_public: true, is_rp: false },
        },
    ])("$name", async ({ room, event, expected }) => {
        // given
        const { result, emit } = await renderLoadedRoom({ rooms: [room] });

        // when
        emit(event);

        // then
        expect(result.current.room.data).toMatchObject(expected);
    });

    it.each<{ name: string; event: RealtimeTestEvent; member: object; message: object }>([
        {
            name: "applies a nickname change to the member list and to their messages",
            event: {
                type: "chat_member_updated",
                data: {
                    room_id: "room-1",
                    user_id: "u2",
                    nickname: "Battler-kun",
                    display_name: "Battler",
                    username: "battler",
                    member_avatar_url: "",
                    nickname_locked: true,
                    timeout_until: "",
                    timeout_set_by_staff: false,
                },
            },
            member: { nickname: "Battler-kun" },
            message: { sender_nickname: "Battler-kun" },
        },
        {
            name: "applies a site role change to the member list and to their messages",
            event: { type: "role_changed", data: { user_id: "u2", role: "moderator" } },
            member: { user: { role: "moderator" } },
            message: { sender: { role: "moderator" } },
        },
    ])("$name", async ({ event, member, message }) => {
        // given
        const { result, emit } = await renderLoadedRoom();
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // when
        emit(event);

        // then
        expect(result.current.members.list[0]).toMatchObject(member);
        expect(result.current.session.messages[0]).toMatchObject(message);
    });

    it.each<{ name: string; event: RealtimeTestEvent; toast: string }>([
        {
            name: "tells the viewer when they are removed from the room",
            event: { type: "chat_kicked", data: { room_id: "room-1", reason: "too much tea" } },
            toast: "You were removed from this room: too much tea",
        },
        {
            name: "tells the viewer when the host deletes the room",
            event: { type: "chat_room_deleted", data: { room_id: "room-1" } },
            toast: "This room was deleted by the host",
        },
    ])("$name", async ({ event, toast }) => {
        // given
        const { result, emit } = await renderLoadedRoom();

        // when
        emit(event);

        // then
        expect(result.current.toast.message).toBe(toast);
    });
});

describe("useRoomController typing", () => {
    it.each([
        { name: "routes a typing broadcast for this room into the roster", roomId: "room-1", typingNames: ["Battler"] },
        { name: "ignores a typing broadcast meant for another room", roomId: "room-2", typingNames: [] },
    ])("$name", async ({ roomId, typingNames }) => {
        // given
        const { result, emit } = await renderLoadedRoom();

        // when
        emit({ type: "typing", data: { room_id: roomId, user_id: "u2" } });

        // then
        expect(result.current.session.typingNames).toEqual(typingNames);
    });
});

describe("useRoomController joining and leaving", () => {
    it("joins the room, refreshes the watch parties and shows the room it just joined", async () => {
        // given
        mocks.joinRoom.mockResolvedValue(makeRoom({ name: "Purgatory" }));
        const { result } = renderRoom({ rooms: [] });

        // when
        await act(async () => {
            await result.current.room.join();
        });

        // then
        expect(mocks.joinRoom).toHaveBeenCalledWith({ roomId: "room-1" });
        expect(mocks.watchPartyRefresh).toHaveBeenCalled();
        expect(result.current.room.joining).toBe(false);
        expect(result.current.room.data?.name).toBe("Purgatory");
    });

    it("reports why joining failed", async () => {
        // given
        mocks.joinRoom.mockRejectedValue(new Error("this room is invite only"));
        const { result } = renderRoom({ rooms: [] });

        // when
        await act(async () => {
            await result.current.room.join();
        });

        // then
        expect(result.current.toast.message).toBe("this room is invite only");
    });

    it.each<{ name: string; action: "leave" | "remove"; confirmed: boolean; mutation: Mock; calls: string[][] }>([
        {
            name: "keeps the viewer in the room when they back out of leaving",
            action: "leave",
            confirmed: false,
            mutation: mocks.leaveRoom,
            calls: [],
        },
        {
            name: "leaves the room once the viewer confirms",
            action: "leave",
            confirmed: true,
            mutation: mocks.leaveRoom,
            calls: [["room-1"]],
        },
        {
            name: "deletes the room once the viewer confirms",
            action: "remove",
            confirmed: true,
            mutation: mocks.deleteRoom,
            calls: [["room-1"]],
        },
        {
            name: "keeps the room when the viewer backs out of deleting it",
            action: "remove",
            confirmed: false,
            mutation: mocks.deleteRoom,
            calls: [],
        },
    ])("$name", async ({ action, confirmed, mutation, calls }) => {
        // given
        answerConfirm(confirmed);
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.room[action]();
        });

        // then
        expect(mutation.mock.calls).toEqual(calls);
    });

    it("reports why leaving failed and stops looking busy", async () => {
        // given
        answerConfirm(true);
        mocks.leaveRoom.mockRejectedValue(new Error("hosts cannot leave"));
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.room.leave();
        });

        // then
        expect(result.current.toast.message).toBe("hosts cannot leave");
        expect(result.current.moderation.busy).toBeNull();
    });
});

describe("useRoomController muting", () => {
    it.each([
        { name: "mutes the room and says so", muted: false, toast: "Notifications muted" },
        { name: "unmutes a room that was already muted", muted: true, toast: "Notifications unmuted" },
    ])("$name", async ({ muted, toast }) => {
        // given
        const { result } = await renderLoadedRoom({ rooms: [makeRoom({ viewer_muted: muted })] });

        // when
        await act(async () => {
            await result.current.room.toggleMute();
        });

        // then
        expect(mocks.setMuted).toHaveBeenCalledWith({ roomId: "room-1", muted: !muted });
        expect(result.current.toast.message).toBe(toast);
        expect(result.current.room.data?.viewer_muted).toBe(!muted);
    });

    it("looks busy on the mute control while the server is still answering", async () => {
        // given
        let release: () => void = () => {};
        mocks.setMuted.mockReturnValue(
            new Promise<void>(resolve => {
                release = () => resolve();
            }),
        );
        const { result } = await renderLoadedRoom();

        // when
        let pending: Promise<void> = Promise.resolve();
        act(() => {
            pending = result.current.room.toggleMute();
        });
        const whileSaving = result.current.moderation.busy;
        await act(async () => {
            release();
            await pending;
        });

        // then
        expect(whileSaving).toBe("mute");
        expect(result.current.moderation.busy).toBeNull();
    });

    it("reports why the mute could not be changed", async () => {
        // given
        mocks.setMuted.mockRejectedValue(new Error("the server is asleep"));
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.room.toggleMute();
        });

        // then
        expect(result.current.toast.message).toBe("the server is asleep");
        expect(result.current.moderation.busy).toBeNull();
    });
});

describe("useRoomController toast", () => {
    it("clears the toast on its own after a few seconds", () => {
        // given
        vi.useFakeTimers();
        const { result } = renderRoom();
        act(() => {
            result.current.toast.show("something went wrong");
        });

        // when
        act(() => {
            vi.advanceTimersByTime(4000);
        });

        // then
        expect(result.current.toast.message).toBeNull();
    });
});
