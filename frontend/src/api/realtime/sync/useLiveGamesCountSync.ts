import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../../queryKeys";
import { REALTIME_EVENTS } from "../events";
import { useRealtimeEvent } from "../useRealtime";

export function useLiveGamesCountSync(): void {
    const qc = useQueryClient();

    useRealtimeEvent(REALTIME_EVENTS.LIVE_GAMES_COUNT, event => {
        const { count } = event.data as { count?: number };

        if (typeof count !== "number") {
            return;
        }

        qc.setQueryData<{ rooms: unknown[]; total: number }>(queryKeys.gameRoom.live(), prev => ({
            rooms: prev?.rooms ?? [],
            total: count,
        }));

        qc.invalidateQueries({ queryKey: queryKeys.gameRoom.live() });
    });
}
