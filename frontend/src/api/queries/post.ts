import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import type { FeedTab, Post } from "../../types/api";
import { getCornerCounts, getPost, getShareCount, listPosts } from "../endpoints";
import { queryKeys } from "../queryKeys";

function generateSeed(): number {
    return Math.floor(Math.random() * 1000000);
}

export function usePostFeed(
    tab: FeedTab,
    corner: string = "general",
    search?: string,
    sort?: string,
    page: number = 1,
    resolved?: string,
) {
    const limit = 20;
    const offset = (page - 1) * limit;
    const seed = useMemo(() => generateSeed(), []);
    const params = { tab, corner, search, sort, resolved, offset, limit, seed };
    const query = useQuery({
        queryKey: queryKeys.post.feed(params),
        queryFn: () =>
            listPosts({
                tab,
                corner,
                search: search || undefined,
                sort: sort || undefined,
                seed,
                limit,
                offset,
                resolved: resolved || undefined,
            }),
    });
    const data = query.data;
    return {
        posts: data?.posts ?? ([] as Post[]),
        total: data?.total ?? 0,
        loading: query.isPending,
        offset,
        limit,
        hasNext: data ? offset + limit < data.total : false,
        hasPrev: offset > 0,
        refresh: query.refetch,
    };
}

export function usePost(id: string) {
    const query = useQuery({
        queryKey: queryKeys.post.detail(id),
        queryFn: () => getPost(id),
        enabled: !!id,
    });
    return { post: query.data ?? null, loading: query.isPending, refresh: query.refetch };
}

export function useShareCount(contentType: string, contentId: string, enabled = true) {
    const query = useQuery({
        queryKey: ["share-count", contentType, contentId],
        queryFn: () => getShareCount(contentType, contentId),
        enabled: enabled && !!contentId,
    });
    return { shareCount: query.data?.share_count ?? 0, loading: query.isPending };
}

export function useCornerCounts() {
    const query = useQuery({
        queryKey: ["post", "corner-counts"],
        queryFn: () => getCornerCounts(),
    });
    return { counts: query.data ?? null, loading: query.isPending };
}
