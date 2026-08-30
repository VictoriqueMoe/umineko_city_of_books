import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../test-utils/render";
import { useRefreshAll } from "./useRefreshAll";

function harness() {
    const queryClient = createTestQueryClient();
    const refetchQueries = vi.spyOn(queryClient, "refetchQueries").mockResolvedValue(undefined);

    return { queryClient, refetchQueries, wrapper: providerWrapper({ queryClient }) };
}

describe("useRefreshAll", () => {
    it("refetches only the queries something is still watching", async () => {
        // given
        const { refetchQueries, wrapper } = harness();
        const { result } = renderHook(() => useRefreshAll(), { wrapper });

        // when
        await result.current();

        // then
        expect(refetchQueries).toHaveBeenCalledExactlyOnceWith({ type: "active" });
    });

    it("waits for the refetch before it settles", async () => {
        // given
        const { refetchQueries, wrapper } = harness();
        let release = () => {};
        refetchQueries.mockReturnValue(
            new Promise<void>(resolve => {
                release = resolve;
            }),
        );
        const { result } = renderHook(() => useRefreshAll(), { wrapper });

        // when
        let settled = false;
        const pending = result.current().then(() => {
            settled = true;
        });

        // then
        await Promise.resolve();
        expect(settled).toBe(false);
        release();
        await pending;
        expect(settled).toBe(true);
    });

    it("hands back the same callback across renders so a listener never has to rebind", () => {
        // given
        const { wrapper } = harness();
        const { result, rerender } = renderHook(() => useRefreshAll(), { wrapper });
        const first = result.current;

        // when
        rerender();

        // then
        expect(result.current).toBe(first);
    });

    it("lets the caller reach the client the provider handed down rather than a module singleton", async () => {
        // given
        const { queryClient, refetchQueries, wrapper } = harness();
        const { result } = renderHook(() => useRefreshAll(), { wrapper });

        // when
        await result.current();

        // then
        expect(refetchQueries.mock.instances[0]).toBe(queryClient);
    });
});
