import { useCallback, useEffect, useState } from "react";
import type { Art, ArtListResponse } from "../types/api";
import { listArt } from "../api/endpoints";

export function useArtFeed(
    corner: string = "general",
    artType?: string,
    search?: string,
    tag?: string,
    sort?: string,
    page: number = 1,
    _refreshKey: number = 0,
) {
    const [data, setData] = useState<ArtListResponse | null>(null);
    const [loading, setLoading] = useState(false);
    const limit = 24;
    const offset = (page - 1) * limit;

    const fetchArt = useCallback(
        async (currentOffset: number, showLoading = true) => {
            if (showLoading) {
                setLoading(true);
            }
            try {
                const result = await listArt({
                    corner,
                    type: artType || undefined,
                    search: search || undefined,
                    tag: tag || undefined,
                    sort: sort || undefined,
                    limit,
                    offset: currentOffset,
                });
                setData(result);
            } catch {
                setData(null);
            } finally {
                setLoading(false);
            }
        },
        [corner, artType, search, tag, sort],
    );

    useEffect(() => {
        fetchArt(offset);
    }, [fetchArt, offset, _refreshKey]);

    return {
        art: data?.art ?? ([] as Art[]),
        total: data?.total ?? 0,
        loading,
        offset,
        limit,
        hasNext: data ? offset + limit < data.total : false,
        hasPrev: offset > 0,
        refresh: () => fetchArt(offset, false),
    };
}
