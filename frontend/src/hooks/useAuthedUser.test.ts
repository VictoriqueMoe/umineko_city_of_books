import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import { useAuthedUser } from "./useAuthedUser";

describe("useAuthedUser", () => {
    it("returns the signed in user without the null branch callers would otherwise handle", () => {
        // given
        const user = makeUser({ id: "u-7", username: "erika", favourite_character: "Erika" });

        // when
        const { result } = renderHook(() => useAuthedUser(), { wrapper: providerWrapper({ user }) });

        // then
        expect(result.current.id).toBe("u-7");
        expect(result.current.favourite_character).toBe("Erika");
    });

    it("throws when the provider has no signed in user", () => {
        // given
        const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
        const wrapper = providerWrapper({ user: null });

        // when
        const attempt = () => renderHook(() => useAuthedUser(), { wrapper });

        // then
        expect(attempt).toThrow("useAuthedUser must be used beneath a ProtectedRoute");
        consoleError.mockRestore();
    });

    it("throws the auth provider error when there is no AuthProvider at all", () => {
        // given
        const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

        // when
        const attempt = () => renderHook(() => useAuthedUser());

        // then
        expect(attempt).toThrow("useAuth must be used within an AuthProvider");
        consoleError.mockRestore();
    });

    it("still returns the user while the session is refreshing in the background", () => {
        // given
        const user = makeUser({ username: "lambdadelta" });
        const wrapper = providerWrapper({ user, auth: { loading: true } });

        // when
        const { result } = renderHook(() => useAuthedUser(), { wrapper });

        // then
        expect(result.current.username).toBe("lambdadelta");
    });
});
