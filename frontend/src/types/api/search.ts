export type SearchEntityType =
    | "theory"
    | "response"
    | "post"
    | "post_comment"
    | "art"
    | "art_comment"
    | "mystery"
    | "mystery_attempt"
    | "mystery_comment"
    | "ship"
    | "ship_comment"
    | "oc"
    | "oc_comment"
    | "announcement"
    | "announcement_comment"
    | "fanfic"
    | "fanfic_comment"
    | "journal"
    | "journal_entry"
    | "journal_comment"
    | "chat_message"
    | "user";

export interface SearchResultAuthor {
    id: string | null;
    username: string;
    display_name: string;
    avatar_url: string;
}

export interface SearchResult {
    type: SearchEntityType;
    id: string;
    parent_id: string | null;
    parent_title: string | null;
    title: string;
    snippet: string;
    url: string;
    author: SearchResultAuthor;
    created_at: string;
}

export interface SearchResponse {
    results: SearchResult[];
    total: number;
}

export interface QuickSearchResponse {
    results: SearchResult[];
}
