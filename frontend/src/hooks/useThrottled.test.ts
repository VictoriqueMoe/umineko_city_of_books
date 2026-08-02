import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useThrottled } from "./useThrottled";

function advance(ms: number) {
    act(() => {
        vi.advanceTimersByTime(ms);
    });
}

describe("useThrottled", () => {
    beforeEach(() => {
        vi.useFakeTimers();
    });

    it("runs the callback straight away on the first call", () => {
        // given
        const fn = vi.fn<(value: string) => void>();
        const { result } = renderHook(() => useThrottled(fn, 100));

        // when
        result.current("first");

        // then
        expect(fn).toHaveBeenCalledTimes(1);
        expect(fn).toHaveBeenCalledWith("first");
    });

    it("swallows calls made inside the throttle window", () => {
        // given
        const fn = vi.fn<(value: string) => void>();
        const { result } = renderHook(() => useThrottled(fn, 100));

        // when
        result.current("a");
        result.current("b");
        result.current("c");

        // then
        expect(fn).toHaveBeenCalledTimes(1);
        expect(fn).toHaveBeenCalledWith("a");
    });

    it("replays only the most recent swallowed call when the window closes", () => {
        // given
        const fn = vi.fn<(value: string) => void>();
        const { result } = renderHook(() => useThrottled(fn, 100));
        result.current("a");
        result.current("b");
        result.current("c");

        // when
        advance(100);

        // then
        expect(fn).toHaveBeenCalledTimes(2);
        expect(fn).toHaveBeenLastCalledWith("c");
    });

    it("holds a swallowed call for the whole delay", () => {
        // given
        const fn = vi.fn<(value: string) => void>();
        const { result } = renderHook(() => useThrottled(fn, 100));
        result.current("a");
        result.current("b");

        // when
        advance(99);

        // then
        expect(fn).toHaveBeenCalledTimes(1);
        advance(1);
        expect(fn).toHaveBeenCalledTimes(2);
    });

    it("keeps the window open for another delay after a replay", () => {
        // given
        const fn = vi.fn<(value: string) => void>();
        const { result } = renderHook(() => useThrottled(fn, 100));
        result.current("a");
        result.current("b");
        advance(100);

        // when
        result.current("c");

        // then
        expect(fn).toHaveBeenCalledTimes(2);
        advance(100);
        expect(fn).toHaveBeenCalledTimes(3);
        expect(fn).toHaveBeenLastCalledWith("c");
    });

    it("closes the window when nothing was swallowed, so the next call is immediate", () => {
        // given
        const fn = vi.fn<(value: string) => void>();
        const { result } = renderHook(() => useThrottled(fn, 100));
        result.current("a");

        // when
        advance(100);

        // then
        expect(fn).toHaveBeenCalledTimes(1);
        expect(vi.getTimerCount()).toBe(0);
        result.current("b");
        expect(fn).toHaveBeenCalledTimes(2);
        expect(fn).toHaveBeenLastCalledWith("b");
    });

    it("passes every argument through to the callback", () => {
        // given
        const fn = vi.fn<(id: string, count: number) => void>();
        const { result } = renderHook(() => useThrottled(fn, 100));

        // when
        result.current("room", 3);

        // then
        expect(fn).toHaveBeenCalledWith("room", 3);
    });

    it("always calls the newest callback it was handed", () => {
        // given
        const first = vi.fn<(value: string) => void>();
        const second = vi.fn<(value: string) => void>();
        const { result, rerender } = renderHook(({ fn }) => useThrottled(fn, 100), {
            initialProps: { fn: first },
        });

        // when
        rerender({ fn: second });
        result.current("hello");

        // then
        expect(first).not.toHaveBeenCalled();
        expect(second).toHaveBeenCalledWith("hello");
    });

    it("keeps a stable throttled function while the delay is unchanged", () => {
        // given
        const fn = vi.fn<(value: string) => void>();
        const { result, rerender } = renderHook(({ delay }) => useThrottled(fn, delay), {
            initialProps: { delay: 100 },
        });
        const original = result.current;

        // when
        rerender({ delay: 100 });

        // then
        expect(result.current).toBe(original);
        rerender({ delay: 200 });
        expect(result.current).not.toBe(original);
    });

    it("drops a swallowed call when the component unmounts", () => {
        // given
        const fn = vi.fn<(value: string) => void>();
        const { result, unmount } = renderHook(() => useThrottled(fn, 100));
        result.current("a");
        result.current("b");

        // when
        unmount();
        advance(500);

        // then
        expect(fn).toHaveBeenCalledTimes(1);
        expect(vi.getTimerCount()).toBe(0);
    });
});
