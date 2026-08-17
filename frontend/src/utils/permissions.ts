export type SiteRole = "super_admin" | "admin" | "moderator";

export const ROLE_GROUPS: { role: SiteRole; label: string }[] = [
    { role: "super_admin", label: "Reality Author" },
    { role: "admin", label: "Voyager Witches" },
    { role: "moderator", label: "Witches" },
];

export function isSiteStaff(role: SiteRole | undefined | null): boolean {
    return role === "super_admin" || role === "admin" || role === "moderator";
}

const ALL_PERMISSIONS = [
    "delete_any_theory",
    "delete_any_response",
    "ban_user",
    "manage_roles",
    "view_admin_panel",
    "manage_settings",
    "view_audit_log",
    "view_stats",
    "view_users",
    "delete_any_user",
    "delete_any_post",
    "delete_any_comment",
    "edit_any_theory",
    "edit_any_post",
    "edit_any_comment",
    "resolve_suggestion",
    "edit_mystery_score",
    "edit_any_journal",
    "delete_any_journal",
    "manage_vanity_roles",
    "manage_banned_words",
    "reset_password",
    "manage_user_account",
    "manage_user_email",
    "set_email_verified",
    "use_chatbot",
] as const;

export type Permission = (typeof ALL_PERMISSIONS)[number];

const rolePermissions: Record<string, Permission[]> = {
    super_admin: [...ALL_PERMISSIONS],
    admin: [...ALL_PERMISSIONS],
    moderator: [
        "delete_any_theory",
        "delete_any_response",
        "delete_any_post",
        "delete_any_comment",
        "edit_any_theory",
        "edit_any_post",
        "edit_any_comment",
        "view_admin_panel",
        "view_stats",
        "view_users",
        "ban_user",
        "edit_mystery_score",
        "edit_any_journal",
        "delete_any_journal",
        "manage_user_account",
        "use_chatbot",
    ],
};

export interface PermissionSubject {
    role?: SiteRole | null;
    permissions?: string[] | null;
}

export type CanSubject = SiteRole | PermissionSubject | undefined | null;

export function can(subject: CanSubject, perm: Permission): boolean {
    if (!subject) {
        return false;
    }

    if (typeof subject === "string") {
        return rolePermissions[subject]?.includes(perm) ?? false;
    }

    if (subject.permissions) {
        return subject.permissions.includes(perm);
    }

    if (!subject.role) {
        return false;
    }

    return rolePermissions[subject.role]?.includes(perm) ?? false;
}

export function canAccessAdmin(subject: CanSubject): boolean {
    return can(subject, "view_admin_panel");
}
