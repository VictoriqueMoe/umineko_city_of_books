import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi, type Mock } from "vitest";
import type { WSMessageHandler } from "../../context/notificationContextValue";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type { GameRoom, GameRoomListResponse, GameScoreboardResponse } from "../../types/api";
import * as endpoints from "../endpoints";
import { queryKeys } from "../queryKeys";
import { useFinishedGameRooms, useGameRoom, useGameScoreboard, useLiveGameRooms, useMyGameRooms } from "./gameRoom";

vi.mock("../endpoints", () => ({
    listMyGameRooms: vi.fn(),
    listLiveGameRooms: vi.fn(),
    listFinishedGameRooms: vi.fn(),
    getGameScoreboard: vi.fn(),
    getGameRoom: vi.fn(),
}));

const listMyGameRooms = vi.mocked(endpoints.listMyGameRooms);
const listLiveGameRooms = vi.mocked(endpoints.listLiveGameRooms);
const listFinishedGameRooms = vi.mocked(endpoints.listFinishedGameRooms);
const getGameScoreboard = vi.mocked(endpoints.getGameScoreboard);
const getGameRoom = vi.mocked(endpoints.getGameRoom);

function makeRoom(id: string, status = "active"): GameRoom {
    return { id, game_type: "chess", status, players: [], watcher_count: 0 } as unknown as GameRoom;
}

function makeRoomList(rooms: GameRoom[], total: number): GameRoomListResponse {
    return { rooms, total };
}

function makeScoreboard(): GameScoreboardResponse {
    return { game_type: "chess", rows: [] };
}

interface WSHarness {
    addWSListener: Mock<(handler: WSMessageHandler) => () => void>;
    sendWSMessage: Mock<(msg: object) => void>;
    emit: (type: string, data: unknown) => Promise<void>;
    unsubscribe: Mock<() => void>;
}

function makeWSHarness(): WSHarness {
    const handlers: WSMessageHandler[] = [];
    const unsubscribe: Mock<() => void> = vi.fn();
    const addWSListener = vi.fn((handler: WSMessageHandler) => {
        handlers.push(handler);
        return unsubscribe;
    });
    const sendWSMessage: Mock<(msg: object) => void> = vi.fn();
    const emit = async (type: string, data: unknown) => {
        await act(async () => {
            for (const handler of handlers) {
                handler({ type, data });
            }
        });
    };
    return { addWSListener, sendWSMessage, emit, unsubscribe };
}

describe("useMyGameRooms", () => {
    it("forwards the filters and caches under the game room list key", async () => {
        // given
        const params = { game_type: "chess", status: "active" } as const;
        listMyGameRooms.mockResolvedValue(makeRoomList([makeRoom("r1")], 1));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useMyGameRooms(params), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(listMyGameRooms).toHaveBeenCalledWith(params);
        expect(queryClient.getQueryData(queryKeys.gameRoom.list(params))).toEqual(makeRoomList([makeRoom("r1")], 1));
    });

    it("registers an empty filter object when no params are given", async () => {
        // given
        listMyGameRooms.mockResolvedValue(makeRoomList([], 0));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useMyGameRooms(), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(listMyGameRooms).toHaveBeenCalledWith(undefined);
        expect(queryClient.getQueryData(queryKeys.gameRoom.list({}))).toEqual(makeRoomList([], 0));
    });

    it("starts with an empty room list and no error", () => {
        // given
        listMyGameRooms.mockReturnValue(new Promise(() => {}));

        // when
        const { result } = renderHook(() => useMyGameRooms(), { wrapper: providerWrapper() });

        // then
        expect(result.current.rooms).toEqual([]);
        expect(result.current.total).toBe(0);
        expect(result.current.error).toBe("");
        expect(result.current.loading).toBe(true);
    });

    it("surfaces the message of a failed request", async () => {
        // given
        listMyGameRooms.mockRejectedValue(new Error("the servants refuse"));

        // when
        const { result } = renderHook(() => useMyGameRooms(), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.error).toBe("the servants refuse"));
        expect(result.current.rooms).toEqual([]);
    });
});

describe("useLiveGameRooms", () => {
    it("keys the query by game type when one is given", async () => {
        // given
        listLiveGameRooms.mockResolvedValue(makeRoomList([makeRoom("r2")], 1));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useLiveGameRooms("chess"), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.rooms).toHaveLength(1));
        expect(listLiveGameRooms).toHaveBeenCalledWith("chess");
        expect(queryClient.getQueryData(["game-rooms", "live", "chess"])).toBeDefined();
    });

    it("uses the unfiltered key when no game type is given", async () => {
        // given
        listLiveGameRooms.mockResolvedValue(makeRoomList([], 0));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useLiveGameRooms(), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(listLiveGameRooms).toHaveBeenCalledWith(undefined);
        expect(queryClient.getQueryData(["game-rooms", "live"])).toEqual(makeRoomList([], 0));
    });

    it("surfaces the message of a failed request", async () => {
        // given
        listLiveGameRooms.mockRejectedValue(new Error("no live boards"));

        // when
        const { result } = renderHook(() => useLiveGameRooms(), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.error).toBe("no live boards"));
    });
});

