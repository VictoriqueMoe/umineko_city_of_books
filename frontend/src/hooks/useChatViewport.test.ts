import { renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useChatViewport } from "./useChatViewport";

type ViewportListener = (event: Event) => void;

interface ViewportHandle {
    setHeight: (height: number) => void;
    setOffsetTop: (offsetTop: number) => void;
    emit: (type: string) => void;
    listenerCount: (type: string) => number;
}

const viewportDescriptor = Object.getOwnPropertyDescriptor(window, "visualViewport");
const innerHeightDescriptor = Object.getOwnPropertyDescriptor(window, "innerHeight");

function restore(key: string, descriptor: PropertyDescriptor | undefined): void {
    if (descriptor) {
        Object.defineProperty(window, key, descriptor);
        return;
    }

    Reflect.deleteProperty(window, key);
}

function setInnerHeight(value: number): void {
    Object.defineProperty(window, "innerHeight", { configurable: true, writable: true, value });
}

function installVisualViewport(height: number, offsetTop = 0): ViewportHandle {
    const listeners = new Map<string, ViewportListener[]>();
    const viewport = {
        height,
        offsetTop,
        addEventListener(type: string, listener: ViewportListener) {
            listeners.set(type, [...(listeners.get(type) ?? []), listener]);
        },
        removeEventListener(type: string, listener: ViewportListener) {
            listeners.set(
                type,
                (listeners.get(type) ?? []).filter(entry => entry !== listener),
            );
        },
    };

    Object.defineProperty(window, "visualViewport", { configurable: true, writable: true, value: viewport });

    return {
        setHeight(next: number) {
            viewport.height = next;
        },
        setOffsetTop(next: number) {
            viewport.offsetTop = next;
        },
        emit(type: string) {
            for (const listener of [...(listeners.get(type) ?? [])]) {
                listener(new Event(type));
            }
        },
        listenerCount(type: string) {
            return (listeners.get(type) ?? []).length;
        },
    };
}

function removeVisualViewport(): void {
    Object.defineProperty(window, "visualViewport", { configurable: true, writable: true, value: undefined });
}

function cssVar(name: string): string {
    return document.documentElement.style.getPropertyValue(name);
}

describe("useChatViewport", () => {
    afterEach(() => {
        restore("visualViewport", viewportDescriptor);
        restore("innerHeight", innerHeightDescriptor);
        document.documentElement.style.removeProperty("--chat-vh");
        document.documentElement.style.removeProperty("--kb-inset");
    });

    it("does nothing when the browser exposes no visual viewport", () => {
        // given
        removeVisualViewport();
        const scrollToBottom = vi.fn();

        // when
        renderHook(() => useChatViewport({ scrollToBottom }));

        // then
        expect(cssVar("--chat-vh")).toBe("");
        expect(cssVar("--kb-inset")).toBe("");
        expect(scrollToBottom).not.toHaveBeenCalled();
    });

    it("publishes the viewport height and the keyboard inset as css variables", () => {
        // given
        setInnerHeight(900);
        installVisualViewport(600);

        // when
        renderHook(() => useChatViewport({ scrollToBottom: vi.fn() }));

        // then
        expect(cssVar("--chat-vh")).toBe("600px");
        expect(cssVar("--kb-inset")).toBe("300px");
    });

    it("subtracts the viewport offset from the keyboard inset", () => {
        // given
        setInnerHeight(900);
        installVisualViewport(600, 100);

        // when
        renderHook(() => useChatViewport({ scrollToBottom: vi.fn() }));

        // then
        expect(cssVar("--kb-inset")).toBe("200px");
    });

    it("clamps the keyboard inset at zero when the viewport is taller than the window", () => {
        // given
        setInnerHeight(800);
        installVisualViewport(900);

        // when
        renderHook(() => useChatViewport({ scrollToBottom: vi.fn() }));

        // then
        expect(cssVar("--kb-inset")).toBe("0px");
    });

    it("refreshes the variables and pins the chat to the bottom when the viewport resizes", () => {
        // given
        setInnerHeight(900);
        const viewport = installVisualViewport(900);
        const scrollToBottom = vi.fn();
        renderHook(() => useChatViewport({ scrollToBottom }));

        // when
        viewport.setHeight(500);
        viewport.emit("resize");

        // then
        expect(cssVar("--chat-vh")).toBe("500px");
        expect(cssVar("--kb-inset")).toBe("400px");
        expect(scrollToBottom).toHaveBeenCalledOnce();
    });

    it("refreshes the variables without scrolling when the viewport is panned", () => {
        // given
        setInnerHeight(900);
        const viewport = installVisualViewport(500);
        const scrollToBottom = vi.fn();
        renderHook(() => useChatViewport({ scrollToBottom }));

        // when
        viewport.setOffsetTop(120);
        viewport.emit("scroll");

        // then
        expect(cssVar("--kb-inset")).toBe("280px");
        expect(scrollToBottom).not.toHaveBeenCalled();
    });

    it("detaches its listeners and clears the variables on unmount", () => {
        // given
        setInnerHeight(900);
        const viewport = installVisualViewport(600);
        const { unmount } = renderHook(() => useChatViewport({ scrollToBottom: vi.fn() }));

        // when
        unmount();

        // then
        expect(viewport.listenerCount("resize")).toBe(0);
        expect(viewport.listenerCount("scroll")).toBe(0);
        expect(cssVar("--chat-vh")).toBe("");
        expect(cssVar("--kb-inset")).toBe("");
    });

    it("resubscribes with the latest scroll callback when it changes", () => {
        // given
        setInnerHeight(900);
        const viewport = installVisualViewport(600);
        const first = vi.fn();
        const second = vi.fn();
        const { rerender } = renderHook(({ scrollToBottom }) => useChatViewport({ scrollToBottom }), {
            initialProps: { scrollToBottom: first },
        });

        // when
        rerender({ scrollToBottom: second });
        viewport.emit("resize");

        // then
        expect(first).not.toHaveBeenCalled();
        expect(second).toHaveBeenCalledOnce();
        expect(viewport.listenerCount("resize")).toBe(1);
    });
});
