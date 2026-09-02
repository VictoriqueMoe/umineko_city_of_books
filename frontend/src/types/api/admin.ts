import type { PaginationFields, SiteRole, User } from "./common";

export interface AdminUserItem {
    id: string;
    username: string;
    display_name: string;
    avatar_url: string;
    role?: SiteRole;
    banned: boolean;
    locked: boolean;
    created_at: string;
}

export interface AdminUserListResponse extends PaginationFields {
    users: AdminUserItem[];
}

export interface AdminUserDetail extends AdminUserItem {
    email?: string;
    email_verified: boolean;
    display_name_locked: boolean;
    ip?: string;
    ban_reason?: string;
    banned_at?: string;
    banned_by?: User;
    lock_reason?: string;
    locked_at?: string;
    approved_at?: string;
    approved_by?: User;
    restricted: boolean;
    theory_count: number;
    response_count: number;
    mystery_score_adjustment: number;
    detective_score: number;
    gm_score_adjustment: number;
    gm_score: number;
}

export interface AdminStats {
    total_users: number;
    total_theories: number;
    total_responses: number;
    total_votes: number;
    total_posts: number;
    total_comments: number;
    new_users_24h: number;
    new_users_7d: number;
    new_users_30d: number;
    new_theories_24h: number;
    new_theories_7d: number;
    new_theories_30d: number;
    new_responses_24h: number;
    new_responses_7d: number;
    new_responses_30d: number;
    new_posts_24h: number;
    new_posts_7d: number;
    new_posts_30d: number;
    posts_by_corner: Record<string, number>;
    most_active_users: {
        id: string;
        username: string;
        display_name: string;
        avatar_url: string;
        action_count: number;
    }[];
}

export interface AuditLogEntry {
    id: number;
    actor_id: string;
    actor_name: string;
    action: string;
    target_type: string;
    target_id: string;
    details: string;
    created_at: string;
    subject_id?: string;
    subject_name?: string;
    subject_username?: string;
}

export interface AuditLogListResponse extends PaginationFields {
    entries: AuditLogEntry[];
}

export interface AdminIPMatches {
    ip: string;
    users: AdminUserItem[];
}

export interface InviteItem {
    code: string;
    created_by: string;
    used_by?: string;
    used_at?: string;
    created_at: string;
}

export interface InviteListResponse extends PaginationFields {
    invites: InviteItem[];
}

export interface ReportItem {
    id: number;
    reporter_name: string;
    reporter_avatar_url: string;
    target_type: string;
    target_id: string;
    context_id?: string;
    reason: string;
    status: string;
    resolved_by?: string;
    created_at: string;
}

export interface ReportListResponse extends PaginationFields {
    reports: ReportItem[];
}

export interface SiteSettings {
    [key: string]: string;
}