describe("useFinishedGameRooms", () => {
    it("defaults to the first page of twenty", async () => {
        // given
        listFinishedGameRooms.mockResolvedValue(makeRoomList([makeRoom("r3", "finished")], 1));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useFinishedGameRooms(), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.rooms).toHaveLength(1));
        expect(listFinishedGameRooms).toHaveBeenCalledWith(undefined, 20, 0);
        expect(queryClient.getQueryData(["game-rooms", "finished", "", { limit: 20, offset: 0 }])).toBeDefined();
    });

    it("forwards an explicit page and game type", async () => {
        // given
        listFinishedGameRooms.mockResolvedValue(makeRoomList([], 90));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useFinishedGameRooms("othello", 5, 10), {
            wrapper: providerWrapper({ queryClient }),
        });

        // then
        await waitFor(() => expect(result.current.total).toBe(90));
        expect(listFinishedGameRooms).toHaveBeenCalledWith("othello", 5, 10);
        expect(queryClient.getQueryData(["game-rooms", "finished", "othello", { limit: 5, offset: 10 }])).toBeDefined();
    });
});

describe("useGameScoreboard", () => {
    it("does not call the endpoint when no game type is chosen", () => {
        // given
        getGameScoreboard.mockResolvedValue(makeScoreboard());

        // when
        const { result } = renderHook(() => useGameScoreboard(undefined), { wrapper: providerWrapper() });

        // then
        expect(getGameScoreboard).not.toHaveBeenCalled();
        expect(result.current.data).toBeNull();
    });

    it("fetches the scoreboard for the chosen game type", async () => {
        // given
        getGameScoreboard.mockResolvedValue(makeScoreboard());
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useGameScoreboard("chess"), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.data).not.toBeNull());
        expect(getGameScoreboard).toHaveBeenCalledWith("chess");
        expect(queryClient.getQueryData(["game-rooms", "scoreboard", "chess"])).toEqual(makeScoreboard());
    });
});

