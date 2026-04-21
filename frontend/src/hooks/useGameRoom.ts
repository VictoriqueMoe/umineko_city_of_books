import { useCallback, useEffect, useRef, useState } from "react";
import * as api from "../api/endpoints";
import type { GameRoom, WSMessage } from "../types/api";
import { useNotifications } from "./useNotifications";

interface UseGameRoomResult {
    room: GameRoom | null;
    loading: boolean;
    error: string;
    refetch: () => Promise<void>;
    wsConnected: boolean;
}

export function useGameRoom(roomId: string | undefined): UseGameRoomResult {
    const { addWSListener, sendWSMessage, wsEpoch } = useNotifications();
    const [room, setRoom] = useState<GameRoom | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const roomIdRef = useRef(roomId);
    roomIdRef.current = roomId;

    const refetch = useCallback(async () => {
        if (!roomId) {
            return;
        }
        setLoading(true);
        setError("");
        try {
            const fresh = await api.getGameRoom(roomId);
            setRoom(fresh);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to load game");
        } finally {
            setLoading(false);
        }
    }, [roomId]);

    useEffect(() => {
        if (!roomId) {
            return;
        }
        void refetch();
    }, [roomId, refetch, wsEpoch]);

    useEffect(() => {
        if (!roomId) {
            return;
        }
        sendWSMessage({ type: "game_room_join", data: { room_id: roomId } });
        return () => {
            sendWSMessage({ type: "game_room_leave", data: { room_id: roomId } });
        };
    }, [roomId, sendWSMessage, wsEpoch]);

    useEffect(() => {
        return addWSListener((msg: WSMessage) => {
            const currentRoomId = roomIdRef.current;
            if (!currentRoomId) {
                return;
            }
            if (
                msg.type !== "game_room_action" &&
                msg.type !== "game_room_started" &&
                msg.type !== "game_room_finished" &&
                msg.type !== "game_room_declined" &&
                msg.type !== "game_room_presence"
            ) {
                return;
            }
            const data = msg.data as { room_id?: string; room?: GameRoom };
            if (data.room_id !== currentRoomId) {
                return;
            }
            if (data.room) {
                setRoom(data.room);
            } else if (msg.type === "game_room_presence") {
                void refetch();
            }
        });
    }, [addWSListener, refetch]);

    return {
        room,
        loading,
        error,
        refetch,
        wsConnected: wsEpoch > 0,
    };
}
