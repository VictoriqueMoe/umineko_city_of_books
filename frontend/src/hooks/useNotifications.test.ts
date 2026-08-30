import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { providerWrapper } from "../test-utils/render";
import { useNotifications } from "./useNotifications";

describe("useNotifications", () => {
    it("returns the unread counters the provider holds", () => {
        // given
        const wrapper = providerWrapper({
            notification: { unreadCount: 3, chatUnreadCount: 12, liveGamesCount: 1, liveStreamsCount: 2 },
        });

        // when
        const { result } = renderHook(() => useNotifications(), { wrapper });

        // then
        expect(result.current.unreadCount).toBe(3);
        expect(result.current.chatUnreadCount).toBe(12);
        expect(result.current.liveGamesCount).toBe(1);
        expect(result.current.liveStreamsCount).toBe(2);
    });

    it("starts every counter at zero when nothing is pending", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useNotifications(), { wrapper });

        // then
        expect(result.current.unreadCount).toBe(0);
        expect(result.current.chatUnreadCount).toBe(0);
        expect(result.current.liveGamesCount).toBe(0);
        expect(result.current.liveStreamsCount).toBe(0);
    });

    it("forwards mark read calls to the provider with the notification id", async () => {
        // given
        const markRead = vi.fn(() => Promise.resolve());
        const markAllRead = vi.fn(() => Promise.resolve());
        const wrapper = providerWrapper({ notification: { markRead, markAllRead } });

        // when
        const { result } = renderHook(() => useNotifications(), { wrapper });
        await result.current.markRead(42);
        await result.current.markAllRead();

        // then
        expect(markRead).toHaveBeenCalledWith(42);
        expect(markAllRead).toHaveBeenCalledOnce();
    });

    it("throws when it is used outside a NotificationProvider", () => {
        // given
        const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

        // when
        const attempt = () => renderHook(() => useNotifications());

        // then
        expect(attempt).toThrow("useNotifications must be used within a NotificationProvider");
        consoleError.mockRestore();
    });
});
