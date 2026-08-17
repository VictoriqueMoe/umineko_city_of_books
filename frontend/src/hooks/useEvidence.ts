import { useCallback, useEffect, useRef, useState } from "react";
import type { EvidenceInput, EvidenceItem, Quote } from "../types/api";
import { type Series, tryGetQuoteByAudioId, tryGetQuoteByIndex } from "../api/endpoints";

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
                    quote = await tryGetQuoteByAudioId(series, ev.audio_id, evLang);
                } else if (ev.quote_index !== undefined) {
                    quote = await tryGetQuoteByIndex(series, ev.quote_index, evLang);
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
