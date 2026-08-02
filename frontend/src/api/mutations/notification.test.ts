import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { markAllNotificationsRead, markNotificationRead } from "../endpoints";
import { queryKeys } from "../queryKeys";
import { useMarkAllNotificationsRead, useMarkNotificationRead } from "./notification";

vi.mock("../endpoints", () => ({
    markAllNotificationsRead: vi.fn(),
    markNotificationRead: vi.fn(),
}));

const markAllNotificationsReadMock = vi.mocked(markAllNotificationsRead);
const markNotificationReadMock = vi.mocked(markNotificationRead);

function setup<T>(hook: () => T) {
    const queryClient = createTestQueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(hook, { wrapper: providerWrapper({ queryClient }) });

    return { result, invalidate };
}

beforeEach(() => {
    markAllNotificationsReadMock.mockResolvedValue(undefined);
    markNotificationReadMock.mockResolvedValue(undefined);
});

describe("useMarkNotificationRead", () => {
    it("marks the numeric notification id it is handed as read", async () => {
        // given
        const { result } = setup(() => useMarkNotificationRead());

        // when
        await act(async () => {
            await result.current.mutateAsync(42);
        });

        // then
        expect(markNotificationReadMock).toHaveBeenCalledWith(42);
    });

    it("invalidates the whole notifications root so the list and the unread badge both refresh", async () => {
        // given
        const { result, invalidate } = setup(() => useMarkNotificationRead());

        // when
        await act(async () => {
            await result.current.mutateAsync(42);
        });

        // then
        expect(invalidate).toHaveBeenCalledWith({ queryKey: queryKeys.notifications.all });
        expect(invalidate).toHaveBeenCalledTimes(1);
    });

    it("leaves the notification caches alone when the request is rejected", async () => {
        // given
        markNotificationReadMock.mockRejectedValue(new Error("gone"));
        const { result, invalidate } = setup(() => useMarkNotificationRead());

        // when
        const attempt = act(async () => {
            await result.current.mutateAsync(42);
        });

        // then
        await expect(attempt).rejects.toThrow("gone");
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("useMarkAllNotificationsRead", () => {
    it("calls the bulk endpoint without any arguments", async () => {
        // given
        const { result } = setup(() => useMarkAllNotificationsRead());

        // when
        await act(async () => {
            await result.current.mutateAsync();
        });

        // then
        expect(markAllNotificationsReadMock).toHaveBeenCalledTimes(1);
        expect(markAllNotificationsReadMock).toHaveBeenCalledWith();
    });

    it("invalidates the whole notifications root once everything is read", async () => {
        // given
        const { result, invalidate } = setup(() => useMarkAllNotificationsRead());

        // when
        await act(async () => {
            await result.current.mutateAsync();
        });

        // then
        expect(invalidate).toHaveBeenCalledWith({ queryKey: queryKeys.notifications.all });
        expect(invalidate).toHaveBeenCalledTimes(1);
    });

    it("leaves the notification caches alone when the bulk request is rejected", async () => {
        // given
        markAllNotificationsReadMock.mockRejectedValue(new Error("offline"));
        const { result, invalidate } = setup(() => useMarkAllNotificationsRead());

        // when
        const attempt = act(async () => {
            await result.current.mutateAsync();
        });

        // then
        await expect(attempt).rejects.toThrow("offline");
        expect(invalidate).not.toHaveBeenCalled();
    });
});
