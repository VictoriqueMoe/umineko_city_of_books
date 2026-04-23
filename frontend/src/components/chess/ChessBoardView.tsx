import { useEffect, useMemo, useState } from "react";
import { Chess } from "chess.js";
import { Chessboard } from "react-chessboard";
import type { ChessState, ChessStats, GameRoom, User } from "../../types/api";
import { Button } from "../Button/Button";
import styles from "./ChessBoardView.module.css";

const DISCONNECT_GRACE_SECONDS = 60;

interface ChessBoardViewProps {
    room: GameRoom;
    viewer: User | null;
    isSpectator: boolean;
    onMove: (move: { from: string; to: string; promotion?: string }) => Promise<void>;
    onResign: () => Promise<void>;
}

function useSecondsTick(active: boolean): number {
    const [now, setNow] = useState(() => Date.now());
    useEffect(() => {
        if (!active) {
            return;
        }
        const id = window.setInterval(() => setNow(Date.now()), 1000);
        return () => window.clearInterval(id);
    }, [active]);
    return now;
}

function getMySlot(room: GameRoom, viewerId: string | null): number | null {
    if (!viewerId) {
        return null;
    }
    const me = room.players.find(p => p.user_id === viewerId);
    return me ? me.slot : null;
}

function formatReason(reason: string): string {
    switch (reason) {
        case "checkmate":
            return "by checkmate";
        case "resignation":
            return "by resignation";
        case "abandoned":
            return "by abandonment";
        case "stalemate":
            return "by stalemate";
        case "insufficient_material":
            return "by insufficient material";
        case "fifty_move_rule":
            return "by fifty-move rule";
        case "repetition":
            return "by threefold repetition";
        case "draw_agreed":
            return "by agreement";
        case "win":
            return "";
        case "draw":
            return "";
        default:
            return reason ? `by ${reason.replace(/_/g, " ")}` : "";
    }
}

