import { useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import {
    endWatchParty,
    forceMuteWatchPartyVoiceParticipant,
    getWatchPartyVoiceToken,
    identifyWatchPartyParticipant,
    joinWatchParty,
    kickWatchPartyParticipant,
    leaveWatchParty,
    startWatchParty,
    transferWatchPartyControl,
} from "../../api/endpoints/watchParty";
import { queryKeys } from "../../api/queryKeys";
import type { WatchPartyType } from "../../types/api";

const DROP_SECRET_ON_SETTLE = {
    gcTime: 0,
} as const;

export interface StartWatchPartyOptions {
    start_url?: string;
    region?: string;
    title?: string;
    type?: WatchPartyType;
    light?: boolean;
}

function requireRoom(roomId: string | null | undefined): string {
    if (!roomId) {
        throw new Error("No chat room is open.");
    }

    return roomId;
}

function refreshSessions(qc: QueryClient, roomId: string | null | undefined): void {
    qc.invalidateQueries({ queryKey: queryKeys.watchParty.list(roomId) });
}

export function useStartWatchParty(roomId: string | null | undefined) {
    const qc = useQueryClient();

    return useMutation({
        mutationFn: (options: StartWatchPartyOptions) => startWatchParty(requireRoom(roomId), options),
        onSuccess: () => refreshSessions(qc, roomId),
        ...DROP_SECRET_ON_SETTLE,
    });
}

export function useJoinWatchParty(roomId: string | null | undefined) {
    const qc = useQueryClient();

    return useMutation({
        mutationFn: (sessionId: string) => joinWatchParty(requireRoom(roomId), sessionId),
        onSuccess: () => refreshSessions(qc, roomId),
        ...DROP_SECRET_ON_SETTLE,
    });
}

export function useLeaveWatchParty(roomId: string | null | undefined) {
    const qc = useQueryClient();

    return useMutation({
        mutationFn: (sessionId: string) => leaveWatchParty(requireRoom(roomId), sessionId),
        onSuccess: () => refreshSessions(qc, roomId),
    });
}

export function useEndWatchParty(roomId: string | null | undefined) {
    const qc = useQueryClient();

    return useMutation({
        mutationFn: (sessionId: string) => endWatchParty(requireRoom(roomId), sessionId),
        onSuccess: () => refreshSessions(qc, roomId),
    });
}

export function useTransferWatchPartyControl(roomId: string | null | undefined) {
    const qc = useQueryClient();

    return useMutation({
        mutationFn: ({ sessionId, userId }: { sessionId: string; userId: string }) =>
            transferWatchPartyControl(requireRoom(roomId), sessionId, userId),
        onSuccess: () => refreshSessions(qc, roomId),
    });
}

export function useKickWatchPartyParticipant(roomId: string | null | undefined) {
    const qc = useQueryClient();

    return useMutation({
        mutationFn: ({ sessionId, userId }: { sessionId: string; userId: string }) =>
            kickWatchPartyParticipant(requireRoom(roomId), sessionId, userId),
        onSuccess: () => refreshSessions(qc, roomId),
    });
}

export function useIdentifyWatchPartyParticipant(roomId: string | null | undefined) {
    return useMutation({
        mutationFn: ({ sessionId, identifier }: { sessionId: string; identifier: string }) =>
            identifyWatchPartyParticipant(requireRoom(roomId), sessionId, identifier),
    });
}

export function useWatchPartyVoiceToken(roomId: string | null | undefined) {
    return useMutation({
        mutationFn: (sessionId: string) => getWatchPartyVoiceToken(requireRoom(roomId), sessionId),
        ...DROP_SECRET_ON_SETTLE,
    });
}

export function useForceMuteWatchPartyVoiceParticipant(roomId: string | null | undefined) {
    return useMutation({
        mutationFn: ({ sessionId, userId, muted }: { sessionId: string; userId: string; muted: boolean }) =>
            forceMuteWatchPartyVoiceParticipant(requireRoom(roomId), sessionId, userId, muted),
    });
}
