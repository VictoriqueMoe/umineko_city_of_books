import { useCallback, useEffect, useRef, useState } from "react";
import type { EvidenceInput, EvidenceItem, Quote, Series } from "../types/api";
import { evidenceQuoteKey, useEvidenceQuotes } from "./queries/quote";

const DEFAULT_LANG = "en";
const NO_EVIDENCE: readonly EvidenceItem[] = [];

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

export function useEvidence(initialEvidence?: EvidenceItem[], series: Series = "umineko") {
    const [evidence, setEvidence] = useState<SelectedEvidence[]>([]);
    const [pickerOpen, setPickerOpen] = useState(false);
    const initialised = useRef(false);

    const seed = initialEvidence ?? NO_EVIDENCE;
    const { quotes, settled } = useEvidenceQuotes(seed, series, DEFAULT_LANG);

    useEffect(() => {
        if (initialised.current || seed.length === 0 || !settled) {
            return;
        }
        initialised.current = true;

        const resolved: SelectedEvidence[] = [];
        for (const ev of seed) {
            const quote = quotes.get(evidenceQuoteKey(ev));
            if (!quote) {
                continue;
            }
            resolved.push({ quote, note: ev.note, lang: ev.lang || DEFAULT_LANG });
        }
        setEvidence(resolved);
    }, [seed, quotes, settled]);

    const addQuote = useCallback((quote: Quote, lang: string = DEFAULT_LANG) => {
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
            if (index < 0 || index >= prev.length) {
                return prev;
            }

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
