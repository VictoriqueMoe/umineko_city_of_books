import { useEffect, useMemo, useState } from "react";
import type { CheckersState, CheckersStats, GameRoom, User } from "../../../types/api.ts";
import { Button } from "../../Button/Button.tsx";
import { DisconnectBanner } from "../DisconnectBanner.tsx";
import { GameOverPanel } from "../GameOverPanel.tsx";
import { GamePlayerBar } from "../GamePlayerBar.tsx";
import { GameStatsGrid } from "../GameStatsGrid.tsx";
import { gameResultLabel, getMySlot, performResignWithConfirm, useDisconnectForfeit } from "../gameRoomHelpers.ts";
import styles from "./CheckersBoardView.module.css";

const BOARD_SIZE = 8;
const SLOT_RED = 0;
const SLOT_BLACK = 1;

interface CheckersBoardViewProps {
    room: GameRoom;
    viewer: User | null;
    isSpectator: boolean;
    onMove: (move: { from: string; path: string[] }) => Promise<void>;
    onResign: () => Promise<void>;
}

type CellChar = "." | "r" | "R" | "b" | "B";

type BoardGrid = CellChar[][];

interface Coord {
    row: number;
    col: number;
}

interface JumpInfo {
    land: Coord;
    captured: Coord;
}

function squareFromCoord(c: Coord): string {
    return String.fromCharCode("a".charCodeAt(0) + c.col) + String(c.row + 1);
}

function isDarkSquare(row: number, col: number): boolean {
    return (row + col) % 2 === 0;
}

function parseBoard(boardStr: string): BoardGrid {
    const grid: BoardGrid = [];
    for (let r = 0; r < BOARD_SIZE; r++) {
        const row: CellChar[] = [];
        for (let c = 0; c < BOARD_SIZE; c++) {
            const ch = boardStr[r * BOARD_SIZE + c] as CellChar | undefined;
            row.push(ch && "rRbB.".includes(ch) ? ch : ".");
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
            if (!isDarkSquare(r, c)) {
                row.push(".");
                continue;
            }
            if (r < 3) {
                row.push("r");
            } else if (r >= 5) {
                row.push("b");
            } else {
                row.push(".");
            }
        }
        grid.push(row);
    }
    return grid;
}

function ownerOf(cell: CellChar): number | null {
    if (cell === "r" || cell === "R") {
        return SLOT_RED;
    }
    if (cell === "b" || cell === "B") {
        return SLOT_BLACK;
    }
    return null;
}

function isKing(cell: CellChar): boolean {
    return cell === "R" || cell === "B";
}

function moveDirsFor(cell: CellChar): Array<[number, number]> {
    if (isKing(cell)) {
        return [
            [-1, -1],
            [-1, 1],
            [1, -1],
            [1, 1],
        ];
    }
    if (cell === "r") {
        return [
            [1, -1],
            [1, 1],
        ];
    }
    if (cell === "b") {
        return [
            [-1, -1],
            [-1, 1],
        ];
    }
    return [];
}

function inBounds(r: number, c: number): boolean {
    return r >= 0 && r < BOARD_SIZE && c >= 0 && c < BOARD_SIZE;
}

function pieceJumps(grid: BoardGrid, r: number, c: number): JumpInfo[] {
    const cell = grid[r][c];
    const owner = ownerOf(cell);
    if (owner === null) {
        return [];
    }
    const out: JumpInfo[] = [];
    for (const [dr, dc] of moveDirsFor(cell)) {
        const midR = r + dr;
        const midC = c + dc;
        const landR = r + 2 * dr;
        const landC = c + 2 * dc;
        if (!inBounds(landR, landC)) {
            continue;
        }
        const mid = grid[midR][midC];
        if (mid === "." || ownerOf(mid) === owner) {
            continue;
        }
        if (grid[landR][landC] !== ".") {
            continue;
        }
        out.push({ land: { row: landR, col: landC }, captured: { row: midR, col: midC } });
    }
    return out;
}

function pieceSimpleMoves(grid: BoardGrid, r: number, c: number): Coord[] {
    const cell = grid[r][c];
    if (ownerOf(cell) === null) {
        return [];
    }
    const out: Coord[] = [];
    for (const [dr, dc] of moveDirsFor(cell)) {
        const nr = r + dr;
        const nc = c + dc;
        if (!inBounds(nr, nc)) {
            continue;
        }
        if (grid[nr][nc] === ".") {
            out.push({ row: nr, col: nc });
        }
    }
    return out;
}

