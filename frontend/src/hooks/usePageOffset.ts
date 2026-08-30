import { useCallback, useState } from "react";
import { useSearchParams } from "react-router";

export interface PageOffsetOptions {
    limit: number;
}

export interface SearchParamPageOptions extends PageOffsetOptions {
    param?: string;
    firstPage?: 0 | 1;
    replace?: boolean;
}

export interface PageOffset {
    offset: number;
    limit: number;
    hasPrev: boolean;
    goNext: () => void;
    goPrev: () => void;
    reset: () => void;
}

export function hasNextPage(page: { offset: number; limit: number }, total: number): boolean {
    return page.offset + page.limit < total;
}

function previousOffset(offset: number, limit: number): number {
    return Math.max(0, offset - limit);
}

function parsePageIndex(raw: string | null, firstPage: number): number {
    if (raw === null || raw.trim() === "") {
        return 0;
    }

    const parsed = Number(raw);
    if (!Number.isFinite(parsed)) {
        return 0;
    }

    return Math.max(0, Math.floor(parsed) - firstPage);
}

export function usePageOffset({ limit }: PageOffsetOptions): PageOffset {
    const [offset, setOffset] = useState(0);

    const goNext = useCallback(() => {
        setOffset(current => current + limit);
    }, [limit]);

    const goPrev = useCallback(() => {
        setOffset(current => previousOffset(current, limit));
    }, [limit]);

    const reset = useCallback(() => {
        setOffset(0);
    }, []);

    return {
        offset,
        limit,
        hasPrev: offset > 0,
        goNext,
        goPrev,
        reset,
    };
}

export function useSearchParamPage({
    limit,
    param = "page",
    firstPage = 1,
    replace = false,
}: SearchParamPageOptions): PageOffset {
    const [searchParams, setSearchParams] = useSearchParams();

    const offset = parsePageIndex(searchParams.get(param), firstPage) * limit;

    const updatePage = useCallback(
        (mapIndex: (current: number) => number) => {
            setSearchParams(
                prev => {
                    const next = new URLSearchParams(prev);
                    const target = Math.max(0, mapIndex(parsePageIndex(prev.get(param), firstPage)));

                    if (target > 0) {
                        next.set(param, String(target + firstPage));
                    } else {
                        next.delete(param);
                    }

                    return next;
                },
                { replace },
            );
        },
        [firstPage, param, replace, setSearchParams],
    );

    const goNext = useCallback(() => {
        updatePage(current => current + 1);
    }, [updatePage]);

    const goPrev = useCallback(() => {
        updatePage(current => current - 1);
    }, [updatePage]);

    const reset = useCallback(() => {
        updatePage(() => 0);
    }, [updatePage]);

    return {
        offset,
        limit,
        hasPrev: offset > 0,
        goNext,
        goPrev,
        reset,
    };
}
