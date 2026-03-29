export interface Quote {
    text: string;
    textHtml: string;
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
    audioCharMap?: Record<string, string>;
    audioTextMap?: Record<string, string>;
}

export interface QuoteBrowseResponse {
    character: string;
    characterId: string;
    quotes: Quote[];
    total: number;
    limit: number;
    offset: number;
}

export interface QuoteSearchResult {
    quote: Quote;
    score: number;
}

export interface QuoteSearchResponse {
    results: QuoteSearchResult[];
    total: number;
    limit: number;
    offset: number;
}

export interface User {
    id: string;
    username: string;
    display_name: string;
    avatar_url?: string;
    role?: string;
}

export interface EvidenceItem {
    id: number;
    audio_id?: string;
    quote_index?: number;
    note: string;
    sort_order: number;
}

export interface Theory {
    id: string;
    title: string;
    body: string;
    episode: number;
    author: User;
    vote_score: number;
    with_love_count: number;
    without_love_count: number;
    user_vote?: number;
    created_at: string;
}

export interface TheoryDetail extends Theory {
    evidence: EvidenceItem[];
    responses: Response[];
}

export interface TheoryListResponse {
    theories: Theory[];
    total: number;
    limit: number;
    offset: number;
}

export interface Response {
    id: string;
    parent_id?: string;
    author: User;
    side: "with_love" | "without_love";
    body: string;
    evidence: EvidenceItem[];
    replies?: Response[];
    vote_score: number;
    user_vote?: number;
    created_at: string;
}

export interface EvidenceInput {
    audio_id?: string;
    quote_index?: number;
    note: string;
}

export interface CreateTheoryPayload {
    title: string;
    body: string;
    episode: number;
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

export interface UserProfile {
    id: string;
    username: string;
    display_name: string;
    bio: string;
    avatar_url: string;
    banner_url: string;
    favourite_character: string;
    gender: string;
    pronoun_subject: string;
    pronoun_possessive: string;
    role?: string;
    online: boolean;
    social_twitter: string;
    social_discord: string;
    social_waifulist: string;
    social_tumblr: string;
    social_github: string;
    website: string;
    created_at: string;
    stats: UserStats;
}

export interface UserStats {
    theory_count: number;
    response_count: number;
    votes_received: number;
}

export interface UpdateProfilePayload {
    display_name: string;
    bio: string;
    avatar_url: string;
    banner_url: string;
    favourite_character: string;
    gender: string;
    pronoun_subject: string;
    pronoun_possessive: string;
    social_twitter: string;
    social_discord: string;
    social_waifulist: string;
    social_tumblr: string;
    social_github: string;
    website: string;
}

export interface ChangePasswordPayload {
    old_password: string;
    new_password: string;
}

export interface DeleteAccountPayload {
    password: string;
}

export interface ActivityItem {
    type: string;
    theory_id: string;
    theory_title: string;
    side?: string;
    body: string;
    created_at: string;
}

export interface ActivityListResponse {
    items: ActivityItem[];
    total: number;
    limit: number;
    offset: number;
}

export type NotificationType = "theory_response" | "response_reply" | "theory_upvote" | "response_upvote";

export interface Notification {
    id: number;
    type: NotificationType;
    reference_id: string;
    theory_id: string;
    theory_title: string;
    actor: User;
    read: boolean;
    created_at: string;
}

export interface NotificationListResponse {
    notifications: Notification[];
    total: number;
    limit: number;
    offset: number;
}

export interface WSMessage {
    type: string;
    data: unknown;
}
