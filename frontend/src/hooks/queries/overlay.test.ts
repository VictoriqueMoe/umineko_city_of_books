import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { queryKeys } from "../../api/queryKeys";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { useOverlayConnection } from "./overlay";

const mocks = vi.hoisted(() => ({
    getOverlayConnection: vi.fn(),
}));

vi.mock("../../api/endpoints/overlay", () => mocks);

function createRetainingQueryClient(): QueryClient {
    return new QueryClient({
        defaultOptions: {
            queries: { retry: false, gcTime: 5 * 60_000, staleTime: 30_000, refetchOnWindowFocus: false },
            mutations: { retry: false },
        },
    });
}

let queryClient: QueryClient;

beforeEach(() => {
    queryClient = createTestQueryClient();
    mocks.getOverlayConnection.mockResolvedValue({
        token: "sammi-token",
        connect_url: "wss://overlay.example",
        connected: false,
    });
});

describe("useOverlayConnection", () => {
    it("reports the connection once it arrives", async () => {
        // given
        const { result } = renderHook(() => useOverlayConnection(), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => {
            expect(result.current.connection?.token).toBe("sammi-token");
        });
        expect(result.current.connection?.connected).toBe(false);
        expect(result.current.error).toBe("");
    });

    it("surfaces the failure the settings section used to render as a friendly default", async () => {
        // given
        mocks.getOverlayConnection.mockRejectedValue(new Error("overlay is down"));

        // when
        const { result } = renderHook(() => useOverlayConnection(), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => {
            expect(result.current.error).toBe("overlay is down");
        });
        expect(result.current.connection).toBeNull();
    });

    it("asks for nothing while it is disabled", () => {
        // given
        renderHook(() => useOverlayConnection(false), { wrapper: providerWrapper({ queryClient }) });

        // then
        expect(mocks.getOverlayConnection).not.toHaveBeenCalled();
    });

    it("drops the overlay token out of the cache the moment the last reader unmounts", async () => {
        // given a client that would otherwise hold every query for five minutes
        const retaining = createRetainingQueryClient();
        const { result, unmount } = renderHook(() => useOverlayConnection(), {
            wrapper: providerWrapper({ queryClient: retaining }),
        });
        await waitFor(() => {
            expect(result.current.connection).not.toBeNull();
        });

        // when
        unmount();

        // then
        await waitFor(() => {
            expect(retaining.getQueryData(queryKeys.overlay.connection())).toBeUndefined();
        });
    });
});
