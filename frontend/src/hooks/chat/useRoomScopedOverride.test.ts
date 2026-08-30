import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useRoomScopedOverride } from "./useRoomScopedOverride";

interface Props {
    roomId: string | undefined;
    base: string[];
}

interface NullableProps {
    roomId: string | undefined;
    base: string[] | null;
}

function renderOverride(initialProps: Props) {
    return renderHook(({ roomId, base }: Props) => useRoomScopedOverride<string[]>(roomId, base), { initialProps });
}

function renderNullableOverride(initialProps: NullableProps) {
    return renderHook(({ roomId, base }: NullableProps) => useRoomScopedOverride<string[] | null>(roomId, base), {
        initialProps,
    });
}

describe("useRoomScopedOverride", () => {
    it("hands back the base value while nothing has been overridden", () => {
        // given
        const base = ["a"];

        // when
        const { result } = renderOverride({ roomId: "room-1", base });

        // then
        expect(result.current[0]).toBe(base);
    });

    it("hands back the override once one is set", () => {
        // given
        const { result } = renderOverride({ roomId: "room-1", base: ["a"] });

        // when
        act(() => {
            result.current[1](["b"]);
        });

        // then
        expect(result.current[0]).toEqual(["b"]);
    });

    it("builds a functional update on top of the base value", () => {
        // given
        const { result } = renderOverride({ roomId: "room-1", base: ["a"] });

        // when
        act(() => {
            result.current[1](prev => [...prev, "b"]);
        });

        // then
        expect(result.current[0]).toEqual(["a", "b"]);
    });

    it("discards the override as soon as the room changes", () => {
        // given
        const { result, rerender } = renderOverride({ roomId: "room-1", base: ["a"] });
        act(() => {
            result.current[1](["b"]);
        });

        // when
        rerender({ roomId: "room-2", base: ["c"] });

        // then
        expect(result.current[0]).toEqual(["c"]);
    });

    it("hands a room its own override back when the viewer returns to it", () => {
        // given
        const { result, rerender } = renderOverride({ roomId: "room-1", base: ["a"] });
        act(() => {
            result.current[1](["b"]);
        });
        rerender({ roomId: "room-2", base: ["c"] });

        // when
        rerender({ roomId: "room-1", base: ["a"] });

        // then
        expect(result.current[0]).toEqual(["b"]);
    });

    it("builds a functional update for the new room on top of that room's base value", () => {
        // given
        const { result, rerender } = renderOverride({ roomId: "room-1", base: ["a"] });
        act(() => {
            result.current[1](["b"]);
        });
        rerender({ roomId: "room-2", base: ["c"] });

        // when
        act(() => {
            result.current[1](prev => [...prev, "d"]);
        });

        // then
        expect(result.current[0]).toEqual(["c", "d"]);
    });

    it("follows the base value again once the override is cleared", () => {
        // given
        const { result, rerender } = renderNullableOverride({ roomId: "room-1", base: null });
        act(() => {
            result.current[1](["b"]);
        });

        // when
        act(() => {
            result.current[1](null);
        });
        rerender({ roomId: "room-1", base: ["a"] });

        // then
        expect(result.current[0]).toEqual(["a"]);
    });
});
