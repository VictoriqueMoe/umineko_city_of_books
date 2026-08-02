import { renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Quote, QuoteBrowseResponse, QuoteSearchResponse } from "../../types/api";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { browseQuotes, searchQuotes } from "../endpoints";
import { useBrowseQuotes, useSearchQuotes } from "./quote";

vi.mock("../endpoints", () => ({
    browseQuotes: vi.fn(),
    searchQuotes: vi.fn(),
}));

const mockedBrowseQuotes = vi.mocked(browseQuotes);
const mockedSearchQuotes = vi.mocked(searchQuotes);

function makeQuote(text: string): Quote {
    return { text, character: "Beatrice", episode: 1, index: 0 } as unknown as Quote;
}

function makeSearchResponse(total: number): QuoteSearchResponse {
    return { results: [{ quote: makeQuote("without love it cannot be seen"), score: 1 }], total, limit: 30, offset: 0 };
}

function makeBrowseResponse(total: number): QuoteBrowseResponse {
    return {
        character: "Beatrice",
        characterId: "beatrice",
        quotes: [makeQuote("the golden witch")],
        total,
        limit: 30,
        offset: 0,
    };
}

function firstKey(qc: QueryClient): readonly unknown[] {
    return qc.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
    mockedSearchQuotes.mockResolvedValue(makeSearchResponse(1));
    mockedBrowseQuotes.mockResolvedValue(makeBrowseResponse(1));
});

describe("useSearchQuotes", () => {
    it("keys the search query by the params it was handed", async () => {
        // given
        const qc = createTestQueryClient();
        const params = { query: "golden", character: "beatrice", episode: 3, lang: "en", limit: 10, offset: 0 };

        // when
        const { result } = renderHook(() => useSearchQuotes(params), { wrapper: providerWrapper({ queryClient: qc }) });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["quotes", "search", params]);
    });

    it("forwards the params to the quote endpoint untouched", async () => {
        // given
        const params = { query: "meta world", series: "higurashi" as const, truth: "red" };

        // when
        const { result } = renderHook(() => useSearchQuotes(params), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(mockedSearchQuotes).toHaveBeenCalledWith(params);
    });

    it("searches by default without being told to", async () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useSearchQuotes({ query: "beato" }), { wrapper });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(mockedSearchQuotes).toHaveBeenCalledOnce();
    });

    it("stays quiet while it is disabled", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useSearchQuotes({ query: "beato" }, false), { wrapper });

        // then
        expect(mockedSearchQuotes).not.toHaveBeenCalled();
        expect(result.current.data).toBeNull();
        expect(result.current.loading).toBe(false);
    });

    it("reports no data while the search is still in flight", () => {
        // given
        mockedSearchQuotes.mockReturnValue(new Promise<QuoteSearchResponse>(() => {}));

        // when
        const { result } = renderHook(() => useSearchQuotes({ query: "beato" }), { wrapper: providerWrapper() });

        // then
        expect(result.current.data).toBeNull();
        expect(result.current.loading).toBe(true);
    });

    it("returns the whole response once the search settles", async () => {
        // given
        const response = makeSearchResponse(12);
        mockedSearchQuotes.mockResolvedValue(response);

        // when
        const { result } = renderHook(() => useSearchQuotes({ query: "beato" }), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.data).toEqual(response);
    });
});

describe("useBrowseQuotes", () => {
    it("keys the browse query by the params it was handed", async () => {
        // given
        const qc = createTestQueryClient();
        const params = { character: "beatrice", episode: 2, arc: "onikakushi", limit: 30, offset: 30 };

        // when
        const { result } = renderHook(() => useBrowseQuotes(params), { wrapper: providerWrapper({ queryClient: qc }) });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["quotes", "browse", params]);
        expect(mockedBrowseQuotes).toHaveBeenCalledWith(params);
    });

    it("stays quiet while it is disabled", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useBrowseQuotes({ character: "beatrice" }, false), { wrapper });

        // then
        expect(mockedBrowseQuotes).not.toHaveBeenCalled();
        expect(result.current.data).toBeNull();
    });

    it("returns the whole response once the browse settles", async () => {
        // given
        const response = makeBrowseResponse(5);
        mockedBrowseQuotes.mockResolvedValue(response);

        // when
        const { result } = renderHook(() => useBrowseQuotes({ character: "beatrice" }), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.data).toEqual(response);
    });

    it("keeps the search and the browse caches apart", async () => {
        // given
        const qc = createTestQueryClient();
        const wrapper = providerWrapper({ queryClient: qc });
        const params = { character: "beatrice" };

        // when
        const { result } = renderHook(() => ({ search: useSearchQuotes(params), browse: useBrowseQuotes(params) }), {
            wrapper,
        });
        await waitFor(() => expect(result.current.browse.loading).toBe(false));

        // then
        expect(qc.getQueryCache().getAll()).toHaveLength(2);
        expect(mockedSearchQuotes).toHaveBeenCalledOnce();
        expect(mockedBrowseQuotes).toHaveBeenCalledOnce();
    });
});
