import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type { SidebarLastVisitedResponse } from "../../types/api";
import { useMarkSidebarVisited } from "./sidebar";

const mocks = vi.hoisted(() => ({
    markSidebarVisited: vi.fn(),
}));

vi.mock("../endpoints", () => mocks);

const cacheKey = ["sidebar", "last-visited"];

function setup() {
    const queryClient = createTestQueryClient();
    queryClient.setQueryDefaults(cacheKey, { gcTime: Infinity });
    const { result } = renderHook(() => useMarkSidebarVisited(), { wrapper: providerWrapper({ queryClient }) });

    return { result, queryClient };
}

function visited(queryClient: ReturnType<typeof setup>["queryClient"]): Record<string, string> | undefined {
    return queryClient.getQueryData<SidebarLastVisitedResponse>(cacheKey)?.visited;
}

beforeEach(() => {
    mocks.markSidebarVisited.mockResolvedValue(undefined);
});

describe("useMarkSidebarVisited", () => {
    it("posts the key it is given to the sidebar endpoint", async () => {
        // given
        const { result } = setup();

        // when
        act(() => {
            result.current.mutate("chat");
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.markSidebarVisited).toHaveBeenCalledWith("chat");
    });

    it("marks the key as visited in the cache before the request has settled", async () => {
        // given
        mocks.markSidebarVisited.mockReturnValue(new Promise(() => {}));
        const { result, queryClient } = setup();

        // when
        act(() => {
            result.current.mutate("theories");
        });

        // then
        await waitFor(() => expect(result.current.isPending).toBe(true));
        expect(Object.keys(visited(queryClient) ?? {})).toEqual(["theories"]);
    });

    it("keeps the keys that were already marked visited", async () => {
        // given
        const { result, queryClient } = setup();
        queryClient.setQueryData<SidebarLastVisitedResponse>(cacheKey, {
            visited: { chat: "2026-07-01T00:00:00.000Z" },
        });

        // when
        act(() => {
            result.current.mutate("theories");
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(visited(queryClient)?.chat).toBe("2026-07-01T00:00:00.000Z");
        expect(visited(queryClient)?.theories).toBeDefined();
    });

    it("stamps the visit with the current time", async () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T09:30:00.000Z"));
        const { result, queryClient } = setup();

        // when
        await act(async () => {
            result.current.mutate("art");
        });

        // then
        expect(visited(queryClient)).toEqual({ art: "2026-08-02T09:30:00.000Z" });
    });

    it("overwrites the timestamp when the same key is visited again", async () => {
        // given
        const { result, queryClient } = setup();
        queryClient.setQueryData<SidebarLastVisitedResponse>(cacheKey, {
            visited: { chat: "2020-01-01T00:00:00.000Z" },
        });

        // when
        act(() => {
            result.current.mutate("chat");
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(visited(queryClient)?.chat).not.toBe("2020-01-01T00:00:00.000Z");
    });

    it("drops the optimistic entry when the request fails so the section stays marked unread", async () => {
        // given
        mocks.markSidebarVisited.mockRejectedValue(new Error("network down"));
        const { result, queryClient } = setup();

        // when
        act(() => {
            result.current.mutate("chat");
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(visited(queryClient)?.chat).toBeUndefined();
    });

    it("puts the previous timestamp back when the request fails", async () => {
        // given
        mocks.markSidebarVisited.mockRejectedValue(new Error("network down"));
        const { result, queryClient } = setup();
        queryClient.setQueryData<SidebarLastVisitedResponse>(cacheKey, {
            visited: { chat: "2020-01-01T00:00:00.000Z" },
        });

        // when
        act(() => {
            result.current.mutate("chat");
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(visited(queryClient)?.chat).toBe("2020-01-01T00:00:00.000Z");
    });

    it("leaves the other visited keys alone when one request fails", async () => {
        // given
        mocks.markSidebarVisited.mockRejectedValue(new Error("network down"));
        const { result, queryClient } = setup();
        queryClient.setQueryData<SidebarLastVisitedResponse>(cacheKey, {
            visited: { theories: "2026-07-01T00:00:00.000Z" },
        });

        // when
        act(() => {
            result.current.mutate("chat");
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(visited(queryClient)).toEqual({ theories: "2026-07-01T00:00:00.000Z" });
    });
});
