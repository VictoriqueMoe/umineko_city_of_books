export const AUDIT_ACTION_LABELS: Record<string, string> = {
    user_created: "Account created",
    set_role: "Role assigned",
    remove_role: "Role removed",
    ban_user: "Banned",
    unban_user: "Unbanned",
    lock_user: "Locked",
    unlock_user: "Unlocked",
    delete_user: "Account deleted",
    reset_password: "Password reset by staff",
    password_reset: "Password reset via email link",
    set_user_email: "Email changed",
    verify_user_email: "Email marked verified",
    unverify_user_email: "Email marked unverified",
    set_display_name: "Display name changed",
    lock_display_name: "Display name locked",
    unlock_display_name: "Display name unlocked",
    force_logout: "All sessions revoked",
    chat_room_ban: "Banned from a chat room",
    chat_room_unban: "Unbanned from a chat room",
    chat_word_filter_kick: "Kicked by the word filter",
    chat_word_filter_delete: "Message deleted by the word filter",
    chat_room_banned_word_create: "Room word filter rule added",
    chat_room_banned_word_update: "Room word filter rule changed",
    chat_room_banned_word_delete: "Room word filter rule removed",
    chat_global_banned_word_create: "Global word filter rule added",
    chat_global_banned_word_update: "Global word filter rule changed",
    chat_global_banned_word_delete: "Global word filter rule removed",
    "watch_party.start": "Watch party started",
    "watch_party.end": "Watch party ended",
    "watch_party.kick": "Kicked from a watch party",
    "watch_party.grant_control": "Given watch party control",
    post_comment_delete: "Post comment deleted",
    post_comment_delete_admin: "Post comment deleted by staff",
    art_comment_delete: "Art comment deleted",
    art_comment_delete_admin: "Art comment deleted by staff",
    fanfic_comment_delete: "Fanfic comment deleted",
    fanfic_comment_delete_admin: "Fanfic comment deleted by staff",
    mystery_comment_delete: "Mystery comment deleted",
    mystery_comment_delete_admin: "Mystery comment deleted by staff",
    journal_comment_delete: "Journal comment deleted",
    journal_comment_delete_admin: "Journal comment deleted by staff",
    ship_comment_delete: "Ship comment deleted",
    ship_comment_delete_admin: "Ship comment deleted by staff",
    secret_comment_delete: "Secret comment deleted",
    secret_comment_delete_admin: "Secret comment deleted by staff",
    announcement_comment_delete: "Announcement comment deleted",
    announcement_comment_delete_admin: "Announcement comment deleted by staff",
    assign_vanity_role: "Vanity role assigned",
    unassign_vanity_role: "Vanity role removed",
    create_vanity_role: "Vanity role created",
    update_vanity_role: "Vanity role changed",
    delete_vanity_role: "Vanity role deleted",
    update_role_permissions: "Role permissions changed",
    update_vanity_role_permissions: "Vanity role permissions changed",
    update_settings: "Site settings changed",
    send_test_email: "Test email sent",
    create_invite: "Invite created",
    delete_invite: "Invite deleted",
    change_password: "Password changed by the member",
    change_email: "Email changed by the member",
    delete_account: "Account deleted by the member",
    login_banned: "Banned member tried to sign in",
    chat_message_delete: "Chat message deleted",
    chat_message_delete_mod: "Chat message deleted by a moderator",
    chat_room_delete: "Chat room deleted",
    chat_room_update: "Chat room settings changed",
    chat_room_kick: "Kicked from a chat room",
    chat_room_timeout: "Timed out in a chat room",
    post_delete: "Post deleted",
    post_delete_admin: "Post deleted by staff",
    art_delete: "Art deleted",
    art_delete_admin: "Art deleted by staff",
    gallery_delete: "Gallery deleted",
    fanfic_delete: "Fanfiction deleted",
    fanfic_delete_admin: "Fanfiction deleted by staff",
    fanfic_update_admin: "Fanfiction edited by staff",
    fanfic_chapter_delete: "Chapter deleted",
    fanfic_chapter_delete_admin: "Chapter deleted by staff",
    fanfic_chapter_update_admin: "Chapter edited by staff",
    journal_delete: "Journal deleted",
    journal_delete_admin: "Journal deleted by staff",
    journal_update_admin: "Journal edited by staff",
    journal_entry_delete: "Journal entry deleted",
    journal_entry_delete_admin: "Journal entry deleted by staff",
    journal_entry_create_admin: "Journal entry written by staff",
    journal_entry_update_admin: "Journal entry edited by staff",
    ship_delete: "Ship deleted",
    ship_delete_admin: "Ship deleted by staff",
    ship_update_admin: "Ship edited by staff",
    oc_delete: "OC deleted",
    oc_delete_admin: "OC deleted by staff",
    oc_comment_delete: "OC comment deleted",
    oc_comment_delete_admin: "OC comment deleted by staff",
    post_comment_update_admin: "Post comment edited by staff",
    art_comment_update_admin: "Art comment edited by staff",
    fanfic_comment_update_admin: "Fanfic comment edited by staff",
    mystery_comment_update_admin: "Mystery comment edited by staff",
    journal_comment_update_admin: "Journal comment edited by staff",
    ship_comment_update_admin: "Ship comment edited by staff",
    secret_comment_update_admin: "Secret comment edited by staff",
    announcement_comment_update_admin: "Announcement comment edited by staff",
    oc_comment_update_admin: "OC comment edited by staff",
    mystery_solved: "Mystery solved",
    mystery_closed: "Mystery closed",
    mystery_delete: "Mystery deleted",
    mystery_delete_admin: "Mystery deleted by staff",
    mystery_update_admin: "Mystery edited by staff",
    mystery_clue_update: "Mystery truth edited",
    mystery_clue_delete: "Mystery truth deleted",
    mystery_attempt_delete_admin: "Mystery attempt deleted by staff",
    theory_refuted: "Theory refuted",
    theory_delete: "Theory deleted",
    theory_delete_admin: "Theory deleted by staff",
    theory_update_admin: "Theory edited by staff",
    theory_response_delete_admin: "Theory response deleted by staff",
    mystery_score_adjust: "Detective score adjusted",
    gm_score_adjust: "Game Master score adjusted",
    announcement_create: "Announcement posted",
    announcement_update: "Announcement edited",
    announcement_delete: "Announcement deleted",
    announcement_pin: "Announcement pinned or unpinned",
    chatbot_create: "Chatbot created",
    chatbot_update: "Chatbot changed",
    chatbot_delete: "Chatbot deleted",
    chatbot_base_prompt_create: "Base prompt created",
    chatbot_base_prompt_update: "Base prompt changed",
    chatbot_base_prompt_delete: "Base prompt deleted",
    chatbot_opt_in_role_migrate: "Chatbot opt-in role migrated",
    banned_gif_create: "Gif banned",
    banned_gif_delete: "Gif ban lifted",
};

