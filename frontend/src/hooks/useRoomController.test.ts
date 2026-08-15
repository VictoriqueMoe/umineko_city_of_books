import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { NotificationContextValue, WSMessageHandler } from "../context/notificationContextValue";
import { makeUser } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import type {
    ChatMessage,
    ChatRoom,
    ChatRoomMember,
    User,
    UserProfile,
    WatchPartySession,
    WSMessage,
} from "../types/api";
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

vi.mock("../api/queries/chat", () => ({
    useUserRooms: mocks.useUserRooms,
    useChatRoomMembers: mocks.useChatRoomMembers,
    fetchRoomMessages: mocks.fetchRoomMessages,
    fetchRoomMessagesBefore: mocks.fetchRoomMessagesBefore,
}));

vi.mock("../api/mutations/chat", () => ({
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

vi.mock("../components/chat/WatchParty/useWatchParty", () => ({ useWatchParty: mocks.useWatchParty }));

vi.mock("../components/chat/Voice/useVoiceChat", () => ({ useVoiceChat: mocks.useVoiceChat }));

vi.mock("./usePresenceReporter", () => ({ usePresenceReporter: () => {} }));

vi.mock("../utils/sound", () => ({
    playMessageSound: mocks.playMessageSound,
    playRemoteAudio: mocks.playRemoteAudio,
}));

const viewer = makeUser({ id: "u1", username: "beatrice", display_name: "Beatrice" });

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return {
        id: "room-1",
        name: "Golden Land",
        description: "a place for tea",
        type: "group",
        is_public: true,
        is_rp: false,
        is_system: false,
        tags: [],
        viewer_muted: false,
        viewer_ghost: false,
        is_member: true,
        member_count: 2,
        hot_score: 0,
        members: [],
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

function makeRoomMember(overrides: Partial<ChatRoomMember> = {}): ChatRoomMember {
    return {
        user: { id: "u2", username: "battler", display_name: "Battler" },
        role: "member",
        joined_at: "2026-01-01T00:00:00Z",
        nickname: "",
        member_avatar_url: "",
        nickname_locked: false,
        ...overrides,
    };
}

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return {
        id: "m1",
        room_id: "room-1",
        sender: { id: "u2", username: "battler", display_name: "Battler" },
        body: "without love it cannot be seen",
        is_system: false,
        created_at: "2026-08-02T10:00:00Z",
        pinned: false,
        reactions: [],
        ...overrides,
    };
}

function makeSession(overrides: Partial<WatchPartySession> = {}): WatchPartySession {
    return {
        id: "party-1",
        room_id: "room-1",
        started_by: "u2",
        controller_id: "u2",
        title: "Higurashi rewatch",
        type: "hyperbeam",
        status: "active",
        started_at: "2026-08-02T09:00:00Z",
        participants: [],
        ...overrides,
    };
}

interface RoomHarnessOptions {
    user?: UserProfile | null;
    rooms?: ChatRoom[];
    roomsLoading?: boolean;
    members?: ChatRoomMember[];
    voiceParticipantIds?: string[];
    sessions?: WatchPartySession[];
    watchPartyLoaded?: boolean;
    route?: string;
    path?: string;
}

function renderRoom(options: RoomHarnessOptions = {}) {
    mocks.useUserRooms.mockReturnValue({
        rooms: options.rooms ?? [makeRoom()],
        loading: options.roomsLoading ?? false,
        refresh: mocks.roomsRefresh,
    });
    mocks.useChatRoomMembers.mockReturnValue({
        members: options.members ?? [makeRoomMember()],
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
        loaded: options.watchPartyLoaded ?? true,
        sessions: options.sessions ?? [],
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

    const handlers: WSMessageHandler[] = [];
    const unsubscribe = vi.fn();
    const addWSListener = vi.fn((handler: WSMessageHandler) => {
        handlers.push(handler);
        return unsubscribe;
    });
    const sendWSMessage = vi.fn();
    const notification: Partial<NotificationContextValue> = { addWSListener, sendWSMessage, wsEpoch: 0 };
    const wrapper = providerWrapper({
        user: options.user === undefined ? viewer : options.user,
        route: options.route ?? "/rooms/room-1",
        path: options.path ?? "/rooms/:roomId",
        notification,
    });
    const rendered = renderHook(() => useRoomController(), { wrapper });

    function emit(msg: WSMessage): void {
        act(() => {
            for (const handler of handlers.slice()) {
                handler(msg);
            }
        });
    }

    function reconnect(): void {
        notification.wsEpoch = (notification.wsEpoch ?? 0) + 1;
        rendered.rerender();
    }

    return { ...rendered, emit, reconnect, sendWSMessage, addWSListener, unsubscribe, notification };
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
    mocks.fetchRoomMessages.mockResolvedValue({ messages: [], total: 0 });
    mocks.fetchRoomMessagesBefore.mockResolvedValue({ messages: [], total: 0 });
    mocks.joinRoom.mockResolvedValue(makeRoom());
    mocks.leaveRoom.mockResolvedValue(undefined);
    mocks.deleteRoom.mockResolvedValue(undefined);
    mocks.setMuted.mockResolvedValue(undefined);
    mocks.kick.mockResolvedValue(undefined);
    mocks.ban.mockResolvedValue(undefined);
    mocks.setNickname.mockResolvedValue(makeRoomMember({ nickname: "Beato" }));
    mocks.unlockNickname.mockResolvedValue(makeRoomMember({ nickname_locked: false }));
    mocks.setMemberTimeout.mockResolvedValue(makeRoomMember({ timeout_until: "2099-01-01T00:00:00Z" }));
    mocks.clearMemberTimeout.mockResolvedValue(makeRoomMember({ timeout_until: undefined }));
    mocks.pin.mockResolvedValue(undefined);
    mocks.unpin.mockResolvedValue(undefined);
    mocks.addReaction.mockResolvedValue(undefined);
    mocks.removeReaction.mockResolvedValue(undefined);
    mocks.watchPartyJoin.mockResolvedValue(undefined);
    mocks.watchPartyRefresh.mockResolvedValue(undefined);
});

describe("useRoomController room loading", () => {
    it("shows the room the url points at", async () => {
        // given
        const options: RoomHarnessOptions = { rooms: [makeRoom(), makeRoom({ id: "room-2", name: "Purgatory" })] };

        // when
        const { result } = await renderLoadedRoom(options);

        // then
        expect(result.current.roomId).toBe("room-1");
        expect(result.current.room?.name).toBe("Golden Land");
    });

    it("reports itself as loading while the viewer's rooms are still on the way", () => {
        // given
        const options: RoomHarnessOptions = { rooms: [], roomsLoading: true };

        // when
        const { result } = renderRoom(options);

        // then
        expect(result.current.loading).toBe(true);
        expect(result.current.room).toBeNull();
    });

    it("has no room to show when the viewer does not belong to it", async () => {
        // given
        const options: RoomHarnessOptions = { rooms: [makeRoom({ id: "room-2" })] };

        // when
        const { result } = renderRoom(options);

        // then
        expect(result.current.room).toBeNull();
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

    it("joins the room over the socket and leaves it when the view closes", async () => {
        // given
        const { sendWSMessage, unmount } = await renderLoadedRoom();

        // when
        const sent = sendWSMessage.mock.calls.map(call => call[0]);
        unmount();

        // then
        expect(sent).toContainEqual({ type: "join_room", data: { room_id: "room-1" } });
        expect(sendWSMessage).toHaveBeenLastCalledWith({ type: "leave_room", data: { room_id: "room-1" } });
    });

    it("never listens to the socket while nobody is signed in", () => {
        // given
        const options: RoomHarnessOptions = { user: null };

        // when
        const { addWSListener } = renderRoom(options);

        // then
        expect(addWSListener).not.toHaveBeenCalled();
    });

    it("stops listening to the socket when the view closes", async () => {
        // given
        const { unmount, unsubscribe } = await renderLoadedRoom();

        // when
        unmount();

        // then
        expect(unsubscribe).toHaveBeenCalled();
    });

    it("refetches the backlog after the socket reconnects", async () => {
        // given
        const { reconnect } = await renderLoadedRoom();
        mocks.fetchRoomMessages.mockClear();

        // when
        act(() => {
            reconnect();
        });

        // then
        await waitFor(() => {
            expect(mocks.fetchRoomMessages).toHaveBeenCalledWith("room-1", 50);
        });
    });
});

describe("useRoomController member list", () => {
    it("groups members by rank with staff at the top", async () => {
        // given
        const options: RoomHarnessOptions = {
            members: [
                makeRoomMember({ user: { id: "u4", username: "ange", display_name: "Ange" } }),
                makeRoomMember({
                    user: { id: "u3", username: "lambda", display_name: "Lambda", role: "admin" },
                }),
                makeRoomMember({ user: { id: "u2", username: "battler", display_name: "Battler" }, role: "host" }),
                makeRoomMember({
                    user: { id: "u1", username: "beatrice", display_name: "Beatrice", role: "super_admin" },
                }),
            ],
        };

        // when
        const { result } = await renderLoadedRoom(options);

        // then
        expect(result.current.memberGroups.map(g => g.label)).toEqual([
            "Reality Author",
            "Host",
            "Voyager Witches",
            "Members",
        ]);
    });

    it("sorts people who are around ahead of people who are not", async () => {
        // given
        const options: RoomHarnessOptions = {
            members: [
                makeRoomMember({ user: { id: "u2", username: "battler", display_name: "Battler" } }),
                makeRoomMember({
                    user: { id: "u3", username: "ange", display_name: "Ange" },
                    presence: "active",
                }),
            ],
        };

        // when
        const { result } = await renderLoadedRoom(options);

        // then
        expect(result.current.memberGroups[0].members.map(m => m.user.id)).toEqual(["u3", "u2"]);
    });

    it("lists the people on the voice call in their own group", async () => {
        // given
        const options: RoomHarnessOptions = {
            members: [
                makeRoomMember({ user: { id: "u2", username: "battler", display_name: "Battler" } }),
                makeRoomMember({ user: { id: "u3", username: "ange", display_name: "Ange" } }),
            ],
            voiceParticipantIds: ["u3"],
        };

        // when
        const { result } = await renderLoadedRoom(options);

        // then
        expect(result.current.memberGroups[0].label).toBe("In Voice");
        expect(result.current.memberGroups[0].members.map(m => m.user.id)).toEqual(["u3"]);
    });

    it("finds the viewer's own membership", async () => {
        // given
        const options: RoomHarnessOptions = {
            members: [makeRoomMember({ user: { id: "u1", username: "beatrice", display_name: "Beatrice" } })],
        };

        // when
        const { result } = await renderLoadedRoom(options);

        // then
        expect(result.current.currentMember?.user.id).toBe("u1");
    });

    it("seeds the presence map from the membership the server sent", async () => {
        // given
        const options: RoomHarnessOptions = {
            members: [makeRoomMember({ presence: "idle" })],
        };

        // when
        const { result } = await renderLoadedRoom(options);

        // then
        expect(result.current.presenceMapMerged).toEqual({ u2: "idle" });
    });

    it("weighs a member who is nowhere to be seen below one who is present", async () => {
        // given
        const options: RoomHarnessOptions = { members: [makeRoomMember({ presence: "active" })] };

        // when
        const { result } = await renderLoadedRoom(options);

        // then
        expect(result.current.memberOnlineWeight("u2")).toBe(0);
        expect(result.current.memberOnlineWeight("ghost")).toBe(1);
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
        expect(result.current.viewerTimedOut).toBe(true);
        expect(result.current.viewerTimeoutUntil).toBe("2099-01-01T00:00:00Z");
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
        expect(result.current.viewerTimedOut).toBe(false);
    });

    it("shows nothing at all when there is no timeout to format", async () => {
        // given
        const { result } = await renderLoadedRoom();

        // when
        const formatted = result.current.formatTimeoutUntil(undefined);

        // then
        expect(formatted).toBe("");
    });

    it("hands back a timestamp it cannot parse untouched", async () => {
        // given
        const { result } = await renderLoadedRoom();

        // when
        const formatted = result.current.formatTimeoutUntil("not a date");

        // then
        expect(formatted).toBe("not a date");
    });
});

describe("useRoomController incoming messages", () => {
    it("shows a message that arrives for this room", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();

        // when
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // then
        expect(result.current.messages.map(m => m.id)).toEqual(["m1"]);
    });

    it("ignores a message meant for another room", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();

        // when
        emit({ type: "chat_message", data: makeMessage({ id: "m1", room_id: "room-2" }) });

        // then
        expect(result.current.messages).toEqual([]);
    });

    it("never shows the viewer's own message twice when the echo arrives", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();
        const own = makeMessage({ id: "m1", sender: { id: "u1", username: "beatrice", display_name: "Beatrice" } });
        act(() => {
            result.current.handleSentMessage(own);
        });

        // when
        emit({ type: "chat_message", data: own });

        // then
        expect(result.current.messages).toHaveLength(1);
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

    it("drops a message the server says was deleted", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // when
        emit({ type: "chat_message_deleted", data: { room_id: "room-1", message_id: "m1" } });

        // then
        expect(result.current.messages).toEqual([]);
    });

    it("applies an edit that arrives over the socket", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // when
        emit({
            type: "chat_message_edited",
            data: makeMessage({ id: "m1", body: "the red truth", edited_at: "2026-08-02T10:05:00Z" }),
        });

        // then
        expect(result.current.messages[0].body).toBe("the red truth");
    });
});

describe("useRoomController reactions and pins", () => {
    it("adds a reaction that arrives over the socket", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // when
        emit({
            type: "chat_reaction_added",
            data: { room_id: "room-1", message_id: "m1", emoji: "🌹", user_id: "u2", display_name: "Battler" },
        });

        // then
        expect(result.current.messages[0].reactions).toEqual([
            { emoji: "🌹", count: 1, viewer_reacted: false, display_names: ["Battler"] },
        ]);
    });

    it("marks a reaction the viewer added themselves", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // when
        emit({
            type: "chat_reaction_added",
            data: { room_id: "room-1", message_id: "m1", emoji: "🌹", user_id: "u1", display_name: "Beatrice" },
        });

        // then
        expect(result.current.messages[0].reactions[0].viewer_reacted).toBe(true);
    });

    it("removes the reaction group once the last reaction is taken back", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });
        emit({
            type: "chat_reaction_added",
            data: { room_id: "room-1", message_id: "m1", emoji: "🌹", user_id: "u2", display_name: "Battler" },
        });

        // when
        emit({
            type: "chat_reaction_removed",
            data: { room_id: "room-1", message_id: "m1", emoji: "🌹", user_id: "u2", display_name: "Battler" },
        });

        // then
        expect(result.current.messages[0].reactions).toEqual([]);
    });

    it("ignores a reaction aimed at another room", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // when
        emit({
            type: "chat_reaction_added",
            data: { room_id: "room-2", message_id: "m1", emoji: "🌹", user_id: "u2", display_name: "Battler" },
        });

        // then
        expect(result.current.messages[0].reactions).toEqual([]);
    });

    it("pins a message and asks the pinned panel to refresh", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });
        const refreshKeyBefore = result.current.pinnedRefreshKey;

        // when
        emit({
            type: "chat_message_pinned",
            data: { room_id: "room-1", message_id: "m1", pinned_at: "2026-08-02T11:00:00Z", pinned_by: "u2" },
        });

        // then
        expect(result.current.messages[0].pinned).toBe(true);
        expect(result.current.pinnedRefreshKey).toBe(refreshKeyBefore + 1);
    });

    it("unpins a message and asks the pinned panel to refresh", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();
        emit({ type: "chat_message", data: makeMessage({ id: "m1", pinned: true }) });

        // when
        emit({ type: "chat_message_unpinned", data: { room_id: "room-1", message_id: "m1" } });

        // then
        expect(result.current.messages[0].pinned).toBe(false);
    });

    it("asks the server to add a reaction the viewer has not left yet", async () => {
        // given
        const { result } = await renderLoadedRoom();
        const message = makeMessage({ id: "m1" });

        // when
        await act(async () => {
            await result.current.handleReactionToggle(message, "🌹");
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
            await result.current.handleReactionToggle(message, "🌹");
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
            await result.current.handleReactionToggle(makeMessage(), "🌹");
        });

        // then
        expect(result.current.toast).toBe("you are timed out");
    });

    it("pins a message the viewer chose to pin", async () => {
        // given
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.handlePinToggle(makeMessage({ id: "m1", pinned: false }));
        });

        // then
        expect(mocks.pin).toHaveBeenCalledWith("m1");
    });

    it("unpins a message that was already pinned", async () => {
        // given
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.handlePinToggle(makeMessage({ id: "m1", pinned: true }));
        });

        // then
        expect(mocks.unpin).toHaveBeenCalledWith("m1");
    });
});

