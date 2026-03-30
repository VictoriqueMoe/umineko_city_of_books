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
}

export function Pagination({ offset, limit, total, hasNext, hasPrev, onNext, onPrev }: PaginationProps) {
    if (total === 0) {
        return null;
    }

    return (
        <div className={styles.pagination}>
            <Button variant="secondary" onClick={onPrev} disabled={!hasPrev}>
                Previous
            </Button>
            <span className={styles.info}>
                {offset + 1}-{Math.min(offset + limit, total)} of {total}
            </span>
            <Button variant="secondary" onClick={onNext} disabled={!hasNext}>
                Next
            </Button>
        </div>
    );
}
