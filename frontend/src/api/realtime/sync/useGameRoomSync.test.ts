import { act, renderHook } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { expectInvalidated } from "../../../test-utils/query";
import { createTestQueryClient, providerWrapper } from "../../../test-utils/render";
import { makeWSHarness, type WSHarness } from "../../../test-utils/ws";
import type { GameRoom } from "../../../types/api";
import { queryKeys } from "../../queryKeys";
import { dispatch } from "../bus";
import type { RealtimeEvent } from "../events";
import type * as OutboundModule from "../outbound";
import { useGameRoomSync } from "./useGameRoomSync";

const holder = vi.hoisted(() => ({ ws: null as unknown as WSHarness }));

vi.mock("../pipeline", () => ({
    ensureRealtimePipeline: () => {},
    getRealtimeEpoch: () => holder.ws.getEpoch(),
    subscribeRealtimeEpoch: (listener: () => void) => holder.ws.subscribeEpoch(listener),
}));

vi.mock("../outbound", async importOriginal => {
    const actual = await importOriginal<typeof OutboundModule>();

    return { ...actual, sendRealtime: (command: OutboundModule.RealtimeCommand) => holder.ws.sendRealtime(command) };
});

function makeRoom(id: string, status = "active"): GameRoom {
    return { id, game_type: "chess", status, players: [], watcher_count: 0 } as unknown as GameRoom;
}

function emit(event: RealtimeEvent): void {
    act(() => {
        dispatch(event);
    });
}

function readRoom(id: string): GameRoom | undefined {
    return queryClient.getQueryData<GameRoom>(queryKeys.gameRoom.detail(id));
}

let queryClient: QueryClient;

function mount(roomId: string | undefined) {
    return renderHook(() => useGameRoomSync(roomId), { wrapper: providerWrapper({ queryClient }) });
}

beforeEach(() => {
    holder.ws = makeWSHarness();
    queryClient = createTestQueryClient();
});

describe("useGameRoomSync membership", () => {
    it("joins the room over the socket on mount and leaves it on unmount", () => {
        // given
        const view = mount("r1");

        // then
        expect(holder.ws.sendRealtime).toHaveBeenCalledWith({ type: "game_room_join", data: { room_id: "r1" } });

        // when
        view.unmount();

        // then
        expect(holder.ws.sendRealtime).toHaveBeenLastCalledWith({
            type: "game_room_leave",
            data: { room_id: "r1" },
        });
    });

    it("does not join when there is no room id", () => {
        // given
        mount(undefined);

        // then
        expect(holder.ws.sendRealtime).not.toHaveBeenCalled();
    });

    it("leaves the old room and joins the new one when the room id changes", () => {
        // given
        const { rerender } = renderHook(({ roomId }: { roomId: string }) => useGameRoomSync(roomId), {
            wrapper: providerWrapper({ queryClient }),
            initialProps: { roomId: "r1" },
        });

        // when
        rerender({ roomId: "r2" });

        // then
        expect(holder.ws.sendRealtime.mock.calls.map(([command]) => command)).toEqual([
            { type: "game_room_join", data: { room_id: "r1" } },
            { type: "game_room_leave", data: { room_id: "r1" } },
            { type: "game_room_join", data: { room_id: "r2" } },
        ]);
    });

    it("joins the room again once the socket has reconnected", () => {
        // given
        mount("r1");

        // when
        holder.ws.reconnect();

        // then
        expect(holder.ws.sendRealtime.mock.calls.map(([command]) => command)).toEqual([
            { type: "game_room_join", data: { room_id: "r1" } },
            { type: "game_room_leave", data: { room_id: "r1" } },
            { type: "game_room_join", data: { room_id: "r1" } },
        ]);
    });

    it("reports the socket as connected only once an epoch has been seen", () => {
        // given
        const { result } = mount("r1");
        expect(result.current.connected).toBe(false);

        // when
        holder.ws.reconnect();

        // then
        expect(result.current.connected).toBe(true);
    });
});

