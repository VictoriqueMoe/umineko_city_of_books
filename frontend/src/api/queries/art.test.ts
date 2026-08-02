import type { QueryClient } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { useAllGalleries, useArt, useArtFeed, useGallery } from "./art";

const endpoints = vi.hoisted(() => ({
    getArt: vi.fn(),
    getGallery: vi.fn(),
    listAllGalleries: vi.fn(),
    listArt: vi.fn(),
}));

vi.mock("../endpoints", () => endpoints);

function setup<T>(hook: () => T) {
    const queryClient = createTestQueryClient();
    const rendered = renderHook(hook, { wrapper: providerWrapper({ queryClient }) });

    return { ...rendered, queryClient };
}

function firstKey(queryClient: QueryClient): readonly unknown[] {
    return queryClient.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
    endpoints.getArt.mockResolvedValue({ id: "art-1", title: "Golden witch" });
    endpoints.getGallery.mockResolvedValue({ gallery: { id: "g-1" }, art: [], total: 0 });
    endpoints.listAllGalleries.mockResolvedValue([]);
    endpoints.listArt.mockResolvedValue({ art: [], total: 0 });
});

describe("useArtFeed", () => {
    it("asks for the general corner in pages of twenty four by default", async () => {
        // given
        const { result } = setup(() => useArtFeed());

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(endpoints.listArt).toHaveBeenCalledWith({
            corner: "general",
            type: undefined,
            search: undefined,
            tag: undefined,
            sort: undefined,
            limit: 24,
            offset: 0,
        });
        expect(result.current.limit).toBe(24);
        expect(result.current.offset).toBe(0);
    });

    it("turns a page number into an offset", async () => {
        // given
        const { result } = setup(() => useArtFeed("general", "", "", "", "", 3));

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(result.current.offset).toBe(48);
        expect(endpoints.listArt).toHaveBeenCalledWith(expect.objectContaining({ offset: 48, limit: 24 }));
    });

    it("drops empty filter strings so the endpoint sees no filter at all", async () => {
        // given
        const { result } = setup(() => useArtFeed("fanart", "", "", "", "", 1));

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(endpoints.listArt).toHaveBeenCalledWith({
            corner: "fanart",
            type: undefined,
            search: undefined,
            tag: undefined,
            sort: undefined,
            limit: 24,
            offset: 0,
        });
    });

    it("forwards every filter it was given", async () => {
        // given
        const { result } = setup(() => useArtFeed("fanart", "sketch", "beato", "witch", "top", 2));

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(endpoints.listArt).toHaveBeenCalledWith({
            corner: "fanart",
            type: "sketch",
            search: "beato",
            tag: "witch",
            sort: "top",
            limit: 24,
            offset: 24,
        });
    });

    it("keys the cache entry by every filter including the refresh key", () => {
        // given
        const { queryClient } = setup(() => useArtFeed("fanart", "sketch", "beato", "witch", "top", 2, 7));

        // when
        const key = firstKey(queryClient);

        // then
        expect(key).toEqual([
            "art",
            "feed",
            {
                corner: "fanart",
                artType: "sketch",
                search: "beato",
                tag: "witch",
                sort: "top",
                offset: 24,
                limit: 24,
                refreshKey: 7,
            },
        ]);
    });

    it("offers a next page while the total runs past the current window", async () => {
        // given
        endpoints.listArt.mockResolvedValue({ art: [{ id: "art-1" }], total: 30 });

        // when
        const { result } = setup(() => useArtFeed("general", undefined, undefined, undefined, undefined, 1));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.hasNext).toBe(true);
        expect(result.current.hasPrev).toBe(false);
        expect(result.current.total).toBe(30);
    });

    it("offers a previous page but no next one on the last page", async () => {
        // given
        endpoints.listArt.mockResolvedValue({ art: [{ id: "art-1" }], total: 30 });

        // when
        const { result } = setup(() => useArtFeed("general", undefined, undefined, undefined, undefined, 2));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.hasNext).toBe(false);
        expect(result.current.hasPrev).toBe(true);
    });

    it("offers no next page before the first response arrives", () => {
        // given
        const { result } = setup(() => useArtFeed("general", undefined, undefined, undefined, undefined, 2));

        // when
        const initial = result.current;

        // then
        expect(initial.hasNext).toBe(false);
        expect(initial.hasPrev).toBe(true);
        expect(initial.art).toEqual([]);
        expect(initial.total).toBe(0);
    });
});

