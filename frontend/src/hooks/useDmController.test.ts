import { act, renderHook, waitFor } from "@testing-library/react";
import { useLocation, useNavigate } from "react-router";
import { beforeEach, describe, expect, it, onTestFinished, vi, type Mock } from "vitest";
import { queryKeys } from "../api/queryKeys";
import type * as BusModule from "../api/realtime/bus";
import type * as OutboundModule from "../api/realtime/outbound";
import type { RoomCapabilities } from "../domain/chat/roomPolicy";
import { makeChatMessage, makeChatRoom, makeDmRoom, makePublicUser, makeUser } from "../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../test-utils/render";
import { makeWSHarness, type RealtimeTestEvent, type RealtimeTestNames, type WSHarness } from "../test-utils/ws";
import type { ChatMessage, ChatRoom, User, UserProfile } from "../types/api";
import type * as ChatMutations from "./mutations/chat";
import type * as ChatQueries from "./queries/chat";
import { useDmController, type DmController } from "./useDmController";

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
    playAudio: vi.fn(),
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
    playAudio: mocks.playAudio,
}));

const viewer = makeUser({ id: "u1", username: "beatrice", display_name: "Beatrice" });
const viewerAsSender = makePublicUser();

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
}

async function settle(): Promise<void> {
    await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
    });
}

