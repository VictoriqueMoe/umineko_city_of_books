import { useQuery } from "@tanstack/react-query";
import { listGiphyFavourites, searchGiphy, trendingGiphy } from "../endpoints";
import { useAuth } from "../../hooks/useAuth";

export function useGiphySearch(query: string, offset = 0, limit = 0, enabled = true) {
    const q = useQuery({
        queryKey: ["giphy", "search", query, { offset, limit }],
        queryFn: () => searchGiphy(query, offset, limit),
        enabled: enabled && !!query,
        staleTime: 5 * 60_000,
    });
    return { data: q.data, loading: q.isLoading, error: q.error, refresh: q.refetch };
}

export function useGiphyTrending(offset = 0, limit = 0, enabled = true) {
    const q = useQuery({
        queryKey: ["giphy", "trending", { offset, limit }],
        queryFn: () => trendingGiphy(offset, limit),
        enabled,
        staleTime: 5 * 60_000,
    });
    return { data: q.data, loading: q.isLoading, error: q.error, refresh: q.refetch };
}

export function useGiphyFavourites(offset = 0, limit = 0) {
    const { user } = useAuth();
    const q = useQuery({
        queryKey: ["giphy", "favourites", { offset, limit }],
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
