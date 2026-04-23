import { useMemo, useState } from "react";
import { Chess } from "chess.js";
import { Chessboard } from "react-chessboard";
import type { ChessState, ChessStats, GameRoom, User } from "../../../types/api.ts";
import { Button } from "../../Button/Button.tsx";
import { DisconnectBanner } from "../DisconnectBanner.tsx";
import { GameOverPanel } from "../GameOverPanel.tsx";
import { GamePlayerBar } from "../GamePlayerBar.tsx";
import { GameStatsGrid } from "../GameStatsGrid.tsx";
import { gameResultLabel, getMySlot, performResignWithConfirm, useDisconnectForfeit } from "../gameRoomHelpers.ts";
import styles from "./ChessBoardView.module.css";

interface ChessBoardViewProps {
    room: GameRoom;
    viewer: User | null;
    isSpectator: boolean;
    onMove: (move: { from: string; to: string; promotion?: string }) => Promise<void>;
    onResign: () => Promise<void>;
}

function formatReason(reason: string): string {
    switch (reason) {
        case "checkmate":
            return "by checkmate";
        case "resignation":
            return "by resignation";
        case "abandoned":
            return "by abandonment";
        case "timeout":
            return "due to inactivity";
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

function isChessStats(x: unknown): x is ChessStats {
    if (!x || typeof x !== "object") {
        return false;
    }
    return "total_ply" in x && "white_moves" in x;
}

export function ChessBoardView({ room, viewer, isSpectator, onMove, onResign }: ChessBoardViewProps) {
    const [error, setError] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const state = room.state as ChessState;

    const viewerId = viewer?.id ?? null;
    const mySlot = getMySlot(room, viewerId);
    const orientation: "white" | "black" = mySlot === 1 ? "black" : "white";
    const isMyTurn = !isSpectator && viewerId !== null && room.turn_user_id === viewerId && room.status === "active";

    const { offlinePlayer, forfeitRemaining, liveDurationSeconds } = useDisconnectForfeit(room);

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

    const [hoveredSquare, setHoveredSquare] = useState<string | null>(null);

    const lastMove = useMemo(() => {
        if (!state?.pgn) {
            return null;
        }
        try {
            const replay = new Chess();
            replay.loadPgn(state.pgn);
            const history = replay.history({ verbose: true }) as Array<{ from: string; to: string }>;
            return history.length > 0 ? history[history.length - 1] : null;
        } catch {
            return null;
        }
    }, [state?.pgn]);

    const checkSquare = useMemo(() => {
        if (!game.isCheck()) {
            return null;
        }
        const board = game.board();
        const files = ["a", "b", "c", "d", "e", "f", "g", "h"];
        for (let rank = 0; rank < 8; rank++) {
            for (let file = 0; file < 8; file++) {
                const piece = board[rank][file];
                if (piece && piece.type === "k" && piece.color === game.turn()) {
                    return files[file] + (8 - rank);
                }
            }
        }
        return null;
    }, [game]);

    const squareStyles = useMemo((): Record<string, React.CSSProperties> => {
        const styles: Record<string, React.CSSProperties> = {};
        if (lastMove) {
            styles[lastMove.from] = { boxShadow: "inset 0 0 0 3px rgba(255, 210, 80, 0.55)" };
            styles[lastMove.to] = { boxShadow: "inset 0 0 0 3px rgba(255, 210, 80, 0.55)" };
        }
        if (checkSquare) {
            styles[checkSquare] = {
                background:
                    "radial-gradient(circle, rgba(255, 0, 0, 0.95) 0%, rgba(230, 50, 50, 0.85) 55%, rgba(180, 20, 20, 0.7) 100%)",
                boxShadow: "inset 0 0 0 4px rgba(255, 0, 0, 0.95)",
            };
        }
        if (hoveredSquare) {
            try {
                const moves = game.moves({ square: hoveredSquare as never, verbose: true }) as Array<{
                    to: string;
                    captured?: string;
                }>;
                for (const m of moves) {
                    if (m.captured) {
                        styles[m.to] = {
                            ...styles[m.to],
                            boxShadow: "inset 0 0 0 4px rgba(80, 80, 80, 0.45)",
                        };
                    } else {
                        styles[m.to] = {
                            ...styles[m.to],
                            background: "radial-gradient(circle, rgba(80, 80, 80, 0.4) 20%, transparent 22%)",
                        };
                    }
                }
            } catch {
                // invalid square, ignore
            }
        }
        return styles;
    }, [lastMove, checkSquare, hoveredSquare, game]);

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
        await performResignWithConfirm(onResign, setSubmitting, setError);
    }

    const result = gameResultLabel(room, viewerId, isSpectator);
    const isOver = room.status === "finished" || room.status === "abandoned";
    const statsAvailable = isChessStats(room.stats);
    const showStats = statsAvailable && (isOver || (room.status === "active" && isSpectator));
    const reasonText = statsAvailable && room.stats ? formatReason((room.stats as ChessStats).result_reason) : "";

    return (
        <div className={styles.wrapper}>
            <GamePlayerBar
                room={room}
                slot0Label="White"
                slot1Label="Black"
                liveDurationSeconds={liveDurationSeconds}
            />

            <DisconnectBanner offlinePlayer={offlinePlayer} forfeitRemaining={forfeitRemaining} />

            {error && <div className={styles.error}>{error}</div>}

            <div className={styles.boardContainer}>
                <Chessboard
                    options={{
                        position: state?.fen || "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
                        boardOrientation: orientation,
                        allowDragging: isMyTurn,
                        squareStyles,
                        onMouseOverSquare: ({ square, piece }) => {
                            if (piece) {
                                setHoveredSquare(square);
                            }
                        },
                        onMouseOutSquare: () => {
                            setHoveredSquare(null);
                        },
                        onPieceDrop: args => {
                            void handleDrop(args);
                            setHoveredSquare(null);
                            return false;
                        },
                    }}
                />
            </div>

            {checkSquare && (
                <div className={styles.checkBanner}>
                    <strong>Check!</strong> The highlighted king is under attack and must be moved out of check, or
                    blocked, or the attacking piece captured.
                </div>
            )}

            <GameOverPanel
                isOver={isOver}
                showChildren={showStats}
                resultText={result.text}
                resultTone={result.tone}
                reasonText={reasonText}
            >
                {showStats && statsAvailable && (
                    <GameStatsGrid
                        slot0Name={room.players.find(p => p.slot === 0)?.display_name ?? "White"}
                        slot1Name={room.players.find(p => p.slot === 1)?.display_name ?? "Black"}
                        isOver={isOver}
                        rows={[
                            {
                                slot0: (room.stats as ChessStats).white_moves,
                                label: "Moves",
                                slot1: (room.stats as ChessStats).black_moves,
                            },
                            {
                                slot0: (room.stats as ChessStats).white_captures,
                                label: "Captures",
                                slot1: (room.stats as ChessStats).black_captures,
                            },
                            {
                                slot0: (room.stats as ChessStats).white_checks,
                                label: "Checks given",
                                slot1: (room.stats as ChessStats).black_checks,
                            },
                        ]}
                        totalLabel="Total ply"
                        totalValue={(room.stats as ChessStats).total_ply}
                        durationSeconds={liveDurationSeconds}
                    />
                )}
            </GameOverPanel>

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