describe("useRoomController membership events", () => {
    it("reloads the member list when somebody new arrives", async () => {
        // given
        const { emit } = await renderLoadedRoom({ rooms: [makeRoom({ member_count: 2 })] });

        // when
        emit({
            type: "chat_member_joined",
            data: { room_id: "room-1", user: { id: "u3", username: "ange", display_name: "Ange" } as User },
        });

        // then
        expect(mocks.membersRefresh).toHaveBeenCalled();
    });

    it("counts a new arrival on the room straight away", async () => {
        // given
        const { result, emit } = await renderLoadedRoom({ rooms: [makeRoom({ member_count: 2 })] });

        // when
        emit({
            type: "chat_member_joined",
            data: { room_id: "room-1", user: { id: "u3", username: "ange", display_name: "Ange" } as User },
        });

        // then
        expect(result.current.room?.member_count).toBe(3);
    });

    it("ignores somebody arriving in another room", async () => {
        // given
        const { emit } = await renderLoadedRoom();
        mocks.membersRefresh.mockClear();

        // when
        emit({
            type: "chat_member_joined",
            data: { room_id: "room-2", user: { id: "u3", username: "ange", display_name: "Ange" } as User },
        });

        // then
        expect(mocks.membersRefresh).not.toHaveBeenCalled();
    });

    it("drops somebody who left the room", async () => {
        // given
        const { result, emit } = await renderLoadedRoom({ rooms: [makeRoom({ member_count: 2 })] });

        // when
        emit({ type: "chat_member_left", data: { room_id: "room-1", user_id: "u2" } });

        // then
        expect(result.current.members).toEqual([]);
    });

    it("takes somebody who left off the room count straight away", async () => {
        // given
        const { result, emit } = await renderLoadedRoom({ rooms: [makeRoom({ member_count: 2 })] });

        // when
        emit({ type: "chat_member_left", data: { room_id: "room-1", user_id: "u2" } });

        // then
        expect(result.current.room?.member_count).toBe(1);
    });

    it("ignores somebody leaving another room", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();

        // when
        emit({ type: "chat_member_left", data: { room_id: "room-2", user_id: "u2" } });

        // then
        expect(result.current.members).toHaveLength(1);
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
        expect(result.current.members[0].nickname).toBe("Battler-kun");
        expect(result.current.messages[0].sender_nickname).toBe("Battler-kun");
    });

    it("tells the viewer when they are removed from the room", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();

        // when
        emit({ type: "chat_kicked", data: { room_id: "room-1", reason: "too much tea" } });

        // then
        expect(result.current.toast).toBe("You were removed from this room: too much tea");
    });

    it("tells the viewer when the host deletes the room", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();

        // when
        emit({ type: "chat_room_deleted", data: { room_id: "room-1" } });

        // then
        expect(result.current.toast).toBe("This room was deleted by the host");
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
        expect(result.current.room?.name).toBe("Purgatory");
        expect(result.current.room?.description).toBe("the seventh twilight");
        expect(result.current.room?.tags).toEqual(["beato", "seventh-twilight"]);
        expect(result.current.room?.is_public).toBe(false);
        expect(result.current.room?.is_rp).toBe(true);
        expect(result.current.room?.viewer_muted).toBe(true);
        expect(result.current.room?.member_count).toBe(7);
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
        expect(result.current.room?.name).toBe("Golden Land");
        expect(result.current.room?.description).toBe("a place for tea");
        expect(result.current.room?.tags).toEqual([]);
        expect(result.current.room?.is_public).toBe(true);
        expect(result.current.room?.is_rp).toBe(false);
    });

    it("records a presence change and forgets somebody who goes offline", async () => {
        // given
        const { result, emit } = await renderLoadedRoom({ members: [] });
        emit({ type: "chat_presence_changed", data: { room_id: "room-1", user_id: "u2", state: "active" } });
        const whilePresent = result.current.presenceMapMerged;

        // when
        emit({ type: "chat_presence_changed", data: { room_id: "room-1", user_id: "u2", state: "offline" } });

        // then
        expect(whilePresent).toEqual({ u2: "active" });
        expect(result.current.presenceMapMerged).toEqual({});
    });

    it("applies a site role change to the member list and to their messages", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });

        // when
        emit({ type: "role_changed", data: { user_id: "u2", role: "moderator" } });

        // then
        expect(result.current.members[0].user.role).toBe("moderator");
        expect(result.current.messages[0].sender.role).toBe("moderator");
    });
});

