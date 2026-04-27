import { useQuery } from "@tanstack/react-query";
import {
    getFanfic,
    getFanficChapter,
    getFanficLanguages,
    getFanficSeries,
    listFanfics,
    searchOCCharacters,
} from "../endpoints";
import { queryKeys } from "../queryKeys";

export function useFanficList(params: Parameters<typeof listFanfics>[0]) {
    const q = useQuery({
        queryKey: queryKeys.fanfic.feed(params),
        queryFn: () => listFanfics(params),
    });
    return { fanfics: q.data?.fanfics ?? [], total: q.data?.total ?? 0, loading: q.isPending };
}

export function useFanfic(id: string) {
    const q = useQuery({
        queryKey: queryKeys.fanfic.detail(id),
        queryFn: () => getFanfic(id),
        enabled: !!id,
    });
    return { fanfic: q.data ?? null, loading: q.isPending, refresh: q.refetch };
}

export function useFanficChapter(fanficId: string, chapterNumber: number) {
    const q = useQuery({
        queryKey: ["fanfic", fanficId, "chapter", chapterNumber],
        queryFn: () => getFanficChapter(fanficId, chapterNumber),
        enabled: !!fanficId && chapterNumber > 0,
    });
    return { chapter: q.data ?? null, loading: q.isPending, refresh: q.refetch };
}

export const fanficQueryFns = {
    fanfic: (id: string) => ({
        queryKey: queryKeys.fanfic.detail(id),
        queryFn: () => getFanfic(id),
    }),
    chapter: (fanficId: string, chapterNumber: number) => ({
        queryKey: ["fanfic", fanficId, "chapter", chapterNumber] as const,
        queryFn: () => getFanficChapter(fanficId, chapterNumber),
    }),
};

export function useFanficLanguages() {
    const q = useQuery({
        queryKey: ["fanfic", "languages"],
        queryFn: () => getFanficLanguages(),
        staleTime: Infinity,
    });
    return { languages: q.data ?? [] };
}

export function useFanficSeries() {
    const q = useQuery({
        queryKey: ["fanfic", "series"],
        queryFn: () => getFanficSeries(),
        staleTime: Infinity,
    });
    return { series: q.data ?? [] };
}

export function useSearchOCCharacters(query: string, enabled = true) {
    const q = useQuery({
        queryKey: ["fanfic", "oc-search", query],
        queryFn: () => searchOCCharacters(query),
        enabled: enabled && !!query,
    });
    return { characters: q.data ?? [], loading: q.isPending };
}
