import type { QueryClient } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { useCharacterList } from "./character";

const endpoints = vi.hoisted(() => ({
    listCharacters: vi.fn(),
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
    endpoints.listCharacters.mockResolvedValue({ characters: [] });
});

describe("useCharacterList", () => {
    it("loads the cast of the series it was given", async () => {
        // given
        endpoints.listCharacters.mockResolvedValue({ characters: [{ id: "beatrice", name: "Beatrice" }] });

        // when
        const { result, queryClient } = setup(() => useCharacterList("umineko"));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.listCharacters).toHaveBeenCalledWith("umineko");
        expect(firstKey(queryClient)).toEqual(["characters", "series", "umineko"]);
        expect(result.current.characters).toEqual([{ id: "beatrice", name: "Beatrice" }]);
    });

    it("keys each series separately", () => {
        // given
        const { queryClient } = setup(() => useCharacterList("higurashi"));

        // when
        const key = firstKey(queryClient);

        // then
        expect(key).toEqual(["characters", "series", "higurashi"]);
    });

    it("stays idle when no series has been chosen", () => {
        // given
        const { result } = setup(() => useCharacterList(""));

        // when
        const current = result.current;

        // then
        expect(endpoints.listCharacters).not.toHaveBeenCalled();
        expect(current.characters).toEqual([]);
        expect(current.loading).toBe(false);
    });

    it("stays idle for original characters because they have no fixed cast", () => {
        // given
        const { result } = setup(() => useCharacterList("oc"));

        // when
        const current = result.current;

        // then
        expect(endpoints.listCharacters).not.toHaveBeenCalled();
        expect(current.characters).toEqual([]);
    });

    it("stays idle while the caller has switched it off", () => {
        // given
        const { result } = setup(() => useCharacterList("umineko", false));

        // when
        const current = result.current;

        // then
        expect(endpoints.listCharacters).not.toHaveBeenCalled();
        expect(current.characters).toEqual([]);
    });

    it("defaults to an empty cast when the response carries no characters", async () => {
        // given
        endpoints.listCharacters.mockResolvedValue({});

        // when
        const { result } = setup(() => useCharacterList("ciconia"));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.characters).toEqual([]);
    });
});
