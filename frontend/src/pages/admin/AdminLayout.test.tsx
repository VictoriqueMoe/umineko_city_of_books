import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { SiteRole } from "../../utils/permissions";
import { AdminLayout } from "./AdminLayout";

const EVERY_TAB = [
    "Dashboard",
    "Users",
    "Reports",
    "Invites",
    "Content Rules",
    "Rules Page",
    "Banned GIFs",
    "Banned Words",
    "Announcements",
    "Settings",
    "Vanity Roles",
    "Permissions",
    "Chatbots",
    "Audit Log",
];

function renderLayout(role: SiteRole | null, route = "/admin") {
    return renderWithProviders(<AdminLayout />, {
        user: role ? makeUser({ role }) : null,
        route,
    });
}

function tabNames(): string[] {
    const names: string[] = [];
    for (const link of screen.getAllByRole("link")) {
        names.push(link.textContent ?? "");
    }

    return names;
}

describe("AdminLayout", () => {
    it("names the panel after moderation for staff who cannot manage settings", () => {
        // given
        const role: SiteRole = "moderator";

        // when
        renderLayout(role);

        // then
        expect(screen.getByRole("heading", { name: "Moderator Panel" })).toBeInTheDocument();
    });

    it("names the panel administration for an admin", () => {
        // given
        const role: SiteRole = "admin";

        // when
        renderLayout(role);

        // then
        expect(screen.getByRole("heading", { name: "Administration" })).toBeInTheDocument();
    });

    it("gives a moderator only the dashboard, users and reports tabs", () => {
        // given
        const role: SiteRole = "moderator";

        // when
        renderLayout(role);

        // then
        expect(tabNames()).toEqual(["Dashboard", "Users", "Reports"]);
    });

    it("gives an admin every tab", () => {
        // given
        const role: SiteRole = "admin";

        // when
        renderLayout(role);

        // then
        expect(tabNames()).toEqual(EVERY_TAB);
    });

    it("gives a super admin every tab", () => {
        // given
        const role: SiteRole = "super_admin";

        // when
        renderLayout(role);

        // then
        expect(tabNames()).toEqual(EVERY_TAB);
    });

    it("leaves a signed out visitor with nothing but the dashboard tab", () => {
        // given
        const role = null;

        // when
        renderLayout(role);

        // then
        expect(tabNames()).toEqual(["Dashboard"]);
        expect(screen.getByRole("heading", { name: "Moderator Panel" })).toBeInTheDocument();
    });

    it("points every tab at its own admin route", () => {
        // given
        const role: SiteRole = "super_admin";

        // when
        renderLayout(role);

        // then
        expect(screen.getByRole("link", { name: "Dashboard" })).toHaveAttribute("href", "/admin");
        expect(screen.getByRole("link", { name: "Users" })).toHaveAttribute("href", "/admin/users");
        expect(screen.getByRole("link", { name: "Reports" })).toHaveAttribute("href", "/admin/reports");
        expect(screen.getByRole("link", { name: "Invites" })).toHaveAttribute("href", "/admin/invites");
        expect(screen.getByRole("link", { name: "Content Rules" })).toHaveAttribute("href", "/admin/content-rules");
        expect(screen.getByRole("link", { name: "Rules Page" })).toHaveAttribute("href", "/admin/rules");
        expect(screen.getByRole("link", { name: "Banned GIFs" })).toHaveAttribute("href", "/admin/banned-gifs");
        expect(screen.getByRole("link", { name: "Banned Words" })).toHaveAttribute("href", "/admin/banned-words");
        expect(screen.getByRole("link", { name: "Announcements" })).toHaveAttribute("href", "/admin/announcements");
        expect(screen.getByRole("link", { name: "Settings" })).toHaveAttribute("href", "/admin/settings");
        expect(screen.getByRole("link", { name: "Vanity Roles" })).toHaveAttribute("href", "/admin/vanity-roles");
        expect(screen.getByRole("link", { name: "Chatbots" })).toHaveAttribute("href", "/admin/chatbots");
        expect(screen.getByRole("link", { name: "Audit Log" })).toHaveAttribute("href", "/admin/audit-log");
    });

    it("marks the tab matching the current route as the active one", () => {
        // given
        const role: SiteRole = "admin";

        // when
        renderLayout(role, "/admin/users");

        // then
        expect(screen.getByRole("link", { name: "Users" })).toHaveAttribute("aria-current", "page");
    });

    it("keeps the dashboard tab inactive while a nested admin page is open", () => {
        // given
        const role: SiteRole = "admin";

        // when
        renderLayout(role, "/admin/audit-log");

        // then
        expect(screen.getByRole("link", { name: "Dashboard" })).not.toHaveAttribute("aria-current");
        expect(screen.getByRole("link", { name: "Audit Log" })).toHaveAttribute("aria-current", "page");
    });
});
