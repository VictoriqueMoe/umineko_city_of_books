import { apiDelete, apiFetch, apiPatch, apiPost } from "../client";
import type {
    JoinWatchPartyResponse,
    StartWatchPartyResponse,
    VoiceTokenResponse,
    WatchPartyListResponse,
} from "../../types/api";

export async function getWatchPartyVoiceToken(roomId: string, sessionId: string): Promise<VoiceTokenResponse> {
    return apiPost<VoiceTokenResponse, Record<string, never>>(
        `/chat/rooms/${roomId}/watch-parties/${sessionId}/voice/token`,
        {},
    );
}

export async function forceMuteWatchPartyVoiceParticipant(
    roomId: string,
    sessionId: string,
    userId: string,
    muted: boolean,
): Promise<void> {
    await apiPost<unknown, { muted: boolean }>(
        `/chat/rooms/${roomId}/watch-parties/${sessionId}/voice/participants/${userId}/mute`,
        { muted },
    );
}

export async function listWatchParties(roomId: string): Promise<WatchPartyListResponse> {
    return apiFetch<WatchPartyListResponse>(`/chat/rooms/${roomId}/watch-parties`);
}

export async function startWatchParty(
    roomId: string,
    options: {
        start_url?: string;
        region?: string;
        title?: string;
        type?: "hyperbeam" | "screenshare";
        light?: boolean;
    },
): Promise<StartWatchPartyResponse> {
    return apiPost<
        StartWatchPartyResponse,
        { start_url?: string; region?: string; title?: string; type?: "hyperbeam" | "screenshare"; light?: boolean }
    >(`/chat/rooms/${roomId}/watch-parties`, options);
}

export async function joinWatchParty(roomId: string, sessionId: string): Promise<JoinWatchPartyResponse> {
    return apiPost<JoinWatchPartyResponse, Record<string, never>>(
        `/chat/rooms/${roomId}/watch-parties/${sessionId}/join`,
        {},
    );
}

export async function leaveWatchParty(roomId: string, sessionId: string): Promise<void> {
    await apiDelete<unknown>(`/chat/rooms/${roomId}/watch-parties/${sessionId}/participants/me`);
}

export async function endWatchParty(roomId: string, sessionId: string): Promise<void> {
    await apiDelete<unknown>(`/chat/rooms/${roomId}/watch-parties/${sessionId}`);
}

export async function transferWatchPartyControl(roomId: string, sessionId: string, userId: string): Promise<void> {
    await apiPatch<unknown, Record<string, never>>(
        `/chat/rooms/${roomId}/watch-parties/${sessionId}/participants/${userId}`,
        {},
    );
}

export async function kickWatchPartyParticipant(roomId: string, sessionId: string, userId: string): Promise<void> {
    await apiDelete<unknown>(`/chat/rooms/${roomId}/watch-parties/${sessionId}/participants/${userId}`);
}

export async function identifyWatchPartyParticipant(
    roomId: string,
    sessionId: string,
    identifier: string,
): Promise<void> {
    await apiPost<unknown, { identifier: string }>(`/chat/rooms/${roomId}/watch-parties/${sessionId}/identify`, {
        identifier,
    });
}
