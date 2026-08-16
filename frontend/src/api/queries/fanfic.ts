import { useQuery } from "@tanstack/react-query";
import { getFanfic, getFanficChapter, getFanficLanguages, getFanficSeries, listFanfics } from "../endpoints";
import { queryKeys } from "../queryKeys";

export function useFanficList(params: Parameters<typeof listFanfics>[0]) {
    const q = useQuery({
        queryKey: queryKeys.fanfic.feed(params),
        queryFn: () => listFanfics(params),
    });
    return { fanfics: q.data?.fanfics ?? [], total: q.data?.total ?? 0, loading: q.isLoading };
}

export function useFanfic(id: string) {
    const q = useQuery({
        queryKey: queryKeys.fanfic.detail(id),
        queryFn: () => getFanfic(id),
        enabled: !!id,
    });
    return { fanfic: q.data ?? null, loading: q.isLoading, refresh: q.refetch };
}

export function useFanficChapter(fanficId: string, chapterNumber: number) {
    const q = useQuery({
        queryKey: queryKeys.fanfic.chapter(fanficId, chapterNumber),
        queryFn: () => getFanficChapter(fanficId, chapterNumber),
        enabled: !!fanficId && chapterNumber > 0,
    });
    return { chapter: q.data ?? null, loading: q.isLoading, refresh: q.refetch };
}

export const fanficQueryFns = {
    fanfic: (id: string) => ({
        queryKey: queryKeys.fanfic.detail(id),
        queryFn: () => getFanfic(id),
    }),
    chapter: (fanficId: string, chapterNumber: number) => ({
        queryKey: queryKeys.fanfic.chapter(fanficId, chapterNumber),
        queryFn: () => getFanficChapter(fanficId, chapterNumber),
    }),
};

export function useFanficLanguages() {
    const q = useQuery({
        queryKey: queryKeys.fanfic.languages(),
        queryFn: () => getFanficLanguages(),
        staleTime: Infinity,
    });
    return { languages: q.data ?? [] };
}

export function useFanficSeries() {
    const q = useQuery({
        queryKey: queryKeys.fanfic.seriesList(),
        queryFn: () => getFanficSeries(),
        staleTime: Infinity,
    });
    return { series: q.data ?? [] };
}
