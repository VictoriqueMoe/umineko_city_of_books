import { useMemo } from "react";
import { useSecondsTick } from "../../hooks/useSecondsTick";
import type { GameRoom, GameRoomPlayer } from "../../types/api";

export const DISCONNECT_GRACE_SECONDS = 60;

export type ResultTone = "win" | "loss" | "draw" | "neutral";

export function getMySlot(room: GameRoom, viewerId: string | null): number | null {
    if (!viewerId) {
        return null;
    }
    const me = room.players.find(p => p.user_id === viewerId);
    return me ? me.slot : null;
}

export function formatDuration(seconds: number): string {
    if (!seconds || seconds < 0) {
        return "-";
    }
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;
    if (h > 0) {
        return `${h}h ${m}m`;
    }
    if (m > 0) {
        return `${m}m ${s}s`;
    }
    return `${s}s`;
}

export function gameResultLabel(
    room: GameRoom,
    viewerId: string | null,
    isSpectator: boolean,
): { text: string; tone: ResultTone } {
    if (room.status !== "finished" && room.status !== "abandoned") {
        return { text: "", tone: "neutral" };
    }
    if (room.result === "timeout") {
        return { text: "Game cancelled", tone: "draw" };
    }
    if (!room.winner_user_id) {
        return { text: "Draw", tone: "draw" };
    }
    if (isSpectator || !viewerId) {
        const winner = room.players.find(p => p.user_id === room.winner_user_id);
        return { text: `${winner?.display_name ?? "?"} won`, tone: "neutral" };
    }
    if (room.winner_user_id === viewerId) {
        return { text: "You won", tone: "win" };
    }
    return { text: "You lost", tone: "loss" };
}

interface DisconnectForfeit {
    offlinePlayer: GameRoomPlayer | undefined;
    forfeitRemaining: number | null;
    liveDurationSeconds: number;
    now: number;
}

export async function performResignWithConfirm(
    onResign: () => Promise<void>,
    setSubmitting: (v: boolean) => void,
    setError: (msg: string) => void,
): Promise<void> {
    if (!window.confirm("Resign this game?")) {
        return;
    }
    setSubmitting(true);
    setError("");
    try {
        await onResign();
    } catch (err) {
        setError(err instanceof Error ? err.message : "Resign failed");
    } finally {
        setSubmitting(false);
    }
}

export function useDisconnectForfeit(room: GameRoom): DisconnectForfeit {
    const offlinePlayer =
        room.status === "active" ? room.players.find(p => !p.connected && p.disconnected_at) : undefined;
    const now = useSecondsTick(Boolean(offlinePlayer) || room.status === "active");

    const forfeitRemaining = useMemo(() => {
        if (!offlinePlayer?.disconnected_at) {
            return null;
        }
        const startedAt = Date.parse(offlinePlayer.disconnected_at);
        if (Number.isNaN(startedAt)) {
            return null;
        }
        const elapsedSec = Math.floor((now - startedAt) / 1000);
        return Math.max(0, DISCONNECT_GRACE_SECONDS - elapsedSec);
    }, [offlinePlayer, now]);

    const liveDurationSeconds = useMemo(() => {
        const start = Date.parse(room.created_at);
        if (Number.isNaN(start)) {
            return 0;
        }
        const end = room.finished_at ? Date.parse(room.finished_at) : now;
        if (Number.isNaN(end)) {
            return 0;
        }
        return Math.max(0, Math.floor((end - start) / 1000));
    }, [room.created_at, room.finished_at, now]);

    return { offlinePlayer, forfeitRemaining, liveDurationSeconds, now };
}
