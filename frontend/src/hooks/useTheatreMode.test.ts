import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useTheatreMode } from "./useTheatreMode";

function press(key: string, init: KeyboardEventInit = {}, target: EventTarget = window) {
    act(() => {
        target.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, ...init }));
    });
}

describe("useTheatreMode", () => {
    it("starts outside theatre mode on every visit", () => {
        // when
        const { result } = renderHook(() => useTheatreMode());

        // then
        expect(result.current.theatre).toBe(false);
    });

    it("enters theatre mode with the T key", () => {
        // given
        const { result } = renderHook(() => useTheatreMode());

        // when
        press("t");

        // then
        expect(result.current.theatre).toBe(true);
    });

    it("leaves theatre mode with a second T", () => {
        // given
        const { result } = renderHook(() => useTheatreMode());
        press("t");

        // when
        press("T");

        // then
        expect(result.current.theatre).toBe(false);
    });

    it("leaves theatre mode on Escape", () => {
        // given
        const { result } = renderHook(() => useTheatreMode());
        press("t");

        // when
        press("Escape");

        // then
        expect(result.current.theatre).toBe(false);
    });

    it("ignores the T key while the viewer is typing", () => {
        // given
        const { result } = renderHook(() => useTheatreMode());
        const input = document.createElement("textarea");
        document.body.appendChild(input);

        // when
        press("t", {}, input);

        // then
        expect(result.current.theatre).toBe(false);
        input.remove();
    });

    it("ignores the T key when it is part of a shortcut", () => {
        // given
        const { result } = renderHook(() => useTheatreMode());

        // when
        press("t", { ctrlKey: true });

        // then
        expect(result.current.theatre).toBe(false);
    });
});
