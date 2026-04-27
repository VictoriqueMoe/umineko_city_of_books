import { useMutation, useQueryClient } from "@tanstack/react-query";
import { markAllNotificationsRead, markNotificationRead, subscribePush, unsubscribePush } from "../endpoints";
import { queryKeys } from "../queryKeys";

export function useMarkNotificationRead() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: number) => markNotificationRead(id),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.notifications.all });
        },
    });
}

export function useMarkAllNotificationsRead() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: () => markAllNotificationsRead(),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.notifications.all });
        },
    });
}

export function useSubscribePush() {
    return useMutation({
        mutationFn: (data: { endpoint: string; keys: { p256dh: string; auth: string } }) => subscribePush(data),
    });
}

export function useUnsubscribePush() {
    return useMutation({
        mutationFn: (data: { endpoint: string }) => unsubscribePush(data),
    });
}
