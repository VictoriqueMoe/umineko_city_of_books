import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { queryKeys } from "../../api/queryKeys";
import { providerWrapper } from "../../test-utils/render";
import type { OverlayConnection } from "../../types/api";
import { useOverlayConnectorFile, useResetOverlayToken, useTestOverlay } from "./overlay";

const mocks = vi.hoisted(() => ({
    fetchOverlayConnectorSEF: vi.fn(),
    resetOverlayToken: vi.fn(),
    testOverlay: vi.fn(),
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

function setup<T>(hook: () => T) {
    const queryClient = createRetainingQueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(hook, { wrapper: providerWrapper({ queryClient }) });

    return {
        result,
        queryClient,
        invalidated: () => invalidate.mock.calls.map(call => JSON.stringify(call[0]?.queryKey)),
    };
}

function seedConnection(connected: boolean): OverlayConnection {
    return { token: "sammi-token", connect_url: "wss://overlay.example", connected };
}

beforeEach(() => {
    mocks.fetchOverlayConnectorSEF.mockResolvedValue('{"sammi":true}');
    mocks.resetOverlayToken.mockResolvedValue(seedConnection(false));
    mocks.testOverlay.mockResolvedValue({ ok: true });
});

describe("useResetOverlayToken", () => {
    it("refreshes the connection query rather than writing the new token into the cache", async () => {
        // given
        const { result, queryClient, invalidated } = setup(() => useResetOverlayToken());

        // when
        act(() => {
            result.current.mutate();
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(invalidated()).toContain(JSON.stringify(queryKeys.overlay.connection()));
        expect(queryClient.getQueryData(queryKeys.overlay.connection())).toBeUndefined();
    });

    it("surfaces a refused reset", async () => {
        // given
        mocks.resetOverlayToken.mockRejectedValue(new Error("could not reset your token"));
        const { result } = setup(() => useResetOverlayToken());

        // when
        act(() => {
            result.current.mutate();
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(result.current.error?.message).toBe("could not reset your token");
    });
});

describe("useTestOverlay", () => {
    it("marks a cached connection as connected when the test lands", async () => {
        // given
        const { result, queryClient } = setup(() => useTestOverlay());
        queryClient.setQueryData<OverlayConnection>(queryKeys.overlay.connection(), seedConnection(false));

        // when
        act(() => {
            result.current.mutate();
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(queryClient.getQueryData<OverlayConnection>(queryKeys.overlay.connection())?.connected).toBe(true);
    });

    it("marks a cached connection as disconnected when the test fails", async () => {
        // given
        mocks.testOverlay.mockRejectedValue(new Error("SAMMI is not listening"));
        const { result, queryClient } = setup(() => useTestOverlay());
        queryClient.setQueryData<OverlayConnection>(queryKeys.overlay.connection(), seedConnection(true));

        // when
        act(() => {
            result.current.mutate();
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(queryClient.getQueryData<OverlayConnection>(queryKeys.overlay.connection())?.connected).toBe(false);
        expect(result.current.error?.message).toBe("SAMMI is not listening");
    });

    it("never invents a connection entry when nothing is cached", async () => {
        // given
        const { result, queryClient } = setup(() => useTestOverlay());

        // when
        act(() => {
            result.current.mutate();
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(queryClient.getQueryData(queryKeys.overlay.connection())).toBeUndefined();
    });
});

describe("useOverlayConnectorFile", () => {
    it("returns the connector as a blob for the caller to save", async () => {
        // given
        const { result } = setup(() => useOverlayConnectorFile());

        // when
        act(() => {
            result.current.mutate();
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(result.current.data).toBeInstanceOf(Blob);
        expect(result.current.data?.type).toBe("text/plain");
        await expect(result.current.data?.text()).resolves.toBe('{"sammi":true}');
    });

    it("surfaces the download failure instead of a silent no-op", async () => {
        // given
        mocks.fetchOverlayConnectorSEF.mockRejectedValue(new Error("Could not download the connector file."));
        const { result } = setup(() => useOverlayConnectorFile());

        // when
        act(() => {
            result.current.mutate();
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(result.current.error?.message).toBe("Could not download the connector file.");
    });
});
