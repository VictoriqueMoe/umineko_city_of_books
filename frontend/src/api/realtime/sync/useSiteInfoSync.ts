import { useCallback, useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../../queryKeys";
import { REALTIME_EVENTS } from "../events";
import { useRealtimeEvent, useRealtimeStatus } from "../useRealtime";

const MIN_REFETCH_INTERVAL_MS = 2000;

const SITE_INFO_EVENTS = [
    REALTIME_EVENTS.TOP_DETECTIVE_CHANGED,
    REALTIME_EVENTS.TOP_GM_CHANGED,
    REALTIME_EVENTS.TOP_CHESS_CHANGED,
    REALTIME_EVENTS.TOP_CHECKERS_CHANGED,
    REALTIME_EVENTS.TOP_OTHELLO_CHANGED,
    REALTIME_EVENTS.TOP_MINESWEEPER_CHANGED,
    REALTIME_EVENTS.VANITY_ROLES_CHANGED,
    REALTIME_EVENTS.RULES_PAGE_CHANGED,
] as const;

export function useSiteInfoSync(): void {
    const queryClient = useQueryClient();
    const epoch = useRealtimeStatus();

    const lastRefreshAt = useRef(0);
    const lastEpoch = useRef(epoch);

    const refresh = useCallback(() => {
        const now = Date.now();
        const dataUpdatedAt = queryClient.getQueryState(queryKeys.auth.siteInfo())?.dataUpdatedAt ?? 0;

        if (now - Math.max(lastRefreshAt.current, dataUpdatedAt) < MIN_REFETCH_INTERVAL_MS) {
            return;
        }

        lastRefreshAt.current = now;
        queryClient.invalidateQueries({ queryKey: queryKeys.auth.siteInfo() });
    }, [queryClient]);

    useRealtimeEvent(SITE_INFO_EVENTS, refresh);

    useEffect(() => {
        if (epoch === lastEpoch.current) {
            return;
        }

        lastEpoch.current = epoch;
        refresh();
    }, [epoch, refresh]);
}
