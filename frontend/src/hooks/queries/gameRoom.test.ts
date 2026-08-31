import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeSpectatorMessage } from "../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type {
    GameRoom,
    GameRoomListResponse,
    GameScoreboardResponse,
    SpectatorChatResponse,
    SpectatorMessage,
} from "../../types/api";
import * as endpoints from "../../api/endpoints/gameRoom";
import { queryKeys } from "../../api/queryKeys";
import { useGameRoomSync } from "../../api/realtime/sync/useGameRoomSync";
import { useGameTurnNotice } from "../useGameTurnNotice";
import {
    useFinishedGameRooms,
    useGameRoom,
    useGameScoreboard,
    useLiveGameRooms,
    useMyGameRooms,
    usePlayerChat,
    useSpectatorChat,
} from "./gameRoom";

vi.mock("../../api/endpoints/gameRoom", () => ({
    listMyGameRooms: vi.fn(),
    listLiveGameRooms: vi.fn(),
    listFinishedGameRooms: vi.fn(),
    getGameScoreboard: vi.fn(),
    getGameRoom: vi.fn(),
    getSpectatorChat: vi.fn(),
    getPlayerChat: vi.fn(),
}));

vi.mock("../../api/realtime/sync/useGameRoomSync", () => ({ useGameRoomSync: vi.fn() }));

vi.mock("../useGameTurnNotice", () => ({ useGameTurnNotice: vi.fn() }));

const listMyGameRooms = vi.mocked(endpoints.listMyGameRooms);
const listLiveGameRooms = vi.mocked(endpoints.listLiveGameRooms);
const listFinishedGameRooms = vi.mocked(endpoints.listFinishedGameRooms);
const getGameScoreboard = vi.mocked(endpoints.getGameScoreboard);
const getGameRoom = vi.mocked(endpoints.getGameRoom);
const getSpectatorChat = vi.mocked(endpoints.getSpectatorChat);
const getPlayerChat = vi.mocked(endpoints.getPlayerChat);
const gameRoomSync = vi.mocked(useGameRoomSync);
const gameTurnNotice = vi.mocked(useGameTurnNotice);

function makeRoom(id: string, status = "active"): GameRoom {
    return { id, game_type: "chess", status, players: [], watcher_count: 0 } as unknown as GameRoom;
}

function makeMessage(id: string, body: string): SpectatorMessage {
    return makeSpectatorMessage({ id, body });
}

function makeChat(...messages: SpectatorMessage[]): SpectatorChatResponse {
    return { messages };
}

function makeRoomList(rooms: GameRoom[], total: number): GameRoomListResponse {
    return { rooms, total };
}

function makeScoreboard(): GameScoreboardResponse {
    return { game_type: "chess", rows: [] };
}

beforeEach(() => {
    gameRoomSync.mockReturnValue({ connected: false });
});

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
        expect(queryClient.getQueryData(queryKeys.gameRoom.live("chess"))).toBeDefined();
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
        expect(queryClient.getQueryData(queryKeys.gameRoom.live())).toEqual(makeRoomList([], 0));
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
        expect(queryClient.getQueryData(queryKeys.gameRoom.finished("", { limit: 20, offset: 0 }))).toBeDefined();
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
        expect(
            queryClient.getQueryData(queryKeys.gameRoom.finished("othello", { limit: 5, offset: 10 })),
        ).toBeDefined();
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
        expect(queryClient.getQueryData(queryKeys.gameRoom.scoreboard("chess"))).toEqual(makeScoreboard());
    });
});

describe("game room query keys", () => {
    it("keeps the live, finished and scoreboard views inside the family a mutation invalidates", async () => {
        // given
        listLiveGameRooms.mockResolvedValue(makeRoomList([], 0));
        listFinishedGameRooms.mockResolvedValue(makeRoomList([], 0));
        getGameScoreboard.mockResolvedValue(makeScoreboard());
        const queryClient = createTestQueryClient();
        const wrapper = providerWrapper({ queryClient });
        const live = renderHook(() => useLiveGameRooms(), { wrapper });
        const finished = renderHook(() => useFinishedGameRooms(), { wrapper });
        const scoreboard = renderHook(() => useGameScoreboard("chess"), { wrapper });
        await waitFor(() => expect(live.result.current.loading).toBe(false));
        await waitFor(() => expect(finished.result.current.loading).toBe(false));
        await waitFor(() => expect(scoreboard.result.current.loading).toBe(false));

        // when
        const matched = queryClient
            .getQueryCache()
            .findAll({ queryKey: queryKeys.gameRoom.all })
            .map(q => q.queryKey);

        // then
        expect(matched).toContainEqual(queryKeys.gameRoom.live());
        expect(matched).toContainEqual(queryKeys.gameRoom.finished("", { limit: 20, offset: 0 }));
        expect(matched).toContainEqual(queryKeys.gameRoom.scoreboard("chess"));
    });

    it("keeps both chats of a room inside the family a mutation invalidates", async () => {
        // given
        getSpectatorChat.mockResolvedValue(makeChat());
        getPlayerChat.mockResolvedValue(makeChat());
        const queryClient = createTestQueryClient();
        const wrapper = providerWrapper({ queryClient });
        const spectator = renderHook(() => useSpectatorChat("r1"), { wrapper });
        const player = renderHook(() => usePlayerChat("r1"), { wrapper });
        await waitFor(() => expect(spectator.result.current.loading).toBe(false));
        await waitFor(() => expect(player.result.current.loading).toBe(false));

        // when
        const matched = queryClient
            .getQueryCache()
            .findAll({ queryKey: queryKeys.gameRoom.all })
            .map(q => q.queryKey);

        // then
        expect(matched).toContainEqual(queryKeys.gameRoom.spectatorChat("r1"));
        expect(matched).toContainEqual(queryKeys.gameRoom.playerChat("r1"));
    });
});

