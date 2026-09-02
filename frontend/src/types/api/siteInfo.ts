import type { VanityRoleDefinition } from "./common";

export interface SiteInfoSecretPiece {
    id: string;
    letter?: string;
    tile?: number;
}

export interface SiteInfoSecret {
    id: string;
    title: string;
    description: string;
    vanity_role_id?: string;
    icon?: string;
    pointer?: string;
    solved_message?: string;
    ready_placeholder?: string;
    pending_hint?: string;
    solved: boolean;
    pieces: SiteInfoSecretPiece[];
}

export interface WebPushConfig {
    vapid_key: string;
    api_key: string;
    project_id: string;
    sender_id: string;
    app_id: string;
}

export interface OtaManifest {
    version: string;
    path: string;
    checksum: string;
    session_key: string;
}

export interface SiteInfo {
    site_name: string;
    site_description: string;
    registration_type: string;
    announcement_banner: string;
    default_theme: string;
    maintenance_mode: boolean;
    maintenance_title: string;
    maintenance_message: string;
    turnstile_enabled: boolean;
    turnstile_site_key: string;
    voice_enabled: boolean;
    email_enabled: boolean;
    chatbot_enabled: boolean;
    chatbot_require_permission: boolean;
    chatbot_context_messages: number;
    chatbot_max_reply_chain: number;
    max_image_size: number;
    max_video_size: number;
    private_mode: boolean;
    max_audio_size: number;
    new_account_hours: number;
    top_detective_ids: string[];
    top_gm_ids: string[];
    top_chess_ids: string[];
    top_checkers_ids: string[];
    top_othello_ids: string[];
    top_minesweeper_ids: string[];
    vanity_roles: VanityRoleDefinition[];
    vanity_role_assignments: Record<string, string[]>;
    listed_secrets: SiteInfoSecret[];
    rules_page: string;
    version: string;
    app_latest_version: string;
    app_download_url: string;
    push_enabled: boolean;
    web_push: WebPushConfig;
}
