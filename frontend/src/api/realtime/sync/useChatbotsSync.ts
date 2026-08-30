import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../../queryKeys";
import { REALTIME_EVENTS } from "../events";
import { useRealtimeEvent } from "../useRealtime";

export function useChatbotsSync(): void {
    const qc = useQueryClient();

    useRealtimeEvent(REALTIME_EVENTS.CHATBOTS_CHANGED, () => {
        qc.invalidateQueries({ queryKey: queryKeys.chatbots.all });
    });
}
