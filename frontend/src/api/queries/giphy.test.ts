import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type { GiphyFavouritesResponse, GiphyResponse } from "../endpoints";
import * as endpoints from "../endpoints";
import { useGiphyFavourites, useGiphySearch, useGiphyTrending } from "./giphy";

vi.mock("../endpoints", () => ({
    searchGiphy: vi.fn(),
    trendingGiphy: vi.fn(),
    listGiphyFavourites: vi.fn(),
}));

const searchGiphy = vi.mocked(endpoints.searchGiphy);
const trendingGiphy = vi.mocked(endpoints.trendingGiphy);
const listGiphyFavourites = vi.mocked(endpoints.listGiphyFavourites);

function makeGiphyResponse(ids: string[]): GiphyResponse {
    return {
        data: ids.map(id => ({ id })),
        pagination: { total_count: ids.length, count: ids.length, offset: 0 },
    } as unknown as GiphyResponse;
}

function makeFavourites(count: number, total: number): GiphyFavouritesResponse {
    const data = [];
    for (let i = 0; i < count; i++) {
        data.push({
            giphy_id: `gif-${i}`,
            url: "https://media.giphy.example/gif.gif",
            title: "A golden butterfly",
            preview_url: "https://media.giphy.example/preview.gif",
            width: 200,
            height: 150,
        });
    }
    return { data, total };
}

describe("useGiphySearch", () => {
    it("searches with the term and the paging window and caches under the search key", async () => {
        // given
        searchGiphy.mockResolvedValue(makeGiphyResponse(["a", "b"]));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useGiphySearch("beatrice", 20, 10), {
            wrapper: providerWrapper({ queryClient }),
        });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(searchGiphy).toHaveBeenCalledWith("beatrice", 20, 10);
        expect(queryClient.getQueryData(["giphy", "search", "beatrice", { offset: 20, limit: 10 }])).toBeDefined();
    });

    it("does not search while the term is empty", () => {
        // given
        searchGiphy.mockResolvedValue(makeGiphyResponse([]));

        // when
        const { result } = renderHook(() => useGiphySearch(""), { wrapper: providerWrapper() });

        // then
        expect(searchGiphy).not.toHaveBeenCalled();
        expect(result.current.data).toBeUndefined();
    });

    it("does not search while the picker is closed", () => {
        // given
        searchGiphy.mockResolvedValue(makeGiphyResponse([]));

        // when
        renderHook(() => useGiphySearch("beatrice", 0, 0, false), { wrapper: providerWrapper() });

        // then
        expect(searchGiphy).not.toHaveBeenCalled();
    });

    it("hands back the raw response and the error when the search fails", async () => {
        // given
        searchGiphy.mockRejectedValue(new Error("giphy is asleep"));

        // when
        const { result } = renderHook(() => useGiphySearch("beatrice"), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.error).toBeInstanceOf(Error));
        expect(result.current.data).toBeUndefined();
    });
});

describe("useGiphyTrending", () => {
    it("fetches the trending page and caches it under the paging window", async () => {
        // given
        trendingGiphy.mockResolvedValue(makeGiphyResponse(["a"]));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useGiphyTrending(0, 25), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(trendingGiphy).toHaveBeenCalledWith(0, 25);
        expect(queryClient.getQueryData(["giphy", "trending", { offset: 0, limit: 25 }])).toBeDefined();
    });

    it("does not fetch while it is disabled", () => {
        // given
        trendingGiphy.mockResolvedValue(makeGiphyResponse([]));

        // when
        renderHook(() => useGiphyTrending(0, 0, false), { wrapper: providerWrapper() });

        // then
        expect(trendingGiphy).not.toHaveBeenCalled();
    });

    it("refetches when refresh is called", async () => {
        // given
        trendingGiphy.mockResolvedValue(makeGiphyResponse(["a"]));
        const { result } = renderHook(() => useGiphyTrending(), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await result.current.refresh();

        // then
        expect(trendingGiphy).toHaveBeenCalledTimes(2);
    });
});

describe("useGiphyFavourites", () => {
    it("lists the favourites of a signed in member", async () => {
        // given
        listGiphyFavourites.mockResolvedValue(makeFavourites(2, 7));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useGiphyFavourites(10, 5), {
            wrapper: providerWrapper({ queryClient, user: makeUser() }),
        });

        // then
        await waitFor(() => expect(result.current.favourites).toHaveLength(2));
        expect(listGiphyFavourites).toHaveBeenCalledWith(10, 5);
        expect(result.current.total).toBe(7);
        expect(queryClient.getQueryData(["giphy", "favourites", { offset: 10, limit: 5 }])).toBeDefined();
    });

    it("does not ask for favourites when nobody is signed in", () => {
        // given
        listGiphyFavourites.mockResolvedValue(makeFavourites(1, 1));

        // when
        const { result } = renderHook(() => useGiphyFavourites(), { wrapper: providerWrapper({ user: null }) });

        // then
        expect(listGiphyFavourites).not.toHaveBeenCalled();
        expect(result.current.favourites).toEqual([]);
        expect(result.current.total).toBe(0);
    });
});
