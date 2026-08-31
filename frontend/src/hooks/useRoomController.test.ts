import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type * as BusModule from "../api/realtime/bus";
import type * as OutboundModule from "../api/realtime/outbound";
import { queryKeys } from "../api/queryKeys";
import type { QueryClient } from "@tanstack/react-query";
import { useLocation } from "react-router";
import { makeChatMessage, makeChatRoom, makeDmRoom, makeRoomMember, makeUser } from "../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../test-utils/render";
import { makeWSHarness, type RealtimeTestEvent, type RealtimeTestNames, type WSHarness } from "../test-utils/ws";
import type { ChatMessage, ChatRoom, ChatRoomMember, User, UserProfile } from "../types/api";
import { useRoomController } from "./useRoomController";

const mocks = vi.hoisted(() => ({
    useUserRooms: vi.fn(),
    useChatRoomMembers: vi.fn(),
    fetchRoomMessages: vi.fn(),
    fetchRoomMessagesBefore: vi.fn(),
    roomsRefresh: vi.fn(),
    membersRefresh: vi.fn(),
    markRead: vi.fn(),
    joinRoom: vi.fn(),
    leaveRoom: vi.fn(),
    deleteRoom: vi.fn(),
    setMuted: vi.fn(),
    kick: vi.fn(),
    ban: vi.fn(),
    setNickname: vi.fn(),
    unlockNickname: vi.fn(),
    setMemberTimeout: vi.fn(),
    clearMemberTimeout: vi.fn(),
    pin: vi.fn(),
    unpin: vi.fn(),
    addReaction: vi.fn(),
    removeReaction: vi.fn(),
    deleteMessage: vi.fn(),
    editMessage: vi.fn(),
    useWatchParty: vi.fn(),
    useVoiceChat: vi.fn(),
    watchPartyJoin: vi.fn(),
    watchPartyRefresh: vi.fn(),
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

vi.mock("./queries/chat", async importOriginal => {
    const actual = await importOriginal<Record<string, unknown>>();

    return {
        ...actual,
        useUserRooms: mocks.useUserRooms,
        useChatRoomMembers: mocks.useChatRoomMembers,
        fetchRoomMessages: mocks.fetchRoomMessages,
        fetchRoomMessagesBefore: mocks.fetchRoomMessagesBefore,
    };
});

vi.mock("./mutations/chat", () => ({
    useMarkChatRoomRead: () => ({ mutate: mocks.markRead, mutateAsync: mocks.markRead }),
    useJoinChatRoom: () => ({ mutateAsync: mocks.joinRoom }),
    useLeaveChatRoom: () => ({ mutateAsync: mocks.leaveRoom }),
    useDeleteChatRoom: () => ({ mutateAsync: mocks.deleteRoom }),
    useSetChatRoomMuted: () => ({ mutateAsync: mocks.setMuted }),
    useKickChatRoomMember: () => ({ mutateAsync: mocks.kick }),
    useBanChatRoomMember: () => ({ mutateAsync: mocks.ban }),
    useSetChatRoomMemberNickname: () => ({ mutateAsync: mocks.setNickname }),
    useUnlockChatRoomMemberNickname: () => ({ mutateAsync: mocks.unlockNickname }),
    useSetChatRoomMemberTimeout: () => ({ mutateAsync: mocks.setMemberTimeout }),
    useClearChatRoomMemberTimeout: () => ({ mutateAsync: mocks.clearMemberTimeout }),
    usePinChatMessage: () => ({ mutateAsync: mocks.pin }),
    useUnpinChatMessage: () => ({ mutateAsync: mocks.unpin }),
    useAddChatMessageReaction: () => ({ mutateAsync: mocks.addReaction }),
    useRemoveChatMessageReaction: () => ({ mutateAsync: mocks.removeReaction }),
    useDeleteChatMessage: () => ({ mutateAsync: mocks.deleteMessage }),
    useEditChatMessage: () => ({ mutateAsync: mocks.editMessage }),
}));

vi.mock("./useWatchParty", () => ({ useWatchParty: mocks.useWatchParty }));

vi.mock("./useVoiceChat", () => ({ useVoiceChat: mocks.useVoiceChat }));

vi.mock("./usePresenceReporter", () => ({ usePresenceReporter: () => {} }));

vi.mock("../platform/sound", () => ({
    playMessageSound: mocks.playMessageSound,
    playRemoteAudio: mocks.playRemoteAudio,
}));

const viewer = makeUser({ id: "u1", username: "beatrice", display_name: "Beatrice" });

const battler = { id: "u2", username: "battler", display_name: "Battler" };

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
    voiceParticipantIds?: string[];
    route?: string;
    path?: string;
}

function primeRoomMocks(options: RoomHarnessOptions = {}) {
    mocks.useUserRooms.mockReturnValue({
        rooms: options.rooms ?? [makeRoom()],
        loading: options.roomsLoading ?? false,
        refresh: mocks.roomsRefresh,
    });
    mocks.useChatRoomMembers.mockReturnValue({
        members: options.members ?? [makeRoomMember({ user: battler })],
        loading: false,
        refresh: mocks.membersRefresh,
    });
    mocks.useVoiceChat.mockReturnValue({
        status: "idle",
        room: null,
        participantIds: options.voiceParticipantIds ?? [],
        join: () => {},
        leave: () => {},
    });
    mocks.useWatchParty.mockReturnValue({
        enabled: false,
        screenShareEnabled: false,
        loaded: true,
        sessions: [],
        activeSession: null,
        openSessionId: null,
        error: null,
        refresh: mocks.watchPartyRefresh,
        start: () => Promise.resolve(null),
        join: mocks.watchPartyJoin,
        leave: () => Promise.resolve(),
        end: () => Promise.resolve(),
        transferControl: () => Promise.resolve(),
        kick: () => Promise.resolve(),
        identify: () => Promise.resolve(),
        openExisting: () => {},
        close: () => {},
        clearError: () => {},
    });
}

function roomWrapper(options: RoomHarnessOptions, queryClient: QueryClient) {
    return providerWrapper({
        user: options.user === undefined ? viewer : options.user,
        route: options.route ?? "/rooms/room-1",
        path: options.path ?? "/rooms/:roomId",
        queryClient,
    });
}

function renderRoom(options: RoomHarnessOptions = {}) {
    primeRoomMocks(options);

    const queryClient = createTestQueryClient();
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const wrapper = roomWrapper(options, queryClient);
    const rendered = renderHook(() => useRoomController(), { wrapper });

    function emit(event: RealtimeTestEvent): void {
        holder.ws.emit(event);
    }

    return {
        ...rendered,
        emit,
        reconnect: holder.ws.reconnect,
        sendRealtime: holder.ws.sendRealtime,
        invalidateQueries,
    };
}

function renderRoomWithLocation(options: RoomHarnessOptions = {}) {
    primeRoomMocks(options);

    const wrapper = roomWrapper(options, createTestQueryClient());

    return renderHook(() => ({ room: useRoomController(), pathname: useLocation().pathname }), { wrapper });
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

beforeEach(() => {
    holder.ws = makeWSHarness();
    mocks.fetchRoomMessages.mockResolvedValue({ messages: [], total: 0 });
    mocks.fetchRoomMessagesBefore.mockResolvedValue({ messages: [], total: 0 });
    mocks.joinRoom.mockResolvedValue(makeRoom());
    mocks.leaveRoom.mockResolvedValue(undefined);
    mocks.deleteRoom.mockResolvedValue(undefined);
    mocks.setMuted.mockResolvedValue(undefined);
    mocks.pin.mockResolvedValue(undefined);
    mocks.unpin.mockResolvedValue(undefined);
    mocks.addReaction.mockResolvedValue(undefined);
    mocks.removeReaction.mockResolvedValue(undefined);
    mocks.watchPartyRefresh.mockResolvedValue(undefined);
});

describe("useRoomController room loading", () => {
    it("shows the room the url points at", async () => {
        // given
        const options: RoomHarnessOptions = { rooms: [makeRoom(), makeRoom({ id: "room-2", name: "Purgatory" })] };

        // when
        const { result } = await renderLoadedRoom(options);

        // then
        expect(result.current.room.id).toBe("room-1");
        expect(result.current.room.data?.name).toBe("Golden Land");
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

    it("sends a dm reached through the rooms url on to the dm page", async () => {
        // given
        const options: RoomHarnessOptions = {
            rooms: [makeDmRoom({ id: "room-1" })],
            route: "/rooms/room-1",
            path: "/:section/:roomId",
        };

        // when
        const { result } = renderRoomWithLocation(options);

        // then
        await waitFor(() => {
            expect(result.current.pathname).toBe("/chat/room-1");
        });
    });

    it("leaves a group room on the rooms url", async () => {
        // given
        const options: RoomHarnessOptions = {
            rooms: [makeRoom({ id: "room-1" })],
            route: "/rooms/room-1",
            path: "/:section/:roomId",
        };

        // when
        const { result } = renderRoomWithLocation(options);
        await waitFor(() => {
            expect(result.current.room.room.data).not.toBeNull();
        });

        // then
        expect(result.current.pathname).toBe("/rooms/room-1");
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

    it("marks the room read once for a given last message", async () => {
        // given
        const options: RoomHarnessOptions = { rooms: [makeRoom({ last_message_at: "2026-08-02T10:00:00Z" })] };

        // when
        const { rerender } = await renderLoadedRoom(options);
        act(() => {
            rerender();
        });

        // then
        expect(mocks.markRead).toHaveBeenCalledExactlyOnceWith("room-1");
    });

    it("describes the room on screen as a group, with its own capabilities", async () => {
        // given
        const options: RoomHarnessOptions = { rooms: [makeRoom()] };

        // when
        const { result } = await renderLoadedRoom(options);

        // then
        expect(result.current.capabilities.kind).toBe("group");
        expect(result.current.capabilities.readReceipts).toBe("none");
        expect(result.current.capabilities.archivesWhenStale).toBe(true);
    });

    it("marks the room read again when the window regains focus", async () => {
        // given
        await renderLoadedRoom();
        mocks.markRead.mockClear();

        // when
        act(() => {
            window.dispatchEvent(new Event("focus"));
        });

        // then
        expect(mocks.markRead).toHaveBeenCalledExactlyOnceWith("room-1");
    });

    it("joins the room over the socket and leaves it when the view closes", async () => {
        // given
        const { sendRealtime, unmount } = await renderLoadedRoom();

        // when
        const sent = sendRealtime.mock.calls.map(call => call[0]);
        unmount();

        // then
        expect(sent).toContainEqual({ type: "join_room", data: { room_id: "room-1" } });
        expect(sendRealtime).toHaveBeenLastCalledWith({ type: "leave_room", data: { room_id: "room-1" } });
    });

    it("ignores everything on the socket while nobody is signed in", () => {
        // given
        const { result, emit } = renderRoom({ user: null });

        // when
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // then
        expect(result.current.session.messages).toEqual([]);
    });

    it("stops listening to the socket when the view closes", async () => {
        // given
        const { unmount } = await renderLoadedRoom();

        // when
        unmount();

        // then
        expect(holder.ws.unsubscribe).toHaveBeenCalled();
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
    it("treats a timeout that has not expired as still in force", async () => {
        // given
        const options: RoomHarnessOptions = {
            members: [
                makeRoomMember({
                    user: { id: "u1", username: "beatrice", display_name: "Beatrice" },
                    timeout_until: "2099-01-01T00:00:00Z",
                }),
            ],
        };

        // when
        const { result } = await renderLoadedRoom(options);

        // then
        expect(result.current.room.viewerTimedOut).toBe(true);
        expect(result.current.room.viewerTimeoutUntil).toBe("2099-01-01T00:00:00Z");
    });

    it("refuses to reopen the viewer's last message while they are timed out", async () => {
        // given
        const options: RoomHarnessOptions = {
            members: [
                makeRoomMember({
                    user: { id: "u1", username: "beatrice", display_name: "Beatrice" },
                    timeout_until: "2099-01-01T00:00:00Z",
                }),
            ],
        };
        const { result, emit } = await renderLoadedRoom(options);
        emit({
            type: "chat_message",
            data: makeMessage({ id: "m1", sender: { id: "u1", username: "beatrice", display_name: "Beatrice" } }),
        });

        // when
        act(() => {
            result.current.session.editLast();
        });

        // then
        expect(result.current.session.editingMessageId).toBeNull();
    });

    it("reopens the viewer's last message once no timeout is in force", async () => {
        // given
        const options: RoomHarnessOptions = {
            members: [makeRoomMember({ user: { id: "u1", username: "beatrice", display_name: "Beatrice" } })],
        };
        const { result, emit } = await renderLoadedRoom(options);
        emit({
            type: "chat_message",
            data: makeMessage({ id: "m1", sender: { id: "u1", username: "beatrice", display_name: "Beatrice" } }),
        });

        // when
        act(() => {
            result.current.session.editLast();
        });

        // then
        expect(result.current.session.editingMessageId).toBe("m1");
    });

    it("treats an expired timeout as over", async () => {
        // given
        const options: RoomHarnessOptions = {
            members: [
                makeRoomMember({
                    user: { id: "u1", username: "beatrice", display_name: "Beatrice" },
                    timeout_until: "2020-01-01T00:00:00Z",
                }),
            ],
        };

        // when
        const { result } = await renderLoadedRoom(options);

        // then
        expect(result.current.room.viewerTimedOut).toBe(false);
    });
});

describe("useRoomController incoming messages", () => {
    it("never shows the viewer's own message twice when the echo arrives", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();
        const own = makeMessage({ id: "m1", sender: { id: "u1", username: "beatrice", display_name: "Beatrice" } });
        act(() => {
            result.current.session.onSent(own);
        });

        // when
        emit({ type: "chat_message", data: own });

        // then
        expect(result.current.session.messages).toHaveLength(1);
    });

    it("plays a sound for somebody else's message while the tab is in the background", async () => {
        // given
        Object.defineProperty(document, "visibilityState", { configurable: true, get: () => "hidden" });
        const { emit } = await renderLoadedRoom();

        // when
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // then
        expect(mocks.playMessageSound).toHaveBeenCalled();
        Reflect.deleteProperty(document, "visibilityState");
    });

    it("stays silent while the room is muted", async () => {
        // given
        Object.defineProperty(document, "visibilityState", { configurable: true, get: () => "hidden" });
        const { emit } = await renderLoadedRoom({ rooms: [makeRoom({ viewer_muted: true })] });

        // when
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // then
        expect(mocks.playMessageSound).not.toHaveBeenCalled();
        Reflect.deleteProperty(document, "visibilityState");
    });
});

describe("useRoomController reactions and pins", () => {
    it("pins a message and stales the pinned panel's query", async () => {
        // given
        const { result, emit, invalidateQueries } = await renderLoadedRoom();
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });
        invalidateQueries.mockClear();

        // when
        emit({
            type: "chat_message_pinned",
            data: { room_id: "room-1", message_id: "m1", pinned_at: "2026-08-02T11:00:00Z", pinned_by: "u2" },
        });

        // then
        expect(result.current.session.messages[0].pinned).toBe(true);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.chat.pinned("room-1") });
    });

    it("unpins a message and stales the pinned panel's query", async () => {
        // given
        const { result, emit, invalidateQueries } = await renderLoadedRoom();
        emit({ type: "chat_message", data: makeMessage({ id: "m1", pinned: true }) });
        invalidateQueries.mockClear();

        // when
        emit({ type: "chat_message_unpinned", data: { room_id: "room-1", message_id: "m1" } });

        // then
        expect(result.current.session.messages[0].pinned).toBe(false);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.chat.pinned("room-1") });
    });

    it("asks the server to add a reaction the viewer has not left yet", async () => {
        // given
        const { result } = await renderLoadedRoom();
        const message = makeMessage({ id: "m1" });

        // when
        await act(async () => {
            await result.current.session.toggleReaction(message, "🌹");
        });

        // then
        expect(mocks.addReaction).toHaveBeenCalledWith({ messageId: "m1", emoji: "🌹" });
    });

    it("takes back a reaction the viewer already left", async () => {
        // given
        const { result } = await renderLoadedRoom();
        const message = makeMessage({
            id: "m1",
            reactions: [{ emoji: "🌹", count: 1, viewer_reacted: true, display_names: ["Beatrice"] }],
        });

        // when
        await act(async () => {
            await result.current.session.toggleReaction(message, "🌹");
        });

        // then
        expect(mocks.removeReaction).toHaveBeenCalledWith({ messageId: "m1", emoji: "🌹" });
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

    it("pins a message the viewer chose to pin", async () => {
        // given
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.session.togglePin(makeMessage({ id: "m1", pinned: false }));
        });

        // then
        expect(mocks.pin).toHaveBeenCalledWith("m1");
    });

    it("unpins a message that was already pinned", async () => {
        // given
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.session.togglePin(makeMessage({ id: "m1", pinned: true }));
        });

        // then
        expect(mocks.unpin).toHaveBeenCalledWith("m1");
    });
});

