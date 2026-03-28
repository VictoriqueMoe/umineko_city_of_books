import { useEffect, useRef, useState } from "react";
import type { EvidenceItem, Quote } from "../types/api";

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

async function fetchQuoteByAudioId(audioId: string): Promise<Quote | null> {
    const firstId = audioId.split(",")[0].trim();
    if (!firstId) {
        return null;
    }
    try {
        const response = await fetch(`${QUOTE_API}/quote/${firstId}`);
        if (!response.ok) {
            return null;
        }
        return response.json();
    } catch {
        return null;
    }
}

async function fetchQuoteByIndex(index: number): Promise<Quote | null> {
    try {
        const response = await fetch(`${QUOTE_API}/quote/index/${index}`);
        if (!response.ok) {
            return null;
        }
        return response.json();
    } catch {
        return null;
    }
}

async function fetchEvidence(ev: EvidenceItem): Promise<Quote | null> {
    if (ev.audio_id) {
        return fetchQuoteByAudioId(ev.audio_id);
    }
    if (ev.quote_index !== undefined) {
        return fetchQuoteByIndex(ev.quote_index);
    }
    return null;
}

export function useResolveQuotes(evidence: EvidenceItem[]) {
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

        Promise.all(toFetch.map(ev => fetchEvidence(ev).then(q => [evidenceKey(ev), q] as const))).then(results => {
            setQuotes(prev => {
                const next = new Map(prev);
                for (const [key, q] of results) {
                    next.set(key, q);
                }
                return next;
            });
        });
    }, [evidence]);

    return quotes;
}
