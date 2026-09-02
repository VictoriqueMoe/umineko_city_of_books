import type { CommentBase, PaginationFields, User } from "./common";

export interface Announcement {
    id: string;
    title: string;
    body: string;
    author: User;
    pinned: boolean;
    created_at: string;
    updated_at: string;
    comments?: AnnouncementComment[];
}

export interface AnnouncementComment extends CommentBase {
    replies?: AnnouncementComment[];
}

export interface AnnouncementListResponse extends PaginationFields {
    announcements: Announcement[];
}
