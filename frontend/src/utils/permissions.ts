export type Permission =
    | "delete_any_theory"
    | "delete_any_response"
    | "ban_user"
    | "manage_roles"
    | "view_admin_panel"
    | "manage_settings"
    | "view_audit_log"
    | "view_stats"
    | "view_users"
    | "delete_any_user"
    | "delete_any_post"
    | "delete_any_comment"
    | "edit_any_theory"
    | "edit_any_post"
    | "edit_any_comment"
    | "resolve_suggestion"
    | "edit_mystery_score"
    | "edit_any_journal"
    | "delete_any_journal";

const rolePermissions: Record<string, Permission[]> = {
    super_admin: [
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
    ],
    admin: [
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
    ],
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
    ],
};

export function can(role: string | undefined, perm: Permission): boolean {
    if (!role) {
        return false;
    }
    return rolePermissions[role]?.includes(perm) ?? false;
}

export function canAccessAdmin(role: string | undefined): boolean {
    return can(role, "view_admin_panel");
}
