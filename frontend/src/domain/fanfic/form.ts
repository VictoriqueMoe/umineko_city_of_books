import type { FanficDetail, ShipCharacter } from "../../types/api";

export const GENRES = [
    "Adventure",
    "Angst",
    "Crime",
    "Drama",
    "Family",
    "Fantasy",
    "Friendship",
    "General",
    "Horror",
    "Humour",
    "Hurt/Comfort",
    "Mystery",
    "Parody",
    "Poetry",
    "Romance",
    "Sci-Fi",
    "Spiritual",
    "Supernatural",
    "Suspense",
    "Tragedy",
    "Western",
];

export const PINNED_SERIES = ["Umineko", "Higurashi", "Ciconia"];
export const OTHER_VALUE = "__other__";

export const MAX_TAGS = 10;
export const MAX_TAG_LENGTH = 30;

export interface EditorFields {
    title: string;
    summary: string;
    rating: string;
    isOneshot: boolean;
    containsLemons: boolean;
    isPairing: boolean;
    status: string;
    characters: ShipCharacter[];
    genreA: string;
    genreB: string;
    tags: string[];
    series: string;
    customSeries: string;
    language: string;
    customLanguage: string;
    coverPreview: string;
}

export interface FanficSeed extends EditorFields {
    chapterCount: number;
}

export interface FanficDetailsPayload {
    title: string;
    summary: string;
    series: string;
    rating: string;
    language: string;
    status: string;
    is_oneshot: boolean;
    contains_lemons: boolean;
    genres: string[];
    tags: string[];
    characters: ShipCharacter[];
    is_pairing: boolean;
}

export function fieldDefaults(): EditorFields {
    return {
        title: "",
        summary: "",
        rating: "K",
        isOneshot: true,
        containsLemons: false,
        isPairing: false,
        status: "in_progress",
        characters: [],
        genreA: "",
        genreB: "",
        tags: [],
        series: "Umineko",
        customSeries: "",
        language: "English",
        customLanguage: "",
        coverPreview: "",
    };
}

export function mergeDefined<T extends object>(base: T, ...overrides: (Partial<T> | null | undefined)[]): T {
    const merged = { ...base };

    for (const override of overrides) {
        if (!override) {
            continue;
        }

        for (const key of Object.keys(base) as (keyof T)[]) {
            const value = override[key];
            if (value !== undefined) {
                merged[key] = value;
            }
        }
    }

    return merged;
}

export function seedFromFanfic(
    fanfic: FanficDetail | null | undefined,
    availableLanguages: string[],
): FanficSeed | null {
    if (!fanfic) {
        return null;
    }

    let seedSeries = fanfic.series;
    let seedCustomSeries = "";
    if (!PINNED_SERIES.includes(fanfic.series)) {
        seedSeries = OTHER_VALUE;
        seedCustomSeries = fanfic.series;
    }

    let seedLanguage = fanfic.language;
    let seedCustomLanguage = "";
    if (availableLanguages.length > 0 && !availableLanguages.includes(fanfic.language)) {
        seedLanguage = OTHER_VALUE;
        seedCustomLanguage = fanfic.language;
    }

    return {
        title: fanfic.title,
        summary: fanfic.summary,
        rating: fanfic.rating,
        isOneshot: fanfic.is_oneshot,
        containsLemons: fanfic.contains_lemons,
        isPairing: fanfic.is_pairing,
        status: fanfic.status,
        characters:
            fanfic.characters?.map((c, i) => ({
                series: c.series,
                character_id: c.character_id,
                character_name: c.character_name,
                sort_order: i,
            })) ?? [],
        genreA: fanfic.genres?.[0] ?? "",
        genreB: fanfic.genres?.[1] ?? "",
        tags: fanfic.tags ?? [],
        series: seedSeries,
        customSeries: seedCustomSeries,
        language: seedLanguage,
        customLanguage: seedCustomLanguage,
        coverPreview: fanfic.cover_image_url ?? "",
        chapterCount: fanfic.chapter_count ?? 0,
    };
}

export function seriesOptions(dynamicSeries: string[]): string[] {
    return [...PINNED_SERIES, ...dynamicSeries.filter(s => !PINNED_SERIES.includes(s))];
}

export function resolveSeries(fields: Pick<EditorFields, "series" | "customSeries">): string {
    return fields.series === OTHER_VALUE ? fields.customSeries.trim() : fields.series;
}

export function resolveLanguage(fields: Pick<EditorFields, "language" | "customLanguage">): string {
    return fields.language === OTHER_VALUE ? fields.customLanguage.trim() : fields.language;
}

export function collapseGenres(genreA: string, genreB: string): string[] {
    const genres: string[] = [];

    if (genreA) {
        genres.push(genreA);
    }

    if (genreB && genreB !== genreA) {
        genres.push(genreB);
    }

    return genres;
}

export function validateDetails(fields: EditorFields): string {
    if (!fields.title.trim()) {
        return "Title is required";
    }

    if (!resolveSeries(fields)) {
        return "Series is required";
    }

    if (!resolveLanguage(fields)) {
        return "Language is required";
    }

    return "";
}

export function toPayload(fields: EditorFields): FanficDetailsPayload {
    return {
        title: fields.title.trim(),
        summary: fields.summary.trim(),
        series: resolveSeries(fields),
        rating: fields.rating,
        language: resolveLanguage(fields),
        status: fields.status,
        is_oneshot: fields.isOneshot,
        contains_lemons: fields.containsLemons,
        genres: collapseGenres(fields.genreA, fields.genreB),
        tags: fields.tags,
        characters: fields.characters,
        is_pairing: fields.isPairing,
    };
}

export function withCharacter(characters: ShipCharacter[], character: ShipCharacter): ShipCharacter[] {
    return [...characters, { ...character, sort_order: characters.length }];
}

export function withoutCharacter(characters: ShipCharacter[], index: number): ShipCharacter[] {
    return characters.filter((_, i) => i !== index).map((c, i) => ({ ...c, sort_order: i }));
}

export function withTag(tags: string[], raw: string): string[] {
    const value = raw.trim().slice(0, MAX_TAG_LENGTH);

    if (!value || tags.length >= MAX_TAGS || tags.some(t => t.toLowerCase() === value.toLowerCase())) {
        return tags;
    }

    return [...tags, value];
}

export function withoutTag(tags: string[], index: number): string[] {
    return tags.filter((_, i) => i !== index);
}
