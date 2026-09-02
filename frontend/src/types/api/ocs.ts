import type { CommentBase, PaginationFields, User } from "./common";

export interface OCImage {
    id: number;
    image_url: string;
    thumbnail_url?: string;
    caption?: string;
    sort_order: number;
}

export interface OC {
    id: string;
    author: User;
    name: string;
    description: string;
    series: string;
    custom_series_name?: string;
    image_url?: string;
    thumbnail_url?: string;
    gallery: OCImage[];
    vote_score: number;
    user_vote?: number;
    favourite_count: number;
    user_favourited: boolean;
    comment_count: number;
    is_crack_oc: boolean;
    created_at: string;
    updated_at?: string;
}

export interface OCComment extends CommentBase {
    replies?: OCComment[];
}

export interface OCDetail extends OC {
    comments: OCComment[];
    viewer_blocked: boolean;
}

export interface OCListResponse extends PaginationFields {
    ocs: OC[];
}

export interface OCSummary {
    id: string;
    name: string;
    series: string;
    custom_series_name?: string;
    thumbnail_url?: string;
}
