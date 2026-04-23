import { formatDuration } from "./gameRoomHelpers";
import styles from "./GameStatsGrid.module.css";

export interface StatsRow {
    slot0: number | string;
    label: string;
    slot1: number | string;
}

interface GameStatsGridProps {
    slot0Name: string;
    slot1Name: string;
    isOver: boolean;
    rows: StatsRow[];
    totalLabel: string;
    totalValue: number | string;
    durationSeconds: number;
}

export function GameStatsGrid({
    slot0Name,
    slot1Name,
    isOver,
    rows,
    totalLabel,
    totalValue,
    durationSeconds,
}: GameStatsGridProps) {
    return (
        <div className={styles.statsGrid}>
            <div className={styles.statsHeader}>
                <span>{slot0Name}</span>
                <span>{isOver ? "" : "Live stats"}</span>
                <span>{slot1Name}</span>
            </div>
            {rows.map(row => (
                <div key={row.label} className={styles.statsRow}>
                    <span>{row.slot0}</span>
                    <span className={styles.statsLabel}>{row.label}</span>
                    <span>{row.slot1}</span>
                </div>
            ))}
            <div className={styles.statsFooter}>
                <span>
                    {totalLabel}: {totalValue}
                </span>
                <span>Duration: {formatDuration(durationSeconds)}</span>
            </div>
        </div>
    );
}
