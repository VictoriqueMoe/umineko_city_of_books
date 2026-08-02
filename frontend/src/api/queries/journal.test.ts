import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type {
    Journal,
    JournalComment,
    JournalDetail,
    JournalEntry,
    JournalListResponse,
    JournalWork,
} from "../../types/api";
import * as endpoints from "../endpoints";
import { queryKeys } from "../queryKeys";
import { useJournal, useJournalEntry, useJournalFeed, type JournalSort } from "./journal";

vi.mock("../endpoints", () => ({
    getJournal: vi.fn(),
    getJournalEntry: vi.fn(),
    listJournals: vi.fn(),
}));

const getJournal = vi.mocked(endpoints.getJournal);
const getJournalEntry = vi.mocked(endpoints.getJournalEntry);
const listJournals = vi.mocked(endpoints.listJournals);

function makeJournal(id: string, title: string): Journal {
    return { id, title, work: "umineko" } as Journal;
}

function makeDetail(id: string, title: string): JournalDetail {
    return { id, title, work: "umineko", entries: [], comments: [] } as unknown as JournalDetail;
}

function makeEntry(entryNumber: number): { entry: JournalEntry; comments: JournalComment[] } {
    return {
        entry: { id: `e${entryNumber}`, entry_number: entryNumber, body: "The rose garden" } as JournalEntry,
        comments: [{ id: "c1" } as JournalComment],
    };
}

function makeListResponse(journals: Journal[], total: number): JournalListResponse {
    return { journals, total, limit: 20, offset: 0 };
}

describe("useJournal", () => {
    it("fetches the journal and caches it under the detail key", async () => {
        // given
        getJournal.mockResolvedValue(makeDetail("j1", "The witch's diary"));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useJournal("j1"), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.journal).not.toBeNull());
        expect(getJournal).toHaveBeenCalledWith("j1");
        expect(queryClient.getQueryData(queryKeys.journal.detail("j1"))).toEqual(makeDetail("j1", "The witch's diary"));
    });

    it("does not call the endpoint when no id is given", () => {
        // given
        getJournal.mockResolvedValue(makeDetail("j1", "Unused"));

        // when
        const { result } = renderHook(() => useJournal(""), { wrapper: providerWrapper() });

        // then
        expect(getJournal).not.toHaveBeenCalled();
        expect(result.current.journal).toBeNull();
    });

    it("refetches the journal when refresh is called", async () => {
        // given
        getJournal.mockResolvedValue(makeDetail("j1", "The witch's diary"));
        const { result } = renderHook(() => useJournal("j1"), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.journal).not.toBeNull());

        // when
        await result.current.refresh();

        // then
        expect(getJournal).toHaveBeenCalledTimes(2);
    });
});

describe("useJournalEntry", () => {
    it("splits the response into the entry and its comments", async () => {
        // given
        getJournalEntry.mockResolvedValue(makeEntry(4));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useJournalEntry("j1", 4), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.entry).not.toBeNull());
        expect(getJournalEntry).toHaveBeenCalledWith("j1", 4);
        expect(result.current.comments).toHaveLength(1);
        expect(queryClient.getQueryData([...queryKeys.journal.detail("j1"), "entry", 4])).toBeDefined();
    });

    it("does not call the endpoint when the entry number is not positive", () => {
        // given
        getJournalEntry.mockResolvedValue(makeEntry(1));

        // when
        const { result } = renderHook(() => useJournalEntry("j1", 0), { wrapper: providerWrapper() });

        // then
        expect(getJournalEntry).not.toHaveBeenCalled();
        expect(result.current.entry).toBeNull();
        expect(result.current.comments).toEqual([]);
    });

    it("does not call the endpoint when the journal id is empty", () => {
        // given
        getJournalEntry.mockResolvedValue(makeEntry(1));

        // when
        renderHook(() => useJournalEntry("", 1), { wrapper: providerWrapper() });

        // then
        expect(getJournalEntry).not.toHaveBeenCalled();
    });
});

