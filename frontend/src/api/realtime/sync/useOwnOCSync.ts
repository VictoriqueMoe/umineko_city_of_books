import { useQueryClient } from "@tanstack/react-query";
import type { OCSummary } from "../../../types/api";
import { queryKeys } from "../../queryKeys";
import { REALTIME_EVENTS } from "../events";
import { useRealtimeEvent } from "../useRealtime";

export function useOwnOCSync(userId: string, currentUserId?: string): void {
    const qc = useQueryClient();

    useRealtimeEvent(REALTIME_EVENTS.USER_OCS_CHANGED, event => {
        if (!userId || userId !== currentUserId) {
            return;
        }

        const { action, oc } = event.data;

        qc.setQueryData<OCSummary[]>(queryKeys.oc.userSummaries(userId), prev => {
            const existing = prev ?? [];
            if (action === "deleted") {
                return existing.filter(item => item.id !== oc.id);
            }
            if (action === "updated") {
                return existing.map(item => (item.id === oc.id ? oc : item));
            }
            if (action === "created") {
                if (existing.some(item => item.id === oc.id)) {
                    return existing;
                }
                return [...existing, oc].sort((a, b) => a.name.localeCompare(b.name));
            }
            return existing;
        });
    });
}
