import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MARK_READ_DEBOUNCE_MS, useDebouncedMarkChatRoomRead } from "./useDebouncedMarkChatRoomRead";

const mocks = vi.hoisted(() => ({
    markRead: vi.fn(),
}));

vi.mock("../mutations/chat", () => ({
    useMarkChatRoomRead: () => ({ mutate: mocks.markRead, mutateAsync: mocks.markRead }),
}));

function setDocumentState(visibility: DocumentVisibilityState, focused: boolean): void {
    Object.defineProperty(document, "visibilityState", { configurable: true, get: () => visibility });
    Object.defineProperty(document, "hasFocus", { configurable: true, value: () => focused });
}

function advance(ms: number): void {
    act(() => {
        vi.advanceTimersByTime(ms);
    });
}

describe("useDebouncedMarkChatRoomRead", () => {
    beforeEach(() => {
        vi.useFakeTimers();
        setDocumentState("visible", true);
    });

    it("makes exactly one read call when ten messages arrive in one second", () => {
        // given
        const { result } = renderHook(() => useDebouncedMarkChatRoomRead());

        // when
        for (let i = 0; i < 10; i++) {
            act(() => {
                result.current("room-1");
            });
            advance(100);
        }
        advance(MARK_READ_DEBOUNCE_MS);

        // then
        expect(mocks.markRead).toHaveBeenCalledExactlyOnceWith("room-1");
    });

    it("holds the call back until the burst has settled", () => {
        // given
        const { result } = renderHook(() => useDebouncedMarkChatRoomRead());

        // when
        act(() => {
            result.current("room-1");
        });
        advance(MARK_READ_DEBOUNCE_MS - 1);

        // then
        expect(mocks.markRead).not.toHaveBeenCalled();

        // when
        advance(1);

        // then
        expect(mocks.markRead).toHaveBeenCalledExactlyOnceWith("room-1");
    });

    it("marks the room the last message belonged to", () => {
        // given
        const { result } = renderHook(() => useDebouncedMarkChatRoomRead());

        // when
        act(() => {
            result.current("room-1");
            result.current("room-2");
        });
        advance(MARK_READ_DEBOUNCE_MS);

        // then
        expect(mocks.markRead).toHaveBeenCalledExactlyOnceWith("room-2");
    });

    it("marks a second burst read once the first has gone through", () => {
        // given
        const { result } = renderHook(() => useDebouncedMarkChatRoomRead());

        // when
        act(() => {
            result.current("room-1");
        });
        advance(MARK_READ_DEBOUNCE_MS);
        act(() => {
            result.current("room-1");
        });
        advance(MARK_READ_DEBOUNCE_MS);

        // then
        expect(mocks.markRead).toHaveBeenCalledTimes(2);
    });

    it("stays quiet while the window has lost focus", () => {
        // given
        setDocumentState("visible", false);
        const { result } = renderHook(() => useDebouncedMarkChatRoomRead());

        // when
        act(() => {
            result.current("room-1");
        });
        advance(MARK_READ_DEBOUNCE_MS);

        // then
        expect(mocks.markRead).not.toHaveBeenCalled();
    });

    it("stays quiet while the tab is hidden", () => {
        // given
        setDocumentState("hidden", true);
        const { result } = renderHook(() => useDebouncedMarkChatRoomRead());

        // when
        act(() => {
            result.current("room-1");
        });
        advance(MARK_READ_DEBOUNCE_MS);

        // then
        expect(mocks.markRead).not.toHaveBeenCalled();
    });

    it("drops a pending call when the view goes away", () => {
        // given
        const { result, unmount } = renderHook(() => useDebouncedMarkChatRoomRead());
        act(() => {
            result.current("room-1");
        });

        // when
        unmount();
        advance(MARK_READ_DEBOUNCE_MS);

        // then
        expect(mocks.markRead).not.toHaveBeenCalled();
    });

    it("honours a caller that wants its own interval", () => {
        // given
        const { result } = renderHook(() => useDebouncedMarkChatRoomRead(50));

        // when
        act(() => {
            result.current("room-1");
        });
        advance(50);

        // then
        expect(mocks.markRead).toHaveBeenCalledExactlyOnceWith("room-1");
    });
});
