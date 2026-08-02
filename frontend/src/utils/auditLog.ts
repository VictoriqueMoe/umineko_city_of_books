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
    assign_vanity_role: "Vanity role assigned",
    unassign_vanity_role: "Vanity role removed",
    create_vanity_role: "Vanity role created",
    update_vanity_role: "Vanity role changed",
    delete_vanity_role: "Vanity role deleted",
    update_settings: "Site settings changed",
    send_test_email: "Test email sent",
    create_invite: "Invite created",
    delete_invite: "Invite deleted",
};

const TARGET_TYPE_LABELS: Record<string, string> = {
    user: "User",
    chat_room: "Chat room",
    chat_watch_party_session: "Watch party",
    banned_word: "Word filter rule",
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