function renderDm(options: HarnessOptions = {}) {
    const queryClient = createTestQueryClient();
    const wrapper = providerWrapper({
        user: options.user === undefined ? viewer : options.user,
        route: options.route ?? "/chat/room-1",
        path: "/chat/:roomId?",
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

function sentCommands(types: readonly OutboundModule.RealtimeCommandName[]): OutboundModule.RealtimeCommand[] {
    return holder.ws.sendRealtime.mock.calls.map(call => call[0]).filter(command => types.includes(command.type));
}

function hideTab(): void {
    Object.defineProperty(document, "visibilityState", { configurable: true, get: () => "hidden" });
    onTestFinished(() => {
        Reflect.deleteProperty(document, "visibilityState");
    });
}

function answerConfirmation(answer: boolean): void {
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(answer);
    onTestFinished(() => {
        confirm.mockRestore();
    });
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
    interface ScreenCase {
        name: string;
        route: string;
        activeRoomId: string | null;
        mobileView: "list" | "room";
        capabilities: Partial<RoomCapabilities>;
        title: string;
        absentTitle: string;
        viewerStates: OutboundModule.RealtimeCommand[];
    }

    const screenCases: ScreenCase[] = [
        {
            name: "a conversation",
            route: "/chat/room-1",
            activeRoomId: "room-1",
            mobileView: "room",
            capabilities: { kind: "pair", readReceipts: "pairwise", canEditSettings: false },
            title: "Battler",
            absentTitle: "Chat",
            viewerStates: [{ type: "viewer_state", data: { room_id: "room-1", state: "active" } }],
        },
        {
            name: "no conversation",
            route: "/chat",
            activeRoomId: null,
            mobileView: "list",
            capabilities: { kind: "group", readReceipts: "none", canEditSettings: false },
            title: "Chat",
            absentTitle: "Battler",
            viewerStates: [],
        },
    ];

    it.each(screenCases)(
        "opens, describes, titles and reports $name as the url names it",
        async ({ route, activeRoomId, mobileView, capabilities, title, absentTitle, viewerStates }) => {
            // given
            const options: HarnessOptions = { route };

            // when
            const { result } = await renderLoadedDm(options);

            // then
            expect(result.current.activeRoomId).toBe(activeRoomId);
            expect(result.current.activeRoom?.id).toBe(activeRoomId ?? undefined);
            expect(result.current.mobileView).toBe(mobileView);
            expect(result.current.capabilities).toMatchObject(capabilities);
            expect(document.title).toContain(title);
            expect(document.title).not.toContain(absentTitle);
            expect(sentCommands(["viewer_state"])).toEqual(viewerStates);
        },
    );

    it("never asks for conversations while nobody is signed in", () => {
        // given
        const options: HarnessOptions = { user: null };

        // when
        const { result } = renderDm(options);

        // then
        expect(mocks.getUserRooms).not.toHaveBeenCalled();
        expect(result.current.loading).toBe(true);
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

    it("keeps group rooms off the list and leaves the list alone, asking for nothing, while they are busy", async () => {
        // given
        mocks.getUserRooms.mockResolvedValue({
            rooms: [makeRoom(), makeRoom({ id: "room-2" }), makeGroupRoom()],
        });
        const { result, emit } = await renderLoadedDm();
        const listed = result.current.rooms.map(r => r.id);
        mocks.getUserRooms.mockClear();

        // when
        await emit({ type: "chat_message", data: makeMessage({ id: "m9", room_id: "room-group" }) });

        // then
        expect(listed).toEqual(["room-1", "room-2"]);
        expect(mocks.getUserRooms).not.toHaveBeenCalled();
        expect(result.current.rooms.map(r => r.id)).toEqual(["room-1", "room-2"]);
        expect(result.current.rooms[0].unread).toBeFalsy();
    });
});

describe("useDmController socket wiring", () => {
    it("joins the active room, marks it read and listens, then leaves and stops listening when the view closes", async () => {
        // given
        const { sendRealtime, subscribe, unsubscribe, unmount } = await renderLoadedDm();

        // when
        const joined = sendRealtime.mock.calls.map(call => call[0]);
        unmount();

        // then
        expect(joined).toContainEqual({ type: "join_room", data: { room_id: "room-1" } });
        expect(mocks.markChatRoomRead).toHaveBeenCalledWith("room-1");
        expect(subscribe).toHaveBeenCalled();
        expect(sendRealtime).toHaveBeenLastCalledWith({ type: "leave_room", data: { room_id: "room-1" } });
        expect(unsubscribe).toHaveBeenCalled();
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

    const typingAnnouncementCases: { name: string; route: string; sent: OutboundModule.RealtimeCommand[] }[] = [
        {
            name: "announces that the viewer is typing in the active conversation",
            route: "/chat/room-1",
            sent: [{ type: "typing", data: { room_id: "room-1" } }],
        },
        { name: "says nothing about typing while no conversation is open", route: "/chat", sent: [] },
    ];

    it.each(typingAnnouncementCases)("$name", async ({ route, sent }) => {
        // given
        const { result, sendRealtime } = await renderLoadedDm({ route });
        sendRealtime.mockClear();

        // when
        act(() => {
            result.current.notifyTyping();
        });

        // then
        expect(sendRealtime.mock.calls.map(call => call[0])).toEqual(sent);
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
        "re-points the open conversation and the socket when the url moves to $name, dropping the reply target but keeping the view mounted",
        async ({ to, expectedRoomId, expectedView, expectedCommands }) => {
            // given
            const { result } = await renderNavigableDm();
            act(() => {
                result.current.dm.setReplyingTo({
                    id: "m1",
                    senderName: "Battler",
                    bodyPreview: "without love it cannot be seen",
                });
                result.current.dm.showToast("the golden land");
            });

            // when
            act(() => {
                result.current.navigate(to);
            });

            // then
            expect(sentCommands(["join_room", "leave_room"])).toEqual(expectedCommands);
            expect(result.current.dm.activeRoomId).toBe(expectedRoomId);
            expect(result.current.dm.mobileView).toBe(expectedView);
            expect(result.current.dm.replyingTo).toBeNull();
            expect(result.current.dm.toast).toBe("the golden land");
            expect(mocks.getUserRooms).toHaveBeenCalledTimes(1);
        },
    );

    it("opens the conversation the viewer picks with its badge cleared, leaving it in the history so back returns to the list", async () => {
        // given
        mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom({ unread: true })] });
        const { result } = await renderNavigableDm("/chat");

        // when
        act(() => {
            result.current.dm.handleRoomSelect("room-1");
        });
        await settle();
        const afterSelect = {
            pathname: result.current.pathname,
            activeRoomId: result.current.dm.activeRoomId,
            unread: result.current.dm.rooms[0].unread,
        };
        act(() => {
            result.current.navigate(-1);
        });

        // then
        expect(afterSelect).toEqual({ pathname: "/chat/room-1", activeRoomId: "room-1", unread: false });
        expect(result.current.pathname).toBe("/chat");
        expect(result.current.dm.activeRoomId).toBeNull();
    });

    it("keeps the mobile back button out of the history and drops any draft, so back returns to the conversation before it", async () => {
        // given
        const { result } = await renderNavigableDm("/chat");
        act(() => {
            result.current.dm.handleRoomSelect("room-1");
        });
        act(() => {
            result.current.dm.handleRoomSelect("room-2");
        });
        act(() => {
            result.current.dm.setDraftRecipient(makeMember({ id: "u5" }));
        });

        // when
        act(() => {
            result.current.dm.handleMobileBack();
        });
        const afterBack = {
            pathname: result.current.pathname,
            activeRoomId: result.current.dm.activeRoomId,
            draftRecipient: result.current.dm.draftRecipient,
        };
        act(() => {
            result.current.navigate(-1);
        });

        // then
        expect(afterBack).toEqual({ pathname: "/chat", activeRoomId: null, draftRecipient: null });
        expect(result.current.pathname).toBe("/chat/room-1");
        expect(result.current.dm.activeRoomId).toBe("room-1");
    });
});

describe("useDmController incoming messages", () => {
    interface LandingCase {
        name: string;
        message: ChatMessage;
        transcript: string[];
        roster: string[];
        unread: boolean;
        typing: string[];
    }

    const landedAt = "2026-08-02T12:00:00Z";

    const landingCases: LandingCase[] = [
        {
            name: "the conversation on screen",
            message: makeMessage({ id: "m1", created_at: landedAt }),
            transcript: ["m1"],
            roster: ["room-1", "room-2"],
            unread: false,
            typing: [],
        },
        {
            name: "another conversation",
            message: makeMessage({ id: "m9", room_id: "room-2", created_at: landedAt }),
            transcript: [],
            roster: ["room-2", "room-1"],
            unread: true,
            typing: ["Battler"],
        },
        {
            name: "another conversation from the viewer themselves",
            message: makeMessage({ id: "m9", room_id: "room-2", sender: viewerAsSender, created_at: landedAt }),
            transcript: [],
            roster: ["room-2", "room-1"],
            unread: false,
            typing: ["Battler"],
        },
    ];

    it.each(landingCases)(
        "updates the transcript, roster order, badge and typing for a message landing in $name",
        async ({ message, transcript, roster, unread, typing }) => {
            // given
            mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom(), makeRoom({ id: "room-2" })] });
            const { result, emit } = await renderLoadedDm();
            await emit({ type: "typing", data: { room_id: "room-1", user_id: "u2" } });

            // when
            await emit({ type: "chat_message", data: message });

            // then
            expect(result.current.messages.map(m => m.id)).toEqual(transcript);
            expect(result.current.rooms.map(r => r.id)).toEqual(roster);
            expect(result.current.rooms[0].unread).toBe(unread);
            expect(result.current.rooms[0].last_message_at).toBe(landedAt);
            expect(result.current.typingNames).toEqual(typing);
        },
    );

    it("never shows the viewer's own message twice when the echo arrives", async () => {
        // given
        const { result, emit } = await renderLoadedDm();
        const own = makeMessage({ id: "m1", sender: viewerAsSender });
        act(() => {
            result.current.handleSentMessage(own);
        });

        // when
        await emit({ type: "chat_message", data: own });

        // then
        expect(result.current.messages).toHaveLength(1);
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

    const soundCases: { name: string; sender: User; sounds: number }[] = [
        {
            name: "plays a sound for someone else's message while the tab is in the background",
            sender: makeMember(),
            sounds: 1,
        },
        { name: "stays silent for the viewer's own message", sender: viewerAsSender, sounds: 0 },
    ];

    it.each(soundCases)("$name", async ({ sender, sounds }) => {
        // given
        hideTab();
        const { emit } = await renderLoadedDm();

        // when
        await emit({ type: "chat_message", data: makeMessage({ id: "m1", sender }) });

        // then
        expect(mocks.playMessageSound).toHaveBeenCalledTimes(sounds);
    });
});

describe("useDmController message failures", () => {
    interface FailureCase {
        name: string;
        refused: Mock;
        reason: string;
        attempt: (dm: DmController) => Promise<unknown>;
    }

    const failureCases: FailureCase[] = [
        {
            name: "tells the viewer why their edit was refused",
            refused: mocks.editChatMessage,
            reason: "that message is too old to edit",
            attempt: dm => dm.handleEditMessage(makeMessage({ id: "m1" }), "rewritten").catch(() => {}),
        },
        {
            name: "tells the viewer why their delete was refused",
            refused: mocks.deleteChatMessage,
            reason: "the witch forbids it",
            attempt: dm => dm.handleDeleteMessage(makeMessage({ id: "m1" })),
        },
        {
            name: "tells the viewer why their reaction was refused",
            refused: mocks.addChatMessageReaction,
            reason: "that emoji is forbidden",
            attempt: dm => dm.handleReactionToggle(makeMessage({ id: "m1" }), "❤"),
        },
    ];

    it.each(failureCases)("$name", async ({ refused, reason, attempt }) => {
        // given
        refused.mockRejectedValue(new Error(reason));
        const { result } = await renderLoadedDm();

        // when
        await act(async () => {
            await attempt(result.current);
        });

        // then
        expect(result.current.toast).toBe(reason);
    });
});

describe("useDmController read receipts", () => {
    interface ReceiptCase {
        name: string;
        route: string;
        receiptRoomId: string;
        screenKind: RoomCapabilities["kind"];
        held: Record<string, string> | undefined;
    }

    const readAt = "2026-08-02T10:30:00Z";

    const receiptCases: ReceiptCase[] = [
        {
            name: "records who has read how far",
            route: "/chat/room-1",
            receiptRoomId: "room-1",
            screenKind: "pair",
            held: { u2: readAt },
        },
        {
            name: "never records a receipt for a room that is not a pair",
            route: "/chat/room-group",
            receiptRoomId: "room-group",
            screenKind: "group",
            held: undefined,
        },
        {
            name: "keeps a receipt for a pair that lands while the viewer is on the conversation list",
            route: "/chat",
            receiptRoomId: "room-1",
            screenKind: "group",
            held: { u2: readAt },
        },
        {
            name: "keeps a receipt for a pair that lands while another kind of room is open",
            route: "/chat/room-group",
            receiptRoomId: "room-1",
            screenKind: "group",
            held: { u2: readAt },
        },
    ];

    it.each(receiptCases)("$name", async ({ route, receiptRoomId, screenKind, held }) => {
        // given
        mocks.getUserRooms.mockResolvedValue({ rooms: [makeRoom(), makeGroupRoom()] });
        const { result, emit } = await renderLoadedDm({ route });

        // when
        await emit({ type: "chat_read_receipt", data: { room_id: receiptRoomId, user_id: "u2", read_at: readAt } });

        // then
        expect(result.current.capabilities.kind).toBe(screenKind);
        expect(result.current.readReceipts[receiptRoomId]).toEqual(held);
    });

    it("ignores a receipt older than the one it already holds", async () => {
        // given
        const { result, emit } = await renderLoadedDm();
        await emit({ type: "chat_read_receipt", data: { room_id: "room-1", user_id: "u2", read_at: readAt } });

        // when
        await emit({
            type: "chat_read_receipt",
            data: { room_id: "room-1", user_id: "u2", read_at: "2026-08-02T09:00:00Z" },
        });

        // then
        expect(result.current.readReceipts["room-1"].u2).toBe(readAt);
    });
});

describe("useDmController typing", () => {
    const typingCases: { name: string; typist: string; names: string[] }[] = [
        { name: "names the person typing in the active conversation", typist: "u2", names: ["Battler"] },
        { name: "never says the viewer is typing to themselves", typist: "u1", names: [] },
        {
            name: "falls back to a placeholder for somebody the room does not list",
            typist: "ghost",
            names: ["Someone"],
        },
    ];

    it.each(typingCases)("$name", async ({ typist, names }) => {
        // given
        const { result, emit } = await renderLoadedDm();

        // when
        await emit({ type: "typing", data: { room_id: "room-1", user_id: typist } });

        // then
        expect(result.current.typingNames).toEqual(names);
    });
});

describe("useDmController selecting conversations", () => {
    it("opens the conversation the viewer already had with the person they picked", async () => {
        // given
        const { result } = await renderLoadedDm({ route: "/chat" });
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

    it("holds the person as a draft and clears the conversation on screen when there is no conversation yet", async () => {
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
        expect(result.current.mobileView).toBe("room");
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
    it("opens the brand new conversation the first message created, showing that message before the server answers", async () => {
        // given
        const { result } = await renderLoadedDm({ route: "/chat" });
        mocks.fetchRoomMessages.mockReturnValue(new Promise(() => {}));
        act(() => {
            result.current.setDraftRecipient(makeMember());
        });

        // when
        act(() => {
            result.current.handleSentMessage(makeMessage({ id: "m1", room_id: "room-9" }), makeRoom({ id: "room-9" }));
        });
        const shownAtOnce = result.current.messages.map(m => m.id);
        await settle();

        // then
        expect(shownAtOnce).toEqual(["m1"]);
        expect(result.current.activeRoomId).toBe("room-9");
        expect(result.current.rooms.map(r => r.id)).toContain("room-9");
        expect(result.current.draftRecipient).toBeNull();
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

describe("useDmController muting a conversation", () => {
    interface MuteCase {
        name: string;
        answer: () => Promise<unknown>;
        viewerMuted: boolean;
        toast: string;
    }

    const muteCases: MuteCase[] = [
        {
            name: "mutes the conversation and reflects it on the room before the roster catches up",
            answer: () => Promise.resolve({ muted: true }),
            viewerMuted: true,
            toast: "Notifications muted",
        },
        {
            name: "leaves the conversation unmuted when the server refuses",
            answer: () => Promise.reject(new Error("nope")),
            viewerMuted: false,
            toast: "nope",
        },
    ];

    it.each(muteCases)("$name", async ({ answer, viewerMuted, toast }) => {
        // given
        mocks.setChatRoomMuted.mockImplementation(answer);
        const { result } = await renderLoadedDm();
        mocks.getUserRooms.mockReturnValue(new Promise(() => {}));

        // when
        await act(async () => {
            await result.current.handleToggleMute();
        });
        await settle();

        // then
        expect(mocks.setChatRoomMuted).toHaveBeenCalledWith("room-1", true);
        expect(result.current.rooms[0].viewer_muted).toBe(viewerMuted);
        expect(result.current.toast).toBe(toast);
        expect(result.current.mutePending).toBe(false);
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
});

describe("useDmController deleting a conversation", () => {
    interface KeptCase {
        name: string;
        confirmed: boolean;
        server: () => Promise<unknown>;
        deleteCalls: string[][];
        toast: string | null;
    }

    const keptCases: KeptCase[] = [
        {
            name: "leaves the conversation alone when the viewer backs out of the confirmation",
            confirmed: false,
            server: () => Promise.resolve(undefined),
            deleteCalls: [],
            toast: null,
        },
        {
            name: "keeps the conversation and says why when the server refuses to delete it",
            confirmed: true,
            server: () => Promise.reject(new Error("the witch forbids it")),
            deleteCalls: [["room-1"]],
            toast: "the witch forbids it",
        },
    ];

    it.each(keptCases)("$name", async ({ confirmed, server, deleteCalls, toast }) => {
        // given
        answerConfirmation(confirmed);
        mocks.deleteChatRoom.mockImplementation(server);
        const { result } = await renderLoadedDm();

        // when
        await act(async () => {
            await result.current.handleDeleteChat();
        });

        // then
        expect(mocks.deleteChatRoom.mock.calls).toEqual(deleteCalls);
        expect(result.current.rooms.map(r => r.id)).toEqual(["room-1"]);
        expect(result.current.activeRoomId).toBe("room-1");
        expect(result.current.toast).toBe(toast);
    });

    it("removes the conversation and returns to the list before the roster catches up", async () => {
        // given
        answerConfirmation(true);
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
    });

    it("keeps the deleted conversation off the roster once it refreshes itself", async () => {
        // given
        answerConfirmation(true);
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
    });
});

describe("useDmController reactions", () => {
    interface ReactionCase {
        name: string;
        reactions: ChatMessage["reactions"];
        added: unknown[][];
        removed: unknown[][];
    }

    const toggled = { messageId: "m1", emoji: "❤" };

    const reactionCases: ReactionCase[] = [
        { name: "adds a reaction the viewer has not given yet", reactions: [], added: [[toggled]], removed: [] },
        {
            name: "takes back the reaction the viewer already gave",
            reactions: [{ emoji: "❤", count: 1, viewer_reacted: true, display_names: ["Beatrice"] }],
            added: [],
            removed: [[toggled]],
        },
        {
            name: "adds to a reaction somebody else started",
            reactions: [{ emoji: "❤", count: 1, viewer_reacted: false, display_names: ["Battler"] }],
            added: [[toggled]],
            removed: [],
        },
    ];

    it.each(reactionCases)("$name", async ({ reactions, added, removed }) => {
        // given
        const message = makeMessage({ id: "m1", reactions });
        const { result } = await renderLoadedDm();

        // when
        await act(async () => {
            await result.current.handleReactionToggle(message, "❤");
        });

        // then
        expect(mocks.addChatMessageReaction.mock.calls).toEqual(added);
        expect(mocks.removeChatMessageReaction.mock.calls).toEqual(removed);
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
});

describe("useDmController finding people", () => {
    it("searches once the viewer stops typing, shows who it found, and clears them when the box is emptied", async () => {
        // given
        vi.useFakeTimers();
        mocks.fetchSearchUsers.mockResolvedValue([makeMember({ id: "u7", display_name: "Ange" })]);
        const { result } = renderDm();
        await act(async () => {
            await vi.advanceTimersByTimeAsync(0);
        });

        // when
        act(() => {
            result.current.setDmSearch("ang");
        });
        act(() => {
            vi.advanceTimersByTime(150);
        });
        const beforeDebounce = mocks.fetchSearchUsers.mock.calls.length;
        await act(async () => {
            await vi.advanceTimersByTimeAsync(100);
        });
        const found = result.current.dmResults.map(u => u.id);
        act(() => {
            result.current.setDmSearch("   ");
        });
        await act(async () => {
            await vi.advanceTimersByTimeAsync(10);
        });

        // then
        expect(beforeDebounce).toBe(0);
        expect(mocks.fetchSearchUsers).toHaveBeenCalledExactlyOnceWith("ang");
        expect(found).toEqual(["u7"]);
        expect(result.current.dmResults).toEqual([]);
    });

    const mutualCases: { name: string; lookup: () => Promise<User[]>; mutuals: string[] }[] = [
        {
            name: "suggests mutual followers when the new conversation panel opens",
            lookup: () => Promise.resolve([makeMember({ id: "u8", display_name: "Maria" })]),
            mutuals: ["u8"],
        },
        {
            name: "shows nobody when the mutual follower lookup fails",
            lookup: () => Promise.reject(new Error("offline")),
            mutuals: [],
        },
    ];

    it.each(mutualCases)("$name", async ({ lookup, mutuals }) => {
        // given
        mocks.fetchMutualFollowers.mockImplementation(lookup);
        const { result } = await renderLoadedDm();

        // when
        act(() => {
            result.current.setShowNewDm(true);
        });
        await settle();

        // then
        await waitFor(() => {
            expect(result.current.dmMutuals.map(u => u.id)).toEqual(mutuals);
        });
        expect(mocks.fetchMutualFollowers).toHaveBeenCalledTimes(1);
    });
});
