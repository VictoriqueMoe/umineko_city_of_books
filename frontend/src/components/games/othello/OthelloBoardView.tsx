import { useMemo, useState } from "react";
import type { GameRoom, OthelloState, OthelloStats, User } from "../../../types/api.ts";
import { Button } from "../../Button/Button.tsx";
import { DisconnectBanner } from "../DisconnectBanner.tsx";
import { GameOverPanel } from "../GameOverPanel.tsx";
import { GamePlayerBar } from "../GamePlayerBar.tsx";
import { GameStatsGrid } from "../GameStatsGrid.tsx";
import { gameResultLabel, getMySlot, performResignWithConfirm, useDisconnectForfeit } from "../gameRoomHelpers.ts";
import styles from "./OthelloBoardView.module.css";

const BOARD_SIZE = 8;
const SLOT_BLACK = 0;
const SLOT_WHITE = 1;

const DIRECTIONS: Array<[number, number]> = [
    [-1, -1],
    [-1, 0],
    [-1, 1],
    [0, -1],
    [0, 1],
    [1, -1],
    [1, 0],
    [1, 1],
];

interface OthelloBoardViewProps {
    room: GameRoom;
    viewer: User | null;
    isSpectator: boolean;
    onMove: (move: { square: string }) => Promise<void>;
    onResign: () => Promise<void>;
}

type CellChar = "." | "B" | "W";
type BoardGrid = CellChar[][];

interface Coord {
    row: number;
    col: number;
}

function squareFromCoord(c: Coord): string {
    return String.fromCharCode("a".charCodeAt(0) + c.col) + String(c.row + 1);
}

function parseBoard(boardStr: string): BoardGrid {
    const grid: BoardGrid = [];
    for (let r = 0; r < BOARD_SIZE; r++) {
        const row: CellChar[] = [];
        for (let c = 0; c < BOARD_SIZE; c++) {
            const ch = boardStr[r * BOARD_SIZE + c] as CellChar | undefined;
            row.push(ch === "B" || ch === "W" ? ch : ".");
        }
        grid.push(row);
    }
    return grid;
}

function initialBoard(): BoardGrid {
    const grid: BoardGrid = [];
    for (let r = 0; r < BOARD_SIZE; r++) {
        const row: CellChar[] = [];
        for (let c = 0; c < BOARD_SIZE; c++) {
            row.push(".");
        }
        grid.push(row);
    }
    grid[3][3] = "W";
    grid[3][4] = "B";
    grid[4][3] = "B";
    grid[4][4] = "W";
    return grid;
}

function inBounds(r: number, c: number): boolean {
    return r >= 0 && r < BOARD_SIZE && c >= 0 && c < BOARD_SIZE;
}

function flipsForPlacement(grid: BoardGrid, row: number, col: number, slot: number): Coord[] {
    if (grid[row][col] !== ".") {
        return [];
    }
    const own: CellChar = slot === SLOT_BLACK ? "B" : "W";
    const opp: CellChar = own === "B" ? "W" : "B";
    const flipped: Coord[] = [];
    for (const [dr, dc] of DIRECTIONS) {
        const line: Coord[] = [];
        let r = row + dr;
        let c = col + dc;
        while (inBounds(r, c) && grid[r][c] === opp) {
            line.push({ row: r, col: c });
            r += dr;
            c += dc;
        }
        if (line.length === 0 || !inBounds(r, c) || grid[r][c] !== own) {
            continue;
        }
        flipped.push(...line);
    }
    return flipped;
}

function legalSquaresFor(grid: BoardGrid, slot: number): Set<string> {
    const out = new Set<string>();
    for (let r = 0; r < BOARD_SIZE; r++) {
        for (let c = 0; c < BOARD_SIZE; c++) {
            if (grid[r][c] !== ".") {
                continue;
            }
            if (flipsForPlacement(grid, r, c, slot).length > 0) {
                out.add(`${r}-${c}`);
            }
        }
    }
    return out;
}

function countDiscsGrid(grid: BoardGrid): { black: number; white: number } {
    let black = 0;
    let white = 0;
    for (let r = 0; r < BOARD_SIZE; r++) {
        for (let c = 0; c < BOARD_SIZE; c++) {
            if (grid[r][c] === "B") {
                black++;
            } else if (grid[r][c] === "W") {
                white++;
            }
        }
    }
    return { black, white };
}