describe("useRoomController membership events", () => {
    it("counts a new arrival on the room straight away", async () => {
        // given
        const { result, emit } = await renderLoadedRoom({ rooms: [makeRoom({ member_count: 2 })] });

        // when
        emit({
            type: "chat_member_joined",
            data: { room_id: "room-1", user: { id: "u3", username: "ange", display_name: "Ange" } as User },
        });

        // then
        expect(result.current.room.data?.member_count).toBe(3);
    });

    it("takes somebody who left off the room count straight away", async () => {
        // given
        const { result, emit } = await renderLoadedRoom({ rooms: [makeRoom({ member_count: 2 })] });

        // when
        emit({ type: "chat_member_left", data: { room_id: "room-1", user_id: "u2" } });

        // then
        expect(result.current.room.data?.member_count).toBe(1);
    });

    it("applies a nickname change to the member list and to their messages", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // when
        emit({
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
        });

        // then
        expect(result.current.members.list[0].nickname).toBe("Battler-kun");
        expect(result.current.session.messages[0].sender_nickname).toBe("Battler-kun");
    });

    it("tells the viewer when they are removed from the room", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();

        // when
        emit({ type: "chat_kicked", data: { room_id: "room-1", reason: "too much tea" } });

        // then
        expect(result.current.toast.message).toBe("You were removed from this room: too much tea");
    });

    it("tells the viewer when the host deletes the room", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();

        // when
        emit({ type: "chat_room_deleted", data: { room_id: "room-1" } });

        // then
        expect(result.current.toast.message).toBe("This room was deleted by the host");
    });

    it("patches the room in place when its settings are edited", async () => {
        // given
        const { result, emit } = await renderLoadedRoom({
            rooms: [makeRoom({ tags: ["beato"], viewer_muted: true, member_count: 7 })],
        });

        // when
        emit({
            type: "chat_room_updated",
            data: {
                room_id: "room-1",
                name: "Purgatory",
                description: "the seventh twilight",
                tags: ["beato", "seventh-twilight"],
                is_public: false,
                is_rp: true,
            },
        });

        // then
        expect(result.current.room.data?.name).toBe("Purgatory");
        expect(result.current.room.data?.description).toBe("the seventh twilight");
        expect(result.current.room.data?.tags).toEqual(["beato", "seventh-twilight"]);
        expect(result.current.room.data?.is_public).toBe(false);
        expect(result.current.room.data?.is_rp).toBe(true);
        expect(result.current.room.data?.viewer_muted).toBe(true);
        expect(result.current.room.data?.member_count).toBe(7);
    });

    it("ignores an edit to another room", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();

        // when
        emit({
            type: "chat_room_updated",
            data: {
                room_id: "room-2",
                name: "Purgatory",
                description: "the seventh twilight",
                tags: ["seventh-twilight"],
                is_public: false,
                is_rp: true,
            },
        });

        // then
        expect(result.current.room.data?.name).toBe("Golden Land");
        expect(result.current.room.data?.description).toBe("a place for tea");
        expect(result.current.room.data?.tags).toEqual([]);
        expect(result.current.room.data?.is_public).toBe(true);
        expect(result.current.room.data?.is_rp).toBe(false);
    });

    it("applies a site role change to the member list and to their messages", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // when
        emit({ type: "role_changed", data: { user_id: "u2", role: "moderator" } });

        // then
        expect(result.current.members.list[0].user.role).toBe("moderator");
        expect(result.current.session.messages[0].sender.role).toBe("moderator");
    });
});

