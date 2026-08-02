import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useScrollToHash } from "./useScrollToHash";

function addTarget(id: string) {
    const el = document.createElement("div");
    el.id = id;
    document.body.appendChild(el);

    return el;
}

function advance(ms: number) {
    act(() => {
        vi.advanceTimersByTime(ms);
    });
}

describe("useScrollToHash", () => {
    beforeEach(() => {
        vi.useFakeTimers();
    });

    afterEach(() => {
        document.body.innerHTML = "";
    });

    it("smoothly centres the target once the settling delay has passed", () => {
        // given
        const target = addTarget("comment-1");
        const scrollIntoView = vi.spyOn(target, "scrollIntoView");
        renderHook(() => useScrollToHash(true, "comment-1"));

        // when
        advance(300);

        // then
        expect(scrollIntoView).toHaveBeenCalledWith({ behavior: "smooth", block: "center" });
    });

    it("waits the full delay before scrolling", () => {
        // given
        const target = addTarget("comment-1");
        const scrollIntoView = vi.spyOn(target, "scrollIntoView");
        renderHook(() => useScrollToHash(true, "comment-1"));

        // when
        advance(299);

        // then
        expect(scrollIntoView).not.toHaveBeenCalled();
        advance(1);
        expect(scrollIntoView).toHaveBeenCalledTimes(1);
    });

    it("does nothing while the page is not ready", () => {
        // given
        const target = addTarget("comment-1");
        const scrollIntoView = vi.spyOn(target, "scrollIntoView");
        renderHook(() => useScrollToHash(false, "comment-1"));

        // when
        advance(1000);

        // then
        expect(scrollIntoView).not.toHaveBeenCalled();
        expect(vi.getTimerCount()).toBe(0);
    });

    it("does nothing when there is no element id", () => {
        // given
        const target = addTarget("comment-1");
        const scrollIntoView = vi.spyOn(target, "scrollIntoView");
        renderHook(() => useScrollToHash(true, null));

        // when
        advance(1000);

        // then
        expect(scrollIntoView).not.toHaveBeenCalled();
        expect(vi.getTimerCount()).toBe(0);
    });

    it("ignores an empty element id", () => {
        // given
        const target = addTarget("comment-1");
        const scrollIntoView = vi.spyOn(target, "scrollIntoView");
        renderHook(() => useScrollToHash(true, ""));

        // when
        advance(1000);

        // then
        expect(scrollIntoView).not.toHaveBeenCalled();
    });

    it("survives a target that is not on the page", () => {
        // given
        renderHook(() => useScrollToHash(true, "missing"));

        // when
        const run = () => advance(300);

        // then
        expect(run).not.toThrow();
    });

    it("scrolls once the target appears on a later render", () => {
        // given
        const { rerender } = renderHook(({ ready }) => useScrollToHash(ready, "comment-1"), {
            initialProps: { ready: false },
        });
        const target = addTarget("comment-1");
        const scrollIntoView = vi.spyOn(target, "scrollIntoView");

        // when
        rerender({ ready: true });
        advance(300);

        // then
        expect(scrollIntoView).toHaveBeenCalledTimes(1);
    });

    it("abandons the pending scroll when the element id changes first", () => {
        // given
        const first = addTarget("comment-1");
        const second = addTarget("comment-2");
        const firstScroll = vi.spyOn(first, "scrollIntoView");
        const secondScroll = vi.spyOn(second, "scrollIntoView");
        const { rerender } = renderHook(({ id }: { id: string | null }) => useScrollToHash(true, id), {
            initialProps: { id: "comment-1" },
        });

        // when
        advance(200);
        rerender({ id: "comment-2" });
        advance(300);

        // then
        expect(firstScroll).not.toHaveBeenCalled();
        expect(secondScroll).toHaveBeenCalledTimes(1);
    });

    it("abandons the pending scroll when the hook unmounts first", () => {
        // given
        const target = addTarget("comment-1");
        const scrollIntoView = vi.spyOn(target, "scrollIntoView");
        const { unmount } = renderHook(() => useScrollToHash(true, "comment-1"));

        // when
        unmount();
        advance(300);

        // then
        expect(scrollIntoView).not.toHaveBeenCalled();
        expect(vi.getTimerCount()).toBe(0);
    });
});
