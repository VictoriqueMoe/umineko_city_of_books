import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useIsMobile } from "./useIsMobile";

const capacitor = vi.hoisted(() => ({ native: false }));

vi.mock("@capacitor/core", () => ({
    Capacitor: {
        isNativePlatform: () => capacitor.native,
        getPlatform: () => (capacitor.native ? "android" : "web"),
    },
}));

interface FakeMediaQueryList {
    matches: boolean;
    media: string;
    onchange: null;
    addListener: () => void;
    removeListener: () => void;
    addEventListener: (type: string, listener: () => void) => void;
    removeEventListener: (type: string, listener: () => void) => void;
    dispatchEvent: () => boolean;
}

function stubMatchMedia(matches: boolean) {
    const listeners = new Set<() => void>();
    const mql: FakeMediaQueryList = {
        matches,
        media: "(max-width: 960px)",
        onchange: null,
        addListener: () => {},
        removeListener: () => {},
        addEventListener: (_type: string, listener: () => void) => {
            listeners.add(listener);
        },
        removeEventListener: (_type: string, listener: () => void) => {
            listeners.delete(listener);
        },
        dispatchEvent: () => false,
    };
    const matchMedia = vi.fn(() => mql);
    vi.stubGlobal("matchMedia", matchMedia);

    function emitChange(next: boolean) {
        mql.matches = next;
        act(() => {
            for (const listener of listeners) {
                listener();
            }
        });
    }

    return { listeners, matchMedia, emitChange };
}

afterEach(() => {
    capacitor.native = false;
});

describe("useIsMobile", () => {
    it("reports a desktop viewport when the query does not match", () => {
        // given
        stubMatchMedia(false);

        // when
        const { result } = renderHook(() => useIsMobile());

        // then
        expect(result.current).toBe(false);
    });

    it("reports a mobile viewport when the query already matches on mount", () => {
        // given
        stubMatchMedia(true);

        // when
        const { result } = renderHook(() => useIsMobile());

        // then
        expect(result.current).toBe(true);
    });

    it("asks for the max-width 960px breakpoint", () => {
        // given
        const { matchMedia } = stubMatchMedia(false);

        // when
        renderHook(() => useIsMobile());

        // then
        expect(matchMedia).toHaveBeenCalledWith("(max-width: 960px)");
    });

    it("switches to mobile when the viewport shrinks past the breakpoint", () => {
        // given
        const { emitChange } = stubMatchMedia(false);
        const { result } = renderHook(() => useIsMobile());

        // when
        emitChange(true);

        // then
        expect(result.current).toBe(true);
    });

    it("switches back to desktop when the viewport grows again", () => {
        // given
        const { emitChange } = stubMatchMedia(true);
        const { result } = renderHook(() => useIsMobile());

        // when
        emitChange(false);

        // then
        expect(result.current).toBe(false);
    });

    it("stops listening for viewport changes once it unmounts", () => {
        // given
        const { listeners } = stubMatchMedia(false);
        const { unmount } = renderHook(() => useIsMobile());
        expect(listeners.size).toBe(1);

        // when
        unmount();

        // then
        expect(listeners.size).toBe(0);
    });

    it("always reports mobile in the native app without consulting the media query", () => {
        // given
        capacitor.native = true;
        const { matchMedia } = stubMatchMedia(false);

        // when
        const { result } = renderHook(() => useIsMobile());

        // then
        expect(result.current).toBe(true);
        expect(matchMedia).not.toHaveBeenCalled();
    });
});