function formatDuration(seconds: number): string {
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

function isChessStats(x: unknown): x is ChessStats {
    if (!x || typeof x !== "object") {
        return false;
    }
    return "total_ply" in x && "white_moves" in x;
}

function resultLabel(
    room: GameRoom,
    viewerId: string | null,
    isSpectator: boolean,
): { text: string; tone: "win" | "loss" | "draw" | "neutral" } {
    if (room.status !== "finished") {
        return { text: "", tone: "neutral" };
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

export function ChessBoardView({ room, viewer, isSpectator, onMove, onResign }: ChessBoardViewProps) {
    const [error, setError] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const state = room.state as ChessState;

    const viewerId = viewer?.id ?? null;
    const mySlot = getMySlot(room, viewerId);
    const orientation: "white" | "black" = mySlot === 1 ? "black" : "white";
    const isMyTurn = !isSpectator && viewerId !== null && room.turn_user_id === viewerId && room.status === "active";

    const offlinePlayer =
        room.status === "active" ? room.players.find(p => !p.connected && p.disconnected_at) : undefined;
    const now = useSecondsTick(Boolean(offlinePlayer));
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

    const game = useMemo(() => {
        const g = new Chess();
        if (state?.fen) {
            try {
                g.load(state.fen);
            } catch {
                // stale state; fall back to initial
            }
        }
        return g;
    }, [state?.fen]);

    async function handleDrop({
        sourceSquare,
        targetSquare,
    }: {
        sourceSquare: string;
        targetSquare: string | null;
    }): Promise<boolean> {
        if (!targetSquare || submitting) {
            return false;
        }
        if (!isMyTurn) {
            return false;
        }

        const moves = game.moves({ square: sourceSquare as never, verbose: true }) as Array<{
            from: string;
            to: string;
            promotion?: string;
        }>;
        const candidate = moves.find(m => m.to === targetSquare);
        if (!candidate) {
            return false;
        }

        setError("");
        setSubmitting(true);
        try {
            await onMove({ from: candidate.from, to: candidate.to, promotion: candidate.promotion });
            return true;
        } catch (err) {
            setError(err instanceof Error ? err.message : "Move failed");
            return false;
        } finally {
            setSubmitting(false);
        }
    }

    async function handleResign() {
        if (submitting) {
            return;
        }
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

    const white = room.players.find(p => p.slot === 0);
    const black = room.players.find(p => p.slot === 1);
    const result = resultLabel(room, viewerId, isSpectator);

    return (
        <div className={styles.wrapper}>
            <div className={styles.status}>
                <div className={styles.statusLeft}>
                    <span className={`${styles.playerDot} ${black?.connected ? styles.playerDotOn : ""}`} />
                    <span className={styles.playerName}>{black?.display_name ?? "Black"}</span>
                    <span
                        className={`${styles.turnMarker} ${
                            isMyTurn && mySlot === 1
                                ? styles.turnMarkerActive
                                : room.turn_user_id === black?.user_id && room.status === "active"
                                  ? styles.turnMarkerActive
                                  : ""
                        }`}
                    >
                        {room.turn_user_id === black?.user_id && room.status === "active" ? "to move" : ""}
                    </span>
                </div>
                <div className={styles.statusCenter}>
                    <span className={styles.watcherCount} title="Spectators watching">
                        👁 {room.watcher_count}
                    </span>
                </div>
                <div className={styles.statusRight}>
                    <span
                        className={`${styles.turnMarker} ${
                            room.turn_user_id === white?.user_id && room.status === "active"
                                ? styles.turnMarkerActive
                                : ""
                        }`}
                    >
                        {room.turn_user_id === white?.user_id && room.status === "active" ? "to move" : ""}
                    </span>
                    <span className={styles.playerName}>{white?.display_name ?? "White"}</span>
                    <span className={`${styles.playerDot} ${white?.connected ? styles.playerDotOn : ""}`} />
                </div>
            </div>

            {offlinePlayer && forfeitRemaining !== null && (
                <div className={styles.disconnectBanner}>
                    {offlinePlayer.display_name} disconnected — forfeits in {forfeitRemaining}s
                </div>
            )}

            {error && <div className={styles.error}>{error}</div>}

            <div className={styles.boardContainer}>
                <Chessboard
                    options={{
                        position: state?.fen || "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
                        boardOrientation: orientation,
                        allowDragging: isMyTurn,
                        onPieceDrop: args => {
                            void handleDrop(args);
                            return false;
                        },
                    }}
                />
            </div>

            {room.status === "finished" && (
                <div className={styles.gameOver}>
                    <div className={styles.result}>
                        <span
                            className={
                                result.tone === "win"
                                    ? styles.resultWin
                                    : result.tone === "loss"
                                      ? styles.resultLoss
                                      : styles.resultDraw
                            }
                        >
                            {result.text}
                        </span>
                        {isChessStats(room.stats) && room.stats.result_reason && (
                            <span className={styles.resultReason}> {formatReason(room.stats.result_reason)}</span>
                        )}
                    </div>
                    {isChessStats(room.stats) && (
                        <div className={styles.statsGrid}>
                            <div className={styles.statsHeader}>
                                <span>{white?.display_name ?? "White"}</span>
                                <span />
                                <span>{black?.display_name ?? "Black"}</span>
                            </div>
                            <div className={styles.statsRow}>
                                <span>{room.stats.white_moves}</span>
                                <span className={styles.statsLabel}>Moves</span>
                                <span>{room.stats.black_moves}</span>
                            </div>
                            <div className={styles.statsRow}>
                                <span>{room.stats.white_captures}</span>
                                <span className={styles.statsLabel}>Captures</span>
                                <span>{room.stats.black_captures}</span>
                            </div>
                            <div className={styles.statsRow}>
                                <span>{room.stats.white_checks}</span>
                                <span className={styles.statsLabel}>Checks given</span>
                                <span>{room.stats.black_checks}</span>
                            </div>
                            <div className={styles.statsFooter}>
                                <span>Total ply: {room.stats.total_ply}</span>
                                <span>Duration: {formatDuration(room.stats.duration_seconds)}</span>
                            </div>
                        </div>
                    )}
                </div>
            )}

            {room.status === "active" && !isSpectator && (
                <div className={styles.controls}>
                    <Button variant="danger" onClick={handleResign} disabled={submitting}>
                        Resign
                    </Button>
                </div>
            )}
        </div>
    );
}