describe("useRoomController typing", () => {
    it("names the person typing in this room", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();

        // when
        emit({ type: "typing", data: { room_id: "room-1", user_id: "u2" } });

        // then
        expect(result.current.typingNames).toEqual(["Battler"]);
    });

    it("prefers the nickname the room gave somebody", async () => {
        // given
        const { result, emit } = await renderLoadedRoom({ members: [makeRoomMember({ nickname: "Battler-kun" })] });

        // when
        emit({ type: "typing", data: { room_id: "room-1", user_id: "u2" } });

        // then
        expect(result.current.typingNames).toEqual(["Battler-kun"]);
    });

    it("falls back to a placeholder for somebody the room does not list", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();

        // when
        emit({ type: "typing", data: { room_id: "room-1", user_id: "ghost" } });

        // then
        expect(result.current.typingNames).toEqual(["Someone"]);
    });

    it("never says the viewer is typing to themselves", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();

        // when
        emit({ type: "typing", data: { room_id: "room-1", user_id: "u1" } });

        // then
        expect(result.current.typingNames).toEqual([]);
    });
});

describe("useRoomController joining and leaving", () => {
    it("joins the room and refreshes the watch parties", async () => {
        // given
        const { result } = renderRoom({ rooms: [] });

        // when
        await act(async () => {
            await result.current.handleJoin();
        });

        // then
        expect(mocks.joinRoom).toHaveBeenCalledWith({ roomId: "room-1" });
        expect(mocks.watchPartyRefresh).toHaveBeenCalled();
        expect(result.current.joining).toBe(false);
    });

    it("shows the room it just joined", async () => {
        // given
        mocks.joinRoom.mockResolvedValue(makeRoom({ name: "Purgatory" }));
        const { result } = renderRoom({ rooms: [] });

        // when
        await act(async () => {
            await result.current.handleJoin();
        });

        // then
        expect(result.current.room?.name).toBe("Purgatory");
    });

    it("reports why joining failed", async () => {
        // given
        mocks.joinRoom.mockRejectedValue(new Error("this room is invite only"));
        const { result } = renderRoom({ rooms: [] });

        // when
        await act(async () => {
            await result.current.handleJoin();
        });

        // then
        expect(result.current.toast).toBe("this room is invite only");
    });

    it("keeps the viewer in the room when they back out of leaving", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.handleLeave();
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
            await result.current.handleLeave();
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
            await result.current.handleLeave();
        });

        // then
        expect(result.current.toast).toBe("hosts cannot leave");
        expect(result.current.busy).toBeNull();
        confirm.mockRestore();
    });

    it("deletes the room once the viewer confirms", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.handleDelete();
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
            await result.current.handleDelete();
        });

        // then
        expect(mocks.deleteRoom).not.toHaveBeenCalled();
        confirm.mockRestore();
    });
});

