import type { PaginationFields, PostMedia, User } from "./common";

export interface ChatRoom {
    id: string;
    name: string;
    description: string;
    type: "dm" | "group";
    is_public: boolean;
    is_rp: boolean;
    is_system: boolean;
    system_kind?: string;
    tags: string[];
    viewer_role?: string;
    viewer_muted: boolean;
    viewer_ghost: boolean;
    is_member: boolean;
    member_count: number;
    hot_score: number;
    members: User[];
    created_at: string;
    last_message_at?: string;
    archived_at?: string;
    unread?: boolean;
    voice_count?: number;
    voice_participants?: string[];
}

export interface UpdateGroupRoomRequest {
    name: string;
    description: string;
    tags: string[];
    is_public: boolean;
    is_rp: boolean;
    confirm_bot_removal: boolean;
}

export interface BotsWillBeKickedResponse {
    error: string;
    code: string;
    bots: User[];
}

export interface ChatRoomMember {
    user: User;
    role: string;
    joined_at: string;
    nickname: string;
    member_avatar_url: string;
    nickname_locked: boolean;
    timeout_until?: string;
    timeout_set_by_staff?: boolean;
    presence?: "active" | "idle" | "";
    ghost?: boolean;
}

export interface ChatRoomBan {
    user: User;
    banned_by?: User;
    reason: string;
    created_at: string;
}

export interface VoiceTokenResponse {
    token: string;
    url: string;
}

export type BannedWordMatchMode = "substring" | "whole_word" | "regex";
export type BannedWordAction = "delete" | "kick";

export interface BannedWordRule {
    id: string;
    scope: "global" | "room";
    room_id?: string;
    pattern: string;
    match_mode: BannedWordMatchMode;
    case_sensitive: boolean;
    action: BannedWordAction;
    created_by_id?: string;
    created_by_name?: string;
    created_at: string;
}

export interface CreateBannedWordRequest {
    pattern: string;
    match_mode: BannedWordMatchMode;
    case_sensitive: boolean;
    action: BannedWordAction;
}

export interface ChatMessageReplyPreview {
    id: string;
    sender_id: string;
    sender_name: string;
    body_preview: string;
}

export interface ReactionGroup {
    emoji: string;
    count: number;
    viewer_reacted: boolean;
    display_names: string[];
}

export interface ChatMessage {
    id: string;
    room_id: string;
    sender: User;
    body: string;
    is_system: boolean;
    created_at: string;
    media?: PostMedia[];
    reply_to?: ChatMessageReplyPreview;
    pinned: boolean;
    pinned_at?: string;
    pinned_by?: string;
    edited_at?: string;
    reactions: ReactionGroup[];
    sender_nickname?: string;
    sender_member_avatar_url?: string;
}

export interface ChatMessageListResponse extends PaginationFields {
    messages: ChatMessage[];
}