describe("useRoomController typing", () => {
    it("routes a typing broadcast for this room into the roster", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();

        // when
        emit({ type: "typing", data: { room_id: "room-1", user_id: "u2" } });

        // then
        expect(result.current.session.typingNames).toEqual(["Battler"]);
    });

    it("ignores a typing broadcast meant for another room", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();

        // when
        emit({ type: "typing", data: { room_id: "room-2", user_id: "u2" } });

        // then
        expect(result.current.session.typingNames).toEqual([]);
    });
});

describe("useRoomController joining and leaving", () => {
    it("joins the room and refreshes the watch parties", async () => {
        // given
        const { result } = renderRoom({ rooms: [] });

        // when
        await act(async () => {
            await result.current.room.join();
        });

        // then
        expect(mocks.joinRoom).toHaveBeenCalledWith({ roomId: "room-1" });
        expect(mocks.watchPartyRefresh).toHaveBeenCalled();
        expect(result.current.room.joining).toBe(false);
    });

    it("shows the room it just joined", async () => {
        // given
        mocks.joinRoom.mockResolvedValue(makeRoom({ name: "Purgatory" }));
        const { result } = renderRoom({ rooms: [] });

        // when
        await act(async () => {
            await result.current.room.join();
        });

        // then
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

    it("keeps the viewer in the room when they back out of leaving", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.room.leave();
        });

        // then
        expect(mocks.leaveRoom).not.toHaveBeenCalled();
        confirm.mockRestore();
    });

    it("leaves the room once the viewer confirms", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.room.leave();
        });

        // then
        expect(mocks.leaveRoom).toHaveBeenCalledWith("room-1");
        confirm.mockRestore();
    });

    it("reports why leaving failed and stops looking busy", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        mocks.leaveRoom.mockRejectedValue(new Error("hosts cannot leave"));
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.room.leave();
        });

        // then
        expect(result.current.toast.message).toBe("hosts cannot leave");
        expect(result.current.moderation.busy).toBeNull();
        confirm.mockRestore();
    });

    it("deletes the room once the viewer confirms", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.room.remove();
        });

        // then
        expect(mocks.deleteRoom).toHaveBeenCalledWith("room-1");
        confirm.mockRestore();
    });

    it("keeps the room when the viewer backs out of deleting it", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.room.remove();
        });

        // then
        expect(mocks.deleteRoom).not.toHaveBeenCalled();
        confirm.mockRestore();
    });
});

describe("useRoomController muting", () => {
    it("mutes the room and says so", async () => {
        // given
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.room.toggleMute();
        });

        // then
        expect(mocks.setMuted).toHaveBeenCalledWith({ roomId: "room-1", muted: true });
        expect(result.current.toast.message).toBe("Notifications muted");
        expect(result.current.room.data?.viewer_muted).toBe(true);
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

    it("unmutes a room that was already muted", async () => {
        // given
        const { result } = await renderLoadedRoom({ rooms: [makeRoom({ viewer_muted: true })] });

        // when
        await act(async () => {
            await result.current.room.toggleMute();
        });

        // then
        expect(mocks.setMuted).toHaveBeenCalledWith({ roomId: "room-1", muted: false });
        expect(result.current.toast.message).toBe("Notifications unmuted");
        expect(result.current.room.data?.viewer_muted).toBe(false);
    });
});

describe("useRoomController toast", () => {
    it("clears the toast on its own after a few seconds", async () => {
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