describe("useGameRoomSync cache", () => {
    it("replaces the cached room when a socket action carries a new one", () => {
        // given
        queryClient.setQueryData(queryKeys.gameRoom.detail("r1"), makeRoom("r1", "active"));
        mount("r1");

        // when
        emit({ type: "game_room_action", data: { room_id: "r1", room: makeRoom("r1", "finished"), by_slot: 0 } });

        // then
        expect(readRoom("r1")?.status).toBe("finished");
    });

    it("replaces the cached room for every room event that carries one", () => {
        // given
        queryClient.setQueryData(queryKeys.gameRoom.detail("r1"), makeRoom("r1", "active"));
        mount("r1");

        // when
        emit({ type: "game_room_started", data: { room_id: "r1", room: makeRoom("r1", "started") } });
        const started = readRoom("r1")?.status;
        emit({ type: "game_room_finished", data: { room_id: "r1", room: makeRoom("r1", "finished") } });
        const finished = readRoom("r1")?.status;
        emit({
            type: "game_room_declined",
            data: { room_id: "r1", room: makeRoom("r1", "declined"), by_user_id: "u1" },
        });
        const declined = readRoom("r1")?.status;
        emit({
            type: "game_draw_offered",
            data: { room_id: "r1", room: makeRoom("r1", "draw-offered"), by_user_id: "u1" },
        });
        const offered = readRoom("r1")?.status;
        emit({
            type: "game_draw_declined",
            data: { room_id: "r1", room: makeRoom("r1", "draw-declined"), by_user_id: "u1" },
        });
        const drawDeclined = readRoom("r1")?.status;

        // then
        expect([started, finished, declined, offered, drawDeclined]).toEqual([
            "started",
            "finished",
            "declined",
            "draw-offered",
            "draw-declined",
        ]);
    });

    it("ignores socket messages meant for a different room", () => {
        // given
        queryClient.setQueryData(queryKeys.gameRoom.detail("r1"), makeRoom("r1", "active"));
        mount("r1");

        // when
        emit({ type: "game_room_finished", data: { room_id: "other", room: makeRoom("other", "finished") } });

        // then
        expect(readRoom("r1")).toEqual(makeRoom("r1", "active"));
        expect(readRoom("other")).toBeUndefined();
    });

    it("ignores socket messages of an unrelated type", () => {
        // given
        queryClient.setQueryData(queryKeys.gameRoom.detail("r1"), makeRoom("r1", "active"));
        mount("r1");

        // when
        emit({ type: "live_games_count", data: { count: 3 } });

        // then
        expect(readRoom("r1")).toEqual(makeRoom("r1", "active"));
    });

    it("ignores room events while there is no room id", () => {
        // given
        mount(undefined);

        // when
        emit({ type: "game_room_action", data: { room_id: "r1", room: makeRoom("r1", "finished"), by_slot: 0 } });

        // then
        expect(readRoom("r1")).toBeUndefined();
    });

    it("refetches the room when a presence message arrives without a room payload", () => {
        // given
        vi.spyOn(queryClient, "invalidateQueries");
        mount("r1");

        // when
        emit({
            type: "game_room_presence",
            data: { room_id: "r1", user_id: "u1", connected: true, as_player: true, watcher_count: 2 },
        });

        // then
        expectInvalidated(queryClient, queryKeys.gameRoom.detail("r1"));
    });

    it("leaves the cache alone when a non-presence event arrives without a room payload", () => {
        // given
        queryClient.setQueryData(queryKeys.gameRoom.detail("r1"), makeRoom("r1", "active"));
        vi.spyOn(queryClient, "invalidateQueries");
        mount("r1");

        // when
        emit({ type: "game_room_finished", data: { room_id: "r1" } } as unknown as RealtimeEvent);

        // then
        expect(readRoom("r1")).toEqual(makeRoom("r1", "active"));
        expect(queryClient.invalidateQueries).not.toHaveBeenCalled();
    });

    it("stops reacting once it is unmounted", () => {
        // given
        queryClient.setQueryData(queryKeys.gameRoom.detail("r1"), makeRoom("r1", "active"));
        const view = mount("r1");

        // when
        view.unmount();
        emit({ type: "game_room_action", data: { room_id: "r1", room: makeRoom("r1", "finished"), by_slot: 0 } });

        // then
        expect(readRoom("r1")?.status).toBe("active");
    });
});
