import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { WSMessageHandler } from "../context/notificationContextValue";
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
        expect(result.current.wsEpoch).toBe(0);
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

    it("registers websocket listeners and hands back the unsubscribe function", () => {
        // given
        const unsubscribe = vi.fn();
        const addWSListener = vi.fn((_handler: WSMessageHandler) => unsubscribe);
        const wrapper = providerWrapper({ notification: { addWSListener } });
        const handler: WSMessageHandler = () => {};

        // when
        const { result } = renderHook(() => useNotifications(), { wrapper });
        const stop = result.current.addWSListener(handler);
        stop();

        // then
        expect(addWSListener).toHaveBeenCalledWith(handler);
        expect(unsubscribe).toHaveBeenCalledOnce();
    });

    it("forwards outgoing websocket messages untouched", () => {
        // given
        const sendWSMessage = vi.fn();
        const wrapper = providerWrapper({ notification: { sendWSMessage } });

        // when
        const { result } = renderHook(() => useNotifications(), { wrapper });
        result.current.sendWSMessage({ type: "subscribe", room: "parlour" });

        // then
        expect(sendWSMessage).toHaveBeenCalledWith({ type: "subscribe", room: "parlour" });
    });

    it("exposes the socket epoch so consumers can resubscribe after a reconnect", () => {
        // given
        const wrapper = providerWrapper({ notification: { wsEpoch: 4 } });

        // when
        const { result } = renderHook(() => useNotifications(), { wrapper });

        // then
        expect(result.current.wsEpoch).toBe(4);
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
