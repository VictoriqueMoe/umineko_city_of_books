import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { providerWrapper } from "../test-utils/render";
import { useTheme } from "./useTheme";

describe("useTheme", () => {
    it("returns the theme and font the provider holds", () => {
        // given
        const wrapper = providerWrapper({ theme: { theme: "bernkastel", font: "im-fell" } });

        // when
        const { result } = renderHook(() => useTheme(), { wrapper });

        // then
        expect(result.current.theme).toBe("bernkastel");
        expect(result.current.font).toBe("im-fell");
    });

    it("defaults to the featherine theme with the default font", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useTheme(), { wrapper });

        // then
        expect(result.current.theme).toBe("featherine");
        expect(result.current.font).toBe("default");
    });

    it("surfaces the layout and particle preferences", () => {
        // given
        const wrapper = providerWrapper({ theme: { wideLayout: true, particlesEnabled: false } });

        // when
        const { result } = renderHook(() => useTheme(), { wrapper });

        // then
        expect(result.current.wideLayout).toBe(true);
        expect(result.current.particlesEnabled).toBe(false);
    });

    it("passes setter calls straight through to the provider", () => {
        // given
        const setTheme = vi.fn();
        const setFont = vi.fn();
        const setWideLayout = vi.fn();
        const wrapper = providerWrapper({ theme: { setTheme, setFont, setWideLayout } });

        // when
        const { result } = renderHook(() => useTheme(), { wrapper });
        result.current.setTheme("beatrice");
        result.current.setFont("im-fell");
        result.current.setWideLayout(true);

        // then
        expect(setTheme).toHaveBeenCalledWith("beatrice");
        expect(setFont).toHaveBeenCalledWith("im-fell");
        expect(setWideLayout).toHaveBeenCalledWith(true);
    });

    it("answers secret unlock questions using the provider implementation", () => {
        // given
        const hasSecret = vi.fn((id: string) => id === "golden-witch");
        const addSecret = vi.fn();
        const wrapper = providerWrapper({ theme: { hasSecret, addSecret } });

        // when
        const { result } = renderHook(() => useTheme(), { wrapper });

        // then
        expect(result.current.hasSecret("golden-witch")).toBe(true);
        expect(result.current.hasSecret("unknown")).toBe(false);
        result.current.addSecret("golden-witch");
        expect(addSecret).toHaveBeenCalledWith("golden-witch");
    });

    it("throws when it is used outside a ThemeProvider", () => {
        // given
        const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

        // when
        const attempt = () => renderHook(() => useTheme());

        // then
        expect(attempt).toThrow("useTheme must be used within a ThemeProvider");
        consoleError.mockRestore();
    });
});
