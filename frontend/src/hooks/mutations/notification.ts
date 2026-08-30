import { useMutation, useQueryClient } from "@tanstack/react-query";
import { markAllNotificationsRead, markNotificationRead } from "../../api/endpoints/notification";
import { queryKeys } from "../../api/queryKeys";

export function useMarkNotificationRead() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: number) => markNotificationRead(id),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.notifications.all });
        },
    });
}

export function useMarkAllNotificationsRead() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: () => markAllNotificationsRead(),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.notifications.all });
        },
    });
}
