import {useCallback, useEffect, useState} from "react";
import type {Theory, TheoryListResponse} from "../types/api";
import type {TheorySort} from "../types/app";
import {listTheories} from "../api/endpoints";

export function useTheoryFeed(sort: TheorySort, episode: number, authorId?: string, search?: string) {
    const [data, setData] = useState<TheoryListResponse | null>(null);
    const [loading, setLoading] = useState(false);
    const [offset, setOffset] = useState(0);
    const limit = 20;

    const fetchTheories = useCallback(
        async (currentOffset: number) => {
            setLoading(true);
            try {
                const result = await listTheories({
                    sort,
                    episode: episode || undefined,
                    author: authorId || undefined,
                    search: search || undefined,
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
        [sort, episode, authorId, search],
    );

    useEffect(() => {
        setOffset(0);
        fetchTheories(0);
    }, [fetchTheories]);

    const goNext = useCallback(() => {
        if (data && offset + limit < data.total) {
            const next = offset + limit;
            setOffset(next);
            fetchTheories(next);
        }
    }, [data, offset, fetchTheories]);

    const goPrev = useCallback(() => {
        if (offset > 0) {
            const prev = Math.max(0, offset - limit);
            setOffset(prev);
            fetchTheories(prev);
        }
    }, [offset, fetchTheories]);

    return {
        theories: data?.theories ?? ([] as Theory[]),
        total: data?.total ?? 0,
        loading,
        offset,
        limit,
        goNext,
        goPrev,
        hasNext: data ? offset + limit < data.total : false,
        hasPrev: offset > 0,
        refresh: () => fetchTheories(offset),
    };
}
