import type { PaginationFields, SiteRole } from "./common";

export interface UserProfile {
    id: string;
    username: string;
    display_name: string;
    bio: string;
    avatar_url: string;
    banner_url: string;
    banner_position: number;
    favourite_character: string;
    gender: string;
    pronoun_subject: string;
    pronoun_possessive: string;
    role?: SiteRole;
    online: boolean;
    social_twitter: string;
    social_discord: string;
    social_waifulist: string;
    social_tumblr: string;
    social_github: string;
    social_bluesky: string;
    website: string;
    dms_enabled: boolean;
    episode_progress: number;
    higurashi_arc_progress: number;
    ciconia_chapter_progress: number;
    secrets: string[];
    dob?: string;
    dob_public?: boolean;
    email?: string;
    email_public?: boolean;
    created_at: string;
    stats: UserStats;
    is_bot?: boolean;
    banned?: boolean;
    ban_reason?: string;
    locked?: boolean;
    lock_reason?: string;
    private?: UserPrivateFields;
}

export interface SessionResponse {
    authenticated: boolean;
    username?: string;
    permissions?: string[];
}

export interface SessionUser extends UserProfile {
    permissions?: string[];
}

export interface UserPrivateFields {
    display_name_locked?: boolean;
    email_verified?: boolean;
    verify_grace_until?: string;
    email_notifications?: boolean;
    play_message_sound?: boolean;
    play_notification_sound?: boolean;
    follow_activity_notifications?: boolean;
    echoes_enabled?: boolean;
    home_page?: string;
    game_board_sort?: string;
    default_profile_tab?: string;
    theme?: string;
    font?: string;
    wide_layout?: boolean;
    chatbot_opted_in?: boolean;
}

export interface UserStats {
    theory_count: number;
    response_count: number;
    votes_received: number;
    ship_count: number;
    mystery_count: number;
    fanfic_count: number;
}

export interface UpdateProfilePayload {
    display_name: string;
    bio: string;
    avatar_url: string;
    banner_url: string;
    banner_position: number;
    favourite_character: string;
    gender: string;
    pronoun_subject: string;
    pronoun_possessive: string;
    social_twitter: string;
    social_discord: string;
    social_waifulist: string;
    social_tumblr: string;
    social_github: string;
    social_bluesky: string;
    website: string;
    dms_enabled: boolean;
    episode_progress: number;
    higurashi_arc_progress: number;
    ciconia_chapter_progress: number;
    dob: string;
    dob_public: boolean;
    email: string;
    email_password?: string;
    email_public: boolean;
    email_notifications: boolean;
    play_message_sound: boolean;
    play_notification_sound: boolean;
    follow_activity_notifications: boolean;
    echoes_enabled: boolean;
    home_page: string;
    game_board_sort: string;
    default_profile_tab: string;
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

export interface ActivityListResponse extends PaginationFields {
    items: ActivityItem[];
}

export interface FollowStats {
    follower_count: number;
    following_count: number;
    is_following: boolean;
    follows_you: boolean;
}

export interface BlockStatus {
    blocking: boolean;
    blocked_by: boolean;
}

export interface BlockedUserItem {
    id: string;
    username: string;
    display_name: string;
    avatar_url: string;
    blocked_at: string;
}
