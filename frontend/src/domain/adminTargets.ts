import type { SiteRole } from "../types/api";
import { can, type CanSubject } from "./permissions";

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
