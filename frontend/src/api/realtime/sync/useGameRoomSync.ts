import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { GameRoom } from "../../../types/api";
import { queryKeys } from "../../queryKeys";
import { REALTIME_EVENTS } from "../events";
import { REALTIME_COMMANDS, sendRealtime } from "../outbound";
import { useRealtimeEvent, useRealtimeStatus } from "../useRealtime";

const GAME_ROOM_EVENTS = [
    REALTIME_EVENTS.GAME_ROOM_ACTION,
    REALTIME_EVENTS.GAME_ROOM_STARTED,
    REALTIME_EVENTS.GAME_ROOM_FINISHED,
    REALTIME_EVENTS.GAME_ROOM_DECLINED,
    REALTIME_EVENTS.GAME_ROOM_PRESENCE,
    REALTIME_EVENTS.GAME_DRAW_OFFERED,
    REALTIME_EVENTS.GAME_DRAW_DECLINED,
] as const;

export interface GameRoomSyncStatus {
    connected: boolean;
}

export function useGameRoomSync<TState = unknown, TStats = unknown>(roomId: string | undefined): GameRoomSyncStatus {
    const qc = useQueryClient();
    const epoch = useRealtimeStatus();

    useEffect(() => {
        if (!roomId) {
            return;
        }

        sendRealtime({ type: REALTIME_COMMANDS.GAME_ROOM_JOIN, data: { room_id: roomId } });

        return () => {
            sendRealtime({ type: REALTIME_COMMANDS.GAME_ROOM_LEAVE, data: { room_id: roomId } });
        };
    }, [roomId, epoch]);

    useRealtimeEvent(GAME_ROOM_EVENTS, event => {
        if (!roomId || event.data.room_id !== roomId) {
            return;
        }

        const room = "room" in event.data ? (event.data.room as GameRoom<TState, TStats> | undefined) : undefined;

        if (room) {
            qc.setQueryData(queryKeys.gameRoom.detail(roomId), room);
            return;
        }

        if (event.type === REALTIME_EVENTS.GAME_ROOM_PRESENCE) {
            qc.invalidateQueries({ queryKey: queryKeys.gameRoom.detail(roomId) });
        }
    });

    return { connected: epoch > 0 };
}
