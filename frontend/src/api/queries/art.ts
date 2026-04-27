import { useQuery } from "@tanstack/react-query";
import type { Art } from "../../types/api";
import { getArt, getGallery, listAllGalleries, listArt } from "../endpoints";
import { queryKeys } from "../queryKeys";

export function useArtFeed(
    corner: string = "general",
    artType?: string,
    search?: string,
    tag?: string,
    sort?: string,
    page: number = 1,
    refreshKey: number = 0,
) {
    const limit = 24;
    const offset = (page - 1) * limit;

    const params = { corner, artType, search, tag, sort, offset, limit, refreshKey };
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
        loading: query.isPending,
        offset,
        limit,
        hasNext: data ? offset + limit < data.total : false,
        hasPrev: offset > 0,
        refresh: query.refetch,
    };
}

export function useArt(id: string) {
    const query = useQuery({
        queryKey: queryKeys.art.detail(id),
        queryFn: () => getArt(id),
        enabled: !!id,
    });
    return { art: query.data ?? null, loading: query.isPending, refresh: query.refetch };
}

export function useGallery(id: string, limit: number = 24, offset: number = 0) {
    const query = useQuery({
        queryKey: ["gallery", id, { limit, offset }],
        queryFn: () => getGallery(id, limit, offset),
        enabled: !!id,
    });
    return {
        gallery: query.data?.gallery ?? null,
        art: query.data?.art ?? [],
        total: query.data?.total ?? 0,
        loading: query.isPending,
        refresh: query.refetch,
    };
}

export function useAllGalleries(corner?: string, enabled = true) {
    const query = useQuery({
        queryKey: ["galleries", "all", corner ?? ""],
        queryFn: () => listAllGalleries(corner),
        enabled,
    });
    return { galleries: query.data ?? [], loading: query.isPending, refresh: query.refetch };
}
