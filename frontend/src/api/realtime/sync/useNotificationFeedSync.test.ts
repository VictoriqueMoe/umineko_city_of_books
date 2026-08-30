import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { expectInvalidated } from "../../../test-utils/query";
import { makeUser } from "../../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../../test-utils/render";
import type { Notification, UserProfile } from "../../../types/api";
import { queryKeys } from "../../queryKeys";
import { dispatch } from "../bus";
import { useNotificationFeedSync, type NotificationFeedSyncDeps } from "./useNotificationFeedSync";

function makeNotification(id: number): Notification {
    return { id, type: "chat_mention" } as Notification;
}

function emitNotification(id: number): void {
    act(() => {
        dispatch({ type: "notification", data: makeNotification(id) });
    });
}

function makeDeps(viewer: UserProfile | null): NotificationFeedSyncDeps {
    return { viewer, showNotification: vi.fn(), playSound: vi.fn() };
}

describe("useNotificationFeedSync", () => {
    it("counts an incoming notification and refreshes the notifications list", () => {
        // given
        const queryClient = createTestQueryClient();
        vi.spyOn(queryClient, "invalidateQueries");
        queryClient.setQueryData(queryKeys.notifications.unreadCount(), { count: 2 });
        renderHook(() => useNotificationFeedSync(makeDeps(makeUser())), {
            wrapper: providerWrapper({ queryClient }),
        });

        // when
        emitNotification(4);

        // then
        expect(queryClient.getQueryData(queryKeys.notifications.unreadCount())).toEqual({ count: 3 });
        expectInvalidated(queryClient, queryKeys.notifications.listAll());
    });

    it("treats a missing unread count as zero", () => {
        // given
        const queryClient = createTestQueryClient();
        renderHook(() => useNotificationFeedSync(makeDeps(makeUser())), {
            wrapper: providerWrapper({ queryClient }),
        });

        // when
        emitNotification(4);

        // then
        expect(queryClient.getQueryData(queryKeys.notifications.unreadCount())).toEqual({ count: 1 });
    });

    it("hands the notification to the injected desktop notifier", () => {
        // given
        const queryClient = createTestQueryClient();
        const deps = makeDeps(makeUser());
        renderHook(() => useNotificationFeedSync(deps), { wrapper: providerWrapper({ queryClient }) });

        // when
        emitNotification(4);

        // then
        expect(deps.showNotification).toHaveBeenCalledWith(makeNotification(4));
    });

    it("plays the sound when the viewer has recorded no preference", () => {
        // given
        const queryClient = createTestQueryClient();
        const deps = makeDeps(makeUser());
        renderHook(() => useNotificationFeedSync(deps), { wrapper: providerWrapper({ queryClient }) });

        // when
        emitNotification(4);

        // then
        expect(deps.playSound).toHaveBeenCalledOnce();
    });

    it("plays the sound when there is no signed in viewer at all", () => {
        // given
        const queryClient = createTestQueryClient();
        const deps = makeDeps(null);
        renderHook(() => useNotificationFeedSync(deps), { wrapper: providerWrapper({ queryClient }) });

        // when
        emitNotification(4);

        // then
        expect(deps.playSound).toHaveBeenCalledOnce();
    });

    it("stays silent when the user has turned the notification sound off", () => {
        // given
        const queryClient = createTestQueryClient();
        const deps = makeDeps(makeUser({ private: { play_notification_sound: false } }));
        renderHook(() => useNotificationFeedSync(deps), { wrapper: providerWrapper({ queryClient }) });

        // when
        emitNotification(4);

        // then
        expect(deps.showNotification).toHaveBeenCalledOnce();
        expect(deps.playSound).not.toHaveBeenCalled();
    });

    it("reads the viewer from the latest render, so the preference needs no ref", () => {
        // given
        const queryClient = createTestQueryClient();
        const playSound = vi.fn();
        const { rerender } = renderHook(
            ({ viewer }: { viewer: UserProfile }) => {
                useNotificationFeedSync({ viewer, showNotification: vi.fn(), playSound });
            },
            {
                wrapper: providerWrapper({ queryClient }),
                initialProps: { viewer: makeUser() },
            },
        );

        // when
        rerender({ viewer: makeUser({ private: { play_notification_sound: false } }) });
        emitNotification(4);

        // then
        expect(playSound).not.toHaveBeenCalled();
    });
});
