import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useWarnBeforeLeaving } from "./useWarnBeforeLeaving";

function leave(): boolean {
    const event = new Event("beforeunload", { cancelable: true });
    window.dispatchEvent(event);

    return event.defaultPrevented;
}

describe("useWarnBeforeLeaving", () => {
    it("asks before leaving while it is active", () => {
        // given
        renderHook(() => useWarnBeforeLeaving(true));

        // when
        const warned = leave();

        // then
        expect(warned).toBe(true);
    });

    it("lets the page go when it is not active", () => {
        // given
        renderHook(() => useWarnBeforeLeaving(false));

        // when
        const warned = leave();

        // then
        expect(warned).toBe(false);
    });

    it("stops asking once it is switched off", () => {
        // given
        const { rerender } = renderHook(({ active }) => useWarnBeforeLeaving(active), {
            initialProps: { active: true },
        });

        // when
        rerender({ active: false });
        const warned = leave();

        // then
        expect(warned).toBe(false);
    });

    it("stops asking once the component is gone", () => {
        // given
        const { unmount } = renderHook(() => useWarnBeforeLeaving(true));

        // when
        unmount();
        const warned = leave();

        // then
        expect(warned).toBe(false);
    });
});
