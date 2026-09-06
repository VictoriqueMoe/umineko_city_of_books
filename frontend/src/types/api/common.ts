export interface PaginationFields {
    total: number;
    limit: number;
    offset: number;
}

export type Series = "umineko" | "higurashi" | "ciconia";

export type SiteRole = "super_admin" | "admin" | "moderator";

export interface VanityRoleDefinition {
    id: string;
    label: string;
    color: string;
    is_system: boolean;
    sort_order: number;
}

export interface User {
    id: string;
    username: string;
    display_name: string;
    avatar_url?: string;
    created_at?: string;
    role?: SiteRole;
    vanity_roles?: VanityRoleDefinition[];
    banned?: boolean;
    ban_reason?: string;
    locked?: boolean;
    lock_reason?: string;
}

export interface PublicUser extends User {
    online: boolean;
}

export interface PostMedia {
    id: number;
    media_url: string;
    media_type: "image" | "video" | "audio";
    thumbnail_url?: string;
    filename?: string;
    sort_order: number;
    width?: number;
    height?: number;
    is_spoiler?: boolean;
}

export type LinkPreviewType = "" | "link" | "youtube" | "image" | "video";

export interface LinkPreview {
    url: string;
    type: LinkPreviewType;
    title?: string;
    description?: string;
    image?: string;
    site_name?: string;
    video_id?: string;
}

export interface CommentBase {
    id: string;
    parent_id?: string;
    author: User;
    body: string;
    media: PostMedia[];
    like_count: number;
    user_liked: boolean;
    replies?: CommentBase[];
    created_at: string;
    updated_at?: string;
}

export interface WSMessage {
    type: string;
    data: unknown;
}
