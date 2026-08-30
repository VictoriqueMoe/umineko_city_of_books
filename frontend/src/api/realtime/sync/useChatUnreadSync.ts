import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../../queryKeys";
import { REALTIME_EVENTS } from "../events";
import { useRealtimeEvent } from "../useRealtime";

const CHAT_UNREAD_EVENTS = [REALTIME_EVENTS.CHAT_UNREAD_BUMPED, REALTIME_EVENTS.CHAT_READ] as const;

export function useChatUnreadSync(): void {
    const qc = useQueryClient();

    useRealtimeEvent(CHAT_UNREAD_EVENTS, event => {
        const { total } = event.data as { total?: number };

        if (typeof total !== "number") {
            return;
        }

        qc.setQueryData<{ count: number }>(queryKeys.chat.unreadCount(), { count: total });
    });
}
