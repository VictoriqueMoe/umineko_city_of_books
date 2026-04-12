import { useCallback, useEffect, useState } from "react";
import type { Journal, JournalListResponse, JournalWork } from "../types/api";
import { listJournals } from "../api/endpoints";

export type JournalSort = "new" | "old" | "recently_active" | "most_followed";

export function useJournalFeed(
    sort: JournalSort,
    work: JournalWork | "",
    search?: string,
    includeArchived?: boolean,
    authorId?: string,
) {
    const [data, setData] = useState<JournalListResponse | null>(null);
    const [loading, setLoading] = useState(false);
    const [offset, setOffset] = useState(0);
    const limit = 20;

    const fetchJournals = useCallback(
        async (currentOffset: number) => {
            setLoading(true);
            try {
                const result = await listJournals({
                    sort,
                    work: work || undefined,
                    author: authorId || undefined,
                    search: search || undefined,
                    includeArchived,
                    limit,
                    offset: currentOffset,
                });
                setData(result);
            } catch {
                setData(null);
            } finally {
                setLoading(false);
            }
        },
        [sort, work, authorId, search, includeArchived],
    );

    useEffect(() => {
        setOffset(0);
        fetchJournals(0);
    }, [fetchJournals]);

    const goNext = useCallback(() => {
        if (data && offset + limit < data.total) {
            const next = offset + limit;
            setOffset(next);
            fetchJournals(next);
        }
    }, [data, offset, fetchJournals]);

    const goPrev = useCallback(() => {
        if (offset > 0) {
            const prev = Math.max(0, offset - limit);
            setOffset(prev);
            fetchJournals(prev);
        }
    }, [offset, fetchJournals]);

    return {
        journals: data?.journals ?? ([] as Journal[]),
        total: data?.total ?? 0,
        loading,
        offset,
        limit,
        goNext,
        goPrev,
        hasNext: data ? offset + limit < data.total : false,
        hasPrev: offset > 0,
        refresh: () => fetchJournals(offset),
    };
}
