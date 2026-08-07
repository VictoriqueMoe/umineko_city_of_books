import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type {
    GMLeaderboardResponse,
    Mystery,
    MysteryDetail,
    MysteryLeaderboardResponse,
    MysteryListResponse,
} from "../../types/api";
import * as endpoints from "../endpoints";
import { queryKeys } from "../queryKeys";
import { useGMLeaderboard, useMystery, useMysteryLeaderboard, useMysteryList } from "./mystery";

vi.mock("../endpoints", () => ({
    getGMLeaderboard: vi.fn(),
    getMystery: vi.fn(),
    getMysteryLeaderboard: vi.fn(),
    listMysteries: vi.fn(),
}));

const getGMLeaderboard = vi.mocked(endpoints.getGMLeaderboard);
const getMystery = vi.mocked(endpoints.getMystery);
const getMysteryLeaderboard = vi.mocked(endpoints.getMysteryLeaderboard);
const listMysteries = vi.mocked(endpoints.listMysteries);

function makeMystery(id: string, title: string): Mystery {
    return { id, title } as Mystery;
}

function makeListResponse(mysteries: Mystery[], total: number): MysteryListResponse {
    return { mysteries, total, limit: 20, offset: 0 };
}

function makeDetail(id: string, title: string): MysteryDetail {
    return { id, title, clues: [], attempts: [], comments: [] } as unknown as MysteryDetail;
}

function makeLeaderboard(count: number): MysteryLeaderboardResponse {
    const entries = [];
    for (let i = 0; i < count; i++) {
        entries.push({ user: { id: `u${i}`, username: `sleuth${i}`, display_name: `Sleuth ${i}` }, solved_count: i });
    }
    return { entries } as unknown as MysteryLeaderboardResponse;
}

function makeGMLeaderboard(count: number): GMLeaderboardResponse {
    const entries = [];
    for (let i = 0; i < count; i++) {
        entries.push({
            user: { id: `g${i}`, username: `gm${i}`, display_name: `GM ${i}` },
            score: i,
            mystery_count: i,
            player_count: i,
        });
    }
    return { entries };
}

describe("useMysteryList", () => {
    it("forwards the filters and caches under the list key", async () => {
        // given
        const params = { sort: "new", solved: "false", limit: 10, offset: 20 };
        listMysteries.mockResolvedValue(makeListResponse([makeMystery("m1", "The sealed room")], 1));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useMysteryList(params), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.mysteries).toHaveLength(1));
        expect(listMysteries).toHaveBeenCalledWith(params);
        expect(queryClient.getQueryData(queryKeys.mystery.list(params))).toBeDefined();
    });

    it("sits inside the mystery family a mystery mutation invalidates", async () => {
        // given
        const params = { sort: "new" };
        listMysteries.mockResolvedValue(makeListResponse([makeMystery("m1", "The sealed room")], 1));
        const queryClient = createTestQueryClient();
        const { result } = renderHook(() => useMysteryList(params), { wrapper: providerWrapper({ queryClient }) });
        await waitFor(() => expect(result.current.mysteries).toHaveLength(1));

        // when
        const matched = queryClient.getQueryCache().findAll({ queryKey: queryKeys.mystery.all });

        // then
        expect(matched.map(q => q.queryKey)).toContainEqual(queryKeys.mystery.list(params));
    });

    it("exposes the total alongside the page", async () => {
        // given
        listMysteries.mockResolvedValue(makeListResponse([], 37));

        // when
        const { result } = renderHook(() => useMysteryList({}), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.total).toBe(37));
    });

    it("starts with an empty page while the request is in flight", () => {
        // given
        listMysteries.mockReturnValue(new Promise(() => {}));

        // when
        const { result } = renderHook(() => useMysteryList({}), { wrapper: providerWrapper() });

        // then
        expect(result.current.mysteries).toEqual([]);
        expect(result.current.total).toBe(0);
        expect(result.current.loading).toBe(true);
    });

    it("refetches the page when refresh is called", async () => {
        // given
        listMysteries.mockResolvedValue(makeListResponse([], 0));
        const { result } = renderHook(() => useMysteryList({}), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await result.current.refresh();

        // then
        expect(listMysteries).toHaveBeenCalledTimes(2);
    });
});

