import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../../queryKeys";
import { REALTIME_EVENTS } from "../events";
import { useRealtimeEvent } from "../useRealtime";

const VIEWER_IDENTITY_EVENTS = [REALTIME_EVENTS.PERMISSIONS_CHANGED, REALTIME_EVENTS.VANITY_ROLES_CHANGED] as const;

export function usePermissionsSync(): void {
    const qc = useQueryClient();

    useRealtimeEvent(VIEWER_IDENTITY_EVENTS, () => {
        qc.invalidateQueries({ queryKey: queryKeys.auth.me() });
    });
}
