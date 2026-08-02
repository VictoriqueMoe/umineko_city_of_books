import { fireEvent, renderHook } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { RefObject } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useClickOutside } from "./useClickOutside";

function mountNodes() {
    const panel = document.createElement("div");
    const child = document.createElement("button");
    panel.appendChild(child);

    const outside = document.createElement("button");
    document.body.append(panel, outside);

    return { panel, child, outside };
}

afterEach(() => {
    document.body.innerHTML = "";
});

describe("useClickOutside", () => {
    it("closes when a mousedown lands outside the referenced element", async () => {
        // given
        const user = userEvent.setup();
        const onClose = vi.fn();
        const { panel, outside } = mountNodes();
        const ref: RefObject<HTMLElement | null> = { current: panel };
        renderHook(() => useClickOutside(ref, onClose));

        // when
        await user.click(outside);

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("stays open when the mousedown starts on the element itself", async () => {
        // given
        const user = userEvent.setup();
        const onClose = vi.fn();
        const { panel } = mountNodes();
        const ref: RefObject<HTMLElement | null> = { current: panel };
        renderHook(() => useClickOutside(ref, onClose));

        // when
        await user.click(panel);

        // then
        expect(onClose).not.toHaveBeenCalled();
    });

    it("stays open when the mousedown starts on a descendant of the element", async () => {
        // given
        const user = userEvent.setup();
        const onClose = vi.fn();
        const { panel, child } = mountNodes();
        const ref: RefObject<HTMLElement | null> = { current: panel };
        renderHook(() => useClickOutside(ref, onClose));

        // when
        await user.click(child);

        // then
        expect(onClose).not.toHaveBeenCalled();
    });

    it("stays quiet while the ref holds no element", async () => {
        // given
        const user = userEvent.setup();
        const onClose = vi.fn();
        const { outside } = mountNodes();
        const ref: RefObject<HTMLElement | null> = { current: null };
        renderHook(() => useClickOutside(ref, onClose));

        // when
        await user.click(outside);

        // then
        expect(onClose).not.toHaveBeenCalled();
    });

    it("reacts to mousedown rather than the completed click", () => {
        // given
        const onClose = vi.fn();
        const { panel, outside } = mountNodes();
        const ref: RefObject<HTMLElement | null> = { current: panel };
        renderHook(() => useClickOutside(ref, onClose));

        // when
        fireEvent.click(outside);
        fireEvent.mouseUp(outside);

        // then
        expect(onClose).not.toHaveBeenCalled();
    });

    it("uses the newest close handler after a rerender", () => {
        // given
        const first = vi.fn();
        const second = vi.fn();
        const { panel, outside } = mountNodes();
        const ref: RefObject<HTMLElement | null> = { current: panel };
        const { rerender } = renderHook(({ onClose }) => useClickOutside(ref, onClose), {
            initialProps: { onClose: first },
        });

        // when
        rerender({ onClose: second });
        fireEvent.mouseDown(outside);

        // then
        expect(first).not.toHaveBeenCalled();
        expect(second).toHaveBeenCalledOnce();
    });

    it("stops listening once the hook unmounts", () => {
        // given
        const onClose = vi.fn();
        const { panel, outside } = mountNodes();
        const ref: RefObject<HTMLElement | null> = { current: panel };
        const removeListener = vi.spyOn(document, "removeEventListener");
        const { unmount } = renderHook(() => useClickOutside(ref, onClose));

        // when
        unmount();
        fireEvent.mouseDown(outside);

        // then
        expect(removeListener).toHaveBeenCalledWith("mousedown", expect.any(Function));
        expect(onClose).not.toHaveBeenCalled();
        removeListener.mockRestore();
    });

    it("fires once per outside mousedown rather than accumulating listeners", () => {
        // given
        const onClose = vi.fn();
        const { panel, outside } = mountNodes();
        const ref: RefObject<HTMLElement | null> = { current: panel };
        const { rerender } = renderHook(() => useClickOutside(ref, onClose));

        // when
        rerender();
        rerender();
        fireEvent.mouseDown(outside);

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });
});