describe("useRoomController moderation", () => {
    it("kicks a member once the viewer confirms and drops them from the list", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.handleKick("u2");
        });

        // then
        expect(mocks.kick).toHaveBeenCalledWith("u2");
        expect(result.current.members).toEqual([]);
        confirm.mockRestore();
    });

    it("leaves the member alone when the viewer backs out of the kick", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.handleKick("u2");
        });

        // then
        expect(mocks.kick).not.toHaveBeenCalled();
        expect(result.current.members).toHaveLength(1);
        confirm.mockRestore();
    });

    it("bans a member with the reason the viewer typed", async () => {
        // given
        const prompt = vi.spyOn(window, "prompt").mockReturnValue("kept saying the same thing");
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.handleBan("u2");
        });

        // then
        expect(mocks.ban).toHaveBeenCalledWith({ userId: "u2", reason: "kept saying the same thing" });
        expect(result.current.toast).toBe("Member banned from the room.");
        prompt.mockRestore();
    });

    it("leaves the member alone when the viewer cancels the ban prompt", async () => {
        // given
        const prompt = vi.spyOn(window, "prompt").mockReturnValue(null);
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.handleBan("u2");
        });

        // then
        expect(mocks.ban).not.toHaveBeenCalled();
        expect(result.current.members).toHaveLength(1);
        prompt.mockRestore();
    });

    it("opens the nickname dialogue on the member the viewer picked", async () => {
        // given
        const member = makeRoomMember({ nickname: "Battler-kun" });
        const { result } = await renderLoadedRoom({ members: [member] });
        act(() => {
            result.current.setOpenMemberMenu("u2");
        });

        // when
        act(() => {
            result.current.openNicknameDialog(member);
        });

        // then
        expect(result.current.nicknameDialogTarget?.user.id).toBe("u2");
        expect(result.current.nicknameDialogValue).toBe("Battler-kun");
        expect(result.current.openMemberMenu).toBeNull();
    });

    it("saves a trimmed nickname and closes the dialogue", async () => {
        // given
        const member = makeRoomMember();
        const { result } = await renderLoadedRoom({ members: [member] });
        act(() => {
            result.current.openNicknameDialog(member);
        });
        act(() => {
            result.current.setNicknameDialogValue("  Beato  ");
        });

        // when
        await act(async () => {
            await result.current.handleModSetNickname();
        });

        // then
        expect(mocks.setNickname).toHaveBeenCalledWith({ userId: "u2", nickname: "Beato" });
        expect(result.current.nicknameDialogTarget).toBeNull();
    });

    it("keeps the nickname dialogue open and explains why the save failed", async () => {
        // given
        mocks.setNickname.mockRejectedValue(new Error("that nickname is taken"));
        const member = makeRoomMember();
        const { result } = await renderLoadedRoom({ members: [member] });
        act(() => {
            result.current.openNicknameDialog(member);
        });

        // when
        await act(async () => {
            await result.current.handleModSetNickname();
        });

        // then
        expect(result.current.nicknameDialogError).toBe("that nickname is taken");
        expect(result.current.nicknameDialogTarget?.user.id).toBe("u2");
        expect(result.current.nicknameDialogSaving).toBe(false);
    });

    it("unlocks a nickname the staff had frozen", async () => {
        // given
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.handleModUnlockNickname("u2");
        });

        // then
        expect(mocks.unlockNickname).toHaveBeenCalledWith("u2");
        expect(result.current.busy).toBeNull();
    });

    it("refuses a timeout that is not a whole number of units", async () => {
        // given
        const member = makeRoomMember();
        const { result } = await renderLoadedRoom({ members: [member] });
        act(() => {
            result.current.openTimeoutDialog(member);
        });
        act(() => {
            result.current.setTimeoutDialogAmount("0");
        });

        // when
        await act(async () => {
            await result.current.handleSetTimeout();
        });

        // then
        expect(result.current.timeoutDialogError).toBe("Enter a whole number greater than zero");
        expect(mocks.setMemberTimeout).not.toHaveBeenCalled();
    });

    it("times a member out for the amount and unit the viewer chose", async () => {
        // given
        const member = makeRoomMember();
        const { result } = await renderLoadedRoom({ members: [member] });
        act(() => {
            result.current.openTimeoutDialog(member);
        });
        act(() => {
            result.current.setTimeoutDialogAmount("3");
            result.current.setTimeoutDialogUnit("days");
        });

        // when
        await act(async () => {
            await result.current.handleSetTimeout();
        });

        // then
        expect(mocks.setMemberTimeout).toHaveBeenCalledWith({ userId: "u2", amount: 3, unit: "days" });
        expect(result.current.members[0].timeout_until).toBe("2099-01-01T00:00:00Z");
        expect(result.current.timeoutDialogTarget).toBeNull();
    });

    it("explains why a timeout could not be set", async () => {
        // given
        mocks.setMemberTimeout.mockRejectedValue(new Error("you cannot time out a host"));
        const member = makeRoomMember();
        const { result } = await renderLoadedRoom({ members: [member] });
        act(() => {
            result.current.openTimeoutDialog(member);
        });

        // when
        await act(async () => {
            await result.current.handleSetTimeout();
        });

        // then
        expect(result.current.timeoutDialogError).toBe("you cannot time out a host");
    });

    it("clears a timeout and puts the fresh member back in the list", async () => {
        // given
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.handleClearTimeout("u2");
        });

        // then
        expect(mocks.clearMemberTimeout).toHaveBeenCalledWith("u2");
        expect(result.current.members[0].timeout_until).toBeUndefined();
        expect(result.current.busy).toBeNull();
    });

    it("mutes the room and says so", async () => {
        // given
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.handleToggleMute();
        });

        // then
        expect(mocks.setMuted).toHaveBeenCalledWith({ roomId: "room-1", muted: true });
        expect(result.current.toast).toBe("Notifications muted");
        expect(result.current.room?.viewer_muted).toBe(true);
    });

    it("reports why the mute could not be changed", async () => {
        // given
        mocks.setMuted.mockRejectedValue(new Error("the server is asleep"));
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.handleToggleMute();
        });

        // then
        expect(result.current.toast).toBe("the server is asleep");
        expect(result.current.busy).toBeNull();
    });

    it("unmutes a room that was already muted", async () => {
        // given
        const { result } = await renderLoadedRoom({ rooms: [makeRoom({ viewer_muted: true })] });

        // when
        await act(async () => {
            await result.current.handleToggleMute();
        });

        // then
        expect(mocks.setMuted).toHaveBeenCalledWith({ roomId: "room-1", muted: false });
        expect(result.current.toast).toBe("Notifications unmuted");
        expect(result.current.room?.viewer_muted).toBe(false);
    });
});

