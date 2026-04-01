import { useCallback, useEffect, useState } from "react";
import type { FeedTab, Post, PostListResponse } from "../types/api";
import { listPosts } from "../api/endpoints";

export function usePostFeed(tab: FeedTab, search?: string, sort?: string) {
    const [data, setData] = useState<PostListResponse | null>(null);
    const [loading, setLoading] = useState(false);
    const [offset, setOffset] = useState(0);
    const limit = 20;

    const fetchPosts = useCallback(
        async (currentOffset: number, showLoading = true) => {
            if (showLoading) {
                setLoading(true);
            }
            try {
                const result = await listPosts({
                    tab,
                    search: search || undefined,
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
        [tab, search, sort],
    );

    useEffect(() => {
        setOffset(0);
        fetchPosts(0);
    }, [fetchPosts]);

    const goNext = useCallback(() => {
        if (data && offset + limit < data.total) {
            const next = offset + limit;
            setOffset(next);
            fetchPosts(next);
        }
    }, [data, offset, fetchPosts]);

    const goPrev = useCallback(() => {
        if (offset > 0) {
            const prev = Math.max(0, offset - limit);
            setOffset(prev);
            fetchPosts(prev);
        }
    }, [offset, fetchPosts]);

    return {
        posts: data?.posts ?? ([] as Post[]),
        total: data?.total ?? 0,
        loading,
        offset,
        limit,
        goNext,
        goPrev,
        hasNext: data ? offset + limit < data.total : false,
        hasPrev: offset > 0,
        refresh: () => fetchPosts(offset, false),
    };
}
