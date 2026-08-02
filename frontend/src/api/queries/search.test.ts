import { renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { QuickSearchResponse, SearchResponse, SearchResult } from "../../types/api";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { quickSearch, searchSite } from "../endpoints";
import { useQuickSearch, useRoomMessageSearch, useSiteSearch } from "./search";

vi.mock("../endpoints", () => ({
    quickSearch: vi.fn(),
    searchSite: vi.fn(),
}));

const mockedQuickSearch = vi.mocked(quickSearch);
const mockedSearchSite = vi.mocked(searchSite);

function makeResult(id: string): SearchResult {
    return { id, title: `result ${id}`, type: "post" } as unknown as SearchResult;
}

function makeSearchResponse(results: SearchResult[], total: number): SearchResponse {
    return { results, total };
}

function firstKey(qc: QueryClient): readonly unknown[] {
    return qc.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
    mockedQuickSearch.mockResolvedValue({ results: [makeResult("q-1")] });
    mockedSearchSite.mockResolvedValue(makeSearchResponse([makeResult("s-1")], 1));
});

describe("useQuickSearch", () => {
    it("keys the query by the trimmed term and asks for three hits per type", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useQuickSearch("  beato  ", true), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["search", "quick", "beato"]);
        expect(mockedQuickSearch).toHaveBeenCalledWith("beato", 3);
    });

    it("stays quiet for a term shorter than two characters", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useQuickSearch("b", true), { wrapper });

        // then
        expect(mockedQuickSearch).not.toHaveBeenCalled();
        expect(result.current.results).toEqual([]);
        expect(result.current.loading).toBe(false);
    });

    it("treats a term of only whitespace as empty", () => {
        // given
        const wrapper = providerWrapper();

        // when
        renderHook(() => useQuickSearch("     ", true), { wrapper });

        // then
        expect(mockedQuickSearch).not.toHaveBeenCalled();
    });

    it("stays quiet while it is switched off", () => {
        // given
        const wrapper = providerWrapper();

        // when
        renderHook(() => useQuickSearch("beato", false), { wrapper });

        // then
        expect(mockedQuickSearch).not.toHaveBeenCalled();
    });

    it("returns the results once the search settles", async () => {
        // given
        mockedQuickSearch.mockResolvedValue({ results: [makeResult("q-1"), makeResult("q-2")] });

        // when
        const { result } = renderHook(() => useQuickSearch("beato", true), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.results).toHaveLength(2);
    });

    it("reports an empty result list while the search is in flight", () => {
        // given
        mockedQuickSearch.mockReturnValue(new Promise<QuickSearchResponse>(() => {}));

        // when
        const { result } = renderHook(() => useQuickSearch("beato", true), { wrapper: providerWrapper() });

        // then
        expect(result.current.results).toEqual([]);
        expect(result.current.loading).toBe(true);
    });
});

describe("useRoomMessageSearch", () => {
    it("keys the query by the room, the trimmed term and the paging", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useRoomMessageSearch("room-1", " beato ", 25, 50, true), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["search", "room", "room-1", "beato", 25, 50]);
    });

    it("asks the site search only for chat messages inside that room", async () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useRoomMessageSearch("room-1", "beato", 20, 0, true), { wrapper });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(mockedSearchSite).toHaveBeenCalledWith("beato", "chat_message", 20, 0, "room-1");
    });

    it("stays quiet without a room to search in", () => {
        // given
        const wrapper = providerWrapper();

        // when
        renderHook(() => useRoomMessageSearch("", "beato", 20, 0, true), { wrapper });

        // then
        expect(mockedSearchSite).not.toHaveBeenCalled();
    });

    it("stays quiet for a term shorter than two characters", () => {
        // given
        const wrapper = providerWrapper();

        // when
        renderHook(() => useRoomMessageSearch("room-1", " b ", 20, 0, true), { wrapper });

        // then
        expect(mockedSearchSite).not.toHaveBeenCalled();
    });

    it("stays quiet while it is switched off", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useRoomMessageSearch("room-1", "beato", 20, 0, false), { wrapper });

        // then
        expect(mockedSearchSite).not.toHaveBeenCalled();
        expect(result.current.results).toEqual([]);
        expect(result.current.total).toBe(0);
    });

    it("returns the results and the total once the search settles", async () => {
        // given
        mockedSearchSite.mockResolvedValue(makeSearchResponse([makeResult("s-1"), makeResult("s-2")], 99));

        // when
        const { result } = renderHook(() => useRoomMessageSearch("room-1", "beato", 20, 0, true), {
            wrapper: providerWrapper(),
        });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.results).toHaveLength(2);
        expect(result.current.total).toBe(99);
    });

    it("keeps showing the previous page while the next one loads", async () => {
        // given
        mockedSearchSite.mockResolvedValue(makeSearchResponse([makeResult("s-1")], 40));
        const wrapper = providerWrapper();
        const { result, rerender } = renderHook(
            props => useRoomMessageSearch("room-1", "beato", 20, props.offset, true),
            {
                wrapper,
                initialProps: { offset: 0 },
            },
        );
        await waitFor(() => expect(result.current.results).toHaveLength(1));

        // when
        mockedSearchSite.mockReturnValue(new Promise<SearchResponse>(() => {}));
        rerender({ offset: 20 });

        // then
        expect(result.current.results).toHaveLength(1);
        expect(result.current.total).toBe(40);
    });
});

describe("useSiteSearch", () => {
    it("keys the query by the trimmed term, the types and the paging", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useSiteSearch(" beato ", "post,art", 20, 40), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["search", "full", "beato", "post,art", 20, 40]);
        expect(mockedSearchSite).toHaveBeenCalledWith("beato", "post,art", 20, 40);
    });

    it("stays quiet for a term shorter than two characters", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useSiteSearch("b", "", 20, 0), { wrapper });

        // then
        expect(mockedSearchSite).not.toHaveBeenCalled();
        expect(result.current.results).toEqual([]);
        expect(result.current.total).toBe(0);
        expect(result.current.loading).toBe(false);
        expect(result.current.fetching).toBe(false);
    });

    it("returns the results and the total once the search settles", async () => {
        // given
        mockedSearchSite.mockResolvedValue(makeSearchResponse([makeResult("s-1")], 3));

        // when
        const { result } = renderHook(() => useSiteSearch("beato", "", 20, 0), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.results).toHaveLength(1);
        expect(result.current.total).toBe(3);
        expect(result.current.fetching).toBe(false);
    });

    it("flags that it is fetching while the request is in flight", () => {
        // given
        mockedSearchSite.mockReturnValue(new Promise<SearchResponse>(() => {}));

        // when
        const { result } = renderHook(() => useSiteSearch("beato", "", 20, 0), { wrapper: providerWrapper() });

        // then
        expect(result.current.loading).toBe(true);
        expect(result.current.fetching).toBe(true);
    });
});
