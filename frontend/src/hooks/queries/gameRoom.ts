import { useQuery } from "@tanstack/react-query";
import * as api from "../../api/endpoints/gameRoom";
import { queryKeys } from "../../api/queryKeys";
import { useGameRoomSync } from "../../api/realtime/sync/useGameRoomSync";
import { useGameTurnNotice } from "../useGameTurnNotice";
import type { GameRoom, GameStatus, GameType, SpectatorMessage } from "../../types/api";

export function useMyGameRooms(params?: { game_type?: GameType; status?: GameStatus }) {
    const q = useQuery({
        queryKey: queryKeys.gameRoom.list(params ?? {}),
        queryFn: () => api.listMyGameRooms(params),
    });
    return {
        rooms: q.data?.rooms ?? [],
        total: q.data?.total ?? 0,
        loading: q.isLoading,
        error: q.error instanceof Error ? q.error.message : "",
        refresh: q.refetch,
    };
}

export function useLiveGameRooms(gameType?: GameType) {
    const q = useQuery({
        queryKey: queryKeys.gameRoom.live(gameType),
        queryFn: () => api.listLiveGameRooms(gameType),
    });
    return {
        rooms: q.data?.rooms ?? [],
        total: q.data?.total ?? 0,
        loading: q.isLoading,
        error: q.error instanceof Error ? q.error.message : "",
        refresh: q.refetch,
    };
}

export function useFinishedGameRooms(gameType?: GameType, limit = 20, offset = 0) {
    const q = useQuery({
        queryKey: queryKeys.gameRoom.finished(gameType ?? "", { limit, offset }),
        queryFn: () => api.listFinishedGameRooms(gameType, limit, offset),
    });
    return { rooms: q.data?.rooms ?? [], total: q.data?.total ?? 0, loading: q.isLoading };
}

export function useGameScoreboard(gameType: GameType | undefined) {
    const q = useQuery({
        queryKey: queryKeys.gameRoom.scoreboard(gameType ?? ""),
        queryFn: () => api.getGameScoreboard(gameType!),
        enabled: !!gameType,
    });
    return { data: q.data ?? null, loading: q.isLoading };
}

const NO_MESSAGES: SpectatorMessage[] = [];

export interface GameChatHistory {
    messages: SpectatorMessage[];
    loading: boolean;
    error: string;
    refresh: () => Promise<unknown>;
}

export function useSpectatorChat(roomId: string, enabled = true): GameChatHistory {
    const q = useQuery({
        queryKey: queryKeys.gameRoom.spectatorChat(roomId),
        queryFn: () => api.getSpectatorChat(roomId),
        enabled: enabled && !!roomId,
    });
    return {
        messages: q.data?.messages ?? NO_MESSAGES,
        loading: q.isLoading,
        error: q.error instanceof Error ? q.error.message : "",
        refresh: q.refetch,
    };
}

export function usePlayerChat(roomId: string, enabled = true): GameChatHistory {
    const q = useQuery({
        queryKey: queryKeys.gameRoom.playerChat(roomId),
        queryFn: () => api.getPlayerChat(roomId),
        enabled: enabled && !!roomId,
    });
    return {
        messages: q.data?.messages ?? NO_MESSAGES,
        loading: q.isLoading,
        error: q.error instanceof Error ? q.error.message : "",
        refresh: q.refetch,
    };
}

interface UseGameRoomResult<TState, TStats> {
    room: GameRoom<TState, TStats> | null;
    loading: boolean;
    error: string;
    refetch: () => Promise<void>;
    wsConnected: boolean;
}

export function useGameRoom<TState = unknown, TStats = unknown>(
    roomId: string | undefined,
): UseGameRoomResult<TState, TStats> {
    const query = useQuery({
        queryKey: queryKeys.gameRoom.detail(roomId ?? ""),
        queryFn: () => api.getGameRoom<TState, TStats>(roomId!),
        enabled: !!roomId,
    });

    const { connected } = useGameRoomSync<TState, TStats>(roomId);
    useGameTurnNotice(roomId);

    return {
        room: query.data ?? null,
        loading: query.isLoading,
        error: query.error instanceof Error ? query.error.message : "",
        refetch: async () => {
            await query.refetch();
        },
        wsConnected: connected,
    };
}
