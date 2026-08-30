import type { ShipCharacter } from "../../types/api";
import type { EditorFields } from "./form";

export const DRAFT_KEY = "fanfic-draft";

export interface DraftData {
    title: string;
    summary: string;
    series: string;
    customSeries: string;
    rating: string;
    language: string;
    customLanguage: string;
    genreA: string;
    genreB: string;
    tags: string[];
    status: string;
    characters: ShipCharacter[];
    isPairing: boolean;
    isOneshot: boolean;
    containsLemons: boolean;
    body: string;
    step: number;
}

export function loadDraft(): DraftData | null {
    try {
        const raw = localStorage.getItem(DRAFT_KEY);
        if (!raw) {
            return null;
        }
        return JSON.parse(raw) as DraftData;
    } catch {
        return null;
    }
}

export function saveDraft(data: DraftData) {
    localStorage.setItem(DRAFT_KEY, JSON.stringify(data));
}

export function clearDraft() {
    localStorage.removeItem(DRAFT_KEY);
}

export function resumableDraft(): DraftData | null {
    const existing = loadDraft();

    return existing && existing.title ? existing : null;
}

export function draftToFields(draft: DraftData): Partial<EditorFields> {
    return {
        title: draft.title,
        summary: draft.summary,
        series: draft.series,
        customSeries: draft.customSeries,
        rating: draft.rating,
        language: draft.language,
        customLanguage: draft.customLanguage,
        genreA: draft.genreA,
        genreB: draft.genreB,
        tags: draft.tags ?? [],
        status: draft.status ?? "in_progress",
        characters: draft.characters,
        isPairing: draft.isPairing,
        isOneshot: draft.isOneshot,
        containsLemons: draft.containsLemons,
    };
}

export function draftFromFields(fields: EditorFields, body: string, step: number): DraftData {
    return {
        title: fields.title,
        summary: fields.summary,
        series: fields.series,
        customSeries: fields.customSeries,
        rating: fields.rating,
        language: fields.language,
        customLanguage: fields.customLanguage,
        genreA: fields.genreA,
        genreB: fields.genreB,
        tags: fields.tags,
        status: fields.status,
        characters: fields.characters,
        isPairing: fields.isPairing,
        isOneshot: fields.isOneshot,
        containsLemons: fields.containsLemons,
        body,
        step,
    };
}
