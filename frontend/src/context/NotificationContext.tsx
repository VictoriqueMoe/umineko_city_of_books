import { type PropsWithChildren, useCallback, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { NotificationContext } from "./notificationContextValue";
import { useAuth } from "../hooks/useAuth";
import { useUnreadCount } from "../hooks/queries/notification";
import { useChatUnreadCount } from "../hooks/queries/chat";
import { useLiveGameRooms } from "../hooks/queries/gameRoom";
import { useLiveStreamsCount } from "../hooks/queries/stream";
import { useMarkAllNotificationsRead, useMarkNotificationRead } from "../hooks/mutations/notification";
import { queryKeys } from "../api/queryKeys";
import { showDesktopNotification } from "../platform/desktopNotifications";
import { playNotificationSound } from "../platform/sound";
import { RealtimeSync } from "../api/realtime/sync/RealtimeSync";

export function NotificationProvider({ children }: PropsWithChildren) {
    const { user } = useAuth();
    const qc = useQueryClient();

    const unreadCountQuery = useUnreadCount();
    const refreshUnreadCount = unreadCountQuery.refresh;
    const chatUnreadCountQuery = useChatUnreadCount();
    const liveGamesQuery = useLiveGameRooms();
    const liveStreamsQuery = useLiveStreamsCount();

    const unreadCount = user ? unreadCountQuery.count : 0;
    const chatUnreadCount = user ? chatUnreadCountQuery.count : 0;
    const liveGamesCount = liveGamesQuery.total ?? 0;
    const liveStreamsCount = liveStreamsQuery.count;

    const { mutateAsync: markReadMutate } = useMarkNotificationRead();
    const { mutateAsync: markAllReadMutate } = useMarkAllNotificationsRead();

    const markRead = useCallback(
        async (id: number) => {
            await markReadMutate(id);
            await refreshUnreadCount();
        },
        [markReadMutate, refreshUnreadCount],
    );

    const markAllRead = useCallback(async () => {
        await markAllReadMutate();
        qc.setQueryData<{ count: number }>(queryKeys.notifications.unreadCount(), { count: 0 });
    }, [markAllReadMutate, qc]);

    const value = useMemo(
        () => ({
            unreadCount,
            chatUnreadCount,
            liveGamesCount,
            liveStreamsCount,
            markRead,
            markAllRead,
        }),
        [unreadCount, chatUnreadCount, liveGamesCount, liveStreamsCount, markRead, markAllRead],
    );

    return (
        <NotificationContext.Provider value={value}>
            <RealtimeSync viewer={user} showNotification={showDesktopNotification} playSound={playNotificationSound} />
            {children}
        </NotificationContext.Provider>
    );
}
