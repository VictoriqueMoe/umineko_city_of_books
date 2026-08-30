import { useCallback, useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../api/queryKeys";
import { REALTIME_EVENTS } from "../api/realtime/events";
import { REALTIME_COMMANDS, sendRealtime } from "../api/realtime/outbound";
import { useRealtimeEvent, useRealtimeStatus } from "../api/realtime/useRealtime";
import type {
    SecretDetailResponse,
    SecretLeaderboardEntry,
    SecretProgressEvent,
    SecretSolvedEvent,
} from "../types/api";
import { useSecret } from "./queries/secret";

export interface SecretRoom {
    detail: SecretDetailResponse | null;
    loading: boolean;
    refresh: () => void;
    solvedByName: string | null;
    dismissSolved: () => void;
}

function sortLeaderboard(rows: SecretLeaderboardEntry[]): SecretLeaderboardEntry[] {
    return [...rows].sort((a, b) => {
        if (a.solved !== b.solved) {
            return a.solved ? -1 : 1;
        }
        if (b.pieces_collected !== a.pieces_collected) {
            return b.pieces_collected - a.pieces_collected;
        }
        return a.user.display_name.localeCompare(b.user.display_name);
    });
}

function withProgress(rows: SecretLeaderboardEntry[], progress: SecretProgressEvent): SecretLeaderboardEntry[] {
    const next = [...rows];
    const existingIdx = next.findIndex(e => e.user.id === progress.user.id);

    if (existingIdx >= 0) {
        next[existingIdx] = { ...next[existingIdx], pieces_collected: progress.pieces_collected };
        return next;
    }

    next.push({ user: progress.user, pieces_collected: progress.pieces_collected, solved: false });

    return next;
}

function withSolver(detail: SecretDetailResponse, solved: SecretSolvedEvent): SecretDetailResponse {
    return {
        ...detail,
        solved: true,
        solver: solved.solver,
        solved_at: solved.solved_at,
        leaderboard: detail.leaderboard.map(e => (e.user.id === solved.solver.id ? { ...e, solved: true } : e)),
    };
}

export function useSecretRoom(id: string): SecretRoom {
    const qc = useQueryClient();
    const epoch = useRealtimeStatus();
    const { data: rawDetail, loading, refresh } = useSecret(id);
    const [solvedByName, setSolvedByName] = useState<string | null>(null);

    useEffect(() => {
        if (!id || epoch === 0) {
            return;
        }

        sendRealtime({ type: REALTIME_COMMANDS.SECRET_JOIN, data: { secret_id: id } });
        refresh();

        return () => {
            sendRealtime({ type: REALTIME_COMMANDS.SECRET_LEAVE, data: { secret_id: id } });
        };
    }, [id, epoch, refresh]);

    useRealtimeEvent([REALTIME_EVENTS.SECRET_PROGRESS, REALTIME_EVENTS.SECRET_SOLVED], event => {
        if (!id || event.data.secret_id !== id) {
            return;
        }

        const detailKey = queryKeys.secrets.detail(id);

        if (event.type === REALTIME_EVENTS.SECRET_PROGRESS) {
            const progress = event.data;
            qc.setQueryData<SecretDetailResponse>(detailKey, prev =>
                prev ? { ...prev, leaderboard: withProgress(prev.leaderboard, progress) } : prev,
            );
            return;
        }

        const solved = event.data;
        setSolvedByName(solved.solver.display_name);
        qc.setQueryData<SecretDetailResponse>(detailKey, prev => (prev ? withSolver(prev, solved) : prev));
    });

    const dismissSolved = useCallback(() => {
        setSolvedByName(null);
    }, []);

    const detail = rawDetail ? { ...rawDetail, leaderboard: sortLeaderboard(rawDetail.leaderboard) } : null;

    return { detail, loading, refresh, solvedByName, dismissSolved };
}
