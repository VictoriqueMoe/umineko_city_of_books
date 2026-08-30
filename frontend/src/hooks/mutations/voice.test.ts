import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { providerWrapper } from "../../test-utils/render";
import { useVoiceToken } from "./voice";

const mocks = vi.hoisted(() => ({
    getVoiceToken: vi.fn(),
}));

vi.mock("../../api/endpoints/chat", () => mocks);

const roomId = "11111111-1111-1111-1111-111111111111";

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
    const { result } = renderHook(hook, { wrapper: providerWrapper({ queryClient }) });

    return { result, queryClient };
}

beforeEach(() => {
    mocks.getVoiceToken.mockResolvedValue({ token: "lk-token", url: "wss://livekit.test" });
});

describe("useVoiceToken", () => {
    it("returns the room voice token and never caches it under a query key", async () => {
        // given
        const { result, queryClient } = setup(() => useVoiceToken());

        // when
        act(() => {
            result.current.mutate(roomId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.getVoiceToken).toHaveBeenCalledWith(roomId);
        expect(result.current.data).toEqual({ token: "lk-token", url: "wss://livekit.test" });
        expect(queryClient.getQueryCache().getAll()).toEqual([]);
    });

    it("surfaces a refused join so the button never silently does nothing", async () => {
        // given
        mocks.getVoiceToken.mockRejectedValue(new Error("no voice for you"));
        const { result } = setup(() => useVoiceToken());

        // when
        act(() => {
            result.current.mutate(roomId);
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(result.current.error?.message).toBe("no voice for you");
    });
});
