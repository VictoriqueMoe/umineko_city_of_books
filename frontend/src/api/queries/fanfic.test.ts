import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type { Fanfic, FanficChapter, FanficDetail, FanficListResponse } from "../../types/api";
import * as endpoints from "../endpoints";
import { queryKeys } from "../queryKeys";
import {
    fanficQueryFns,
    useFanfic,
    useFanficChapter,
    useFanficLanguages,
    useFanficList,
    useFanficSeries,
} from "./fanfic";

vi.mock("../endpoints", () => ({
    listFanfics: vi.fn(),
    getFanfic: vi.fn(),
    getFanficChapter: vi.fn(),
    getFanficLanguages: vi.fn(),
    getFanficSeries: vi.fn(),
}));

const listFanfics = vi.mocked(endpoints.listFanfics);
const getFanfic = vi.mocked(endpoints.getFanfic);
const getFanficChapter = vi.mocked(endpoints.getFanficChapter);
const getFanficLanguages = vi.mocked(endpoints.getFanficLanguages);
const getFanficSeries = vi.mocked(endpoints.getFanficSeries);

function makeFanfic(id: string, title: string): Fanfic {
    return { id, title } as Fanfic;
}

function makeListResponse(fanfics: Fanfic[], total: number): FanficListResponse {
    return { fanfics, total, limit: 20, offset: 0 };
}

function makeDetail(id: string, title: string): FanficDetail {
    return { id, title, chapters: [], comments: [] } as unknown as FanficDetail;
}

function makeChapter(chapterNumber: number): FanficChapter {
    return { id: `c${chapterNumber}`, chapter_number: chapterNumber, title: "The golden witch" } as FanficChapter;
}

describe("useFanficList", () => {
    it("forwards the filter params to the list endpoint and caches under the feed key", async () => {
        // given
        const params = { sort: "new", series: "umineko", limit: 20 };
        listFanfics.mockResolvedValue(makeListResponse([makeFanfic("f1", "Golden Land")], 1));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useFanficList(params), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(listFanfics).toHaveBeenCalledWith(params);
        expect(queryClient.getQueryData(queryKeys.fanfic.feed(params))).toEqual(
            makeListResponse([makeFanfic("f1", "Golden Land")], 1),
        );
    });

    it("exposes the fanfics and the total once the response arrives", async () => {
        // given
        listFanfics.mockResolvedValue(
            makeListResponse([makeFanfic("f1", "Rokkenjima"), makeFanfic("f2", "Beato")], 42),
        );

        // when
        const { result } = renderHook(() => useFanficList({}), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.fanfics).toHaveLength(2));
        expect(result.current.total).toBe(42);
    });

    it("starts with an empty list and a zero total while it is still loading", () => {
        // given
        listFanfics.mockReturnValue(new Promise(() => {}));

        // when
        const { result } = renderHook(() => useFanficList({}), { wrapper: providerWrapper() });

        // then
        expect(result.current.fanfics).toEqual([]);
        expect(result.current.total).toBe(0);
        expect(result.current.loading).toBe(true);
    });
});

describe("useFanfic", () => {
    it("fetches the fanfic and caches it under the detail key", async () => {
        // given
        getFanfic.mockResolvedValue(makeDetail("f1", "Legend of the golden witch"));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useFanfic("f1"), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.fanfic).not.toBeNull());
        expect(getFanfic).toHaveBeenCalledWith("f1");
        expect(queryClient.getQueryData(queryKeys.fanfic.detail("f1"))).toEqual(
            makeDetail("f1", "Legend of the golden witch"),
        );
    });

    it("does not call the endpoint when no id is given", () => {
        // given
        getFanfic.mockResolvedValue(makeDetail("f1", "Unused"));

        // when
        const { result } = renderHook(() => useFanfic(""), { wrapper: providerWrapper() });

        // then
        expect(getFanfic).not.toHaveBeenCalled();
        expect(result.current.fanfic).toBeNull();
    });

    it("refetches the fanfic when refresh is called", async () => {
        // given
        getFanfic.mockResolvedValue(makeDetail("f1", "Turn of the golden witch"));
        const { result } = renderHook(() => useFanfic("f1"), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.fanfic).not.toBeNull());

        // when
        await result.current.refresh();

        // then
        expect(getFanfic).toHaveBeenCalledTimes(2);
    });
});

