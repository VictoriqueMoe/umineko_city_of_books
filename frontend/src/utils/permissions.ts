export type Permission =
    | "delete_any_theory"
    | "delete_any_response"
    | "ban_user"
    | "manage_roles"
    | "view_admin_panel"
    | "manage_settings"
    | "view_audit_log"
    | "view_stats"
    | "delete_any_user";

const rolePermissions: Record<string, Permission[]> = {
    admin: [
        "delete_any_theory",
        "delete_any_response",
        "ban_user",
        "manage_roles",
        "view_admin_panel",
        "manage_settings",
        "view_audit_log",
        "view_stats",
        "delete_any_user",
    ],
    moderator: ["delete_any_theory", "delete_any_response", "view_admin_panel", "view_stats"],
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
