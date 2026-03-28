import {useCallback, useState} from "react";
import type {EvidenceInput, Quote} from "../types/api";

export interface SelectedEvidence {
    quote: Quote;
    note: string;
}

function quoteKey(quote: Quote): string {
    if (quote.audioId) {
        return `audio:${quote.audioId}`;
    }
    return `index:${quote.index}`;
}

export function useEvidence() {
    const [evidence, setEvidence] = useState<SelectedEvidence[]>([]);
    const [pickerOpen, setPickerOpen] = useState(false);

    const addQuote = useCallback((quote: Quote) => {
        const key = quoteKey(quote);
        setEvidence(prev => {
            if (prev.some(e => quoteKey(e.quote) === key)) {
                return prev;
            }
            return [...prev, { quote, note: "" }];
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
