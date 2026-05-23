import { useMemo } from "react";
import type { MinesweeperState } from "../../../types/api";
import styles from "./MinesweeperMiniBoard.module.css";

interface MinesweeperMiniBoardProps {
    state: MinesweeperState;
    slot: number;
}

export function MinesweeperMiniBoard({ state, slot }: MinesweeperMiniBoardProps) {
    const cells = useMemo(() => {
        const revealedArr = state.revealed?.[slot] ?? [];
        const flaggedArr = state.flagged?.[slot] ?? [];
        const mines = state.mines ?? [];
        const out: { revealed: boolean; flagged: boolean; mine: boolean }[][] = [];
        for (let y = 0; y < state.height; y++) {
            const row: { revealed: boolean; flagged: boolean; mine: boolean }[] = [];
            for (let x = 0; x < state.width; x++) {
                const idx = y * state.width + x;
                row.push({
                    revealed: Boolean(revealedArr[idx]),
                    flagged: Boolean(flaggedArr[idx]),
                    mine: Boolean(mines[idx]),
                });
            }
            out.push(row);
        }
        return out;
    }, [state, slot]);

    return (
        <div className={styles.miniWrap}>
            <div
                className={styles.miniBoard}
                style={{
                    gridTemplateColumns: `repeat(${state.width}, 1fr)`,
                    gridTemplateRows: `repeat(${state.height}, 1fr)`,
                }}
            >
                {cells.map((row, y) =>
                    row.map((c, x) => {
                        const classes = [styles.miniCell];
                        if (c.revealed) {
                            classes.push(c.mine ? styles.mine : styles.revealed);
                        } else if (c.flagged) {
                            classes.push(styles.flagged);
                        }
                        return <div key={`${x}-${y}`} className={classes.join(" ")} />;
                    }),
                )}
            </div>
        </div>
    );
}