describe("useRoomController jumping to a message", () => {
    it("highlights a message that is already on screen", async () => {
        // given
        const { result, emit } = await renderLoadedRoom();
        emit({ type: "chat_message", data: makeMessage({ id: "m1" }) });
        const element = document.createElement("div");
        element.id = "chat-msg-m1";
        document.body.appendChild(element);

        // when
        await act(async () => {
            await result.current.handleJumpToMessage("m1");
        });

        // then
        expect(result.current.highlightedMsgId).toBe("m1");
        element.remove();
    });

    it("says so when the message cannot be found anywhere in the history", async () => {
        // given
        const { result } = await renderLoadedRoom();

        // when
        await act(async () => {
            await result.current.handleJumpToMessage("ghost");
        });

        // then
        expect(result.current.toast).toBe("Couldn't locate that message.");
    });
});

describe("useRoomController view preferences", () => {
    it("remembers that the viewer collapsed the sidebar", async () => {
        // given
        const { result } = await renderLoadedRoom();

        // when
        act(() => {
            result.current.toggleSidebar();
        });

        // then
        expect(result.current.sidebarCollapsed).toBe(true);
        expect(localStorage.getItem("ut-room-sidebar-collapsed-room-1")).toBe("1");
    });

    it("opens the sidebar again and forgets the preference", async () => {
        // given
        localStorage.setItem("ut-room-sidebar-collapsed-room-1", "1");
        const { result } = await renderLoadedRoom();

        // when
        act(() => {
            result.current.toggleSidebar();
        });

        // then
        expect(result.current.sidebarCollapsed).toBe(false);
        expect(localStorage.getItem("ut-room-sidebar-collapsed-room-1")).toBeNull();
    });

    it("remembers that the viewer expanded the room description", async () => {
        // given
        const { result } = await renderLoadedRoom();

        // when
        act(() => {
            result.current.toggleDescExpanded();
        });

        // then
        expect(result.current.descExpanded).toBe(true);
        expect(localStorage.getItem("roomInfoExpanded:room-1")).toBe("true");
    });

    it("honours a stored description preference over the screen size", async () => {
        // given
        localStorage.setItem("roomInfoExpanded:room-1", "true");

        // when
        const { result } = await renderLoadedRoom();

        // then
        expect(result.current.descExpanded).toBe(true);
    });

    it("clears the toast on its own after a few seconds", async () => {
        // given
        vi.useFakeTimers();
        const { result } = renderRoom();
        act(() => {
            result.current.setToast("something went wrong");
        });

        // when
        act(() => {
            vi.advanceTimersByTime(4000);
        });

        // then
        expect(result.current.toast).toBeNull();
    });
});

describe("useRoomController watch party invites", () => {
    it("opens the watch party the invite link points at", async () => {
        // given
        const options: RoomHarnessOptions = {
            route: "/rooms/room-1?party=party-1",
            sessions: [makeSession()],
        };

        // when
        await renderLoadedRoom(options);

        // then
        await waitFor(() => {
            expect(mocks.watchPartyJoin).toHaveBeenCalledWith("party-1");
        });
    });

    it("reports an invite that points at a watch party which has ended", async () => {
        // given
        const options: RoomHarnessOptions = { route: "/rooms/room-1?party=party-1", sessions: [] };

        // when
        const { result } = await renderLoadedRoom(options);

        // then
        expect(result.current.invitedPartyMissing).toBe(true);
        expect(mocks.watchPartyJoin).not.toHaveBeenCalled();
    });

    it("says nothing about invites while the watch parties are still loading", async () => {
        // given
        const options: RoomHarnessOptions = {
            route: "/rooms/room-1?party=party-1",
            sessions: [],
            watchPartyLoaded: false,
        };

        // when
        const { result } = await renderLoadedRoom(options);

        // then
        expect(result.current.invitedPartyMissing).toBe(false);
    });
});
