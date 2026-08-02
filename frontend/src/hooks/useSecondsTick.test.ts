import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useSecondsTick } from "./useSecondsTick";

const start = new Date("2026-08-02T12:00:00.000Z");

function advance(ms: number) {
    act(() => {
        vi.advanceTimersByTime(ms);
    });
}

describe("useSecondsTick", () => {
    beforeEach(() => {
        vi.useFakeTimers();
        vi.setSystemTime(start);
    });

    it("returns the current time on the first render", () => {
        // given
        const active = true;

        // when
        const { result } = renderHook(() => useSecondsTick(active));

        // then
        expect(result.current).toBe(start.getTime());
    });

    it("moves forward once every second while it is active", () => {
        // given
        const { result } = renderHook(() => useSecondsTick(true));

        // when
        advance(1000);

        // then
        expect(result.current).toBe(start.getTime() + 1000);
        advance(1000);
        expect(result.current).toBe(start.getTime() + 2000);
    });

    it("does not move part way through a second", () => {
        // given
        const { result } = renderHook(() => useSecondsTick(true));

        // when
        advance(999);

        // then
        expect(result.current).toBe(start.getTime());
    });

    it("stays frozen while it is inactive", () => {
        // given
        const { result } = renderHook(() => useSecondsTick(false));

        // when
        advance(5000);

        // then
        expect(result.current).toBe(start.getTime());
        expect(vi.getTimerCount()).toBe(0);
    });

    it("starts ticking when it becomes active", () => {
        // given
        const { result, rerender } = renderHook(({ active }) => useSecondsTick(active), {
            initialProps: { active: false },
        });
        advance(5000);

        // when
        rerender({ active: true });
        advance(1000);

        // then
        expect(result.current).toBe(start.getTime() + 6000);
    });

    it("stops ticking when it goes inactive", () => {
        // given
        const { result, rerender } = renderHook(({ active }) => useSecondsTick(active), {
            initialProps: { active: true },
        });
        advance(1000);

        // when
        rerender({ active: false });
        advance(5000);

        // then
        expect(result.current).toBe(start.getTime() + 1000);
        expect(vi.getTimerCount()).toBe(0);
    });

    it("clears its interval when it unmounts", () => {
        // given
        const { unmount } = renderHook(() => useSecondsTick(true));
        expect(vi.getTimerCount()).toBe(1);

        // when
        unmount();

        // then
        expect(vi.getTimerCount()).toBe(0);
    });
});
