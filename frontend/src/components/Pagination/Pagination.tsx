import { Button } from "../Button/Button";
import styles from "./Pagination.module.css";

interface PaginationProps {
    offset: number;
    limit: number;
    total: number;
    hasNext: boolean;
    hasPrev: boolean;
    onNext: () => void;
    onPrev: () => void;
    onFirst?: () => void;
    onLast?: () => void;
    size?: "small" | "medium";
    compact?: boolean;
}

export function Pagination({
    offset,
    limit,
    total,
    hasNext,
    hasPrev,
    onNext,
    onPrev,
    onFirst,
    onLast,
    size = "medium",
    compact = false,
}: PaginationProps) {
    if (total === 0) {
        return null;
    }

    const classes = [styles.pagination, compact ? styles.compact : ""].filter(Boolean).join(" ");

    return (
        <div className={classes}>
            {onFirst && (
                <Button variant="secondary" size={size} onClick={onFirst} disabled={!hasPrev}>
                    {"« First"}
                </Button>
            )}
            <Button variant="secondary" size={size} onClick={onPrev} disabled={!hasPrev}>
                Previous
            </Button>
            <span className={styles.info}>
                {offset + 1}-{Math.min(offset + limit, total)} of {total}
            </span>
            <Button variant="secondary" size={size} onClick={onNext} disabled={!hasNext}>
                Next
            </Button>
            {onLast && (
                <Button variant="secondary" size={size} onClick={onLast} disabled={!hasNext}>
                    {"Last »"}
                </Button>
            )}
        </div>
    );
}