function playerHasCapture(grid: BoardGrid, slot: number): boolean {
    for (let r = 0; r < BOARD_SIZE; r++) {
        for (let c = 0; c < BOARD_SIZE; c++) {
            if (ownerOf(grid[r][c]) !== slot) {
                continue;
            }
            if (pieceJumps(grid, r, c).length > 0) {
                return true;
            }
        }
    }
    return false;
}

function cloneGrid(grid: BoardGrid): BoardGrid {
    return grid.map(row => row.slice());
}

function applyJumpLocal(grid: BoardGrid, from: Coord, jump: JumpInfo): { grid: BoardGrid; crowned: boolean } {
    const next = cloneGrid(grid);
    const piece = next[from.row][from.col];
    next[from.row][from.col] = ".";
    next[jump.captured.row][jump.captured.col] = ".";
    let landing: CellChar = piece;
    let crowned = false;
    if (piece === "r" && jump.land.row === BOARD_SIZE - 1) {
        landing = "R";
        crowned = true;
    } else if (piece === "b" && jump.land.row === 0) {
        landing = "B";
        crowned = true;
    }
    next[jump.land.row][jump.land.col] = landing;
    return { grid: next, crowned };
}

function formatReason(reason: string): string {
    switch (reason) {
        case "no_pieces":
            return "by capturing all pieces";
        case "no_moves":
            return "by blocking all moves";
        case "resignation":
            return "by resignation";
        case "abandoned":
            return "by abandonment";
        case "timeout":
            return "due to inactivity";
        case "forty_move_rule":
            return "by forty-move rule";
        default:
            return reason ? `by ${reason.replace(/_/g, " ")}` : "";
    }
}

function isCheckersStats(x: unknown): x is CheckersStats {
    if (!x || typeof x !== "object") {
        return false;
    }
    return "total_moves" in x && "red_captures" in x;
}

