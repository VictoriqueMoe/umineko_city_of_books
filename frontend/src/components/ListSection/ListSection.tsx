import type { ReactNode } from "react";
import { hasNextPage } from "../../hooks/usePageOffset";
import { Pagination } from "../Pagination/Pagination";
import styles from "./ListSection.module.css";

export interface ListSectionPage {
    offset: number;
    limit: number;
    hasPrev: boolean;
    goNext: () => void;
    goPrev: () => void;
}

interface ListSectionProps<T> {
    items: T[];
    loading: boolean;
    total: number;
    page: ListSectionPage | null;
    loadingText: string;
    emptyText: string;
    children: (items: T[]) => ReactNode;
}

export function ListSection<T>({ items, loading, total, page, loadingText, emptyText, children }: ListSectionProps<T>) {
    if (loading) {
        return <div className={styles.loading}>{loadingText}</div>;
    }

    if (items.length === 0) {
        return <div className={styles.empty}>{emptyText}</div>;
    }

    return (
        <>
            {children(items)}
            {page !== null && total > page.limit && (
                <Pagination
                    offset={page.offset}
                    limit={page.limit}
                    total={total}
                    hasNext={hasNextPage(page, total)}
                    hasPrev={page.hasPrev}
                    onNext={page.goNext}
                    onPrev={page.goPrev}
                />
            )}
        </>
    );
}
