import { useMemo } from "react";
import { useQueries, useQuery } from "@tanstack/react-query";
import { browseQuotes, searchQuotes, tryGetQuoteByAudioId, tryGetQuoteByIndex } from "../../api/endpoints/quote";
import type { EvidenceItem, Quote, Series } from "../../types/api";
import { queryKeys } from "../../api/queryKeys";

export function useSearchQuotes(
    params: {
        query?: string;
        character?: string;
        episode?: number;
        arc?: string;
        chapter?: string;
        truth?: string;
        lang?: string;
        limit?: number;
        offset?: number;
        series?: Series;
    },
    enabled = true,
) {
    const q = useQuery({
        queryKey: queryKeys.quotes.search(params),
        queryFn: () => searchQuotes(params),
        enabled,
    });
    return { data: q.data ?? null, loading: q.isLoading };
}

export function useBrowseQuotes(
    params: {
        character?: string;
        episode?: number;
        truth?: string;
        arc?: string;
        chapter?: string;
        lang?: string;
        limit?: number;
        offset?: number;
        series?: Series;
    },
    enabled = true,
) {
    const q = useQuery({
        queryKey: queryKeys.quotes.browse(params),
        queryFn: () => browseQuotes(params),
        enabled,
    });
    return { data: q.data ?? null, loading: q.isLoading };
}

export interface EvidenceQuotes {
    quotes: ReadonlyMap<string, Quote | null>;
    settled: boolean;
}

interface EvidenceQuoteQuery {
    key: string;
    queryKey: readonly unknown[];
    queryFn: () => Promise<Quote | null>;
}

export function evidenceQuoteKey(ev: EvidenceItem): string {
    if (ev.audio_id) {
        return `audio:${ev.audio_id}`;
    }
    if (ev.quote_index !== undefined) {
        return `index:${ev.quote_index}`;
    }

    return "";
}

function evidenceQuoteQueries(
    evidence: readonly EvidenceItem[],
    series: Series,
    fallbackLang: string | undefined,
): EvidenceQuoteQuery[] {
    const planned: EvidenceQuoteQuery[] = [];
    for (const ev of evidence) {
        const lang = ev.lang || fallbackLang;
        const audioId = ev.audio_id;
        if (audioId) {
            planned.push({
                key: evidenceQuoteKey(ev),
                queryKey: queryKeys.quotes.byAudioId(series, audioId, lang),
                queryFn: () => tryGetQuoteByAudioId(series, audioId, lang),
            });
            continue;
        }

        const index = ev.quote_index;
        if (index !== undefined) {
            planned.push({
                key: evidenceQuoteKey(ev),
                queryKey: queryKeys.quotes.byIndex(series, index, lang),
                queryFn: () => tryGetQuoteByIndex(series, index, lang),
            });
        }
    }

    return planned;
}

export function useEvidenceQuotes(
    evidence: readonly EvidenceItem[],
    series: Series,
    fallbackLang?: string,
): EvidenceQuotes {
    const planned = useMemo(
        () => evidenceQuoteQueries(evidence, series, fallbackLang),
        [evidence, series, fallbackLang],
    );
    const results = useQueries({
        queries: planned.map(p => ({ queryKey: p.queryKey, queryFn: p.queryFn })),
    });

    return useMemo(() => {
        const quotes = new Map<string, Quote | null>();
        let settled = true;
        for (const [i, result] of results.entries()) {
            if (result.isPending) {
                settled = false;
                continue;
            }
            if (result.data !== undefined) {
                quotes.set(planned[i].key, result.data);
            }
        }

        return { quotes, settled };
    }, [planned, results]);
}
