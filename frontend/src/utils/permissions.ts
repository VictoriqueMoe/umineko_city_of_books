type Permission =
    | "delete_any_theory"
    | "delete_any_response"
    | "ban_user"
    | "manage_roles";

const rolePermissions: Record<string, Permission[]> = {
    admin: ["delete_any_theory", "delete_any_response", "ban_user", "manage_roles"],
    moderator: ["delete_any_theory", "delete_any_response"],
};

export function can(role: string | undefined, perm: Permission): boolean {
    if (!role) {
        return false;
    }
    return rolePermissions[role]?.includes(perm) ?? false;
}
