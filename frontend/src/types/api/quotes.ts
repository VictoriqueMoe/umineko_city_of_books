import type { PaginationFields } from "./common";

export interface Quote {
    text: string;
    textHtml: string;
    textJp?: string;
    textJpHtml?: string;
    characterId: string;
    character: string;
    audioId: string;
    episode: number;
    contentType: string;
    hasRedTruth: boolean;
    hasBlueTruth: boolean;
    hasGoldTruth: boolean;
    hasPurpleTruth: boolean;
    index: number;
    arc?: string;
    audioCharMap?: Record<string, string>;
    audioTextMap?: Record<string, string>;
}

export interface QuoteBrowseResponse extends PaginationFields {
    character: string;
    characterId: string;
    quotes: Quote[];
}

export interface QuoteSearchResult {
    quote: Quote;
    score: number;
}

export interface QuoteSearchResponse extends PaginationFields {
    results: QuoteSearchResult[];
}

export interface CharacterGroups {
    main: Record<string, string>;
    additional: Record<string, string>;
}

export interface CharacterListEntry {
    id: string;
    name: string;
    group?: "main" | "additional";
}

export interface CharacterListResponse {
    series: string;
    characters: CharacterListEntry[];
}
