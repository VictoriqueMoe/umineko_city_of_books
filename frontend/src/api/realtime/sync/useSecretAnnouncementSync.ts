import { useCallback, useState } from "react";
import { REALTIME_EVENTS, type SecretClosedPayload } from "../events";
import { useRealtimeEvent } from "../useRealtime";

export interface SecretAnnouncement {
    announcement: SecretClosedPayload | null;
    dismiss: () => void;
}

export function useSecretAnnouncementSync(): SecretAnnouncement {
    const [announcement, setAnnouncement] = useState<SecretClosedPayload | null>(null);

    useRealtimeEvent(REALTIME_EVENTS.SECRET_CLOSED, event => {
        const closed: SecretClosedPayload | undefined = event.data;

        if (!closed?.secret_id || !closed.solver) {
            return;
        }

        setAnnouncement(closed);
    });

    const dismiss = useCallback(() => {
        setAnnouncement(null);
    }, []);

    return { announcement, dismiss };
}