describe("useArt", () => {
    it("loads the artwork behind the id it was given", async () => {
        // given
        const { result, queryClient } = setup(() => useArt("art-1"));

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(endpoints.getArt).toHaveBeenCalledWith("art-1");
        expect(firstKey(queryClient)).toEqual(["art", "detail", "art-1"]);
        expect(result.current.art).toEqual({ id: "art-1", title: "Golden witch" });
    });

    it("stays idle and reports no artwork when the id is empty", () => {
        // given
        const { result } = setup(() => useArt(""));

        // when
        const current = result.current;

        // then
        expect(endpoints.getArt).not.toHaveBeenCalled();
        expect(current.art).toBeNull();
        expect(current.loading).toBe(false);
    });
});

describe("useGallery", () => {
    it("asks for the first twenty four pieces of a gallery by default", async () => {
        // given
        const { result, queryClient } = setup(() => useGallery("g-1"));

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(endpoints.getGallery).toHaveBeenCalledWith("g-1", 24, 0);
        expect(firstKey(queryClient)).toEqual(["gallery", "g-1", { limit: 24, offset: 0 }]);
    });

    it("forwards a chosen page window and keys the cache entry by it", async () => {
        // given
        endpoints.getGallery.mockResolvedValue({ gallery: { id: "g-1" }, art: [{ id: "art-1" }], total: 3 });

        // when
        const { result, queryClient } = setup(() => useGallery("g-1", 10, 20));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.getGallery).toHaveBeenCalledWith("g-1", 10, 20);
        expect(firstKey(queryClient)).toEqual(["gallery", "g-1", { limit: 10, offset: 20 }]);
        expect(result.current.gallery).toEqual({ id: "g-1" });
        expect(result.current.art).toEqual([{ id: "art-1" }]);
        expect(result.current.total).toBe(3);
    });

    it("stays idle and reports nothing when the id is empty", () => {
        // given
        const { result } = setup(() => useGallery(""));

        // when
        const current = result.current;

        // then
        expect(endpoints.getGallery).not.toHaveBeenCalled();
        expect(current.gallery).toBeNull();
        expect(current.art).toEqual([]);
        expect(current.total).toBe(0);
    });
});

describe("useAllGalleries", () => {
    it("asks for every gallery when no corner is named", async () => {
        // given
        endpoints.listAllGalleries.mockResolvedValue([{ id: "g-1" }]);

        // when
        const { result, queryClient } = setup(() => useAllGalleries());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.listAllGalleries).toHaveBeenCalledWith(undefined);
        expect(firstKey(queryClient)).toEqual(["galleries", "all", ""]);
        expect(result.current.galleries).toEqual([{ id: "g-1" }]);
    });

    it("keys the cache entry by the corner it was given", async () => {
        // given
        const { result, queryClient } = setup(() => useAllGalleries("fanart"));

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(endpoints.listAllGalleries).toHaveBeenCalledWith("fanart");
        expect(firstKey(queryClient)).toEqual(["galleries", "all", "fanart"]);
    });

    it("stays idle while it is switched off", () => {
        // given
        const { result } = setup(() => useAllGalleries("fanart", false));

        // when
        const current = result.current;

        // then
        expect(endpoints.listAllGalleries).not.toHaveBeenCalled();
        expect(current.galleries).toEqual([]);
        expect(current.loading).toBe(false);
    });
});
