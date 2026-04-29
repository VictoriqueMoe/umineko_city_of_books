import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { quickSearch, searchSite } from "../endpoints";

export function useQuickSearch(query: string, enabled: boolean) {
    const trimmed = query.trim();
    const q = useQuery({
        queryKey: ["search", "quick", trimmed],
        queryFn: () => quickSearch(trimmed, 3),
        enabled: enabled && trimmed.length >= 2,
        staleTime: 30_000,
    });
    return {
        results: q.data?.results ?? [],
        loading: q.isLoading || q.isFetching,
    };
}

export function useSiteSearch(query: string, types: string, limit: number, offset: number) {
    const trimmed = query.trim();
    const q = useQuery({
        queryKey: ["search", "full", trimmed, types, limit, offset],
        queryFn: () => searchSite(trimmed, types, limit, offset),
        enabled: trimmed.length >= 2,
        placeholderData: keepPreviousData,
    });
    return {
        results: q.data?.results ?? [],
        total: q.data?.total ?? 0,
        loading: q.isLoading,
        fetching: q.isFetching,
    };
}
