import { useCallback, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../api/queryKeys";
import { REALTIME_EVENTS } from "../api/realtime/events";
import { useRealtimeEvent } from "../api/realtime/useRealtime";
import { prependNotification, type NotificationPage } from "../domain/notifications";
import type { Notification } from "../types/api";
import { useNotifications as useNotificationsQuery } from "./queries/notification";
import { useNotifications } from "./useNotifications";

export const NOTIFICATION_PAGE_SIZE = 50;

export interface NotificationFeed {
    notifications: Notification[];
    total: number;
    loading: boolean;
    hasMore: boolean;
    loadMore: () => void;
    markingId: number | null;
    markRead: (notif: Notification) => Promise<void>;
    markAllRead: () => Promise<void>;
}

export function useNotificationFeed(): NotificationFeed {
    const qc = useQueryClient();
    const { markRead: markReadOnServer, markAllRead: markAllReadOnServer } = useNotifications();
    const [limit, setLimit] = useState(NOTIFICATION_PAGE_SIZE);
    const [markingId, setMarkingId] = useState<number | null>(null);

    const query = useNotificationsQuery(limit, 0);
    const listKey = useMemo(() => queryKeys.notifications.list({ limit, offset: 0 }), [limit]);

    useRealtimeEvent(REALTIME_EVENTS.NOTIFICATION, event => {
        const arrival = event.data;

        qc.setQueryData<NotificationPage>(listKey, prev => (prev ? prependNotification(prev, arrival) : prev));
    });

    const loadMore = useCallback(() => {
        setLimit(current => current + NOTIFICATION_PAGE_SIZE);
    }, []);

    const markRead = useCallback(
        async (notif: Notification) => {
            if (notif.read || markingId === notif.id) {
                return;
            }

            setMarkingId(notif.id);

            try {
                await markReadOnServer(notif.id);
                qc.setQueryData<NotificationPage>(listKey, prev => {
                    if (!prev) {
                        return prev;
                    }

                    return {
                        ...prev,
                        notifications: prev.notifications.map(n => (n.id === notif.id ? { ...n, read: true } : n)),
                    };
                });
            } catch {
                return;
            } finally {
                setMarkingId(current => (current === notif.id ? null : current));
            }
        },
        [markReadOnServer, markingId, qc, listKey],
    );

    const markAllRead = useCallback(async () => {
        try {
            await markAllReadOnServer();
        } catch {
            return;
        }

        qc.setQueryData<NotificationPage>(listKey, prev =>
            prev ? { ...prev, notifications: prev.notifications.map(n => ({ ...n, read: true })) } : prev,
        );
    }, [markAllReadOnServer, qc, listKey]);

    return {
        notifications: query.notifications,
        total: query.total,
        loading: query.loading,
        hasMore: query.notifications.length < query.total,
        loadMore,
        markingId,
        markRead,
        markAllRead,
    };
}
