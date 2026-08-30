import { useLocation } from "react-router";
import { REALTIME_EVENTS } from "../api/realtime/events";
import { useRealtimeEvent } from "../api/realtime/useRealtime";
import { useAuth } from "./useAuth";

export function useNewAnnouncementAlert(onAnnouncement: () => void): void {
    const { user } = useAuth();
    const location = useLocation();

    useRealtimeEvent(REALTIME_EVENTS.NEW_ANNOUNCEMENT, event => {
        if (event.data.author_id === user?.id) {
            return;
        }

        if (location.pathname.startsWith("/announcement")) {
            return;
        }

        onAnnouncement();
    });
}
