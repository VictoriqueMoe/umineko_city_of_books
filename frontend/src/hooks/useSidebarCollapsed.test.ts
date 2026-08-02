import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useSidebarCollapsed } from "./useSidebarCollapsed";

const STORAGE_KEY = "sidebar-collapsed";

afterEach(() => {
    vi.restoreAllMocks();
});

describe("useSidebarCollapsed", () => {
    it("starts expanded when nothing has been stored yet", () => {
        // given
        expect(window.localStorage.getItem(STORAGE_KEY)).toBeNull();

        // when
        const { result } = renderHook(() => useSidebarCollapsed());

        // then
        expect(result.current[0]).toBe(false);
    });

    it("starts collapsed when the stored preference says so", () => {
        // given
        window.localStorage.setItem(STORAGE_KEY, "true");

        // when
        const { result } = renderHook(() => useSidebarCollapsed());

        // then
        expect(result.current[0]).toBe(true);
    });

    it("treats any other stored value as expanded", () => {
        // given
        window.localStorage.setItem(STORAGE_KEY, "TRUE");

        // when
        const { result } = renderHook(() => useSidebarCollapsed());

        // then
        expect(result.current[0]).toBe(false);
    });

    it("writes the starting state to storage on the first render", () => {
        // given
        const stored = window.localStorage.getItem(STORAGE_KEY);
        expect(stored).toBeNull();

        // when
        renderHook(() => useSidebarCollapsed());

        // then
        expect(window.localStorage.getItem(STORAGE_KEY)).toBe("false");
    });

    it("persists the collapsed state so it survives a reload", () => {
        // given
        const { result } = renderHook(() => useSidebarCollapsed());

        // when
        act(() => {
            result.current[1](true);
        });

        // then
        expect(result.current[0]).toBe(true);
        expect(window.localStorage.getItem(STORAGE_KEY)).toBe("true");
    });

    it("persists an expansion back over a stored collapse", () => {
        // given
        window.localStorage.setItem(STORAGE_KEY, "true");
        const { result } = renderHook(() => useSidebarCollapsed());

        // when
        act(() => {
            result.current[1](false);
        });

        // then
        expect(result.current[0]).toBe(false);
        expect(window.localStorage.getItem(STORAGE_KEY)).toBe("false");
    });

    it("toggles from the previous value when given an updater", () => {
        // given
        const { result } = renderHook(() => useSidebarCollapsed());

        // when
        act(() => {
            result.current[1](prev => !prev);
        });

        // then
        expect(result.current[0]).toBe(true);
        act(() => {
            result.current[1](prev => !prev);
        });
        expect(result.current[0]).toBe(false);
    });

    it("keeps the setter identity stable across renders", () => {
        // given
        const { result, rerender } = renderHook(() => useSidebarCollapsed());
        const setter = result.current[1];

        // when
        act(() => {
            result.current[1](true);
        });
        rerender();

        // then
        expect(result.current[1]).toBe(setter);
    });

    it("falls back to expanded when storage cannot be read", () => {
        // given
        vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
            throw new Error("storage is blocked");
        });

        // when
        const { result } = renderHook(() => useSidebarCollapsed());

        // then
        expect(result.current[0]).toBe(false);
    });

    it("keeps working when storage cannot be written", () => {
        // given
        vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
            throw new Error("storage is blocked");
        });
        const { result } = renderHook(() => useSidebarCollapsed());

        // when
        const update = () =>
            act(() => {
                result.current[1](true);
            });

        // then
        expect(update).not.toThrow();
        expect(result.current[0]).toBe(true);
    });
});
