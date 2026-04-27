import { useCallback, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import type { Theory } from "../../types/api";
import type { TheorySort } from "../../types/app";
import { getTheory, listTheories, type Series } from "../endpoints";
import { queryKeys } from "../queryKeys";

export function useTheory(id: string) {
    const query = useQuery({
        queryKey: queryKeys.theory.detail(id),
        queryFn: () => getTheory(id),
        enabled: !!id,
    });
    return {
        theory: query.data ?? null,
        loading: query.isPending,
        refresh: query.refetch,
    };
}

export function useTheoryFeed(sort: TheorySort, episode: number, authorId?: string, search?: string, series?: Series) {
    const limit = 20;
    const [offset, setOffset] = useState(0);

    const params = { sort, episode, authorId, search, series, offset, limit };
    const query = useQuery({
        queryKey: queryKeys.theory.feed(params),
        queryFn: () =>
            listTheories({
                sort,
                episode: episode || undefined,
                author: authorId || undefined,
                search: search || undefined,
                series: series ?? "umineko",
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
        theories: data?.theories ?? ([] as Theory[]),
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
