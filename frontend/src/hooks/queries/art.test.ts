import type { QueryClient } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { TagCount } from "../../types/api";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { useAllGalleries, useArt, useArtCornerCounts, useArtFeed, useGallery, usePopularTags } from "./art";

const endpoints = vi.hoisted(() => ({
    getArt: vi.fn(),
    getArtCornerCounts: vi.fn(),
    getGallery: vi.fn(),
    getPopularTags: vi.fn(),
    listAllGalleries: vi.fn(),
    listArt: vi.fn(),
}));

vi.mock("../../api/endpoints/art", () => endpoints);

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
    endpoints.getArtCornerCounts.mockResolvedValue({});
    endpoints.getGallery.mockResolvedValue({ gallery: { id: "g-1" }, art: [], total: 0 });
    endpoints.getPopularTags.mockResolvedValue([]);
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
    });

    it("asks for the window of art it was given", async () => {
        // given
        const { result } = setup(() => useArtFeed("general", "", "", "", "", 48, 12));

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(endpoints.listArt).toHaveBeenCalledWith(expect.objectContaining({ offset: 48, limit: 12 }));
    });

    it("drops empty filter strings so the endpoint sees no filter at all", async () => {
        // given
        const { result } = setup(() => useArtFeed("fanart", "", "", "", "", 0));

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
        const { result } = setup(() => useArtFeed("fanart", "sketch", "beato", "witch", "top", 24));

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

    it("keys the cache entry by every filter it was given", () => {
        // given
        const { queryClient } = setup(() => useArtFeed("fanart", "sketch", "beato", "witch", "top", 24, 24));

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
            },
        ]);
    });

    it("hands back the art and the total the server sent", async () => {
        // given
        endpoints.listArt.mockResolvedValue({ art: [{ id: "art-1" }], total: 30 });

        // when
        const { result } = setup(() => useArtFeed("general", undefined, undefined, undefined, undefined, 0));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.art).toEqual([{ id: "art-1" }]);
        expect(result.current.total).toBe(30);
    });

    it("reports nothing at all before the first response arrives", () => {
        // given
        const { result } = setup(() => useArtFeed("general", undefined, undefined, undefined, undefined, 24));

        // when
        const initial = result.current;

        // then
        expect(initial.art).toEqual([]);
        expect(initial.total).toBe(0);
        expect(initial.loading).toBe(true);
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

describe("useArtCornerCounts", () => {
    it("returns the art counts under the art corner counts key", async () => {
        // given
        endpoints.getArtCornerCounts.mockResolvedValue({ gallery: 9 });
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => useArtCornerCounts(), {
            wrapper: providerWrapper({ queryClient: client }),
        });

        // then
        await waitFor(() => expect(result.current.counts).toEqual({ gallery: 9 }));
        expect(client.getQueryData(["art", "corner-counts"])).toEqual({ gallery: 9 });
    });

    it("falls back to an empty map while loading", () => {
        // given
        endpoints.getArtCornerCounts.mockReturnValue(new Promise(() => {}));

        // when
        const { result } = renderHook(() => useArtCornerCounts(), { wrapper: providerWrapper() });

        // then
        expect(result.current.counts).toEqual({});
    });
});

describe("usePopularTags", () => {
    it("keys the tags by the corner it was given", async () => {
        // given
        const tags: TagCount[] = [{ tag: "beatrice", count: 12 }];
        endpoints.getPopularTags.mockResolvedValue(tags);
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => usePopularTags("gallery"), {
            wrapper: providerWrapper({ queryClient: client }),
        });

        // then
        await waitFor(() => expect(result.current.tags).toEqual(tags));
        expect(endpoints.getPopularTags).toHaveBeenCalledWith("gallery");
        expect(client.getQueryData(["art", "popular-tags", "gallery"])).toEqual(tags);
    });

    it("uses the empty corner key when no corner is given", async () => {
        // given
        endpoints.getPopularTags.mockResolvedValue([]);
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => usePopularTags(), { wrapper: providerWrapper({ queryClient: client }) });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.getPopularTags).toHaveBeenCalledWith(undefined);
        expect(client.getQueryData(["art", "popular-tags", ""])).toEqual([]);
    });
});