function formatReason(reason: string): string {
    switch (reason) {
        case "most_discs":
            return "with the most discs";
        case "no_moves":
            return "by leaving no legal moves";
        case "draw":
            return "as a tied disc count";
        case "resignation":
            return "by resignation";
        case "abandoned":
            return "by abandonment";
        case "timeout":
            return "due to inactivity";
        default:
            return reason ? `by ${reason.replace(/_/g, " ")}` : "";
    }
}

function isOthelloStats(x: unknown): x is OthelloStats {
    if (!x || typeof x !== "object") {
        return false;
    }
    return "black_discs" in x && "white_discs" in x;
}

export function OthelloBoardView({ room, viewer, isSpectator, onMove, onResign }: OthelloBoardViewProps) {
    const [error, setError] = useState("");
    const [submitting, setSubmitting] = useState(false);

    const state = room.state as Partial<OthelloState> | undefined;
    const stateBoard = state?.board;
    const grid = useMemo(() => {
        if (stateBoard && stateBoard.length === 64) {
            return parseBoard(stateBoard);
        }
        return initialBoard();
    }, [stateBoard]);

    const viewerId = viewer?.id ?? null;
    const mySlot = getMySlot(room, viewerId);
    const isMyTurn = !isSpectator && viewerId !== null && room.turn_user_id === viewerId && room.status === "active";

    const { offlinePlayer, forfeitRemaining, liveDurationSeconds } = useDisconnectForfeit(room);

    const legalSquares = useMemo(() => {
        if (!isMyTurn || mySlot === null) {
            return new Set<string>();
        }
        return legalSquaresFor(grid, mySlot);
    }, [grid, isMyTurn, mySlot]);

    const turnUserSlot = room.players.find(p => p.user_id === room.turn_user_id)?.slot ?? null;
    const opponentPassed =
        room.status === "active" &&
        state?.last_move !== undefined &&
        state.last_move !== null &&
        turnUserSlot !== null &&
        state.last_move.slot === turnUserSlot;

    const lastMove = state?.last_move;
    const lastMoveSquareKey = useMemo(() => {
        if (!lastMove) {
            return null;
        }
        const sq = lastMove.square;
        if (sq.length !== 2) {
            return null;
        }
        const col = sq.charCodeAt(0) - "a".charCodeAt(0);
        const row = sq.charCodeAt(1) - "1".charCodeAt(0);
        if (col < 0 || col >= BOARD_SIZE || row < 0 || row >= BOARD_SIZE) {
            return null;
        }
        return `${row}-${col}`;
    }, [lastMove]);

    const lastFlippedSet = useMemo(() => {
        const out = new Set<string>();
        if (!lastMove) {
            return out;
        }
        for (const sq of lastMove.flipped) {
            if (sq.length !== 2) {
                continue;
            }
            const col = sq.charCodeAt(0) - "a".charCodeAt(0);
            const row = sq.charCodeAt(1) - "1".charCodeAt(0);
            if (col < 0 || col >= BOARD_SIZE || row < 0 || row >= BOARD_SIZE) {
                continue;
            }
            out.add(`${row}-${col}`);
        }
        return out;
    }, [lastMove]);

    const liveDiscs = useMemo(() => countDiscsGrid(grid), [grid]);

    async function handleSquareClick(row: number, col: number) {
        if (!isMyTurn || submitting) {
            return;
        }
        if (!legalSquares.has(`${row}-${col}`)) {
            return;
        }
        setSubmitting(true);
        setError("");
        try {
            await onMove({ square: squareFromCoord({ row, col }) });
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

    const result = gameResultLabel(room, viewerId, isSpectator);
    const isOver = room.status === "finished" || room.status === "abandoned";
    const statsAvailable = isOthelloStats(room.stats);
    const showStats = statsAvailable && (isOver || (room.status === "active" && isSpectator));
    const reasonText = statsAvailable && room.stats ? formatReason((room.stats as OthelloStats).result_reason) : "";

    const displayRows: number[] = [];
    for (let r = BOARD_SIZE - 1; r >= 0; r--) {
        displayRows.push(r);
    }
    const displayCols: number[] = [];
    for (let c = 0; c < BOARD_SIZE; c++) {
        displayCols.push(c);
    }

    function cellContent(cell: CellChar) {
        if (cell === ".") {
            return null;
        }
        const pieceClass = cell === "B" ? styles.pieceBlack : styles.pieceWhite;
        return <span className={`${styles.piece} ${pieceClass}`} />;
    }

    return (
        <div className={styles.wrapper}>
            <GamePlayerBar
                room={room}
                slot0Label="Black"
                slot1Label="White"
                liveDurationSeconds={liveDurationSeconds}
            />

            <DisconnectBanner offlinePlayer={offlinePlayer} forfeitRemaining={forfeitRemaining} />

            {error && <div className={styles.error}>{error}</div>}
            {opponentPassed && !error && (
                <div className={styles.info}>Opponent had no legal moves and passed. Your turn again.</div>
            )}
            {!opponentPassed && isMyTurn && legalSquares.size === 0 && room.status === "active" && (
                <div className={styles.info}>
                    You have no legal moves; the server will pass for you on the next sync.
                </div>
            )}

            <div className={styles.boardContainer}>
                <div className={styles.board}>
                    {displayRows.map(r => (
                        <div className={styles.boardRow} key={r}>
                            {displayCols.map(c => {
                                const key = `${r}-${c}`;
                                const cell = grid[r][c];
                                const isLegal = legalSquares.has(key);
                                const isLastMove = lastMoveSquareKey === key;
                                const isFlipped = lastFlippedSet.has(key);
                                return (
                                    <button
                                        type="button"
                                        key={key}
                                        className={[
                                            styles.square,
                                            isLegal ? styles.squareLegal : "",
                                            isLastMove ? styles.squareLastMove : "",
                                            isFlipped ? styles.squareFlipped : "",
                                        ]
                                            .filter(Boolean)
                                            .join(" ")}
                                        onClick={() => handleSquareClick(r, c)}
                                        disabled={!isMyTurn || !isLegal || submitting}
                                        aria-label={squareFromCoord({ row: r, col: c })}
                                    >
                                        {cellContent(cell)}
                                    </button>
                                );
                            })}
                        </div>
                    ))}
                </div>
            </div>

            <div className={styles.discTally}>
                <span>
                    <span className={`${styles.tallyDot} ${styles.pieceBlack}`} /> Black: {liveDiscs.black}
                </span>
                <span>
                    <span className={`${styles.tallyDot} ${styles.pieceWhite}`} /> White: {liveDiscs.white}
                </span>
            </div>

            <GameOverPanel
                isOver={isOver}
                showChildren={showStats}
                resultText={result.text}
                resultTone={result.tone}
                reasonText={reasonText}
            >
                {showStats && statsAvailable && (
                    <GameStatsGrid
                        slot0Name={room.players.find(p => p.slot === SLOT_BLACK)?.display_name ?? "Black"}
                        slot1Name={room.players.find(p => p.slot === SLOT_WHITE)?.display_name ?? "White"}
                        isOver={isOver}
                        rows={[
                            {
                                slot0: (room.stats as OthelloStats).black_discs,
                                label: "Discs",
                                slot1: (room.stats as OthelloStats).white_discs,
                            },
                            {
                                slot0: (room.stats as OthelloStats).black_moves,
                                label: "Moves",
                                slot1: (room.stats as OthelloStats).white_moves,
                            },
                            {
                                slot0: (room.stats as OthelloStats).black_flips,
                                label: "Flips",
                                slot1: (room.stats as OthelloStats).white_flips,
                            },
                            {
                                slot0: (room.stats as OthelloStats).black_corners,
                                label: "Corners",
                                slot1: (room.stats as OthelloStats).white_corners,
                            },
                            {
                                slot0: (room.stats as OthelloStats).black_passes,
                                label: "Passes",
                                slot1: (room.stats as OthelloStats).white_passes,
                            },
                        ]}
                        totalLabel="Total moves"
                        totalValue={(room.stats as OthelloStats).total_moves}
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
