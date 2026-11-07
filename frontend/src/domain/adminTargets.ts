import type { SiteRole } from "../types/api";
import { can, type CanSubject, type Permission } from "./permissions";

export type AdminTab = {
    to: string;
    label: string;
    permission: Permission;
    end?: boolean;
};

const ADMIN_TABS: AdminTab[] = [
    { to: "/admin", label: "Dashboard", permission: "view_stats", end: true },
    { to: "/admin/users", label: "Users", permission: "view_users" },
    { to: "/admin/reports", label: "Reports", permission: "view_users" },
    { to: "/admin/invites", label: "Invites", permission: "manage_roles" },
    { to: "/admin/content-rules", label: "Content Rules", permission: "manage_settings" },
    { to: "/admin/rules", label: "Rules Page", permission: "manage_settings" },
    { to: "/admin/banned-gifs", label: "Banned GIFs", permission: "manage_settings" },
    { to: "/admin/banned-words", label: "Banned Words", permission: "manage_banned_words" },
    { to: "/admin/announcements", label: "Announcements", permission: "manage_settings" },
    { to: "/admin/settings", label: "Settings", permission: "manage_settings" },
    { to: "/admin/vanity-roles", label: "Vanity Roles", permission: "manage_vanity_roles" },
    { to: "/admin/permissions", label: "Permissions", permission: "manage_roles" },
    { to: "/admin/chatbots", label: "Chatbots", permission: "manage_settings" },
    { to: "/admin/audit-log", label: "Audit Log", permission: "view_audit_log" },
];

export function adminTabsFor(user: CanSubject): AdminTab[] {
    if (!can(user, "view_admin_panel")) {
        return [];
    }

    return ADMIN_TABS.filter(tab => can(user, tab.permission));
}

export interface AdminTarget {
    role?: SiteRole | null;
}

export interface AdminUserCapabilities {
    isProtected: boolean;
    canManageAccount: boolean;
    canManageEmail: boolean;
    canSetEmailVerified: boolean;
    canEditMysteryScore: boolean;
    canManageRoles: boolean;
    canBan: boolean;
    canLock: boolean;
    canRevokeSessions: boolean;
    canResetPassword: boolean;
    canDeleteUser: boolean;
}

export interface StaffPanelLabels {
    heading: string;
    navSection: string;
    navLink: string;
}

export function isProtectedTarget(target: AdminTarget | null | undefined): boolean {
    return target?.role === "super_admin";
}

export function adminUserCapabilities(
    actor: CanSubject,
    target: AdminTarget | null | undefined,
): AdminUserCapabilities {
    const isProtected = isProtectedTarget(target);
    const targetIsAdmin = target?.role === "admin";

    const canBan = can(actor, "ban_user") && !isProtected;

    return {
        isProtected,
        canManageAccount: can(actor, "manage_user_account") && !isProtected,
        canManageEmail: can(actor, "manage_user_email") && !isProtected,
        canSetEmailVerified: can(actor, "set_email_verified") && !isProtected,
        canEditMysteryScore: can(actor, "edit_mystery_score") && !isProtected,
        canManageRoles: can(actor, "manage_roles") && !isProtected,
        canBan,
        canLock: canBan && !targetIsAdmin,
        canRevokeSessions: canBan,
        canResetPassword: can(actor, "reset_password") && !isProtected,
        canDeleteUser: can(actor, "delete_any_user") && !isProtected,
    };
}

export function staffPanelLabel(user: CanSubject): StaffPanelLabels {
    if (can(user, "manage_settings")) {
        return { heading: "Administration", navSection: "Admin", navLink: "Admin Panel" };
    }

    return { heading: "Moderator Panel", navSection: "Moderation", navLink: "Moderator Panel" };
}