describe("useJournalFeed", () => {
    it("drops the empty work, search and author filters before calling the endpoint", async () => {
        // given
        listJournals.mockResolvedValue(makeListResponse([], 0));

        // when
        const { result } = renderHook(() => useJournalFeed("new", "", "", false, ""), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(listJournals).toHaveBeenCalledWith({
            sort: "new",
            work: undefined,
            author: undefined,
            search: undefined,
            includeArchived: false,
            limit: 20,
            offset: 0,
        });
    });

    it("passes the chosen filters through and caches under the feed key", async () => {
        // given
        listJournals.mockResolvedValue(makeListResponse([makeJournal("j1", "Rokkenjima days")], 1));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useJournalFeed("most_followed", "higurashi", "rika", true, "u1"), {
            wrapper: providerWrapper({ queryClient }),
        });

        // then
        await waitFor(() => expect(result.current.journals).toHaveLength(1));
        expect(listJournals).toHaveBeenCalledWith({
            sort: "most_followed",
            work: "higurashi",
            author: "u1",
            search: "rika",
            includeArchived: true,
            limit: 20,
            offset: 0,
        });
        expect(
            queryClient.getQueryData(
                queryKeys.journal.feed({
                    sort: "most_followed",
                    work: "higurashi",
                    search: "rika",
                    includeArchived: true,
                    authorId: "u1",
                    offset: 0,
                    limit: 20,
                }),
            ),
        ).toBeDefined();
    });

    it("starts on the first page with no previous page", async () => {
        // given
        listJournals.mockResolvedValue(makeListResponse([makeJournal("j1", "Rokkenjima days")], 50));

        // when
        const { result } = renderHook(() => useJournalFeed("new", ""), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.total).toBe(50));
        expect(result.current.offset).toBe(0);
        expect(result.current.limit).toBe(20);
        expect(result.current.hasPrev).toBe(false);
        expect(result.current.hasNext).toBe(true);
    });

    it("advances a page and asks for the next window", async () => {
        // given
        listJournals.mockResolvedValue(makeListResponse([makeJournal("j1", "Rokkenjima days")], 50));
        const { result } = renderHook(() => useJournalFeed("new", ""), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.total).toBe(50));

        // when
        act(() => result.current.goNext());

        // then
        await waitFor(() => expect(result.current.offset).toBe(20));
        expect(listJournals).toHaveBeenLastCalledWith(expect.objectContaining({ offset: 20, limit: 20 }));
    });

    it("refuses to advance past the last page", async () => {
        // given
        listJournals.mockResolvedValue(makeListResponse([makeJournal("j1", "Rokkenjima days")], 12));
        const { result } = renderHook(() => useJournalFeed("new", ""), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.total).toBe(12));

        // when
        act(() => result.current.goNext());

        // then
        expect(result.current.offset).toBe(0);
        expect(result.current.hasNext).toBe(false);
    });

    it("steps back a page and never below the first one", async () => {
        // given
        listJournals.mockResolvedValue(makeListResponse([makeJournal("j1", "Rokkenjima days")], 50));
        const { result } = renderHook(() => useJournalFeed("new", ""), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.total).toBe(50));
        act(() => result.current.goNext());
        await waitFor(() => expect(result.current.offset).toBe(20));

        // when
        act(() => result.current.goPrev());
        await waitFor(() => expect(result.current.offset).toBe(0));
        act(() => result.current.goPrev());

        // then
        expect(result.current.offset).toBe(0);
        expect(result.current.hasPrev).toBe(false);
    });

    it("returns to the first page when the sort changes", async () => {
        // given
        listJournals.mockResolvedValue(makeListResponse([makeJournal("j1", "Rokkenjima days")], 50));
        const { result, rerender } = renderHook(({ sort }: { sort: JournalSort }) => useJournalFeed(sort, ""), {
            initialProps: { sort: "new" as JournalSort },
            wrapper: providerWrapper(),
        });
        await waitFor(() => expect(result.current.total).toBe(50));
        act(() => result.current.goNext());
        await waitFor(() => expect(result.current.offset).toBe(20));

        // when
        rerender({ sort: "old" });

        // then
        expect(result.current.offset).toBe(0);
        await waitFor(() => expect(listJournals).toHaveBeenLastCalledWith(expect.objectContaining({ offset: 0 })));
    });

    it("returns to the first page when the search changes", async () => {
        // given
        listJournals.mockResolvedValue(makeListResponse([makeJournal("j1", "Rokkenjima days")], 50));
        const { result, rerender } = renderHook(({ search }: { search: string }) => useJournalFeed("new", "", search), {
            initialProps: { search: "" },
            wrapper: providerWrapper(),
        });
        await waitFor(() => expect(result.current.total).toBe(50));
        act(() => result.current.goNext());
        await waitFor(() => expect(result.current.offset).toBe(20));

        // when
        rerender({ search: "beatrice" });

        // then
        expect(result.current.offset).toBe(0);
        await waitFor(() =>
            expect(listJournals).toHaveBeenLastCalledWith(expect.objectContaining({ search: "beatrice", offset: 0 })),
        );
    });

    it("never asks for the stale offset under the new filters", async () => {
        // given
        listJournals.mockResolvedValue(makeListResponse([makeJournal("j1", "Rokkenjima days")], 50));
        const { result, rerender } = renderHook(({ work }: { work: JournalWork | "" }) => useJournalFeed("new", work), {
            initialProps: { work: "" as JournalWork | "" },
            wrapper: providerWrapper(),
        });
        await waitFor(() => expect(result.current.total).toBe(50));
        act(() => result.current.goNext());
        await waitFor(() => expect(result.current.offset).toBe(20));

        // when
        rerender({ work: "higurashi" });
        await waitFor(() => expect(result.current.journals).toHaveLength(1));

        // then
        expect(listJournals).not.toHaveBeenCalledWith(expect.objectContaining({ work: "higurashi", offset: 20 }));
    });

    it("keeps the page when nothing but an unrelated rerender happens", async () => {
        // given
        listJournals.mockResolvedValue(makeListResponse([makeJournal("j1", "Rokkenjima days")], 50));
        const { result, rerender } = renderHook(() => useJournalFeed("new", ""), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.total).toBe(50));
        act(() => result.current.goNext());
        await waitFor(() => expect(result.current.offset).toBe(20));

        // when
        rerender();

        // then
        expect(result.current.offset).toBe(20);
    });

    it("starts with an empty page while the request is in flight", () => {
        // given
        listJournals.mockReturnValue(new Promise(() => {}));

        // when
        const { result } = renderHook(() => useJournalFeed("new", ""), { wrapper: providerWrapper() });

        // then
        expect(result.current.journals).toEqual([]);
        expect(result.current.total).toBe(0);
        expect(result.current.hasNext).toBe(false);
        expect(result.current.loading).toBe(true);
    });
});
