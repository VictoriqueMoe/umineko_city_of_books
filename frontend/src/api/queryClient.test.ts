import { QueryClient } from "@tanstack/react-query";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "./client";
import { queryClient } from "./queryClient";

afterEach(() => {
    queryClient.clear();
});

describe("queryClient", () => {
    it("is a single shared query client for the whole app", async () => {
        // when
        const reimported = await import("./queryClient");

        // then
        expect(queryClient).toBeInstanceOf(QueryClient);
        expect(reimported.queryClient).toBe(queryClient);
    });

    it("keeps fetched data fresh for thirty seconds and cached for five minutes", () => {
        // when
        const queries = queryClient.getDefaultOptions().queries;

        // then
        expect(queries?.staleTime).toBe(30_000);
        expect(queries?.gcTime).toBe(300_000);
    });

    it("does not refetch when the window regains focus", () => {
        // then
        expect(queryClient.getDefaultOptions().queries?.refetchOnWindowFocus).toBe(false);
    });

    it("retries a query once but never retries a mutation", () => {
        // when
        const defaults = queryClient.getDefaultOptions();
        const retry = defaults.queries?.retry as (failureCount: number, error: Error) => boolean;

        // then
        expect(retry(0, new Error("network down"))).toBe(true);
        expect(retry(1, new Error("network down"))).toBe(false);
        expect(defaults.mutations?.retry).toBe(0);
    });

    it("never retries a client error, because the answer will not change", () => {
        // given
        const retry = queryClient.getDefaultOptions().queries?.retry as (failureCount: number, error: Error) => boolean;

        // then
        expect(retry(0, new ApiError(404, "post not found", null))).toBe(false);
        expect(retry(0, new ApiError(403, "forbidden", null))).toBe(false);
        expect(retry(0, new ApiError(500, "server exploded", null))).toBe(true);
    });

    it("serves a second fetch of the same key from the cache while it is still fresh", async () => {
        // given
        const queryFn = vi.fn((): Promise<string> => Promise.resolve("beatrice"));

        // when
        await queryClient.fetchQuery({ queryKey: ["queryClient", "fresh"], queryFn });
        const second = await queryClient.fetchQuery({ queryKey: ["queryClient", "fresh"], queryFn });

        // then
        expect(second).toBe("beatrice");
        expect(queryFn).toHaveBeenCalledOnce();
    });

    it("calls a failing query function twice before giving up", async () => {
        // given
        vi.useFakeTimers();
        const queryFn = vi.fn((): Promise<string> => Promise.reject(new Error("boom")));
        const settled = queryClient
            .fetchQuery({ queryKey: ["queryClient", "retry"], queryFn })
            .catch((error: unknown) => error);

        // when
        await vi.advanceTimersByTimeAsync(10_000);

        // then
        await expect(settled).resolves.toBeInstanceOf(Error);
        expect(queryFn).toHaveBeenCalledTimes(2);
    });

    it("stores and reads data under a query key", () => {
        // given
        queryClient.setQueryData(["queryClient", "manual"], { name: "Beatrice" });

        // when
        const stored = queryClient.getQueryData(["queryClient", "manual"]);

        // then
        expect(stored).toEqual({ name: "Beatrice" });
        expect(queryClient.getQueryData(["queryClient", "other"])).toBeUndefined();
    });

    it("drops every cached query when it is cleared", () => {
        // given
        queryClient.setQueryData(["queryClient", "manual"], { name: "Beatrice" });

        // when
        queryClient.clear();

        // then
        expect(queryClient.getQueryData(["queryClient", "manual"])).toBeUndefined();
        expect(queryClient.getQueryCache().getAll()).toHaveLength(0);
    });
});
