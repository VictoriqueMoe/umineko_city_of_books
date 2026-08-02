import { act, renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Ship, ShipDetail, ShipListResponse } from "../../types/api";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { getShip, listShips } from "../endpoints";
import { useShip, useShipList } from "./ship";

vi.mock("../endpoints", () => ({
    getShip: vi.fn(),
    listShips: vi.fn(),
}));

const mockedGetShip = vi.mocked(getShip);
const mockedListShips = vi.mocked(listShips);

function makeShip(id: string): Ship {
    return { id, title: `ship ${id}` } as unknown as Ship;
}

function makeShipList(ships: Ship[], total: number): ShipListResponse {
    return { ships, total, limit: 20, offset: 0 };
}

function firstKey(qc: QueryClient): readonly unknown[] {
    return qc.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
    mockedListShips.mockResolvedValue(makeShipList([makeShip("s-1")], 1));
    mockedGetShip.mockResolvedValue({ id: "s-1", title: "ship s-1" } as unknown as ShipDetail);
});

describe("useShipList", () => {
    it("keys the feed query by the params it was handed", async () => {
        // given
        const qc = createTestQueryClient();
        const params = { sort: "popular", series: "umineko", limit: 20, offset: 0 };

        // when
        const { result } = renderHook(() => useShipList(params), { wrapper: providerWrapper({ queryClient: qc }) });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["ship", "feed", params]);
    });

    it("forwards every filter to the list endpoint untouched", async () => {
        // given
        const params = { sort: "new", series: "higurashi", character: "rika", crackships: true, limit: 5, offset: 10 };

        // when
        const { result } = renderHook(() => useShipList(params), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(mockedListShips).toHaveBeenCalledWith(params);
    });

    it("reports an empty list while the request is still in flight", () => {
        // given
        mockedListShips.mockReturnValue(new Promise<ShipListResponse>(() => {}));

        // when
        const { result } = renderHook(() => useShipList({}), { wrapper: providerWrapper() });

        // then
        expect(result.current.ships).toEqual([]);
        expect(result.current.total).toBe(0);
        expect(result.current.loading).toBe(true);
    });

    it("exposes the ships and the total once the response arrives", async () => {
        // given
        mockedListShips.mockResolvedValue(makeShipList([makeShip("s-1"), makeShip("s-2")], 42));

        // when
        const { result } = renderHook(() => useShipList({}), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.ships).toHaveLength(2);
        expect(result.current.total).toBe(42);
    });

    it("falls back to empty values when the response carries no ships", async () => {
        // given
        mockedListShips.mockResolvedValue({} as unknown as ShipListResponse);

        // when
        const { result } = renderHook(() => useShipList({}), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.ships).toEqual([]);
        expect(result.current.total).toBe(0);
    });
});

describe("useShip", () => {
    it("keys the detail query by the ship id", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useShip("s-9"), { wrapper: providerWrapper({ queryClient: qc }) });
        await waitFor(() => expect(result.current.ship).not.toBeNull());

        // then
        expect(firstKey(qc)).toEqual(["ship", "detail", "s-9"]);
    });

    it("does not ask the server for a ship without an id", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useShip(""), { wrapper });

        // then
        expect(mockedGetShip).not.toHaveBeenCalled();
        expect(result.current.ship).toBeNull();
        expect(result.current.loading).toBe(false);
    });

    it("returns the ship once it has loaded", async () => {
        // given
        mockedGetShip.mockResolvedValue({ id: "s-3", title: "the golden witch" } as unknown as ShipDetail);

        // when
        const { result } = renderHook(() => useShip("s-3"), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.ship).toEqual({ id: "s-3", title: "the golden witch" });
        expect(mockedGetShip).toHaveBeenCalledWith("s-3");
    });

    it("fetches the ship again when refresh is called", async () => {
        // given
        const { result } = renderHook(() => useShip("s-1"), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await act(async () => {
            await result.current.refresh();
        });

        // then
        expect(mockedGetShip).toHaveBeenCalledTimes(2);
    });
});
