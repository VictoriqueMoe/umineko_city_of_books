import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { STREAM_CHAT_POPOUT_CLOSED } from "../platform/streamChatPopout";
import { useStreamChatPopout, useStreamChatPopoutReporter } from "./useStreamPopout";

function makePopoutWindow() {
    return { focus: vi.fn(), close: vi.fn() };
}

function stubWindowOpen(opened: { focus: () => void; close: () => void } | null) {
    const open = vi.fn(() => opened as unknown as Window | null);
    vi.stubGlobal("open", open);

    return open;
}

function sendClosedMessage(streamId: string, origin = window.location.origin): void {
    act(() => {
        window.dispatchEvent(
            new MessageEvent("message", {
                data: { type: STREAM_CHAT_POPOUT_CLOSED, streamId },
                origin,
            }),
        );
    });
}

beforeEach(() => {
    stubWindowOpen(makePopoutWindow());
});

afterEach(() => {
    vi.unstubAllGlobals();
});

describe("useStreamChatPopout", () => {
    it("keeps the chat where it is until it is asked to pop out", () => {
        // given
        const view = renderHook(() => useStreamChatPopout("beatrice", "stream-1"));

        // when
        const { poppedOut } = view.result.current;

        // then
        expect(poppedOut).toBe(false);
    });

    it("opens the chat in a window of its own", () => {
        // given
        const open = stubWindowOpen(makePopoutWindow());
        const view = renderHook(() => useStreamChatPopout("beatrice", "stream-7"));

        // when
        act(() => {
            view.result.current.popOut();
        });

        // then
        expect(open).toHaveBeenCalledWith(
            "/beatrice/live/chat",
            "stream-chat-stream-7",
            expect.stringContaining("popup=yes"),
        );
        expect(view.result.current.poppedOut).toBe(true);
    });

    it("brings the popped out window to the front", () => {
        // given
        const popout = makePopoutWindow();
        stubWindowOpen(popout);
        const view = renderHook(() => useStreamChatPopout("beatrice", "stream-1"));

        // when
        act(() => {
            view.result.current.popOut();
        });

        // then
        expect(popout.focus).toHaveBeenCalled();
    });

    it("leaves the chat where it is when the browser blocks the popup", () => {
        // given
        stubWindowOpen(null);
        const view = renderHook(() => useStreamChatPopout("beatrice", "stream-1"));

        // when
        act(() => {
            view.result.current.popOut();
        });

        // then
        expect(view.result.current.poppedOut).toBe(false);
    });

    it("opens nothing without a stream to chat about", () => {
        // given
        const open = stubWindowOpen(makePopoutWindow());
        const view = renderHook(() => useStreamChatPopout("beatrice", undefined));

        // when
        act(() => {
            view.result.current.popOut();
        });

        // then
        expect(open).not.toHaveBeenCalled();
    });

    it("closes the popped out window and takes the chat back", () => {
        // given
        const popout = makePopoutWindow();
        stubWindowOpen(popout);
        const view = renderHook(() => useStreamChatPopout("beatrice", "stream-1"));
        act(() => {
            view.result.current.popOut();
        });

        // when
        act(() => {
            view.result.current.bringBack();
        });

        // then
        expect(popout.close).toHaveBeenCalled();
        expect(view.result.current.poppedOut).toBe(false);
    });

    it("takes the chat back when the popped out window says it has closed", () => {
        // given
        const view = renderHook(() => useStreamChatPopout("beatrice", "stream-1"));
        act(() => {
            view.result.current.popOut();
        });

        // when
        sendClosedMessage("stream-1");

        // then
        expect(view.result.current.poppedOut).toBe(false);
    });

    it("ignores a close message about a different stream", () => {
        // given
        const view = renderHook(() => useStreamChatPopout("beatrice", "stream-1"));
        act(() => {
            view.result.current.popOut();
        });

        // when
        sendClosedMessage("stream-2");

        // then
        expect(view.result.current.poppedOut).toBe(true);
    });

    it("ignores a close message sent from another site", () => {
        // given
        const view = renderHook(() => useStreamChatPopout("beatrice", "stream-1"));
        act(() => {
            view.result.current.popOut();
        });

        // when
        sendClosedMessage("stream-1", "https://evil.example");

        // then
        expect(view.result.current.poppedOut).toBe(true);
    });

    it("ignores a message that is not the popout handshake", () => {
        // given
        const view = renderHook(() => useStreamChatPopout("beatrice", "stream-1"));
        act(() => {
            view.result.current.popOut();
        });

        // when
        act(() => {
            window.dispatchEvent(
                new MessageEvent("message", {
                    data: { type: "something-else", streamId: "stream-1" },
                    origin: window.location.origin,
                }),
            );
        });

        // then
        expect(view.result.current.poppedOut).toBe(true);
    });

    it("stops listening for the handshake once the page is gone", () => {
        // given
        const removeEventListener = vi.spyOn(window, "removeEventListener");
        const view = renderHook(() => useStreamChatPopout("beatrice", "stream-1"));
        act(() => {
            view.result.current.popOut();
        });

        // when
        view.unmount();

        // then
        expect(removeEventListener).toHaveBeenCalledWith("message", expect.any(Function));
        removeEventListener.mockRestore();
    });

    it("listens for the handshake only while the chat is popped out", () => {
        // given
        const addEventListener = vi.spyOn(window, "addEventListener");

        // when
        renderHook(() => useStreamChatPopout("beatrice", "stream-1"));

        // then
        expect(addEventListener).not.toHaveBeenCalledWith("message", expect.any(Function));
        addEventListener.mockRestore();
    });
});

describe("useStreamChatPopoutReporter", () => {
    it("tells the window it came from when it is closed", () => {
        // given
        const postMessage = vi.fn();
        vi.stubGlobal("opener", { closed: false, postMessage });
        renderHook(() => useStreamChatPopoutReporter("stream-1"));

        // when
        window.dispatchEvent(new Event("pagehide"));

        // then
        expect(postMessage).toHaveBeenCalledWith(
            { type: STREAM_CHAT_POPOUT_CLOSED, streamId: "stream-1" },
            window.location.origin,
        );
    });

    it("stays quiet when the window it came from has already gone", () => {
        // given
        const postMessage = vi.fn();
        vi.stubGlobal("opener", { closed: true, postMessage });
        renderHook(() => useStreamChatPopoutReporter("stream-1"));

        // when
        window.dispatchEvent(new Event("pagehide"));

        // then
        expect(postMessage).not.toHaveBeenCalled();
    });

    it("reports nothing when the window was opened directly rather than popped out", () => {
        // given
        vi.stubGlobal("opener", null);
        const addEventListener = vi.spyOn(window, "addEventListener");

        // when
        renderHook(() => useStreamChatPopoutReporter("stream-1"));

        // then
        expect(addEventListener).not.toHaveBeenCalledWith("pagehide", expect.any(Function));
        addEventListener.mockRestore();
    });

    it("stops reporting once it is unmounted", () => {
        // given
        const postMessage = vi.fn();
        vi.stubGlobal("opener", { closed: false, postMessage });
        const view = renderHook(() => useStreamChatPopoutReporter("stream-1"));

        // when
        view.unmount();
        window.dispatchEvent(new Event("pagehide"));

        // then
        expect(postMessage).not.toHaveBeenCalled();
    });
});
