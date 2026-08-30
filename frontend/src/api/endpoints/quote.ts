import { buildQueryString } from "../client";
import type { CharacterGroups, Quote, QuoteBrowseResponse, QuoteSearchResponse, Series } from "../../types/api";

const QUOTE_API = "https://quotes.auaurora.moe/api/v1";

export async function searchQuotes(params: {
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
}): Promise<QuoteSearchResponse> {
    const series = params.series ?? "umineko";
    const qs = buildQueryString({
        q: params.query,
        character: params.character,
        episode: params.episode,
        arc: params.arc,
        chapter: params.chapter,
        truth: params.truth,
        lang: series === "umineko" ? params.lang : undefined,
        limit: params.limit ?? 30,
        offset: params.offset,
    });
    const response = await fetch(`${QUOTE_API}/${series}/search${qs}`);
    if (!response.ok) {
        throw new Error(`Quote API error: ${response.status}`);
    }
    return response.json();
}

export async function browseQuotes(params: {
    character?: string;
    episode?: number;
    truth?: string;
    arc?: string;
    chapter?: string;
    lang?: string;
    limit?: number;
    offset?: number;
    series?: Series;
}): Promise<QuoteBrowseResponse> {
    const series = params.series ?? "umineko";
    const qs = buildQueryString({
        character: params.character,
        episode: params.episode,
        truth: params.truth,
        arc: params.arc,
        chapter: params.chapter,
        lang: series === "umineko" ? params.lang : undefined,
        limit: params.limit ?? 30,
        offset: params.offset,
    });
    const response = await fetch(`${QUOTE_API}/${series}/browse${qs}`);
    if (!response.ok) {
        throw new Error(`Quote API error: ${response.status}`);
    }
    return response.json();
}

export async function tryGetQuoteByAudioId(series: Series, audioId: string, lang?: string): Promise<Quote | null> {
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

export async function tryGetQuoteByIndex(series: Series, index: number, lang?: string): Promise<Quote | null> {
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

export async function getCharacters(series: Series = "umineko"): Promise<Record<string, string>> {
    const groups = await getCharacterGroups(series);
    return { ...groups.main, ...groups.additional };
}

export async function getCharacterGroups(series: Series = "umineko"): Promise<CharacterGroups> {
    const response = await fetch(`${QUOTE_API}/${series}/characters`);
    if (!response.ok) {
        throw new Error(`Quote API error: ${response.status}`);
    }
    const data = await response.json();
    return {
        main: data.characters ?? {},
        additional: data.additional ?? {},
    };
}
