import type { CommentBase, PaginationFields, PostMedia, User } from "./common";

export interface PollOption {
    id: number;
    label: string;
    vote_count: number;
    percent: number;
}

export interface Poll {
    id: string;
    options: PollOption[];
    total_votes: number;
    user_voted_option: number | null;
    expired: boolean;
    expires_at: string;
    duration_seconds: number;
}

export interface CreatePollPayload {
    options: { label: string }[];
    duration_seconds: number;
}

export interface SharedContentPreview {
    id: string;
    content_type: string;
    title?: string;
    body?: string;
    image_url?: string;
    media?: PostMedia[];
    author?: User;
    deleted: boolean;
    url: string;
    difficulty?: string;
    solved?: boolean;
    series?: string;
    vote_score?: number;
    credibility_score?: number;
    rating?: string;
    word_count?: number;
    chapter_count?: number;
    corner?: string;
    like_count?: number;
    comment_count?: number;
}

export interface Post {
    id: string;
    author: User;
    body: string;
    media: PostMedia[];
    poll?: Poll;
    shared_content?: SharedContentPreview;
    share_count: number;
    like_count: number;
    comment_count: number;
    view_count: number;
    user_liked: boolean;
    resolved_status?: string;
    created_at: string;
    updated_at?: string;
}

export interface PostDetail extends Post {
    comments: PostComment[];
    liked_by: User[];
    viewer_blocked: boolean;
}

export interface PostComment extends CommentBase {
    replies?: PostComment[];
}

export interface PostListResponse extends PaginationFields {
    posts: Post[];
}

export type FeedTab = "following" | "everyone";
