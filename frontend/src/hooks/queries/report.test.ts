import type { QueryClient } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { useReports } from "./report";

const endpoints = vi.hoisted(() => ({
    getReports: vi.fn(),
}));

vi.mock("../../api/endpoints/report", () => endpoints);

function setup<T>(hook: () => T) {
    const queryClient = createTestQueryClient();
    const rendered = renderHook(hook, { wrapper: providerWrapper({ queryClient }) });

    return { ...rendered, queryClient };
}

function firstKey(queryClient: QueryClient): readonly unknown[] {
    return queryClient.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
    endpoints.getReports.mockResolvedValue({ reports: [] });
});

describe("useReports", () => {
    it("asks for the reports of a given status and keys the cache entry by it", async () => {
        // given
        endpoints.getReports.mockResolvedValue({ reports: [{ id: "r-1" }] });

        // when
        const { result, queryClient } = setup(() => useReports("resolved"));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.getReports).toHaveBeenCalledWith("resolved");
        expect(firstKey(queryClient)).toEqual(["admin", "reports", { status: "resolved" }]);
        expect(result.current.reports).toEqual([{ id: "r-1" }]);
    });

    it("defaults to an empty report list", async () => {
        // given
        endpoints.getReports.mockResolvedValue({});

        // when
        const { result } = setup(() => useReports("open"));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.reports).toEqual([]);
    });
});
