import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type { Notification, NotificationListResponse } from "../../types/api";
import * as endpoints from "../endpoints";
import { queryKeys } from "../queryKeys";
import { useNotifications, useUnreadCount } from "./notification";

vi.mock("../endpoints", () => ({
    getNotifications: vi.fn(),
    getUnreadCount: vi.fn(),
}));

const getNotifications = vi.mocked(endpoints.getNotifications);
const getUnreadCount = vi.mocked(endpoints.getUnreadCount);

function makeNotification(id: number): Notification {
    return { id, type: "mention", read: false } as unknown as Notification;
}

function makeListResponse(items: Notification[], total: number): NotificationListResponse {
    return { notifications: items, total, limit: 20, offset: 0 };
}

describe("useNotifications", () => {
    it("asks for the default page of twenty and caches it under the list key", async () => {
        // given
        getNotifications.mockResolvedValue(makeListResponse([makeNotification(1)], 1));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useNotifications(), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.notifications).toHaveLength(1));
        expect(getNotifications).toHaveBeenCalledWith({ limit: 20, offset: 0 });
        expect(queryClient.getQueryData(queryKeys.notifications.list({ limit: 20, offset: 0 }))).toBeDefined();
    });

    it("forwards an explicit page window", async () => {
        // given
        getNotifications.mockResolvedValue(makeListResponse([], 84));
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useNotifications(5, 15), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => expect(result.current.total).toBe(84));
        expect(getNotifications).toHaveBeenCalledWith({ limit: 5, offset: 15 });
        expect(queryClient.getQueryData(queryKeys.notifications.list({ limit: 5, offset: 15 }))).toBeDefined();
    });

    it("starts with an empty page while the request is in flight", () => {
        // given
        getNotifications.mockReturnValue(new Promise(() => {}));

        // when
        const { result } = renderHook(() => useNotifications(), { wrapper: providerWrapper() });

        // then
        expect(result.current.notifications).toEqual([]);
        expect(result.current.total).toBe(0);
        expect(result.current.loading).toBe(true);
    });

    it("refetches the page when refresh is called", async () => {
        // given
        getNotifications.mockResolvedValue(makeListResponse([], 0));
        const { result } = renderHook(() => useNotifications(), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await result.current.refresh();

        // then
        expect(getNotifications).toHaveBeenCalledTimes(2);
    });
});

describe("useUnreadCount", () => {
    it("reports the unread count for a signed in member", async () => {
        // given
        getUnreadCount.mockResolvedValue({ count: 6 });
        const queryClient = createTestQueryClient();

        // when
        const { result } = renderHook(() => useUnreadCount(), {
            wrapper: providerWrapper({ queryClient, user: makeUser() }),
        });

        // then
        await waitFor(() => expect(result.current.count).toBe(6));
        expect(queryClient.getQueryData(queryKeys.notifications.unreadCount())).toEqual({ count: 6 });
    });

    it("does not ask for a count when nobody is signed in", () => {
        // given
        getUnreadCount.mockResolvedValue({ count: 6 });

        // when
        const { result } = renderHook(() => useUnreadCount(), { wrapper: providerWrapper({ user: null }) });

        // then
        expect(getUnreadCount).not.toHaveBeenCalled();
        expect(result.current.count).toBe(0);
    });

    it("refetches the count when refresh is called", async () => {
        // given
        getUnreadCount.mockResolvedValue({ count: 1 });
        const { result } = renderHook(() => useUnreadCount(), { wrapper: providerWrapper({ user: makeUser() }) });
        await waitFor(() => expect(result.current.count).toBe(1));

        // when
        await result.current.refresh();

        // then
        expect(getUnreadCount).toHaveBeenCalledTimes(2);
    });
});
