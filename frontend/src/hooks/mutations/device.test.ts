import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useDeviceTokens } from "./device";

const { endpoints } = vi.hoisted(() => ({
    endpoints: { registerDeviceToken: vi.fn(), unregisterDeviceToken: vi.fn() },
}));

vi.mock("../../api/endpoints/auth", () => endpoints);

beforeEach(() => {
    endpoints.registerDeviceToken.mockResolvedValue(undefined);
    endpoints.unregisterDeviceToken.mockResolvedValue(undefined);
});

describe("useDeviceTokens", () => {
    it("registers a token against the account for the platform that produced it", async () => {
        // given
        const { result } = renderHook(() => useDeviceTokens());

        // when
        await result.current.registerToken("fid-abc123", "web");

        // then
        expect(endpoints.registerDeviceToken).toHaveBeenCalledWith("fid-abc123", "web");
    });

    it("withdraws a token from the account", async () => {
        // given
        const { result } = renderHook(() => useDeviceTokens());

        // when
        await result.current.unregisterToken("fid-abc123");

        // then
        expect(endpoints.unregisterDeviceToken).toHaveBeenCalledWith("fid-abc123");
    });

    it("keeps the same pair across renders, so an effect that depends on it does not re-run", () => {
        // given
        const { result, rerender } = renderHook(() => useDeviceTokens());
        const first = result.current;

        // when
        rerender();

        // then
        expect(result.current).toBe(first);
    });
});
