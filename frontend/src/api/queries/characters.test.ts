import type { QueryClient } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { useAllCharacters, useCharacterGroups, useCharactersFlat, useOCCharacters } from "./characters";

const endpoints = vi.hoisted(() => ({
    getCharacterGroups: vi.fn(),
    getCharacters: vi.fn(),
    searchOCCharacters: vi.fn(),
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
    endpoints.getCharacterGroups.mockResolvedValue({ main: {}, additional: {} });
    endpoints.getCharacters.mockResolvedValue({});
    endpoints.searchOCCharacters.mockResolvedValue([]);
});

describe("useAllCharacters", () => {
    it("gathers the umineko and higurashi casts alongside the ciconia groups", async () => {
        // given
        endpoints.getCharacters.mockImplementation((series: string) =>
            Promise.resolve(series === "umineko" ? { beatrice: "Beatrice" } : { rika: "Rika" }),
        );
        endpoints.getCharacterGroups.mockResolvedValue({ main: { miyao: "Miyao" }, additional: {} });

        // when
        const { result } = setup(() => useAllCharacters());

        // then
        await waitFor(() => expect(result.current.umineko).toEqual({ beatrice: "Beatrice" }));
        expect(endpoints.getCharacters).toHaveBeenCalledWith("umineko");
        expect(endpoints.getCharacters).toHaveBeenCalledWith("higurashi");
        expect(endpoints.getCharacterGroups).toHaveBeenCalledWith("ciconia");
        expect(result.current.higurashi).toEqual({ rika: "Rika" });
        expect(result.current.ciconia).toEqual({ main: { miyao: "Miyao" }, additional: {} });
    });

    it("registers a single cache entry for the combined cast", async () => {
        // given
        const { result, queryClient } = setup(() => useAllCharacters());

        // when
        await waitFor(() => expect(queryClient.getQueryData(["characters", "all"])).toBeDefined());

        // then
        expect(firstKey(queryClient)).toEqual(["characters", "all"]);
        expect(result.current.ciconia).toEqual({ main: {}, additional: {} });
    });

    it("hands back an empty cast of the right shape before anything has loaded", () => {
        // given
        const { result } = setup(() => useAllCharacters());

        // when
        const initial = result.current;

        // then
        expect(initial).toEqual({ umineko: {}, higurashi: {}, ciconia: { main: {}, additional: {} } });
    });

    it("keeps handing back the same empty cast while the request is in flight", () => {
        // given
        const { result, rerender } = setup(() => useAllCharacters());
        const first = result.current;

        // when
        rerender();

        // then
        expect(result.current).toBe(first);
    });
});

describe("useCharactersFlat", () => {
    it("loads the flattened cast of the series it was given", async () => {
        // given
        endpoints.getCharacters.mockResolvedValue({ beatrice: "Beatrice" });

        // when
        const { result, queryClient } = setup(() => useCharactersFlat("umineko"));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.getCharacters).toHaveBeenCalledWith("umineko");
        expect(firstKey(queryClient)).toEqual(["characters", "flat", "umineko"]);
        expect(result.current.characters).toEqual({ beatrice: "Beatrice" });
    });

    it("keys each series separately", () => {
        // given
        const { queryClient } = setup(() => useCharactersFlat("ciconia"));

        // when
        const key = firstKey(queryClient);

        // then
        expect(key).toEqual(["characters", "flat", "ciconia"]);
    });

    it("hands back an empty cast before the request settles", async () => {
        // given
        const { result } = setup(() => useCharactersFlat("higurashi"));

        // when
        const initial = result.current;

        // then
        expect(initial.loading).toBe(true);
        expect(initial.characters).toEqual({});
        await waitFor(() => expect(result.current.loading).toBe(false));
    });
});

describe("useOCCharacters", () => {
    it("searches with an empty term by default", async () => {
        // given
        const { result, queryClient } = setup(() => useOCCharacters());

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(endpoints.searchOCCharacters).toHaveBeenCalledWith("");
        expect(firstKey(queryClient)).toEqual(["characters", "oc", ""]);
    });

    it("forwards the search term and keys the cache entry by it", async () => {
        // given
        endpoints.searchOCCharacters.mockResolvedValue(["Ange", "Angelica"]);

        // when
        const { result, queryClient } = setup(() => useOCCharacters("ange"));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.searchOCCharacters).toHaveBeenCalledWith("ange");
        expect(firstKey(queryClient)).toEqual(["characters", "oc", "ange"]);
        expect(result.current.characters).toEqual(["Ange", "Angelica"]);
    });

    it("hands back an empty list before the search settles", async () => {
        // given
        const { result } = setup(() => useOCCharacters("ange"));

        // when
        const initial = result.current;

        // then
        expect(initial.characters).toEqual([]);
        await waitFor(() => expect(result.current.loading).toBe(false));
    });
});

describe("useCharacterGroups", () => {
    it("loads the grouped cast of the series it was given", async () => {
        // given
        endpoints.getCharacterGroups.mockResolvedValue({ main: { miyao: "Miyao" }, additional: { jayden: "Jayden" } });

        // when
        const { result, queryClient } = setup(() => useCharacterGroups("ciconia"));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.getCharacterGroups).toHaveBeenCalledWith("ciconia");
        expect(firstKey(queryClient)).toEqual(["character-groups", "ciconia"]);
        expect(result.current.groups).toEqual({ main: { miyao: "Miyao" }, additional: { jayden: "Jayden" } });
    });

    it("hands back empty groups before the request settles", async () => {
        // given
        const { result } = setup(() => useCharacterGroups("umineko"));

        // when
        const initial = result.current;

        // then
        expect(initial.loading).toBe(true);
        expect(initial.groups).toEqual({ main: {}, additional: {} });
        await waitFor(() => expect(result.current.loading).toBe(false));
    });

    it("keeps handing back the same empty groups while the request is in flight", () => {
        // given
        const { result, rerender } = setup(() => useCharacterGroups("umineko"));
        const first = result.current.groups;

        // when
        rerender();

        // then
        expect(result.current.groups).toBe(first);
    });
});
