import { useMemo, useState } from "react";
import { Chess, Square } from "chess.js";
import { Chessboard } from "react-chessboard";
import type { ChessState, ChessStats, GameRoom, User } from "../../../types/api.ts";
import { Button } from "../../Button/Button.tsx";
import { Input } from "../../Input/Input.tsx";
import { DisconnectBanner } from "../DisconnectBanner.tsx";
import { GameOverPanel } from "../GameOverPanel.tsx";
import { GamePlayerBar } from "../GamePlayerBar.tsx";
import { GameStatsGrid } from "../GameStatsGrid.tsx";
import { gameResultLabel, getMySlot, performResignWithConfirm, useDisconnectForfeit } from "../gameRoomHelpers.ts";
import styles from "./ChessBoardView.module.css";

const UCI_RE = /^([a-h][1-8])([a-h][1-8])([qrbn])?$/;

type PromotionPiece = "q" | "r" | "b" | "n";

type ParsedUci = {
    from: Square;
    to: Square;
    promotion?: PromotionPiece;
};

function parseUci(input: string): ParsedUci | null {
    const m = input.trim().toLowerCase().match(UCI_RE);

    if (!m) {
        return null;
    }

    return {
        from: m[1] as Square,
        to: m[2] as Square,
        promotion: m[3] as PromotionPiece | undefined,
    };
}

