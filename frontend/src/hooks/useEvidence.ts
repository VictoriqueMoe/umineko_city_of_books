import { useCallback, useEffect, useRef, useState } from "react";
import type { EvidenceInput, EvidenceItem, Quote } from "../types/api";
import type { Series } from "../api/endpoints";

const QUOTE_API = "https://quotes.auaurora.moe/api/v1";

export interface SelectedEvidence {
    quote: Quote;
    note: string;
    lang: string;
}

function quoteKey(quote: Quote): string {
    if (quote.audioId) {
        return `audio:${quote.audioId}`;
    }
    return `index:${quote.index}`;
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

export function useEvidence(initialEvidence?: EvidenceItem[], series: Series = "umineko") {
    const [evidence, setEvidence] = useState<SelectedEvidence[]>([]);
    const [pickerOpen, setPickerOpen] = useState(false);
    const initialised = useRef(false);

    useEffect(() => {
        if (initialised.current || !initialEvidence || initialEvidence.length === 0) {
            return;
        }
        initialised.current = true;

        Promise.all(
            initialEvidence.map(async ev => {
                let quote: Quote | null = null;
                const evLang = ev.lang || "en";
                if (ev.audio_id) {
                    quote = await fetchQuoteByAudioId(series, ev.audio_id, evLang);
                } else if (ev.quote_index !== undefined) {
                    quote = await fetchQuoteByIndex(series, ev.quote_index, evLang);
                }
                if (!quote) {
                    return null;
                }
                return { quote, note: ev.note, lang: evLang } as SelectedEvidence;
            }),
        ).then(results => {
            const resolved = results.filter((r): r is SelectedEvidence => r !== null);
            setEvidence(resolved);
        });
    }, [initialEvidence, series]);

    const addQuote = useCallback((quote: Quote, lang: string = "en") => {
        const key = quoteKey(quote);
        setEvidence(prev => {
            if (prev.some(e => quoteKey(e.quote) === key)) {
                return prev;
            }
            return [...prev, { quote, note: "", lang }];
        });
        setPickerOpen(false);
    }, []);

    const updateNote = useCallback((index: number, note: string) => {
        setEvidence(prev => {
            const updated = [...prev];
            updated[index] = { ...updated[index], note };
            return updated;
        });
    }, []);

    const removeAt = useCallback((index: number) => {
        setEvidence(prev => prev.filter((_, i) => i !== index));
    }, []);

    const clear = useCallback(() => {
        setEvidence([]);
    }, []);

    const openPicker = useCallback(() => setPickerOpen(true), []);
    const closePicker = useCallback(() => setPickerOpen(false), []);

    const toInput = useCallback((): EvidenceInput[] => {
        return evidence.map(ev => ({
            audio_id: ev.quote.audioId || undefined,
            quote_index: ev.quote.audioId ? undefined : ev.quote.index,
            note: ev.note,
            lang: ev.lang,
        }));
    }, [evidence]);

    return {
        evidence,
        pickerOpen,
        addQuote,
        updateNote,
        removeAt,
        clear,
        openPicker,
        closePicker,
        toInput,
        selectedKeys: evidence.map(e => quoteKey(e.quote)),
    };
}
