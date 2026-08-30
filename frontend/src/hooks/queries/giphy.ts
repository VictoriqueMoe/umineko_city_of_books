import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { ApiError } from "../../api/client";
import { listGiphyFavourites, searchGiphy, trendingGiphy } from "../../api/endpoints/giphy";
import { queryKeys } from "../../api/queryKeys";
import { useAuth } from "../useAuth";

const RATE_LIMITED_STATUS = 429;

export interface GiphyRateLimit {
    resetAt: string | null;
}

function readRateLimit(error: unknown): GiphyRateLimit | null {
    if (!(error instanceof ApiError) || error.status !== RATE_LIMITED_STATUS) {
        return null;
    }

    const body = error.body as { reset_at?: string } | null;

    return { resetAt: body?.reset_at ?? null };
}

export function useGiphySearch(query: string, offset = 0, limit = 0, enabled = true) {
    const q = useQuery({
        queryKey: queryKeys.giphy.search(query, { offset, limit }),
        queryFn: () => searchGiphy(query, offset, limit),
        enabled: enabled && !!query,
        staleTime: 5 * 60_000,
    });
    const rateLimit = useMemo(() => readRateLimit(q.error), [q.error]);

    return { data: q.data, loading: q.isLoading, error: q.error, rateLimit, refresh: q.refetch };
}

export function useGiphyTrending(offset = 0, limit = 0, enabled = true) {
    const q = useQuery({
        queryKey: queryKeys.giphy.trending({ offset, limit }),
        queryFn: () => trendingGiphy(offset, limit),
        enabled,
        staleTime: 5 * 60_000,
    });
    const rateLimit = useMemo(() => readRateLimit(q.error), [q.error]);

    return { data: q.data, loading: q.isLoading, error: q.error, rateLimit, refresh: q.refetch };
}

export function useGiphyFavourites(offset = 0, limit = 0) {
    const { user } = useAuth();
    const q = useQuery({
        queryKey: queryKeys.giphy.favouritesList({ offset, limit }),
        queryFn: () => listGiphyFavourites(offset, limit),
        enabled: !!user,
    });
    return {
        favourites: q.data?.data ?? [],
        total: q.data?.total ?? 0,
        loading: q.isLoading,
        refresh: q.refetch,
    };
}
