import type { CommentBase, PaginationFields, User } from "./common";

export interface ShipCharacter {
    series: string;
    character_id?: string;
    character_name: string;
    sort_order: number;
}

export interface Ship {
    id: string;
    author: User;
    title: string;
    description: string;
    image_url?: string;
    thumbnail_url?: string;
    characters: ShipCharacter[];
    vote_score: number;
    user_vote?: number;
    comment_count: number;
    is_crackship: boolean;
    created_at: string;
    updated_at?: string;
}

export interface ShipComment extends CommentBase {
    replies?: ShipComment[];
}

export interface ShipDetail extends Ship {
    comments: ShipComment[];
    viewer_blocked: boolean;
}

export interface ShipListResponse extends PaginationFields {
    ships: Ship[];
}
