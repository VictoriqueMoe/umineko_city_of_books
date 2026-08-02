import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type { GiphyFavourite } from "../endpoints";
import { useAddGiphyFavourite, useRemoveGiphyFavourite } from "./giphy";

const mocks = vi.hoisted(() => ({
    addGiphyFavourite: vi.fn(),
    removeGiphyFavourite: vi.fn(),
}));

vi.mock("../endpoints", () => mocks);

const favouritesKey = ["giphy", "favourites"];

const favourite: GiphyFavourite = {
    giphy_id: "beatrice-laughs",
    url: "https://media.example.test/beatrice.gif",
    title: "the golden witch laughs",
    preview_url: "https://media.example.test/beatrice-preview.gif",
    width: 320,
    height: 240,
};

function harness() {
    const queryClient = createTestQueryClient();
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

    return { invalidateQueries, queryClient, wrapper: providerWrapper({ queryClient }) };
}

beforeEach(() => {
    mocks.addGiphyFavourite.mockResolvedValue(undefined);
    mocks.removeGiphyFavourite.mockResolvedValue(undefined);
});

describe("useAddGiphyFavourite", () => {
    it("sends the whole favourite to the api", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useAddGiphyFavourite(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(favourite);
        });

        // then
        expect(mocks.addGiphyFavourite).toHaveBeenCalledWith(favourite);
    });

    it("refreshes the saved favourites once the gif is stored", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useAddGiphyFavourite(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(favourite);
        });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: favouritesKey });
    });

    it("leaves the saved favourites alone when the gif could not be stored", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.addGiphyFavourite.mockRejectedValue(new Error("giphy is down"));
        const { result } = renderHook(() => useAddGiphyFavourite(), { wrapper });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync(favourite)).rejects.toThrow("giphy is down");
        });

        // then
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});

describe("useRemoveGiphyFavourite", () => {
    it("removes the favourite by its giphy id alone", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useRemoveGiphyFavourite(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync("beatrice-laughs");
        });

        // then
        expect(mocks.removeGiphyFavourite).toHaveBeenCalledWith("beatrice-laughs");
    });

    it("refreshes the saved favourites once the gif is gone", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useRemoveGiphyFavourite(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync("beatrice-laughs");
        });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: favouritesKey });
    });

    it("leaves the saved favourites alone when the removal fails", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.removeGiphyFavourite.mockRejectedValue(new Error("not yours"));
        const { result } = renderHook(() => useRemoveGiphyFavourite(), { wrapper });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync("beatrice-laughs")).rejects.toThrow("not yours");
        });

        // then
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});
