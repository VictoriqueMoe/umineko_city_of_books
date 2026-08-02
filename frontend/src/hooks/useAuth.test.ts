import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import { useAuth } from "./useAuth";

describe("useAuth", () => {
    it("returns the signed in user supplied by the provider", () => {
        // given
        const user = makeUser({ username: "battler", display_name: "Battler" });

        // when
        const { result } = renderHook(() => useAuth(), { wrapper: providerWrapper({ user }) });

        // then
        expect(result.current.user).toEqual(user);
        expect(result.current.loading).toBe(false);
    });

    it("returns a null user when nobody is signed in", () => {
        // given
        const wrapper = providerWrapper({ user: null });

        // when
        const { result } = renderHook(() => useAuth(), { wrapper });

        // then
        expect(result.current.user).toBeNull();
    });

    it("surfaces the loading flag while the session is being resolved", () => {
        // given
        const wrapper = providerWrapper({ auth: { loading: true } });

        // when
        const { result } = renderHook(() => useAuth(), { wrapper });

        // then
        expect(result.current.loading).toBe(true);
    });

    it("hands back the very callbacks the provider was given", async () => {
        // given
        const logoutUser = vi.fn(() => Promise.resolve());
        const setUser = vi.fn();
        const wrapper = providerWrapper({ auth: { logoutUser, setUser } });

        // when
        const { result } = renderHook(() => useAuth(), { wrapper });
        await result.current.logoutUser();
        result.current.setUser(null);

        // then
        expect(logoutUser).toHaveBeenCalledOnce();
        expect(setUser).toHaveBeenCalledWith(null);
    });

    it("throws when it is used outside an AuthProvider", () => {
        // given
        const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

        // when
        const attempt = () => renderHook(() => useAuth());

        // then
        expect(attempt).toThrow("useAuth must be used within an AuthProvider");
        consoleError.mockRestore();
    });
});
