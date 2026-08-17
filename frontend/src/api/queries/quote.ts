import { useQuery } from "@tanstack/react-query";
import { browseQuotes, searchQuotes, type Series } from "../endpoints";
import { queryKeys } from "../queryKeys";

export function useSearchQuotes(
    params: {
        query?: string;
        character?: string;
        episode?: number;
        arc?: string;
        chapter?: string;
        truth?: string;
        lang?: string;
        limit?: number;
        offset?: number;
        series?: Series;
    },
    enabled = true,
) {
    const q = useQuery({
        queryKey: queryKeys.quotes.search(params),
        queryFn: () => searchQuotes(params),
        enabled,
    });
    return { data: q.data ?? null, loading: q.isLoading };
}

export function useBrowseQuotes(
    params: {
        character?: string;
        episode?: number;
        truth?: string;
        arc?: string;
        chapter?: string;
        lang?: string;
        limit?: number;
        offset?: number;
        series?: Series;
    },
    enabled = true,
) {
    const q = useQuery({
        queryKey: queryKeys.quotes.browse(params),
        queryFn: () => browseQuotes(params),
        enabled,
    });
    return { data: q.data ?? null, loading: q.isLoading };
}
