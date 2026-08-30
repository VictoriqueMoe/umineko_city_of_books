import { act, renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Theory, TheoryDetail, TheoryListResponse } from "../../types/api";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { getTheory, listTheories } from "../../api/endpoints/theory";
import { useTheory, useTheoryFeed } from "./theory";

vi.mock("../../api/endpoints/theory", () => ({
    getTheory: vi.fn(),
    listTheories: vi.fn(),
}));

const mockedGetTheory = vi.mocked(getTheory);
const mockedListTheories = vi.mocked(listTheories);

function makeTheory(id: string): Theory {
    return { id, title: `theory ${id}` } as unknown as Theory;
}

function makeTheoryList(theories: Theory[], total: number): TheoryListResponse {
    return { theories, total, limit: 20, offset: 0 };
}

function firstKey(qc: QueryClient): readonly unknown[] {
    return qc.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
    mockedGetTheory.mockResolvedValue({ id: "t-1", title: "theory t-1" } as unknown as TheoryDetail);
    mockedListTheories.mockResolvedValue(makeTheoryList([makeTheory("t-1")], 1));
});

describe("useTheory", () => {
    it("keys the detail query by the theory id", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useTheory("t-8"), { wrapper: providerWrapper({ queryClient: qc }) });
        await waitFor(() => expect(result.current.theory).not.toBeNull());

        // then
        expect(firstKey(qc)).toEqual(["theory", "detail", "t-8"]);
        expect(mockedGetTheory).toHaveBeenCalledWith("t-8");
    });

    it("does not ask the server for a theory without an id", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useTheory(""), { wrapper });

        // then
        expect(mockedGetTheory).not.toHaveBeenCalled();
        expect(result.current.theory).toBeNull();
        expect(result.current.loading).toBe(false);
    });

    it("fetches the theory again when refresh is called", async () => {
        // given
        const { result } = renderHook(() => useTheory("t-1"), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await act(async () => {
            await result.current.refresh();
        });

        // then
        expect(mockedGetTheory).toHaveBeenCalledTimes(2);
    });
});

describe("useTheoryFeed", () => {
    it("keys the feed by its filters and its paging", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(
            () =>
                useTheoryFeed({
                    sort: "popular",
                    episode: 4,
                    authorId: "author-1",
                    search: "beato",
                    series: "ciconia",
                }),
            { wrapper: providerWrapper({ queryClient: qc }) },
        );
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual([
            "theory",
            "feed",
            {
                sort: "popular",
                episode: 4,
                authorId: "author-1",
                search: "beato",
                series: "ciconia",
                offset: 0,
                limit: 20,
            },
        ]);
    });

    it("forwards the filters it was given to the list endpoint", async () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(
            () =>
                useTheoryFeed({ sort: "new", episode: 2, authorId: "author-1", search: "beato", series: "higurashi" }),
            { wrapper },
        );
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(mockedListTheories).toHaveBeenCalledWith({
            sort: "new",
            episode: 2,
            author: "author-1",
            search: "beato",
            series: "higurashi",
            limit: 20,
            offset: 0,
        });
    });

    it("drops the episode, author and search filters when they are empty", async () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useTheoryFeed({ sort: "new", episode: 0, authorId: "", search: "" }), {
            wrapper,
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(mockedListTheories).toHaveBeenCalledWith({
            sort: "new",
            episode: undefined,
            author: undefined,
            search: undefined,
            series: "umineko",
            limit: 20,
            offset: 0,
        });
    });

    it("reports empty values while the feed is loading", () => {
        // given
        mockedListTheories.mockReturnValue(new Promise<TheoryListResponse>(() => {}));

        // when
        const { result } = renderHook(() => useTheoryFeed({ sort: "new", episode: 0 }), {
            wrapper: providerWrapper(),
        });

        // then
        expect(result.current.theories).toEqual([]);
        expect(result.current.total).toBe(0);
        expect(result.current.loading).toBe(true);
    });

    it("asks for the window of theories the caller is paged to", async () => {
        // given
        mockedListTheories.mockResolvedValue(makeTheoryList([makeTheory("t-1")], 45));
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useTheoryFeed({ sort: "new", episode: 0, offset: 20, limit: 10 }), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(mockedListTheories).toHaveBeenCalledWith(expect.objectContaining({ offset: 20, limit: 10 }));
        expect(firstKey(qc)).toEqual(["theory", "feed", expect.objectContaining({ offset: 20, limit: 10 })]);
        expect(result.current.total).toBe(45);
    });

    it("stays idle while it is switched off", () => {
        // given
        const { result } = renderHook(() => useTheoryFeed({ sort: "new", episode: 0, enabled: false }), {
            wrapper: providerWrapper(),
        });

        // then
        expect(mockedListTheories).not.toHaveBeenCalled();
        expect(result.current.theories).toEqual([]);
        expect(result.current.total).toBe(0);
        expect(result.current.loading).toBe(false);
    });

    it("fetches the feed again when refresh is called", async () => {
        // given
        const { result } = renderHook(() => useTheoryFeed({ sort: "new", episode: 0 }), {
            wrapper: providerWrapper(),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await act(async () => {
            await result.current.refresh();
        });

        // then
        expect(mockedListTheories).toHaveBeenCalledTimes(2);
    });
});
