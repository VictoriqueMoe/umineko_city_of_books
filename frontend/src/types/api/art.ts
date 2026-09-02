import type { CommentBase, PaginationFields, User } from "./common";

export interface Art {
    id: string;
    author: User;
    corner: string;
    art_type: string;
    title: string;
    description: string;
    image_url: string;
    thumbnail_url: string;
    gallery_id?: string;
    tags: string[];
    like_count: number;
    comment_count: number;
    view_count: number;
    user_liked: boolean;
    is_spoiler: boolean;
    created_at: string;
    updated_at?: string;
}

export interface ArtDetail extends Art {
    comments: ArtComment[];
    liked_by: User[];
    viewer_blocked: boolean;
}

export interface ArtComment extends CommentBase {
    replies?: ArtComment[];
}

export interface ArtListResponse extends PaginationFields {
    art: Art[];
}

export interface TagCount {
    tag: string;
    count: number;
}

export interface Gallery {
    id: string;
    author: User;
    name: string;
    description: string;
    cover_image_url: string;
    cover_thumbnail_url: string;
    preview_images?: { thumbnail_url: string; full_url: string }[];
    art_count: number;
    created_at: string;
    updated_at?: string;
}

export interface GalleryDetailResponse extends PaginationFields {
    gallery: Gallery;
    art: Art[];
}
