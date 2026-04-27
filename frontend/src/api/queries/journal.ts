import { useCallback, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import type { Journal, JournalWork } from "../../types/api";
import { getJournal, listJournals } from "../endpoints";
import { queryKeys } from "../queryKeys";

export type JournalSort = "new" | "old" | "recently_active" | "most_followed";

export function useJournal(id: string) {
    const q = useQuery({
        queryKey: queryKeys.journal.detail(id),
        queryFn: () => getJournal(id),
        enabled: !!id,
    });
    return {
        journal: q.data ?? null,
        loading: q.isPending,
        refresh: q.refetch,
    };
}

export function useJournalFeed(
    sort: JournalSort,
    work: JournalWork | "",
    search?: string,
    includeArchived?: boolean,
    authorId?: string,
) {
    const limit = 20;
    const [offset, setOffset] = useState(0);

    const params = { sort, work, search, includeArchived, authorId, offset, limit };
    const query = useQuery({
        queryKey: queryKeys.journal.feed(params),
        queryFn: () =>
            listJournals({
                sort,
                work: work || undefined,
                author: authorId || undefined,
                search: search || undefined,
                includeArchived,
                limit,
                offset,
            }),
    });

    const data = query.data;

    const goNext = useCallback(() => {
        if (data && offset + limit < data.total) {
            setOffset(offset + limit);
        }
    }, [data, offset]);

    const goPrev = useCallback(() => {
        if (offset > 0) {
            setOffset(Math.max(0, offset - limit));
        }
    }, [offset]);

    return {
        journals: data?.journals ?? ([] as Journal[]),
        total: data?.total ?? 0,
        loading: query.isPending,
        offset,
        limit,
        goNext,
        goPrev,
        hasNext: data ? offset + limit < data.total : false,
        hasPrev: offset > 0,
        refresh: query.refetch,
    };
}
