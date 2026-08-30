import { describe, expect, it } from "vitest";
import { adminUserCapabilities, isProtectedTarget, staffPanelLabel, type AdminUserCapabilities } from "./adminTargets";

const gateNames: (keyof AdminUserCapabilities)[] = [
    "canManageAccount",
    "canManageEmail",
    "canSetEmailVerified",
    "canEditMysteryScore",
    "canManageRoles",
    "canBan",
    "canLock",
    "canRevokeSessions",
    "canResetPassword",
    "canDeleteUser",
];

describe("isProtectedTarget", () => {
    it("protects the reality author and nobody else", () => {
        // given
        const targets = [{ role: "super_admin" as const }, { role: "admin" as const }, { role: "moderator" as const }];

        // when
        const protectedFlags = targets.map(isProtectedTarget);

        // then
        expect(protectedFlags).toEqual([true, false, false]);
    });

    it("treats a missing target as unprotected", () => {
        // given
        const target = undefined;

        // when
        const isProtected = isProtectedTarget(target);

        // then
        expect(isProtected).toBe(false);
    });
});

describe("adminUserCapabilities", () => {
    it("closes every gate against a super admin target no matter who is asking", () => {
        // given
        const target = { role: "super_admin" as const };

        // when
        const capabilities = adminUserCapabilities("super_admin", target);

        // then
        expect(capabilities.isProtected).toBe(true);
        for (const gate of gateNames) {
            expect(capabilities[gate]).toBe(false);
        }
    });

    it("opens every gate for a super admin acting on an ordinary member", () => {
        // given
        const target = { role: undefined };

        // when
        const capabilities = adminUserCapabilities("super_admin", target);

        // then
        for (const gate of gateNames) {
            expect(capabilities[gate]).toBe(true);
        }
    });

    it("closes every gate against a signed out actor", () => {
        // given
        const target = { role: undefined };

        // when
        const capabilities = adminUserCapabilities(null, target);

        // then
        for (const gate of gateNames) {
            expect(capabilities[gate]).toBe(false);
        }
    });

    it("withholds the gates a moderator has no permission for", () => {
        // given
        const target = { role: undefined };

        // when
        const capabilities = adminUserCapabilities("moderator", target);

        // then
        expect(capabilities.canBan).toBe(true);
        expect(capabilities.canLock).toBe(true);
        expect(capabilities.canRevokeSessions).toBe(true);
        expect(capabilities.canManageAccount).toBe(true);
        expect(capabilities.canEditMysteryScore).toBe(true);
        expect(capabilities.canManageRoles).toBe(false);
        expect(capabilities.canManageEmail).toBe(false);
        expect(capabilities.canSetEmailVerified).toBe(false);
        expect(capabilities.canResetPassword).toBe(false);
        expect(capabilities.canDeleteUser).toBe(false);
    });

    it("closes locking alone against an admin target while banning stays open", () => {
        // given
        const target = { role: "admin" as const };

        // when
        const capabilities = adminUserCapabilities("super_admin", target);

        // then
        expect(capabilities.canBan).toBe(true);
        expect(capabilities.canRevokeSessions).toBe(true);
        expect(capabilities.canLock).toBe(false);
    });
});

describe("staffPanelLabel", () => {
    it("names the panel for an administrator", () => {
        // given
        const user = { role: "admin" as const };

        // when
        const labels = staffPanelLabel(user);

        // then
        expect(labels).toEqual({ heading: "Administration", navSection: "Admin", navLink: "Admin Panel" });
    });

    it("names the panel for a moderator", () => {
        // given
        const user = { role: "moderator" as const };

        // when
        const labels = staffPanelLabel(user);

        // then
        expect(labels).toEqual({ heading: "Moderator Panel", navSection: "Moderation", navLink: "Moderator Panel" });
    });

    it("falls back to the moderator wording for a viewer with no role", () => {
        // given
        const user = undefined;

        // when
        const labels = staffPanelLabel(user);

        // then
        expect(labels.heading).toBe("Moderator Panel");
    });
});
