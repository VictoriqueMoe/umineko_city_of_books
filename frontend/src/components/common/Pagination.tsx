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
        <div className="pagination">
            <button className="pagination-btn" onClick={onPrev} disabled={!hasPrev}>
                Previous
            </button>
            <span className="pagination-info">
                {offset + 1}-{Math.min(offset + limit, total)} of {total}
            </span>
            <button className="pagination-btn" onClick={onNext} disabled={!hasNext}>
                Next
            </button>
        </div>
    );
}