describe("useGameRoom", () => {
    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("joins the room over the socket on mount and leaves it on unmount", async () => {
        // given
        getGameRoom.mockResolvedValue(makeRoom("r1"));
        const ws = makeWSHarness();

        // when
        const { result, unmount } = renderHook(() => useGameRoom("r1"), {
            wrapper: providerWrapper({ notification: ws }),
        });
        await waitFor(() => expect(result.current.room).not.toBeNull());

        // then
        expect(ws.sendWSMessage).toHaveBeenCalledWith({ type: "game_room_join", data: { room_id: "r1" } });
        unmount();
        expect(ws.sendWSMessage).toHaveBeenLastCalledWith({ type: "game_room_leave", data: { room_id: "r1" } });
    });

    it("does not fetch or join when there is no room id", () => {
        // given
        getGameRoom.mockResolvedValue(makeRoom("r1"));
        const ws = makeWSHarness();

        // when
        const { result } = renderHook(() => useGameRoom(undefined), { wrapper: providerWrapper({ notification: ws }) });

        // then
        expect(getGameRoom).not.toHaveBeenCalled();
        expect(ws.sendWSMessage).not.toHaveBeenCalled();
        expect(result.current.room).toBeNull();
    });

    it("replaces the cached room when a socket action carries a new one", async () => {
        // given
        getGameRoom.mockResolvedValue(makeRoom("r1", "active"));
        const ws = makeWSHarness();
        const { result } = renderHook(() => useGameRoom("r1"), { wrapper: providerWrapper({ notification: ws }) });
        await waitFor(() => expect(result.current.room).not.toBeNull());

        // when
        await ws.emit("game_room_action", { room_id: "r1", room: makeRoom("r1", "finished") });

        // then
        await waitFor(() => expect(result.current.room?.status).toBe("finished"));
        expect(getGameRoom).toHaveBeenCalledTimes(1);
    });

    it("ignores socket messages meant for a different room", async () => {
        // given
        getGameRoom.mockResolvedValue(makeRoom("r1", "active"));
        const ws = makeWSHarness();
        const queryClient = createTestQueryClient();
        const { result } = renderHook(() => useGameRoom("r1"), {
            wrapper: providerWrapper({ queryClient, notification: ws }),
        });
        await waitFor(() => expect(result.current.room).not.toBeNull());

        // when
        await ws.emit("game_room_finished", { room_id: "other", room: makeRoom("other", "finished") });

        // then
        expect(queryClient.getQueryData(queryKeys.gameRoom.detail("r1"))).toEqual(makeRoom("r1", "active"));
        expect(queryClient.getQueryData(queryKeys.gameRoom.detail("other"))).toBeUndefined();
    });

    it("ignores socket messages of an unrelated type", async () => {
        // given
        getGameRoom.mockResolvedValue(makeRoom("r1", "active"));
        const ws = makeWSHarness();
        const queryClient = createTestQueryClient();
        const { result } = renderHook(() => useGameRoom("r1"), {
            wrapper: providerWrapper({ queryClient, notification: ws }),
        });
        await waitFor(() => expect(result.current.room).not.toBeNull());

        // when
        await ws.emit("chat_message", { room_id: "r1", room: makeRoom("r1", "finished") });

        // then
        expect(queryClient.getQueryData(queryKeys.gameRoom.detail("r1"))).toEqual(makeRoom("r1", "active"));
    });

    it("refetches the room when a presence message arrives without a room payload", async () => {
        // given
        getGameRoom.mockResolvedValue(makeRoom("r1", "active"));
        const ws = makeWSHarness();
        const { result } = renderHook(() => useGameRoom("r1"), { wrapper: providerWrapper({ notification: ws }) });
        await waitFor(() => expect(result.current.room).not.toBeNull());
        getGameRoom.mockResolvedValue(makeRoom("r1", "finished"));

        // when
        await ws.emit("game_room_presence", { room_id: "r1" });

        // then
        await waitFor(() => expect(result.current.room?.status).toBe("finished"));
        expect(getGameRoom).toHaveBeenCalledTimes(2);
    });

    it("raises a desktop notification for your turn when the tab is not focused", async () => {
        // given
        getGameRoom.mockResolvedValue(makeRoom("r1"));
        const ws = makeWSHarness();
        const created: { title: string; options?: NotificationOptions }[] = [];
        class FakeNotification {
            static permission = "granted";
            onclick: (() => void) | null = null;
            close = vi.fn();
            constructor(title: string, options?: NotificationOptions) {
                created.push({ title, options });
            }
        }
        vi.stubGlobal("Notification", FakeNotification);
        vi.spyOn(document, "hasFocus").mockReturnValue(false);
        const { result } = renderHook(() => useGameRoom("r1"), { wrapper: providerWrapper({ notification: ws }) });
        await waitFor(() => expect(result.current.room).not.toBeNull());

        // when
        await ws.emit("game_your_turn", { room_id: "r1", game_type: "chess" });

        // then
        expect(created).toHaveLength(1);
        expect(created[0].title).toBe("Your move in chess");
        expect(created[0].options?.tag).toBe("game-turn-r1");
    });

    it("stays quiet about your turn while the tab is focused", async () => {
        // given
        getGameRoom.mockResolvedValue(makeRoom("r1"));
        const ws = makeWSHarness();
        const created: string[] = [];
        class FakeNotification {
            static permission = "granted";
            onclick: (() => void) | null = null;
            close = vi.fn();
            constructor(title: string) {
                created.push(title);
            }
        }
        vi.stubGlobal("Notification", FakeNotification);
        vi.spyOn(document, "hasFocus").mockReturnValue(true);
        const { result } = renderHook(() => useGameRoom("r1"), { wrapper: providerWrapper({ notification: ws }) });
        await waitFor(() => expect(result.current.room).not.toBeNull());

        // when
        await ws.emit("game_your_turn", { room_id: "r1", game_type: "chess" });

        // then
        expect(created).toEqual([]);
    });

    it("stays quiet about your turn when notification permission was never granted", async () => {
        // given
        getGameRoom.mockResolvedValue(makeRoom("r1"));
        const ws = makeWSHarness();
        const created: string[] = [];
        class FakeNotification {
            static permission = "default";
            constructor(title: string) {
                created.push(title);
            }
        }
        vi.stubGlobal("Notification", FakeNotification);
        vi.spyOn(document, "hasFocus").mockReturnValue(false);
        const { result } = renderHook(() => useGameRoom("r1"), { wrapper: providerWrapper({ notification: ws }) });
        await waitFor(() => expect(result.current.room).not.toBeNull());

        // when
        await ws.emit("game_your_turn", { room_id: "r1", game_type: "chess" });

        // then
        expect(created).toEqual([]);
    });

    it("reports the socket as connected only once an epoch has been seen", async () => {
        // given
        getGameRoom.mockResolvedValue(makeRoom("r1"));
        const ws = makeWSHarness();

        // when
        const offline = renderHook(() => useGameRoom("r1"), { wrapper: providerWrapper({ notification: ws }) });
        const online = renderHook(() => useGameRoom("r1"), {
            wrapper: providerWrapper({ notification: { ...ws, wsEpoch: 2 } }),
        });

        // then
        await waitFor(() => expect(offline.result.current.room).not.toBeNull());
        expect(offline.result.current.wsConnected).toBe(false);
        expect(online.result.current.wsConnected).toBe(true);
    });

    it("surfaces the message of a failed room fetch", async () => {
        // given
        getGameRoom.mockRejectedValue(new Error("that room is sealed"));
        const ws = makeWSHarness();

        // when
        const { result } = renderHook(() => useGameRoom("r1"), { wrapper: providerWrapper({ notification: ws }) });

        // then
        await waitFor(() => expect(result.current.error).toBe("that room is sealed"));
    });
});