interface ChessBoardViewProps {
    room: GameRoom;
    viewer: User | null;
    isSpectator: boolean;
    onMove: (move: { from: string; to: string; promotion?: string }) => Promise<void>;
    onResign: () => Promise<void>;
    onOfferDraw: () => Promise<void>;
    onAcceptDraw: () => Promise<void>;
    onDeclineDraw: () => Promise<void>;
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

export function ChessBoardView({
    room,
    viewer,
    isSpectator,
    onMove,
    onResign,
    onOfferDraw,
    onAcceptDraw,
    onDeclineDraw,
}: ChessBoardViewProps) {
    const [error, setError] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [coordInput, setCoordInput] = useState("");
    const state = room.state as ChessState;

    const viewerId = viewer?.id ?? null;
    const mySlot = getMySlot(room, viewerId);
    const orientation: "white" | "black" = mySlot === 1 ? "black" : "white";
    const isMyTurn = !isSpectator && viewerId !== null && room.turn_user_id === viewerId && room.status === "active";

    const { offlinePlayer, forfeitRemaining, liveDurationSeconds } = useDisconnectForfeit(room);

    const stateFen = state?.fen;
    const statePgn = state?.pgn;

    const game = useMemo(() => {
        const g = new Chess();
        if (stateFen) {
            try {
                g.load(stateFen);
            } catch {
                // stale state; fall back to initial
            }
        }
        return g;
    }, [stateFen]);

    const [hoveredSquare, setHoveredSquare] = useState<string | null>(null);

    const lastMove = useMemo(() => {
        if (!statePgn) {
            return null;
        }
        try {
            const replay = new Chess();
            replay.loadPgn(statePgn);
            const history = replay.history({ verbose: true }) as Array<{ from: string; to: string }>;
            return history.length > 0 ? history[history.length - 1] : null;
        } catch {
            return null;
        }
    }, [statePgn]);

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
        sourceSquare: Square;
        targetSquare: string | null;
    }): Promise<boolean> {
        if (!targetSquare || submitting) {
            return false;
        }
        if (!isMyTurn) {
            return false;
        }

        const moves = game.moves({ square: sourceSquare, verbose: true }) as Array<{
            from: string;
            to: string;
            promotion?: string;
        }>;
        const matches = moves.filter(m => m.to === targetSquare);
        if (matches.length === 0) {
            return false;
        }
        const candidate = matches.find(m => m.promotion === "q") ?? matches[0];

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

    async function handleCoordSubmit() {
        if (submitting || !isMyTurn) {
            return;
        }
        const parsed = parseUci(coordInput);
        if (!parsed) {
            setError("Invalid coordinate. Use e.g. e2e4 or e7e8q.");
            return;
        }
        const legal = game.moves({ square: parsed.from, verbose: true }) as Array<{
            from: string;
            to: string;
            promotion?: string;
        }>;
        const candidates = legal.filter(m => m.to === parsed.to);
        if (candidates.length === 0) {
            setError(`Illegal move: ${parsed.from}${parsed.to}`);
            return;
        }
        let chosen = candidates[0];
        if (candidates.some(m => m.promotion)) {
            if (!parsed.promotion) {
                setError("Promotion required: append q, r, b, or n.");
                return;
            }
            const match = candidates.find(m => m.promotion === parsed.promotion);
            if (!match) {
                setError(`No promotion to ${parsed.promotion} available.`);
                return;
            }
            chosen = match;
        } else if (parsed.promotion) {
            setError("That move is not a promotion.");
            return;
        }
        setError("");
        setSubmitting(true);
        try {
            await onMove({ from: chosen.from, to: chosen.to, promotion: chosen.promotion });
            setCoordInput("");
        } catch (err) {
            setError(err instanceof Error ? err.message : "Move failed");
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

    async function handleDrawAction(fn: () => Promise<void>) {
        if (submitting) {
            return;
        }
        setError("");
        setSubmitting(true);
        try {
            await fn();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Action failed");
        } finally {
            setSubmitting(false);
        }
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
                        onPieceDrop: ({ sourceSquare, targetSquare }) => {
                            void handleDrop({ sourceSquare: sourceSquare as Square, targetSquare });
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

            {room.status === "active" && !isSpectator && room.draw_offer_from_user_id && (
                <>
                    {room.draw_offer_from_user_id !== viewerId ? (
                        <div className={styles.drawBanner}>
                            <span className={styles.drawBannerText}>
                                Your opponent has offered a draw. Accept to end the game as a draw, or decline to keep
                                playing.
                            </span>
                            <div className={styles.drawBannerActions}>
                                <Button
                                    variant="primary"
                                    size="small"
                                    onClick={() => handleDrawAction(onAcceptDraw)}
                                    disabled={submitting}
                                >
                                    Accept draw
                                </Button>
                                <Button
                                    variant="ghost"
                                    size="small"
                                    onClick={() => handleDrawAction(onDeclineDraw)}
                                    disabled={submitting}
                                >
                                    Decline
                                </Button>
                            </div>
                        </div>
                    ) : (
                        <div className={styles.drawPending}>
                            Draw offered. Waiting for your opponent to respond. It is withdrawn if you make a move.
                        </div>
                    )}
                </>
            )}

            {room.status === "active" && !isSpectator && (
                <>
                    <form
                        className={styles.coordForm}
                        onSubmit={e => {
                            e.preventDefault();
                            void handleCoordSubmit();
                        }}
                    >
                        <Input
                            type="text"
                            value={coordInput}
                            onChange={e => setCoordInput(e.target.value)}
                            placeholder="Type a move (e.g. e2e4, e1g1, e7e8q)"
                            autoComplete="off"
                            spellCheck={false}
                            disabled={!isMyTurn || submitting}
                            maxLength={5}
                            fullWidth
                        />
                        <Button
                            type="submit"
                            variant="primary"
                            disabled={!isMyTurn || submitting || coordInput.trim() === ""}
                        >
                            Play
                        </Button>
                    </form>
                    <div className={styles.controls}>
                        {!room.draw_offer_from_user_id && (
                            <Button variant="ghost" onClick={() => handleDrawAction(onOfferDraw)} disabled={submitting}>
                                Offer draw
                            </Button>
                        )}
                        <Button variant="danger" onClick={handleResign} disabled={submitting}>
                            Resign
                        </Button>
                    </div>
                </>
            )}
        </div>
    );
}
