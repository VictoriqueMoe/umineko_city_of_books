import { useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import * as api from "../endpoints";
import { queryKeys } from "../queryKeys";
import { useNotifications } from "../../hooks/useNotifications";
import type { GameRoom, GameStatus, GameType, WSMessage } from "../../types/api";

export function useMyGameRooms(params?: { game_type?: GameType; status?: GameStatus }) {
    const q = useQuery({
        queryKey: queryKeys.gameRoom.list(params ?? {}),
        queryFn: () => api.listMyGameRooms(params),
    });
    return {
        rooms: q.data?.rooms ?? [],
        total: q.data?.total ?? 0,
        loading: q.isPending,
        error: q.error instanceof Error ? q.error.message : "",
        refresh: q.refetch,
    };
}

export function useLiveGameRooms(gameType?: GameType) {
    const q = useQuery({
        queryKey: gameType ? ["game-rooms", "live", gameType] : ["game-rooms", "live"],
        queryFn: () => api.listLiveGameRooms(gameType),
    });
    return {
        rooms: q.data?.rooms ?? [],
        total: q.data?.total ?? 0,
        loading: q.isPending,
        error: q.error instanceof Error ? q.error.message : "",
        refresh: q.refetch,
    };
}

export function useFinishedGameRooms(gameType?: GameType, limit = 20, offset = 0) {
    const q = useQuery({
        queryKey: ["game-rooms", "finished", gameType ?? "", { limit, offset }],
        queryFn: () => api.listFinishedGameRooms(gameType, limit, offset),
    });
    return { rooms: q.data?.rooms ?? [], total: q.data?.total ?? 0, loading: q.isPending };
}

export function useGameScoreboard(gameType: GameType | undefined) {
    const q = useQuery({
        queryKey: ["game-rooms", "scoreboard", gameType ?? ""],
        queryFn: () => api.getGameScoreboard(gameType!),
        enabled: !!gameType,
    });
    return { data: q.data ?? null, loading: q.isPending };
}

export function useSpectatorChat(roomId: string, enabled = true) {
    const q = useQuery({
        queryKey: ["game-rooms", roomId, "spectator-chat"],
        queryFn: () => api.getSpectatorChat(roomId),
        enabled: enabled && !!roomId,
    });
    return { data: q.data ?? null, loading: q.isPending, refresh: q.refetch };
}

export function usePlayerChat(roomId: string, enabled = true) {
    const q = useQuery({
        queryKey: ["game-rooms", roomId, "player-chat"],
        queryFn: () => api.getPlayerChat(roomId),
        enabled: enabled && !!roomId,
    });
    return { data: q.data ?? null, loading: q.isPending, refresh: q.refetch };
}

interface UseGameRoomResult {
    room: GameRoom | null;
    loading: boolean;
    error: string;
    refetch: () => Promise<void>;
    wsConnected: boolean;
}

export function useGameRoom(roomId: string | undefined): UseGameRoomResult {
    const queryClient = useQueryClient();
    const { addWSListener, sendWSMessage, wsEpoch } = useNotifications();

    const query = useQuery({
        queryKey: queryKeys.gameRoom.detail(roomId ?? ""),
        queryFn: () => api.getGameRoom(roomId!),
        enabled: !!roomId,
    });

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
            if (!roomId) {
                return;
            }
            if (msg.type === "game_your_turn") {
                const data = msg.data as { room_id?: string; game_type?: string };
                if (data.room_id !== roomId) {
                    return;
                }
                const tabHidden = document.visibilityState !== "visible" || !document.hasFocus();
                if (!tabHidden) {
                    return;
                }
                if (
                    typeof window !== "undefined" &&
                    "Notification" in window &&
                    window.Notification.permission === "granted"
                ) {
                    const label = data.game_type ?? "game";
                    const notif = new window.Notification(`Your move in ${label}`, {
                        body: "It's your turn.",
                        icon: "/favicon/android-chrome-192x192.png",
                        tag: `game-turn-${roomId}`,
                    });
                    notif.onclick = () => {
                        window.focus();
                        window.location.href = `/games/${data.game_type ?? "chess"}/${roomId}`;
                        notif.close();
                    };
                }
                return;
            }
            if (
                msg.type !== "game_room_action" &&
                msg.type !== "game_room_started" &&
                msg.type !== "game_room_finished" &&
                msg.type !== "game_room_declined" &&
                msg.type !== "game_room_presence" &&
                msg.type !== "game_draw_offered" &&
                msg.type !== "game_draw_declined"
            ) {
                return;
            }
            const data = msg.data as { room_id?: string; room?: GameRoom };
            if (data.room_id !== roomId) {
                return;
            }
            if (data.room) {
                queryClient.setQueryData(queryKeys.gameRoom.detail(roomId), data.room);
            } else if (msg.type === "game_room_presence") {
                void queryClient.invalidateQueries({ queryKey: queryKeys.gameRoom.detail(roomId) });
            }
        });
    }, [addWSListener, roomId, queryClient]);

    return {
        room: query.data ?? null,
        loading: query.isPending,
        error: query.error instanceof Error ? query.error.message : "",
        refetch: async () => {
            await query.refetch();
        },
        wsConnected: wsEpoch > 0,
    };
}
