import type { PaginationFields, User } from "./common";

export interface EvidenceItem {
    id: number;
    audio_id?: string;
    quote_index?: number;
    note: string;
    lang: string;
    sort_order: number;
}

export interface Theory {
    id: string;
    title: string;
    body: string;
    episode: number;
    series: string;
    author: User;
    vote_score: number;
    with_love_count: number;
    without_love_count: number;
    user_vote?: number;
    credibility_score: number;
    status: "open" | "contested" | "refuted";
    created_at: string;
}

export interface TheoryDetail extends Theory {
    evidence: EvidenceItem[];
    responses: TheoryResponse[];
    refuted_by_response_id?: string;
    refuted_by?: User;
    refuted_at?: string;
}

export interface TheoryListResponse extends PaginationFields {
    theories: Theory[];
}

export interface TheoryResponse {
    id: string;
    parent_id?: string;
    author: User;
    side: "with_love" | "without_love";
    body: string;
    evidence: EvidenceItem[];
    replies?: TheoryResponse[];
    vote_score: number;
    user_vote?: number;
    created_at: string;
}

export interface EvidenceInput {
    audio_id?: string;
    quote_index?: number;
    note: string;
    lang?: string;
}

export interface CreateTheoryPayload {
    title: string;
    body: string;
    episode: number;
    series: string;
    evidence: EvidenceInput[];
}

export interface CreateResponsePayload {
    parent_id?: string;
    side: "with_love" | "without_love";
    body: string;
    evidence: EvidenceInput[];
}

export interface VotePayload {
    value: number;
}
