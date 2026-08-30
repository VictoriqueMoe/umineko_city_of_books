import { REALTIME_EVENTS } from "../api/realtime/events";
import { useRealtimeEvent } from "../api/realtime/useRealtime";
import { turnNotice } from "../domain/games/turnNotice";
import { isTabHidden, showGameTurnNotification } from "../platform/desktopNotifications";

export function useGameTurnNotice(roomId: string | undefined): void {
    useRealtimeEvent(REALTIME_EVENTS.GAME_YOUR_TURN, event => {
        const notice = turnNotice({
            watchedRoomId: roomId,
            eventRoomId: event.data.room_id,
            gameType: event.data.game_type,
            tabHidden: isTabHidden(),
        });

        if (!notice) {
            return;
        }

        showGameTurnNotification(notice);
    });
}
