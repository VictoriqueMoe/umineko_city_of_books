import { describe, expect, it } from "vitest";
import { can, canAccessAdmin, isSiteStaff, ROLE_GROUPS, type Permission, type SiteRole } from "./permissions";

const ALL_PERMISSIONS: Permission[] = [
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
];

const MODERATOR_ALLOWED: Permission[] = [
    "ban_user",
    "delete_any_comment",
    "delete_any_journal",
    "delete_any_post",
    "delete_any_response",
    "delete_any_theory",
    "edit_any_comment",
    "edit_any_journal",
    "edit_any_post",
    "edit_any_theory",
    "edit_mystery_score",
    "manage_user_account",
    "view_admin_panel",
    "view_stats",
    "view_users",
    "use_chatbot",
];

const MODERATOR_DENIED: Permission[] = [
    "manage_user_email",
    "manage_settings",
    "view_audit_log",
    "manage_roles",
    "delete_any_user",
    "reset_password",
    "set_email_verified",
    "resolve_suggestion",
    "manage_vanity_roles",
    "manage_banned_words",
];

function grantedTo(role: SiteRole): Permission[] {
    const granted: Permission[] = [];
    for (const permission of ALL_PERMISSIONS) {
        if (can(role, permission)) {
            granted.push(permission);
        }
    }

    return granted.sort();
}

describe("can", () => {
    it("grants a super admin every permission that exists", () => {
        // given
        const expected = [...ALL_PERMISSIONS].sort();

        // when
        const granted = grantedTo("super_admin");

        // then
        expect(granted).toEqual(expected);
    });

    it("grants an admin every permission that exists", () => {
        // given
        const expected = [...ALL_PERMISSIONS].sort();

        // when
        const granted = grantedTo("admin");

        // then
        expect(granted).toEqual(expected);
    });

    it("grants a moderator only the content and account moderation permissions", () => {
        // given
        const expected = [...MODERATOR_ALLOWED].sort();

        // when
        const granted = grantedTo("moderator");

        // then
        expect(granted).toEqual(expected);
    });

    it("withholds the sensitive site and identity permissions from a moderator", () => {
        // given
        const denied = MODERATOR_DENIED;

        // when
        const stillAllowed = denied.filter(permission => can("moderator", permission));

        // then
        expect(stillAllowed).toEqual([]);
    });

    it("denies everything when nobody is signed in", () => {
        // given
        const role = undefined;

        // when
        const granted = ALL_PERMISSIONS.filter(permission => can(role, permission));

        // then
        expect(granted).toEqual([]);
    });

    it("denies everything for a role that is not in the matrix", () => {
        // given
        const role = "member" as SiteRole;

        // when
        const granted = ALL_PERMISSIONS.filter(permission => can(role, permission));

        // then
        expect(granted).toEqual([]);
    });

    it("does not treat an unknown permission as granted for any role", () => {
        // given
        const unknown = "rewrite_the_gameboard" as Permission;

        // then
        expect(can("super_admin", unknown)).toBe(false);
        expect(can("admin", unknown)).toBe(false);
        expect(can("moderator", unknown)).toBe(false);
    });
});

describe("canAccessAdmin", () => {
    it("lets every staff role into the admin panel", () => {
        // then
        expect(canAccessAdmin("super_admin")).toBe(true);
        expect(canAccessAdmin("admin")).toBe(true);
        expect(canAccessAdmin("moderator")).toBe(true);
    });

    it("keeps signed out visitors out of the admin panel", () => {
        // given
        const role = undefined;

        // when
        const allowed = canAccessAdmin(role);

        // then
        expect(allowed).toBe(false);
    });

    it("keeps an unrecognised role out of the admin panel", () => {
        // given
        const role = "member" as SiteRole;

        // when
        const allowed = canAccessAdmin(role);

        // then
        expect(allowed).toBe(false);
    });
});

describe("isSiteStaff", () => {
    it("recognises each of the three staff roles", () => {
        // then
        expect(isSiteStaff("super_admin")).toBe(true);
        expect(isSiteStaff("admin")).toBe(true);
        expect(isSiteStaff("moderator")).toBe(true);
    });

    it("treats a missing role as not staff", () => {
        // then
        expect(isSiteStaff(undefined)).toBe(false);
        expect(isSiteStaff(null)).toBe(false);
    });

    it("treats an unrecognised role as not staff", () => {
        // given
        const role = "member" as SiteRole;

        // when
        const staff = isSiteStaff(role);

        // then
        expect(staff).toBe(false);
    });
});

describe("ROLE_GROUPS", () => {
    it("lists the staff roles from most to least privileged with their in-world labels", () => {
        // then
        expect(ROLE_GROUPS).toEqual([
            { role: "super_admin", label: "Reality Author" },
            { role: "admin", label: "Voyager Witches" },
            { role: "moderator", label: "Witches" },
        ]);
    });

    it("only lists roles the permission matrix actually knows about", () => {
        // when
        const withoutPermissions = ROLE_GROUPS.filter(group => !canAccessAdmin(group.role));

        // then
        expect(withoutPermissions).toEqual([]);
    });
});

describe("can with a server-supplied permission list", () => {
    it("prefers the server list over the static role map", () => {
        // given
        const user = { role: "moderator" as SiteRole, permissions: ["view_admin_panel"] };

        // when
        const stillHasBan = can(user, "ban_user");
        const hasAdminPanel = can(user, "view_admin_panel");

        // then
        expect(stillHasBan).toBe(false);
        expect(hasAdminPanel).toBe(true);
    });

    it("grants a permission a vanity role carries even with no site role", () => {
        // given
        const user = { permissions: ["use_chatbot"] };

        // when
        const allowed = can(user, "use_chatbot");

        // then
        expect(allowed).toBe(true);
    });

    it("falls back to the static map when the payload carries no permission list", () => {
        // given
        const user = { role: "moderator" as SiteRole };

        // when
        const allowed = can(user, "ban_user");

        // then
        expect(allowed).toBe(true);
    });

    it("denies everything to an empty server list", () => {
        // given
        const user = { role: "moderator" as SiteRole, permissions: [] };

        // when
        const allowed = can(user, "ban_user");

        // then
        expect(allowed).toBe(false);
    });
});