export function CheckersBoardView({ room, viewer, isSpectator, onMove, onResign }: CheckersBoardViewProps) {
    const [error, setError] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [selected, setSelected] = useState<Coord | null>(null);
    const [jumpPath, setJumpPath] = useState<Coord[]>([]);
    const [jumpOriginPiece, setJumpOriginPiece] = useState<CellChar | null>(null);

    const state = room.state as Partial<CheckersState> | undefined;
    const persistedGrid = useMemo(() => {
        if (state?.board && state.board.length === 64) {
            return parseBoard(state.board);
        }
        return initialBoard();
    }, [state?.board]);

    const viewerId = viewer?.id ?? null;
    const mySlot = getMySlot(room, viewerId);
    const orientation: "red" | "black" = mySlot === SLOT_BLACK ? "black" : "red";
    const isMyTurn = !isSpectator && viewerId !== null && room.turn_user_id === viewerId && room.status === "active";

    const { offlinePlayer, forfeitRemaining, liveDurationSeconds } = useDisconnectForfeit(room);

    useEffect(() => {
        setSelected(null);
        setJumpPath([]);
        setJumpOriginPiece(null);
        setError("");
    }, [room.id, room.turn_user_id, room.status, state?.board]);

    const inProgressJump = jumpPath.length > 0 && selected !== null;
    const displayGrid = useMemo(() => {
        if (!inProgressJump || !selected || jumpOriginPiece === null) {
            return persistedGrid;
        }
        const g = cloneGrid(persistedGrid);
        let curPiece = jumpOriginPiece;
        let curR = selected.row;
        let curC = selected.col;
        g[curR][curC] = ".";
        for (const step of jumpPath) {
            const midR = curR + (step.row - curR) / 2;
            const midC = curC + (step.col - curC) / 2;
            g[midR][midC] = ".";
            let landing: CellChar = curPiece;
            if (curPiece === "r" && step.row === BOARD_SIZE - 1) {
                landing = "R";
            } else if (curPiece === "b" && step.row === 0) {
                landing = "B";
            }
            g[step.row][step.col] = landing;
            curPiece = landing;
            curR = step.row;
            curC = step.col;
        }
        return g;
    }, [persistedGrid, inProgressJump, selected, jumpOriginPiece, jumpPath]);

    const activeCoord: Coord | null = useMemo(() => {
        if (selected && jumpPath.length > 0) {
            return jumpPath[jumpPath.length - 1];
        }
        return selected;
    }, [selected, jumpPath]);

    const availableTargets = useMemo((): {
        jumps: JumpInfo[];
        simples: Coord[];
        mustJump: boolean;
    } => {
        if (!isMyTurn || !activeCoord) {
            return { jumps: [], simples: [], mustJump: false };
        }
        const mustJumpAnywhere = playerHasCapture(displayGrid, mySlot ?? 0);
        const jumps = pieceJumps(displayGrid, activeCoord.row, activeCoord.col);
        if (inProgressJump) {
            return { jumps, simples: [], mustJump: true };
        }
        const simples = mustJumpAnywhere ? [] : pieceSimpleMoves(displayGrid, activeCoord.row, activeCoord.col);
        const pieceCanJump = jumps.length > 0;
        const effectiveJumps = mustJumpAnywhere && !pieceCanJump ? [] : jumps;
        return { jumps: effectiveJumps, simples, mustJump: mustJumpAnywhere };
    }, [isMyTurn, activeCoord, displayGrid, mySlot, inProgressJump]);

    async function submitMove(from: Coord, path: Coord[]) {
        const fromSq = squareFromCoord(from);
        const pathSq = path.map(squareFromCoord);
        setSubmitting(true);
        setError("");
        try {
            await onMove({ from: fromSq, path: pathSq });
            setSelected(null);
            setJumpPath([]);
            setJumpOriginPiece(null);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Move failed");
            setSelected(null);
            setJumpPath([]);
            setJumpOriginPiece(null);
        } finally {
            setSubmitting(false);
        }
    }

    function handleSquareClick(row: number, col: number) {
        if (!isMyTurn || submitting) {
            return;
        }
        if (!isDarkSquare(row, col)) {
            return;
        }

        const cell = displayGrid[row][col];
        const owner = ownerOf(cell);

        if (inProgressJump) {
            const jumpHit = availableTargets.jumps.find(j => j.land.row === row && j.land.col === col);
            if (!jumpHit) {
                return;
            }
            const nextPath = [...jumpPath, { row, col }];
            const afterGrid = (() => {
                const g = cloneGrid(persistedGrid);
                let piece = jumpOriginPiece ?? persistedGrid[selected!.row][selected!.col];
                let r = selected!.row;
                let c = selected!.col;
                g[r][c] = ".";
                for (const step of nextPath) {
                    const midR = r + (step.row - r) / 2;
                    const midC = c + (step.col - c) / 2;
                    g[midR][midC] = ".";
                    let landing: CellChar = piece;
                    let crowned = false;
                    if (piece === "r" && step.row === BOARD_SIZE - 1) {
                        landing = "R";
                        crowned = true;
                    } else if (piece === "b" && step.row === 0) {
                        landing = "B";
                        crowned = true;
                    }
                    g[step.row][step.col] = landing;
                    piece = landing;
                    r = step.row;
                    c = step.col;
                    if (crowned) {
                        return { g, stop: true, endR: r, endC: c, endPiece: piece };
                    }
                }
                return { g, stop: false, endR: r, endC: c, endPiece: piece };
            })();

            const hasMore = !afterGrid.stop && pieceJumps(afterGrid.g, afterGrid.endR, afterGrid.endC).length > 0;
            if (hasMore) {
                setJumpPath(nextPath);
                return;
            }
            void submitMove(selected!, nextPath);
            return;
        }

        if (selected && owner === mySlot) {
            setSelected({ row, col });
            setJumpPath([]);
            setJumpOriginPiece(null);
            return;
        }

        if (!selected) {
            if (owner !== mySlot) {
                return;
            }
            setSelected({ row, col });
            setJumpPath([]);
            setJumpOriginPiece(cell);
            return;
        }

        const jumpHit = availableTargets.jumps.find(j => j.land.row === row && j.land.col === col);
        if (jumpHit) {
            const { grid: afterJump, crowned } = applyJumpLocal(persistedGrid, selected, jumpHit);
            if (!crowned && pieceJumps(afterJump, row, col).length > 0) {
                setJumpPath([{ row, col }]);
                setJumpOriginPiece(persistedGrid[selected.row][selected.col]);
                return;
            }
            void submitMove(selected, [{ row, col }]);
            return;
        }

        const simpleHit = availableTargets.simples.find(s => s.row === row && s.col === col);
        if (simpleHit) {
            void submitMove(selected, [{ row, col }]);
            return;
        }

        if (owner === null) {
            setSelected(null);
            setJumpOriginPiece(null);
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
    const statsAvailable = isCheckersStats(room.stats);
    const showStats = statsAvailable && (isOver || (room.status === "active" && isSpectator));
    const reasonText = statsAvailable && room.stats ? formatReason((room.stats as CheckersStats).result_reason) : "";

    const highlightTargets = new Set<string>();
    for (const j of availableTargets.jumps) {
        highlightTargets.add(`${j.land.row}-${j.land.col}`);
    }
    for (const s of availableTargets.simples) {
        highlightTargets.add(`${s.row}-${s.col}`);
    }

    const displayRows: number[] = [];
    for (let r = 0; r < BOARD_SIZE; r++) {
        displayRows.push(r);
    }
    if (orientation === "red") {
        displayRows.reverse();
    }
    const displayCols: number[] = [];
    for (let c = 0; c < BOARD_SIZE; c++) {
        displayCols.push(c);
    }
    if (orientation === "black") {
        displayCols.reverse();
    }

    function cellContent(cell: CellChar) {
        if (cell === ".") {
            return null;
        }
        const pieceClass = cell === "r" || cell === "R" ? styles.pieceRed : styles.pieceBlack;
        return (
            <span className={`${styles.piece} ${pieceClass}`}>
                {isKing(cell) && <span className={styles.crown}>♛</span>}
            </span>
        );
    }

    return (
        <div className={styles.wrapper}>
            <GamePlayerBar room={room} slot0Label="Red" slot1Label="Black" liveDurationSeconds={liveDurationSeconds} />

            <DisconnectBanner offlinePlayer={offlinePlayer} forfeitRemaining={forfeitRemaining} />

            {error && <div className={styles.error}>{error}</div>}
            {inProgressJump && !error && (
                <div className={styles.info}>Keep jumping - another capture is required before your turn ends.</div>
            )}
            {availableTargets.mustJump && !inProgressJump && isMyTurn && (
                <div className={styles.info}>Capture is mandatory this turn.</div>
            )}

            <div className={styles.boardContainer}>
                <div className={styles.board}>
                    {displayRows.map(r => (
                        <div className={styles.boardRow} key={r}>
                            {displayCols.map(c => {
                                const dark = isDarkSquare(r, c);
                                const isSelected = activeCoord && activeCoord.row === r && activeCoord.col === c;
                                const isTarget = highlightTargets.has(`${r}-${c}`);
                                const cell = displayGrid[r][c];
                                return (
                                    <button
                                        type="button"
                                        key={`${r}-${c}`}
                                        className={[
                                            styles.square,
                                            dark ? styles.squareDark : styles.squareLight,
                                            isSelected ? styles.squareSelected : "",
                                            isTarget ? styles.squareTarget : "",
                                        ]
                                            .filter(Boolean)
                                            .join(" ")}
                                        onClick={() => handleSquareClick(r, c)}
                                        disabled={!isMyTurn || !dark || submitting}
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

            <GameOverPanel
                isOver={isOver}
                showChildren={showStats}
                resultText={result.text}
                resultTone={result.tone}
                reasonText={reasonText}
            >
                {showStats && statsAvailable && (
                    <GameStatsGrid
                        slot0Name={room.players.find(p => p.slot === SLOT_RED)?.display_name ?? "Red"}
                        slot1Name={room.players.find(p => p.slot === SLOT_BLACK)?.display_name ?? "Black"}
                        isOver={isOver}
                        rows={[
                            {
                                slot0: (room.stats as CheckersStats).red_moves,
                                label: "Moves",
                                slot1: (room.stats as CheckersStats).black_moves,
                            },
                            {
                                slot0: (room.stats as CheckersStats).red_captures,
                                label: "Captures",
                                slot1: (room.stats as CheckersStats).black_captures,
                            },
                            {
                                slot0: (room.stats as CheckersStats).red_crownings,
                                label: "Kings crowned",
                                slot1: (room.stats as CheckersStats).black_crownings,
                            },
                            {
                                slot0: (room.stats as CheckersStats).red_pieces_left,
                                label: "Pieces left",
                                slot1: (room.stats as CheckersStats).black_pieces_left,
                            },
                        ]}
                        totalLabel="Total moves"
                        totalValue={(room.stats as CheckersStats).total_moves}
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
