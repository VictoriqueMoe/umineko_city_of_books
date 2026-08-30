import styles from "./MysteryPages.module.css";

export interface MysteryBadgeSubject {
    difficulty: string;
    solved: boolean;
    paused: boolean;
    gm_away: boolean;
    free_for_all: boolean;
    keep_open_after_solve: boolean;
    solver_count: number;
}

export function MysteryBadges({ mystery, playerCount }: { mystery: MysteryBadgeSubject; playerCount?: number }) {
    return (
        <div className={styles.cardBadges}>
            <span className={`${styles.badge} ${styles.badgeDifficulty}`}>{mystery.difficulty}</span>
            <span className={`${styles.badge} ${mystery.solved ? styles.badgeSolved : styles.badgeOpen}`}>
                {mystery.solved ? "Solved" : "Open"}
            </span>
            {mystery.paused && <span className={`${styles.badge} ${styles.badgePaused}`}>Paused</span>}
            {mystery.gm_away && !mystery.paused && (
                <span className={`${styles.badge} ${styles.badgeAway}`}>GM Away</span>
            )}
            {mystery.free_for_all && <span className={`${styles.badge} ${styles.badgeFreeForAll}`}>Free-for-all</span>}
            {mystery.keep_open_after_solve && !mystery.solved && (
                <span className={`${styles.badge} ${styles.badgeFreeForAll}`}>Ongoing</span>
            )}
            {mystery.solver_count > 0 && (
                <span className={`${styles.badge} ${styles.badgeSolved}`}>
                    {mystery.solver_count} solver{mystery.solver_count !== 1 ? "s" : ""}
                </span>
            )}
            {playerCount !== undefined && (
                <span className={`${styles.badge} ${styles.badgePieces}`}>
                    {playerCount} piece{playerCount !== 1 ? "s" : ""} attempting
                </span>
            )}
        </div>
    );
}
