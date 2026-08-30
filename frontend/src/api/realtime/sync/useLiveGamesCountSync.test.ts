import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { expectInvalidated } from "../../../test-utils/query";
import { createTestQueryClient, providerWrapper } from "../../../test-utils/render";
import { queryKeys } from "../../queryKeys";
import { dispatch } from "../bus";
import type { RealtimeEvent } from "../events";
import { useLiveGamesCountSync } from "./useLiveGamesCountSync";

function emit(event: RealtimeEvent): void {
    act(() => {
        dispatch(event);
    });
}

describe("useLiveGamesCountSync", () => {
    it("writes the count as the total and keeps the rooms already cached", () => {
        // given
        const queryClient = createTestQueryClient();
        queryClient.setQueryData(queryKeys.gameRoom.live(), { rooms: [{ id: "room-1" }], total: 1 });
        renderHook(() => useLiveGamesCountSync(), { wrapper: providerWrapper({ queryClient }) });

        // when
        emit({ type: "live_games_count", data: { count: 4 } });

        // then
        expect(queryClient.getQueryData(queryKeys.gameRoom.live())).toEqual({ rooms: [{ id: "room-1" }], total: 4 });
    });

    it("starts from an empty room list when nothing is cached yet", () => {
        // given
        const queryClient = createTestQueryClient();
        renderHook(() => useLiveGamesCountSync(), { wrapper: providerWrapper({ queryClient }) });

        // when
        emit({ type: "live_games_count", data: { count: 2 } });

        // then
        expect(queryClient.getQueryData(queryKeys.gameRoom.live())).toEqual({ rooms: [], total: 2 });
    });

    it("invalidates the live game room list so the rooms themselves are refetched", () => {
        // given
        const queryClient = createTestQueryClient();
        vi.spyOn(queryClient, "invalidateQueries");
        renderHook(() => useLiveGamesCountSync(), { wrapper: providerWrapper({ queryClient }) });

        // when
        emit({ type: "live_games_count", data: { count: 2 } });

        // then
        expectInvalidated(queryClient, queryKeys.gameRoom.live());
    });

    it("ignores a payload with no numeric count", () => {
        // given
        const queryClient = createTestQueryClient();
        vi.spyOn(queryClient, "invalidateQueries");
        queryClient.setQueryData(queryKeys.gameRoom.live(), { rooms: [], total: 1 });
        renderHook(() => useLiveGamesCountSync(), { wrapper: providerWrapper({ queryClient }) });

        // when
        emit({ type: "live_games_count", data: {} as never });

        // then
        expect(queryClient.getQueryData(queryKeys.gameRoom.live())).toEqual({ rooms: [], total: 1 });
        expect(queryClient.invalidateQueries).not.toHaveBeenCalled();
    });
});