describe("useGameRoom", () => {
    it("caches the room under the detail key for the id it was given", async () => {
        // given
        getGameRoom.mockResolvedValue(makeRoom("r1"));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useGameRoom("r1"), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.room).not.toBeNull());
        expect(getGameRoom).toHaveBeenCalledWith("r1");
        expect(queryClient.getQueryData(queryKeys.gameRoom.detail("r1"))).toEqual(makeRoom("r1"));
    });

    it("hands the room it is showing to the sync hook and to the turn notice", async () => {
        // given
        getGameRoom.mockResolvedValue(makeRoom("r1"));

        // when
        const { result } = renderHook(() => useGameRoom("r1"), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.room).not.toBeNull());

        // then
        expect(gameRoomSync).toHaveBeenCalledWith("r1");
        expect(gameTurnNotice).toHaveBeenCalledWith("r1");
    });

    it("does not fetch when there is no room id", () => {
        // given
        getGameRoom.mockResolvedValue(makeRoom("r1"));

        // when
        const { result } = renderHook(() => useGameRoom(undefined), { wrapper: providerWrapper() });

        // then
        expect(getGameRoom).not.toHaveBeenCalled();
        expect(gameRoomSync).toHaveBeenCalledWith(undefined);
        expect(result.current.room).toBeNull();
    });

    it("reports the socket as connected only once the sync hook says so", async () => {
        // given
        getGameRoom.mockResolvedValue(makeRoom("r1"));
        const offline = renderHook(() => useGameRoom("r1"), { wrapper: providerWrapper() });
        await waitFor(() => expect(offline.result.current.room).not.toBeNull());
        expect(offline.result.current.wsConnected).toBe(false);

        // when
        gameRoomSync.mockReturnValue({ connected: true });
        const online = renderHook(() => useGameRoom("r1"), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(online.result.current.room).not.toBeNull());
        expect(online.result.current.wsConnected).toBe(true);
    });

    it("fetches the room again when refetch is called", async () => {
        // given
        getGameRoom.mockResolvedValue(makeRoom("r1"));
        const { result } = renderHook(() => useGameRoom("r1"), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.room).not.toBeNull());

        // when
        await result.current.refetch();

        // then
        expect(getGameRoom).toHaveBeenCalledTimes(2);
    });

    it("surfaces the message of a failed room fetch", async () => {
        // given
        getGameRoom.mockRejectedValue(new Error("that room is sealed"));

        // when
        const { result } = renderHook(() => useGameRoom("r1"), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.error).toBe("that room is sealed"));
    });
});

describe("useSpectatorChat", () => {
    it("caches the watchers' messages under the spectator chat key", async () => {
        // given
        getSpectatorChat.mockResolvedValue(makeChat(makeMessage("m1", "the golden truth")));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useSpectatorChat("r1"), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.messages).toHaveLength(1));
        expect(getSpectatorChat).toHaveBeenCalledWith("r1");
        expect(queryClient.getQueryData(queryKeys.gameRoom.spectatorChat("r1"))).toEqual(
            makeChat(makeMessage("m1", "the golden truth")),
        );
    });

    it("starts with no messages and no error", () => {
        // given
        getSpectatorChat.mockReturnValue(new Promise(() => {}));

        // when
        const { result } = renderHook(() => useSpectatorChat("r1"), { wrapper: providerWrapper() });

        // then
        expect(result.current.messages).toEqual([]);
        expect(result.current.error).toBe("");
        expect(result.current.loading).toBe(true);
    });

    it("asks for nothing while the spectator chat is not the one on screen", () => {
        // given
        getSpectatorChat.mockResolvedValue(makeChat());

        // when
        const { result } = renderHook(() => useSpectatorChat("r1", false), { wrapper: providerWrapper() });

        // then
        expect(getSpectatorChat).not.toHaveBeenCalled();
        expect(result.current.messages).toEqual([]);
        expect(result.current.loading).toBe(false);
    });

    it("surfaces the message of a failed request", async () => {
        // given
        getSpectatorChat.mockRejectedValue(new Error("the watchers were sent away"));

        // when
        const { result } = renderHook(() => useSpectatorChat("r1"), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.error).toBe("the watchers were sent away"));
        expect(result.current.messages).toEqual([]);
    });
});

describe("usePlayerChat", () => {
    it("caches the private messages under a key of its own so the spectator chat cannot answer for it", async () => {
        // given
        getPlayerChat.mockResolvedValue(makeChat(makeMessage("m2", "your move")));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => usePlayerChat("r1"), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.messages).toHaveLength(1));
        expect(getPlayerChat).toHaveBeenCalledWith("r1");
        expect(queryClient.getQueryData(queryKeys.gameRoom.playerChat("r1"))).toEqual(
            makeChat(makeMessage("m2", "your move")),
        );
        expect(queryClient.getQueryData(queryKeys.gameRoom.spectatorChat("r1"))).toBeUndefined();
    });

    it("asks for nothing while the player chat is not the one on screen", () => {
        // given
        getPlayerChat.mockResolvedValue(makeChat());

        // when
        const { result } = renderHook(() => usePlayerChat("r1", false), { wrapper: providerWrapper() });

        // then
        expect(getPlayerChat).not.toHaveBeenCalled();
        expect(result.current.messages).toEqual([]);
        expect(result.current.loading).toBe(false);
    });
});