describe("useFanficChapter", () => {
    it("fetches the chapter and caches it under the chapter key", async () => {
        // given
        getFanficChapter.mockResolvedValue(makeChapter(3));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useFanficChapter("f1", 3), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.chapter).not.toBeNull());
        expect(getFanficChapter).toHaveBeenCalledWith("f1", 3);
        expect(queryClient.getQueryData(["fanfic", "f1", "chapter", 3])).toEqual(makeChapter(3));
    });

    it("does not call the endpoint when the chapter number is not positive", () => {
        // given
        getFanficChapter.mockResolvedValue(makeChapter(1));

        // when
        const { result } = renderHook(() => useFanficChapter("f1", 0), { wrapper: providerWrapper() });

        // then
        expect(getFanficChapter).not.toHaveBeenCalled();
        expect(result.current.chapter).toBeNull();
    });

    it("does not call the endpoint when the fanfic id is empty", () => {
        // given
        getFanficChapter.mockResolvedValue(makeChapter(1));

        // when
        renderHook(() => useFanficChapter("", 1), { wrapper: providerWrapper() });

        // then
        expect(getFanficChapter).not.toHaveBeenCalled();
    });
});

describe("fanficQueryFns", () => {
    it("builds a prefetchable descriptor for a fanfic detail", async () => {
        // given
        getFanfic.mockResolvedValue(makeDetail("f9", "End of the golden witch"));

        // when
        const descriptor = fanficQueryFns.fanfic("f9");
        const data = await descriptor.queryFn();

        // then
        expect(descriptor.queryKey).toEqual(queryKeys.fanfic.detail("f9"));
        expect(getFanfic).toHaveBeenCalledWith("f9");
        expect(data).toEqual(makeDetail("f9", "End of the golden witch"));
    });

    it("builds a prefetchable descriptor for a chapter", async () => {
        // given
        getFanficChapter.mockResolvedValue(makeChapter(7));

        // when
        const descriptor = fanficQueryFns.chapter("f9", 7);
        const data = await descriptor.queryFn();

        // then
        expect(descriptor.queryKey).toEqual(["fanfic", "f9", "chapter", 7]);
        expect(getFanficChapter).toHaveBeenCalledWith("f9", 7);
        expect(data).toEqual(makeChapter(7));
    });
});

describe("useFanficLanguages", () => {
    it("returns the languages the endpoint reports", async () => {
        // given
        getFanficLanguages.mockResolvedValue(["English", "Japanese"]);

        // when
        const { result } = renderHook(() => useFanficLanguages(), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.languages).toEqual(["English", "Japanese"]));
    });

    it("falls back to an empty list before the response arrives", () => {
        // given
        getFanficLanguages.mockReturnValue(new Promise(() => {}));

        // when
        const { result } = renderHook(() => useFanficLanguages(), { wrapper: providerWrapper() });

        // then
        expect(result.current.languages).toEqual([]);
    });
});

describe("useFanficSeries", () => {
    it("returns the series the endpoint reports", async () => {
        // given
        getFanficSeries.mockResolvedValue(["Umineko", "Higurashi"]);

        // when
        const { result } = renderHook(() => useFanficSeries(), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.series).toEqual(["Umineko", "Higurashi"]));
    });

    it("falls back to an empty list before the response arrives", () => {
        // given
        getFanficSeries.mockReturnValue(new Promise(() => {}));

        // when
        const { result } = renderHook(() => useFanficSeries(), { wrapper: providerWrapper() });

        // then
        expect(result.current.series).toEqual([]);
    });
});
