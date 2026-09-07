import { act, renderHook } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { useResetOnChange, useSyncedState } from "./useResetOnChange";

describe("useResetOnChange", () => {
    it("leaves the state alone while the key stays the same", () => {
        // given
        const reset = vi.fn();

        // when
        const { rerender } = renderHook(({ key }) => useResetOnChange(key, reset), {
            initialProps: { key: "post-1" },
        });
        rerender({ key: "post-1" });

        // then
        expect(reset).not.toHaveBeenCalled();
    });

    it("resets once when the key changes", () => {
        // given
        const reset = vi.fn();
        const { rerender } = renderHook(({ key }) => useResetOnChange(key, reset), {
            initialProps: { key: "post-1" },
        });

        // when
        rerender({ key: "post-2" });
        rerender({ key: "post-2" });

        // then
        expect(reset).toHaveBeenCalledTimes(1);
    });

    it("renders the reset state rather than a frame of the previous entity", () => {
        // given
        const { result, rerender } = renderHook(
            ({ key }) => {
                const [draft, setDraft] = useState("");
                useResetOnChange(key, () => setDraft(""));

                return { draft, setDraft };
            },
            { initialProps: { key: "post-1" } },
        );
        act(() => {
            result.current.setDraft("Beato did it");
        });

        // when
        rerender({ key: "post-2" });

        // then
        expect(result.current.draft).toBe("");
    });
});

describe("useSyncedState", () => {
    it("keeps local changes while the source stays the same", () => {
        // given
        const { result, rerender } = renderHook(({ source }) => useSyncedState(source), {
            initialProps: { source: 3 },
        });

        // when
        act(() => {
            result.current[1](4);
        });
        rerender({ source: 3 });

        // then
        expect(result.current[0]).toBe(4);
    });

    it("takes the new source when it changes", () => {
        // given
        const { result, rerender } = renderHook(({ source }) => useSyncedState(source), {
            initialProps: { source: 3 },
        });
        act(() => {
            result.current[1](4);
        });

        // when
        rerender({ source: 9 });

        // then
        expect(result.current[0]).toBe(9);
    });
});
