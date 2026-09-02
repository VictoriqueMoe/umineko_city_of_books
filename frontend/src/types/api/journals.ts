import type { CommentBase, PaginationFields, PostMedia, User } from "./common";

export type JournalWork = "general" | "umineko" | "higurashi" | "ciconia" | "higanbana" | "roseguns";

export interface Journal {
    id: string;
    title: string;
    work: JournalWork;
    author: User;
    follower_count: number;
    is_following: boolean;
    is_archived: boolean;
    is_paused: boolean;
    comment_count: number;
    entry_count: number;
    latest_entry_number?: number;
    latest_entry_title?: string | null;
    latest_entry_excerpt: string;
    latest_entry_at?: string;
    created_at: string;
    updated_at?: string;
    last_author_activity_at: string;
    archived_at?: string;
}

export interface JournalComment extends CommentBase {
    replies?: JournalComment[];
    is_author: boolean;
    entry_id?: string;
}

export interface JournalEntrySummary {
    id: string;
    entry_number: number;
    title?: string | null;
    word_count: number;
    is_draft: boolean;
    created_at: string;
}

export interface JournalEntry {
    id: string;
    journal_id: string;
    entry_number: number;
    title?: string | null;
    body: string;
    word_count: number;
    is_draft: boolean;
    has_prev: boolean;
    has_next: boolean;
    created_at: string;
    updated_at?: string;
    media: PostMedia[];
}

export interface JournalDetail extends Journal {
    entries: JournalEntrySummary[];
    latest_entry?: JournalEntry | null;
    comments: JournalComment[];
}

export interface JournalListResponse extends PaginationFields {
    journals: Journal[];
}

export interface CreateJournalPayload {
    title: string;
    work: JournalWork;
}

export interface JournalEntryPayload {
    title: string;
    body: string;
    is_draft: boolean;
}
