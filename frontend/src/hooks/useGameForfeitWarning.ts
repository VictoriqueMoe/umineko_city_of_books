import { useCallback, useState } from "react";
import { REALTIME_EVENTS } from "../api/realtime/events";
import { useRealtimeEvent } from "../api/realtime/useRealtime";
import { useAuth } from "./useAuth";

export interface PendingForfeit {
    roomId: string;
    gameType: string;
    forfeitAt: number;
}

export interface FinalForfeit {
    roomId: string;
    gameType: string;
}

export interface GameForfeitWarningState {
    pending: PendingForfeit | null;
    finalForfeit: FinalForfeit | null;
    dismissFinalForfeit: () => void;
}

const FORFEIT_EVENTS = [
    REALTIME_EVENTS.GAME_FORFEIT_WARNING,
    REALTIME_EVENTS.GAME_FORFEIT_CLEARED,
    REALTIME_EVENTS.GAME_ROOM_FINISHED,
] as const;

export function useGameForfeitWarning(): GameForfeitWarningState {
    const { user } = useAuth();
    const [pending, setPending] = useState<PendingForfeit | null>(null);
    const [finalForfeit, setFinalForfeit] = useState<FinalForfeit | null>(null);

    useRealtimeEvent(FORFEIT_EVENTS, event => {
        switch (event.type) {
            case "game_forfeit_warning": {
                const data = event.data;

                if (!data.room_id || !data.disconnected_at || typeof data.grace_seconds !== "number") {
                    return;
                }

                const startedAt = Date.parse(data.disconnected_at);

                if (Number.isNaN(startedAt)) {
                    return;
                }

                setPending({
                    roomId: data.room_id,
                    gameType: data.game_type ?? "chess",
                    forfeitAt: startedAt + data.grace_seconds * 1000,
                });

                return;
            }
            case "game_forfeit_cleared": {
                const data = event.data;

                setPending(prev => (prev && prev.roomId === data.room_id ? null : prev));

                return;
            }
            case "game_room_finished": {
                const data = event.data;

                if (!data.room_id || !data.abandoned_by || !user) {
                    return;
                }

                if (data.abandoned_by !== user.id) {
                    return;
                }

                const roomId = data.room_id;

                setPending(prev => {
                    const gameType = prev && prev.roomId === roomId ? prev.gameType : "chess";
                    setFinalForfeit({ roomId, gameType });

                    return null;
                });
            }
        }
    });

    const dismissFinalForfeit = useCallback(() => {
        setFinalForfeit(null);
    }, []);

    return { pending, finalForfeit, dismissFinalForfeit };
}
