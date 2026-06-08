import { useMemo } from "react";
import type { CSSProperties } from "react";
import type { MinesweeperState } from "../../../types/api";
import { MinesweeperCell } from "./MinesweeperCell";
import styles from "./MinesweeperBoard.module.css";

interface MinesweeperBoardProps {
    state: MinesweeperState;
    slot: number;
    interactive: boolean;
    cellSize?: number;
    flagMode?: boolean;
    pendingClick?: { x: number; y: number } | null;
    onReveal?: (x: number, y: number) => void;
    onFlag?: (x: number, y: number) => void;
}

export function MinesweeperBoard({
    state,
    slot,
    interactive,
    cellSize = 32,
    flagMode = false,
    pendingClick,
    onReveal,
    onFlag,
}: MinesweeperBoardProps) {
    const cells = useMemo(() => {
        const rows: { x: number; y: number; revealed: boolean; flagged: boolean; mine: boolean; value: number }[][] =
            [];
        const revealedArr = state.revealed?.[slot] ?? [];
        const flaggedArr = state.flagged?.[slot] ?? [];
        const valuesArr = state.values?.[slot] ?? [];
        const mines = state.mines ?? [];
        for (let y = 0; y < state.height; y++) {
            const row: { x: number; y: number; revealed: boolean; flagged: boolean; mine: boolean; value: number }[] =
                [];
            for (let x = 0; x < state.width; x++) {
                const idx = y * state.width + x;
                const revealed = Boolean(revealedArr[idx]);
                const flagged = Boolean(flaggedArr[idx]);
                const mine = Boolean(mines[idx]);
                const value = revealed && !mine ? (valuesArr[idx] ?? 0) : 0;
                row.push({ x, y, revealed, flagged, mine, value });
            }
            rows.push(row);
        }
        return rows;
    }, [state, slot]);

    const hitX = state.hit_mine_x;
    const hitY = state.hit_mine_y;
    const isMini = cellSize < 20;
    const isFinished = state.phase === "finished";
    const serverPending = state.pending_clicks?.[slot] ?? null;
    const effectivePending =
        pendingClick ?? (serverPending !== null ? { x: serverPending[0], y: serverPending[1] } : null);
    const winnerSlot = state.winner_slot;
    const loserSlot = winnerSlot !== undefined ? 1 - winnerSlot : -1;
    const isLoser = slot === loserSlot;
    const isWinner = slot === winnerSlot;
    const outcomeClass = isFinished ? (isWinner ? styles.boardWinner : isLoser ? styles.boardLoser : "") : "";
    const wrapperClass = [styles.boardWrap, isMini ? styles.mini : "", outcomeClass].filter(Boolean).join(" ");
    const boardClass = isMini ? `${styles.board} ${styles.boardMini}` : styles.board;

    const cssVars: CSSProperties = {
        ["--cell-max" as string]: `${cellSize}px`,
        ["--cols" as string]: String(state.width),
        ["--rows" as string]: String(state.height),
    };

    return (
        <div className={wrapperClass} style={cssVars}>
            {isFinished && isWinner && <div className={`${styles.outcomeBadge} ${styles.outcomeWinner}`}>Winner</div>}
            {isFinished && isLoser && <div className={`${styles.outcomeBadge} ${styles.outcomeLoser}`}>Defeated</div>}
            <div className={boardClass}>
                {cells.map(row =>
                    row.map(c => {
                        const isPending =
                            !isMini &&
                            effectivePending !== null &&
                            effectivePending.x === c.x &&
                            effectivePending.y === c.y &&
                            !c.revealed;
                        const isHitMine = isLoser && hitX === c.x && hitY === c.y;
                        return (
                            <MinesweeperCell
                                key={`${c.x}-${c.y}`}
                                revealed={c.revealed}
                                flagged={c.flagged}
                                mine={c.mine}
                                value={c.value}
                                isHitMine={isHitMine}
                                isPending={isPending}
                                forceShowMine={isFinished && c.mine}
                                hideContent={isMini}
                                disabled={!interactive || state.phase !== "playing" || c.revealed}
                                onClick={
                                    isMini ? undefined : () => (flagMode ? onFlag?.(c.x, c.y) : onReveal?.(c.x, c.y))
                                }
                                onRightClick={isMini ? undefined : () => onFlag?.(c.x, c.y)}
                            />
                        );
                    }),
                )}
            </div>
        </div>
    );
}