const TARGET_TYPE_LABELS: Record<string, string> = {
    user: "User",
    chat_room: "Chat room",
    chat_watch_party_session: "Watch party",
    banned_word: "Word filter rule",
    role: "Role",
    vanity_role: "Vanity role",
    settings: "Settings",
    invite: "Invite",
    post_comment: "Post comment",
    art_comment: "Art comment",
    journal_comment: "Journal comment",
    mystery_comment: "Mystery comment",
    fanfic_comment: "Fanfic comment",
    secret_comment: "Secret comment",
    ship_comment: "Ship comment",
    announcement_comment: "Announcement comment",
    oc_comment: "OC comment",
    post: "Post",
    art: "Art",
    gallery: "Gallery",
    fanfic: "Fanfiction",
    fanfic_chapter: "Chapter",
    journal: "Journal",
    journal_entry: "Journal entry",
    ship: "Ship",
    oc: "OC",
    announcement: "Announcement",
    mystery: "Mystery",
    mystery_attempt: "Mystery attempt",
    theory: "Theory",
    theory_response: "Theory response",
    chatbot: "Chatbot",
    chatbot_base_prompt: "Base prompt",
    banned_gif: "Banned gif",
};

export function auditActionLabel(action: string): string {
    return AUDIT_ACTION_LABELS[action] ?? action.replace(/[._]/g, " ");
}

export function auditTargetLabel(targetType: string): string {
    return TARGET_TYPE_LABELS[targetType] ?? targetType.replace(/_/g, " ");
}

export function shortId(id: string): string {
    if (id.length <= 8) {
        return id;
    }
    return `${id.slice(0, 8)}...`;
}

export interface AuditDetailPart {
    key: string;
    value: string;
}

function unquote(value: string): string {
    const trimmed = value.trim();
    if (trimmed.length >= 2 && trimmed.startsWith('"') && trimmed.endsWith('"')) {
        return trimmed.slice(1, -1).replace(/\\"/g, '"').replace(/\\\\/g, "\\");
    }
    return trimmed;
}

export function parseAuditDetails(details: string): AuditDetailPart[] {
    const raw = details.trim();
    if (!raw) {
        return [];
    }

    if (raw.startsWith("{")) {
        try {
            const parsed: unknown = JSON.parse(raw);
            if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
                return Object.entries(parsed as Record<string, unknown>).map(([key, value]) => ({
                    key,
                    value: String(value),
                }));
            }
        } catch {
            return [{ key: "", value: raw }];
        }
        return [{ key: "", value: raw }];
    }

    const keyPattern = /(?:^|\s)([a-z_]+)=/g;
    const starts: { key: string; from: number; valueFrom: number }[] = [];
    for (let m = keyPattern.exec(raw); m !== null; m = keyPattern.exec(raw)) {
        starts.push({ key: m[1], from: m.index, valueFrom: m.index + m[0].length });
    }

    if (starts.length === 0) {
        return [{ key: "", value: raw }];
    }

    const parts: AuditDetailPart[] = [];
    for (let i = 0; i < starts.length; i++) {
        const end = i + 1 < starts.length ? starts[i + 1].from : raw.length;
        const value = unquote(raw.slice(starts[i].valueFrom, end));
        if (value) {
            parts.push({ key: starts[i].key, value });
        }
    }
    return parts;
}
