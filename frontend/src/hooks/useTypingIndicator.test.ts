import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useTypingIndicator } from "./useTypingIndicator";

describe("useTypingIndicator", () => {
    it("starts with nobody typing", () => {
        // given
        const scope = "room-1";

        // when
        const { result } = renderHook(() => useTypingIndicator(scope));

        // then
        expect(result.current.typingUserIds).toEqual([]);
    });

    it("records a user as typing", () => {
        // given
        const { result } = renderHook(() => useTypingIndicator("room-1"));

        // when
        act(() => {
            result.current.noteTyping("u1");
        });

        // then
        expect(result.current.typingUserIds).toEqual(["u1"]);
    });

    it("keeps the order users started typing in and never lists one twice", () => {
        // given
        const { result } = renderHook(() => useTypingIndicator("room-1"));

        // when
        act(() => {
            result.current.noteTyping("u1");
            result.current.noteTyping("u2");
            result.current.noteTyping("u1");
        });

        // then
        expect(result.current.typingUserIds).toEqual(["u1", "u2"]);
    });

    it("removes a user once they are cleared", () => {
        // given
        const { result } = renderHook(() => useTypingIndicator("room-1"));
        act(() => {
            result.current.noteTyping("u1");
            result.current.noteTyping("u2");
        });

        // when
        act(() => {
            result.current.clearUser("u1");
        });

        // then
        expect(result.current.typingUserIds).toEqual(["u2"]);
    });

    it("leaves the list untouched when clearing someone who was not typing", () => {
        // given
        const { result } = renderHook(() => useTypingIndicator("room-1"));
        act(() => {
            result.current.noteTyping("u1");
        });
        const before = result.current.typingUserIds;

        // when
        act(() => {
            result.current.clearUser("ghost");
        });

        // then
        expect(result.current.typingUserIds).toBe(before);
    });

    it("forgets everyone when reset", () => {
        // given
        const { result } = renderHook(() => useTypingIndicator("room-1"));
        act(() => {
            result.current.noteTyping("u1");
            result.current.noteTyping("u2");
        });

        // when
        act(() => {
            result.current.reset();
        });

        // then
        expect(result.current.typingUserIds).toEqual([]);
    });

    it("expires a user five seconds after their last keystroke", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        const { result } = renderHook(() => useTypingIndicator("room-1"));
        act(() => {
            result.current.noteTyping("u1");
        });

        // when
        act(() => {
            vi.advanceTimersByTime(4000);
        });
        const stillTyping = result.current.typingUserIds;
        act(() => {
            vi.advanceTimersByTime(2000);
        });

        // then
        expect(stillTyping).toEqual(["u1"]);
        expect(result.current.typingUserIds).toEqual([]);
    });

    it("keeps a user typing while their keystrokes keep arriving", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        const { result } = renderHook(() => useTypingIndicator("room-1"));
        act(() => {
            result.current.noteTyping("u1");
        });

        // when
        act(() => {
            vi.advanceTimersByTime(4000);
        });
        act(() => {
            result.current.noteTyping("u1");
        });
        act(() => {
            vi.advanceTimersByTime(3000);
        });

        // then
        expect(result.current.typingUserIds).toEqual(["u1"]);
    });

    it("expires only the users whose window has passed", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        const { result } = renderHook(() => useTypingIndicator("room-1"));
        act(() => {
            result.current.noteTyping("u1");
        });

        // when
        act(() => {
            vi.advanceTimersByTime(3000);
        });
        act(() => {
            result.current.noteTyping("u2");
        });
        act(() => {
            vi.advanceTimersByTime(3000);
        });

        // then
        expect(result.current.typingUserIds).toEqual(["u2"]);
    });

    it("reports nobody typing after the scope changes", () => {
        // given
        const { result, rerender } = renderHook(({ scope }) => useTypingIndicator(scope), {
            initialProps: { scope: "room-1" },
        });
        act(() => {
            result.current.noteTyping("u1");
        });

        // when
        rerender({ scope: "room-2" });

        // then
        expect(result.current.typingUserIds).toEqual([]);
    });

    it("does not carry typists across a scope change", () => {
        // given
        const { result, rerender } = renderHook(({ scope }) => useTypingIndicator(scope), {
            initialProps: { scope: "room-1" },
        });
        act(() => {
            result.current.noteTyping("u1");
        });
        rerender({ scope: "room-2" });

        // when
        act(() => {
            result.current.noteTyping("u2");
        });

        // then
        expect(result.current.typingUserIds).toEqual(["u2"]);
    });

    it("stops its expiry timer when unmounted", () => {
        // given
        vi.useFakeTimers();
        const clearIntervalSpy = vi.spyOn(globalThis, "clearInterval");
        const { unmount } = renderHook(() => useTypingIndicator("room-1"));

        // when
        unmount();

        // then
        expect(clearIntervalSpy).toHaveBeenCalled();
        clearIntervalSpy.mockRestore();
    });
});
