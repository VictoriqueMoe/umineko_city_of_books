import { useCallback, useEffect, useRef, useState } from "react";
import type { FeedTab, Post, PostListResponse } from "../types/api";
import { listPosts } from "../api/endpoints";

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
    const [data, setData] = useState<PostListResponse | null>(null);
    const [loading, setLoading] = useState(false);
    const seedRef = useRef(generateSeed());
    const limit = 20;
    const offset = (page - 1) * limit;

    const fetchPosts = useCallback(
        async (currentOffset: number, showLoading = true) => {
            if (showLoading) {
                setLoading(true);
            }
            try {
                const result = await listPosts({
                    tab,
                    corner,
                    search: search || undefined,
                    sort: sort || undefined,
                    seed: seedRef.current,
                    limit,
                    offset: currentOffset,
                    resolved: resolved || undefined,
                });
                setData(result);
            } catch {
                setData(null);
            } finally {
                setLoading(false);
            }
        },
        [tab, corner, search, sort, resolved],
    );

    useEffect(() => {
        fetchPosts(offset);
    }, [fetchPosts, offset]);

    return {
        posts: data?.posts ?? ([] as Post[]),
        total: data?.total ?? 0,
        loading,
        offset,
        limit,
        hasNext: data ? offset + limit < data.total : false,
        hasPrev: offset > 0,
        refresh: () => fetchPosts(offset, false),
    };
}
