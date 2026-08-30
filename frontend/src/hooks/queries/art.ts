import { useQuery } from "@tanstack/react-query";
import type { Art } from "../../types/api";
import {
    getArt,
    getArtCornerCounts,
    getGallery,
    getPopularTags,
    listAllGalleries,
    listArt,
} from "../../api/endpoints/art";
import { queryKeys } from "../../api/queryKeys";

export function useArtCornerCounts() {
    const q = useQuery({ queryKey: queryKeys.art.cornerCounts(), queryFn: () => getArtCornerCounts() });
    return { counts: q.data ?? {}, loading: q.isLoading };
}

export function usePopularTags(corner?: string) {
    const q = useQuery({
        queryKey: queryKeys.art.popularTags(corner ?? ""),
        queryFn: () => getPopularTags(corner),
    });
    return { tags: q.data ?? [], loading: q.isLoading };
}

export function useArtFeed(
    corner: string = "general",
    artType?: string,
    search?: string,
    tag?: string,
    sort?: string,
    offset: number = 0,
    limit: number = 24,
) {
    const params = { corner, artType, search, tag, sort, offset, limit };
    const query = useQuery({
        queryKey: queryKeys.art.feed(params),
        queryFn: () =>
            listArt({
                corner,
                type: artType || undefined,
                search: search || undefined,
                tag: tag || undefined,
                sort: sort || undefined,
                limit,
                offset,
            }),
    });

    const data = query.data;
    return {
        art: data?.art ?? ([] as Art[]),
        total: data?.total ?? 0,
        loading: query.isLoading,
        refresh: query.refetch,
    };
}

export function useArt(id: string) {
    const query = useQuery({
        queryKey: queryKeys.art.detail(id),
        queryFn: () => getArt(id),
        enabled: !!id,
    });
    return { art: query.data ?? null, loading: query.isLoading, refresh: query.refetch };
}

export function useGallery(id: string, limit: number = 24, offset: number = 0) {
    const query = useQuery({
        queryKey: queryKeys.gallery.detail(id, { limit, offset }),
        queryFn: () => getGallery(id, limit, offset),
        enabled: !!id,
    });
    return {
        gallery: query.data?.gallery ?? null,
        art: query.data?.art ?? [],
        total: query.data?.total ?? 0,
        loading: query.isLoading,
        refresh: query.refetch,
    };
}

export function useAllGalleries(corner?: string, enabled = true) {
    const query = useQuery({
        queryKey: queryKeys.gallery.list(corner ?? ""),
        queryFn: () => listAllGalleries(corner),
        enabled,
    });
    return { galleries: query.data ?? [], loading: query.isLoading, refresh: query.refetch };
}
