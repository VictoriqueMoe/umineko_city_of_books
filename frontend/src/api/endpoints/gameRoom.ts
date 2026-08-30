import { apiFetch, apiPost, buildQueryString } from "../client";
import type {
    GameRoom,
    GameRoomListResponse,
    GameScoreboardResponse,
    GameStatus,
    GameType,
    SpectatorChatResponse,
    SpectatorMessage,
} from "../../types/api";

export async function inviteToGame(opponentId: string, gameType: GameType): Promise<GameRoom> {
    return apiPost<GameRoom, { opponent_id: string; game_type: GameType }>(`/game-rooms`, {
        opponent_id: opponentId,
        game_type: gameType,
    });
}

export async function listMyGameRooms(params?: {
    game_type?: GameType;
    status?: GameStatus;
}): Promise<GameRoomListResponse> {
    const qs = buildQueryString({ game_type: params?.game_type ?? "", status: params?.status ?? "" });
    return apiFetch<GameRoomListResponse>(`/game-rooms${qs}`);
}

export async function getGameRoom<TState = unknown, TStats = unknown>(id: string): Promise<GameRoom<TState, TStats>> {
    return apiFetch<GameRoom<TState, TStats>>(`/game-rooms/${id}`);
}

export async function acceptGameInvite(id: string): Promise<GameRoom> {
    return apiPost<GameRoom, Record<string, never>>(`/game-rooms/${id}/accept`, {});
}

export async function declineGameInvite(id: string): Promise<void> {
    await apiPost<unknown, Record<string, never>>(`/game-rooms/${id}/decline`, {});
}

export async function cancelGameInvite(id: string): Promise<void> {
    await apiPost<unknown, Record<string, never>>(`/game-rooms/${id}/cancel`, {});
}

export async function submitGameAction(id: string, action: Record<string, unknown>): Promise<GameRoom> {
    return apiPost<GameRoom, { action: Record<string, unknown> }>(`/game-rooms/${id}/action`, { action });
}

export async function resignGame(id: string): Promise<GameRoom> {
    return apiPost<GameRoom, Record<string, never>>(`/game-rooms/${id}/resign`, {});
}

export async function offerDraw(id: string): Promise<GameRoom> {
    return apiPost<GameRoom, Record<string, never>>(`/game-rooms/${id}/offer-draw`, {});
}

export async function acceptDraw(id: string): Promise<GameRoom> {
    return apiPost<GameRoom, Record<string, never>>(`/game-rooms/${id}/accept-draw`, {});
}

export async function declineDraw(id: string): Promise<GameRoom> {
    return apiPost<GameRoom, Record<string, never>>(`/game-rooms/${id}/decline-draw`, {});
}

export async function getGameScoreboard(gameType: GameType): Promise<GameScoreboardResponse> {
    return apiFetch<GameScoreboardResponse>(`/games/${encodeURIComponent(gameType)}/scoreboard`);
}

export async function listLiveGameRooms(gameType?: GameType): Promise<GameRoomListResponse> {
    const qs = buildQueryString({ game_type: gameType ?? "" });
    return apiFetch<GameRoomListResponse>(`/game-rooms/live${qs}`);
}

export async function listFinishedGameRooms(
    gameType?: GameType,
    limit: number = 20,
    offset: number = 0,
): Promise<GameRoomListResponse> {
    const qs = buildQueryString({ game_type: gameType ?? "", limit, offset });
    return apiFetch<GameRoomListResponse>(`/game-rooms/finished${qs}`);
}

export async function getSpectatorChat(roomId: string): Promise<SpectatorChatResponse> {
    return apiFetch<SpectatorChatResponse>(`/game-rooms/${roomId}/chat`);
}

export async function postSpectatorChat(roomId: string, body: string): Promise<SpectatorMessage> {
    return apiPost<SpectatorMessage, { body: string }>(`/game-rooms/${roomId}/chat`, { body });
}

export async function getPlayerChat(roomId: string): Promise<SpectatorChatResponse> {
    return apiFetch<SpectatorChatResponse>(`/game-rooms/${roomId}/player-chat`);
}

export async function postPlayerChat(roomId: string, body: string): Promise<SpectatorMessage> {
    return apiPost<SpectatorMessage, { body: string }>(`/game-rooms/${roomId}/player-chat`, { body });
}
