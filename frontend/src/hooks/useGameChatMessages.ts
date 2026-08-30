import { useEffect, useMemo, useRef, useState } from "react";
import { REALTIME_EVENTS } from "../api/realtime/events";
import { useRealtimeEvent, useRealtimeStatus } from "../api/realtime/useRealtime";
import type { SpectatorMessage } from "../types/api";
import type { GameChatHistory } from "./queries/gameRoom";

export type GameChatEventName =
    typeof REALTIME_EVENTS.PLAYER_CHAT_MESSAGE | typeof REALTIME_EVENTS.SPECTATOR_CHAT_MESSAGE;

interface ArrivalBuffer {
    roomId: string;
    wsType: GameChatEventName;
    messages: SpectatorMessage[];
}

const NO_ARRIVALS: SpectatorMessage[] = [];

function arrivalsFor(buffer: ArrivalBuffer, roomId: string, wsType: GameChatEventName): SpectatorMessage[] {
    return buffer.roomId === roomId && buffer.wsType === wsType ? buffer.messages : NO_ARRIVALS;
}

function withArrivals(history: SpectatorMessage[], arrivals: SpectatorMessage[]): SpectatorMessage[] {
    if (arrivals.length === 0) {
        return history;
    }

    const known = new Set(history.map(m => m.id));
    const unseen = arrivals.filter(m => !known.has(m.id));

    return unseen.length === 0 ? history : [...history, ...unseen];
}

export function useGameChatMessages(
    roomId: string,
    wsType: GameChatEventName,
    history: GameChatHistory,
): SpectatorMessage[] {
    const [buffer, setBuffer] = useState<ArrivalBuffer>({ roomId, wsType, messages: NO_ARRIVALS });

    const realtimeEpoch = useRealtimeStatus();
    const refresh = history.refresh;
    const refreshedAtRef = useRef(realtimeEpoch);
    useEffect(() => {
        if (refreshedAtRef.current === realtimeEpoch) {
            return;
        }
        refreshedAtRef.current = realtimeEpoch;

        refresh().catch(() => {});
    }, [realtimeEpoch, refresh]);

    useRealtimeEvent(wsType, event => {
        const message = event.data.message;
        if (event.data.room_id !== roomId || !message) {
            return;
        }

        setBuffer(prev => {
            const seen = arrivalsFor(prev, roomId, wsType);
            if (seen.some(m => m.id === message.id)) {
                return prev;
            }
            return { roomId, wsType, messages: [...seen, message] };
        });
    });

    const arrivals = arrivalsFor(buffer, roomId, wsType);

    return useMemo(() => withArrivals(history.messages, arrivals), [history.messages, arrivals]);
}
