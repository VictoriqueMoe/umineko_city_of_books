import type { PaginationFields, User } from "./common";
import type { PostComment } from "./posts";

export interface FanficCharacter {
    series: string;
    character_id?: string;
    character_name: string;
    sort_order: number;
}

export interface Fanfic {
    id: string;
    author: User;
    title: string;
    summary: string;
    series: string;
    rating: string;
    language: string;
    status: string;
    is_oneshot: boolean;
    contains_lemons: boolean;
    cover_image_url?: string;
    cover_thumbnail_url?: string;
    genres: string[];
    tags: string[];
    characters: FanficCharacter[];
    is_pairing: boolean;
    word_count: number;
    chapter_count: number;
    favourite_count: number;
    view_count: number;
    comment_count: number;
    user_favourited: boolean;
    published_at: string;
    created_at: string;
    updated_at?: string;
}

export interface FanficChapterSummary {
    id: string;
    chapter_number: number;
    title: string;
    word_count: number;
}

export interface FanficChapter {
    id: string;
    chapter_number: number;
    title: string;
    body: string;
    word_count: number;
    has_prev: boolean;
    has_next: boolean;
    created_at: string;
    updated_at?: string;
}

export interface FanficDetail extends Fanfic {
    chapters: FanficChapterSummary[];
    comments: PostComment[];
    reading_progress: number;
    viewer_blocked: boolean;
}

export interface FanficListResponse extends PaginationFields {
    fanfics: Fanfic[];
}
