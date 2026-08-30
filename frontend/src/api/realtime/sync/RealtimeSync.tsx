import { useEffect } from "react";
import type { Notification, UserProfile } from "../../../types/api";
import { closeRealtimeSocket, openRealtimeSocket } from "../socket";
import { useChatUnreadSync } from "./useChatUnreadSync";
import { useChatbotsSync } from "./useChatbotsSync";
import { useLiveGamesCountSync } from "./useLiveGamesCountSync";
import { useNotificationFeedSync } from "./useNotificationFeedSync";
import { usePermissionsSync } from "./usePermissionsSync";
import { useSiteInfoSync } from "./useSiteInfoSync";
import { useStreamDirectorySync } from "./useStreamDirectorySync";
import { useUserIdentitySync } from "./useUserIdentitySync";

export interface RealtimeSyncProps {
    viewer: UserProfile | null;
    showNotification: (notification: Notification) => void;
    playSound: () => void;
}

export function RealtimeSync({ viewer, showNotification, playSound }: RealtimeSyncProps): null {
    useNotificationFeedSync({ viewer, showNotification, playSound });
    useUserIdentitySync();
    usePermissionsSync();
    useChatUnreadSync();
    useLiveGamesCountSync();
    useChatbotsSync();
    useStreamDirectorySync();
    useSiteInfoSync();

    const viewerId = viewer?.id;

    useEffect(() => {
        if (!viewerId) {
            return;
        }

        openRealtimeSocket(viewerId);

        return () => {
            closeRealtimeSocket(viewerId);
        };
    }, [viewerId]);

    return null;
}
