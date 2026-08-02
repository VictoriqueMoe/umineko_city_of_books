import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, vi } from "vitest";

class MockObserver {
    observe(): void {}
    unobserve(): void {}
    disconnect(): void {}
    takeRecords(): [] {
        return [];
    }
}

function define(target: object, key: string, value: unknown): void {
    Object.defineProperty(target, key, { configurable: true, writable: true, value });
}

if (!window.matchMedia) {
    define(window, "matchMedia", (query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: () => {},
        removeListener: () => {},
        addEventListener: () => {},
        removeEventListener: () => {},
        dispatchEvent: () => false,
    }));
}

define(globalThis, "ResizeObserver", MockObserver);
define(globalThis, "IntersectionObserver", MockObserver);

define(Element.prototype, "scrollIntoView", () => {});
define(Element.prototype, "scrollTo", () => {});
define(window, "scrollTo", () => {});

define(HTMLMediaElement.prototype, "play", () => Promise.resolve());
define(HTMLMediaElement.prototype, "pause", () => {});
define(HTMLMediaElement.prototype, "load", () => {});

if (!URL.createObjectURL) {
    define(URL, "createObjectURL", () => "blob:mock");
    define(URL, "revokeObjectURL", () => {});
}

if (!navigator.clipboard) {
    define(navigator, "clipboard", {
        writeText: () => Promise.resolve(),
        readText: () => Promise.resolve(""),
    });
}

afterEach(() => {
    cleanup();
    localStorage.clear();
    vi.useRealTimers();
});
