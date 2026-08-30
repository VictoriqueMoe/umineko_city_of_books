import { useQueryClient } from "@tanstack/react-query";
import type { Notification, UserProfile } from "../../../types/api";
import { queryKeys } from "../../queryKeys";
import { REALTIME_EVENTS } from "../events";
import { useRealtimeEvent } from "../useRealtime";

export interface NotificationFeedSyncDeps {
    viewer: UserProfile | null;
    showNotification: (notification: Notification) => void;
    playSound: () => void;
}

export function useNotificationFeedSync({ viewer, showNotification, playSound }: NotificationFeedSyncDeps): void {
    const qc = useQueryClient();

    useRealtimeEvent(REALTIME_EVENTS.NOTIFICATION, event => {
        const notif = event.data;

        qc.setQueryData<{ count: number }>(queryKeys.notifications.unreadCount(), prev => ({
            count: (prev?.count ?? 0) + 1,
        }));
        qc.invalidateQueries({ queryKey: queryKeys.notifications.listAll() });
        showNotification(notif);

        if (viewer?.private?.play_notification_sound ?? true) {
            playSound();
        }
    });
}
