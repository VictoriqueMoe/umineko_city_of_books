import { useEffect, useRef, useState } from "react";
import type { EvidenceItem, Quote } from "../types/api";
import type { Series } from "../api/endpoints";

const QUOTE_API = "https://quotes.auaurora.moe/api/v1";

function evidenceKey(ev: EvidenceItem): string {
    if (ev.audio_id) {
        return `audio:${ev.audio_id}`;
    }
    if (ev.quote_index !== undefined) {
        return `index:${ev.quote_index}`;
    }
    return "";
}

async function fetchQuoteByAudioId(series: Series, audioId: string, lang?: string): Promise<Quote | null> {
    const firstId = audioId.split(",")[0].trim();
    if (!firstId) {
        return null;
    }
    try {
        const qs = lang ? `?lang=${lang}` : "";
        const response = await fetch(`${QUOTE_API}/${series}/quote/${firstId}${qs}`);
        if (!response.ok) {
            return null;
        }
        return response.json();
    } catch {
        return null;
    }
}

async function fetchQuoteByIndex(series: Series, index: number, lang?: string): Promise<Quote | null> {
    try {
        const qs = lang ? `?lang=${lang}` : "";
        const response = await fetch(`${QUOTE_API}/${series}/quote/index/${index}${qs}`);
        if (!response.ok) {
            return null;
        }
        return response.json();
    } catch {
        return null;
    }
}

async function fetchEvidence(series: Series, ev: EvidenceItem): Promise<Quote | null> {
    const lang = ev.lang || undefined;
    if (ev.audio_id) {
        return fetchQuoteByAudioId(series, ev.audio_id, lang);
    }
    if (ev.quote_index !== undefined) {
        return fetchQuoteByIndex(series, ev.quote_index, lang);
    }
    return null;
}

export function useResolveQuotes(evidence: EvidenceItem[], series: Series = "umineko") {
    const [quotes, setQuotes] = useState<Map<string, Quote | null>>(new Map());
    const attempted = useRef<Set<string>>(new Set());

    useEffect(() => {
        const toFetch = evidence.filter(ev => {
            const key = evidenceKey(ev);
            return key !== "" && !attempted.current.has(key);
        });
        if (toFetch.length === 0) {
            return;
        }

        for (const ev of toFetch) {
            attempted.current.add(evidenceKey(ev));
        }

        Promise.all(toFetch.map(ev => fetchEvidence(series, ev).then(q => [evidenceKey(ev), q] as const))).then(
            results => {
                setQuotes(prev => {
                    const next = new Map(prev);
                    for (const [key, q] of results) {
                        next.set(key, q);
                    }
                    return next;
                });
            },
        );
    }, [evidence, series]);

    return quotes;
}