describe("useMystery", () => {
    it("fetches the mystery and caches it under the detail key", async () => {
        // given
        getMystery.mockResolvedValue(makeDetail("m1", "The sealed room"));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useMystery("m1"), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.mystery).not.toBeNull());
        expect(getMystery).toHaveBeenCalledWith("m1");
        expect(queryClient.getQueryData(queryKeys.mystery.detail("m1"))).toEqual(makeDetail("m1", "The sealed room"));
    });

    it("does not call the endpoint when no id is given", () => {
        // given
        getMystery.mockResolvedValue(makeDetail("m1", "Unused"));

        // when
        const { result } = renderHook(() => useMystery(""), { wrapper: providerWrapper() });

        // then
        expect(getMystery).not.toHaveBeenCalled();
        expect(result.current.mystery).toBeNull();
    });

    it("refetches the mystery when refresh is called", async () => {
        // given
        getMystery.mockResolvedValue(makeDetail("m1", "The sealed room"));
        const { result } = renderHook(() => useMystery("m1"), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.mystery).not.toBeNull());

        // when
        await result.current.refresh();

        // then
        expect(getMystery).toHaveBeenCalledTimes(2);
    });
});

describe("useMysteryLeaderboard", () => {
    it("keys the leaderboard by the limit it was given", async () => {
        // given
        getMysteryLeaderboard.mockResolvedValue(makeLeaderboard(3));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useMysteryLeaderboard(5), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.entries).toHaveLength(3));
        expect(getMysteryLeaderboard).toHaveBeenCalledWith(5);
        expect(queryClient.getQueryData(queryKeys.mystery.leaderboard(5))).toBeDefined();
    });

    it("uses a null limit in the key when none is given", async () => {
        // given
        getMysteryLeaderboard.mockResolvedValue(makeLeaderboard(0));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useMysteryLeaderboard(), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(getMysteryLeaderboard).toHaveBeenCalledWith(undefined);
        expect(queryClient.getQueryData(queryKeys.mystery.leaderboard(null))).toBeDefined();
    });

    it("falls back to no entries while loading", () => {
        // given
        getMysteryLeaderboard.mockReturnValue(new Promise(() => {}));

        // when
        const { result } = renderHook(() => useMysteryLeaderboard(), { wrapper: providerWrapper() });

        // then
        expect(result.current.entries).toEqual([]);
        expect(result.current.loading).toBe(true);
    });
});

describe("useGMLeaderboard", () => {
    it("keys the game master leaderboard by the limit it was given", async () => {
        // given
        getGMLeaderboard.mockResolvedValue(makeGMLeaderboard(2));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useGMLeaderboard(10), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.entries).toHaveLength(2));
        expect(getGMLeaderboard).toHaveBeenCalledWith(10);
        expect(queryClient.getQueryData(queryKeys.mystery.gmLeaderboard(10))).toBeDefined();
    });

    it("uses a null limit in the key when none is given", async () => {
        // given
        getGMLeaderboard.mockResolvedValue(makeGMLeaderboard(0));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useGMLeaderboard(), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(getGMLeaderboard).toHaveBeenCalledWith(undefined);
        expect(queryClient.getQueryData(queryKeys.mystery.gmLeaderboard(null))).toBeDefined();
    });

    it("falls back to no entries while loading", () => {
        // given
        getGMLeaderboard.mockReturnValue(new Promise(() => {}));

        // when
        const { result } = renderHook(() => useGMLeaderboard(), { wrapper: providerWrapper() });

        // then
        expect(result.current.entries).toEqual([]);
    });
});
