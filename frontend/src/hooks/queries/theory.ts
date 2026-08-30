import { useQuery } from "@tanstack/react-query";
import type { Series, Theory } from "../../types/api";
import type { TheorySort } from "../../types/app";
import { getTheory, listTheories } from "../../api/endpoints/theory";
import { queryKeys } from "../../api/queryKeys";

export function useTheory(id: string) {
    const query = useQuery({
        queryKey: queryKeys.theory.detail(id),
        queryFn: () => getTheory(id),
        enabled: !!id,
    });
    return {
        theory: query.data ?? null,
        loading: query.isLoading,
        refresh: query.refetch,
    };
}

export interface TheoryFeedOptions {
    sort: TheorySort;
    episode: number;
    authorId?: string;
    search?: string;
    series?: Series;
    offset?: number;
    limit?: number;
    enabled?: boolean;
}

export function useTheoryFeed({
    sort,
    episode,
    authorId,
    search,
    series,
    offset = 0,
    limit = 20,
    enabled = true,
}: TheoryFeedOptions) {
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
        enabled,
    });

    const data = query.data;

    return {
        theories: data?.theories ?? ([] as Theory[]),
        total: data?.total ?? 0,
        loading: query.isLoading,
        refresh: query.refetch,
    };
}
