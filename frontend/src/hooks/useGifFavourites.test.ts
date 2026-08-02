import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { GiphyFavourite } from "../api/endpoints";
import { providerWrapper } from "../test-utils/render";
import { useGifFavourites } from "./useGifFavourites";

function makeGif(overrides: Partial<GiphyFavourite> = {}): GiphyFavourite {
    return {
        giphy_id: "gif-1",
        url: "https://media.giphy.com/gif-1.gif",
        title: "A golden butterfly",
        preview_url: "https://media.giphy.com/gif-1-preview.gif",
        width: 200,
        height: 150,
        ...overrides,
    };
}

describe("useGifFavourites", () => {
    it("returns the favourites the provider holds", () => {
        // given
        const favourites = [makeGif(), makeGif({ giphy_id: "gif-2", title: "Seagulls" })];
        const wrapper = providerWrapper({ gifFavourites: { favourites, ids: new Set(["gif-1", "gif-2"]) } });

        // when
        const { result } = renderHook(() => useGifFavourites(), { wrapper });

        // then
        expect(result.current.favourites).toHaveLength(2);
        expect(result.current.ids.has("gif-2")).toBe(true);
    });

    it("returns an empty list and an empty id set when nothing is favourited", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useGifFavourites(), { wrapper });

        // then
        expect(result.current.favourites).toEqual([]);
        expect(result.current.ids.size).toBe(0);
        expect(result.current.isFavourite("gif-1")).toBe(false);
    });

    it("answers isFavourite using the provider implementation", () => {
        // given
        const isFavourite = vi.fn((giphyID: string) => giphyID === "gif-1");
        const wrapper = providerWrapper({ gifFavourites: { isFavourite } });

        // when
        const { result } = renderHook(() => useGifFavourites(), { wrapper });

        // then
        expect(result.current.isFavourite("gif-1")).toBe(true);
        expect(result.current.isFavourite("gif-9")).toBe(false);
        expect(isFavourite).toHaveBeenCalledTimes(2);
    });

    it("forwards the whole favourite object when toggling", async () => {
        // given
        const toggle = vi.fn(() => Promise.resolve());
        const gif = makeGif({ giphy_id: "gif-3" });
        const wrapper = providerWrapper({ gifFavourites: { toggle } });

        // when
        const { result } = renderHook(() => useGifFavourites(), { wrapper });
        await result.current.toggle(gif);

        // then
        expect(toggle).toHaveBeenCalledWith(gif);
    });

    it("forwards refresh calls to the provider", async () => {
        // given
        const refresh = vi.fn(() => Promise.resolve());
        const wrapper = providerWrapper({ gifFavourites: { refresh } });

        // when
        const { result } = renderHook(() => useGifFavourites(), { wrapper });
        await result.current.refresh();

        // then
        expect(refresh).toHaveBeenCalledOnce();
    });

    it("throws when it is used outside a GifFavouritesProvider", () => {
        // given
        const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

        // when
        const attempt = () => renderHook(() => useGifFavourites());

        // then
        expect(attempt).toThrow("useGifFavourites must be used within a GifFavouritesProvider");
        consoleError.mockRestore();
    });
});
