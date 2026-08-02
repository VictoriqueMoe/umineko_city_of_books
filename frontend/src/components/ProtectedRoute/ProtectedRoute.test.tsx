import { screen } from "@testing-library/react";
import { Route, Routes } from "react-router";
import { describe, expect, it } from "vitest";
import type { UserProfile } from "../../types/api";
import type { Permission, SiteRole } from "../../utils/permissions";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { ProtectedRoute } from "./ProtectedRoute";

interface ProtectedOptions {
    permission?: Permission;
    user?: UserProfile | null;
    loading?: boolean;
}

function renderProtected(options: ProtectedOptions = {}) {
    return renderWithProviders(
        <Routes>
            <Route element={<ProtectedRoute permission={options.permission} />}>
                <Route path="/secret" element={<p>the golden truth</p>} />
            </Route>
            <Route path="/login" element={<p>sign in first</p>} />
            <Route path="/" element={<p>front page</p>} />
        </Routes>,
        { route: "/secret", user: options.user ?? null, auth: { loading: options.loading ?? false } },
    );
}

interface RoleCase {
    role: SiteRole;
    permission: Permission;
    allowed: boolean;
}

const roleCases: RoleCase[] = [
    { role: "super_admin", permission: "manage_settings", allowed: true },
    { role: "admin", permission: "manage_settings", allowed: true },
    { role: "moderator", permission: "manage_settings", allowed: false },
    { role: "super_admin", permission: "view_admin_panel", allowed: true },
    { role: "admin", permission: "view_admin_panel", allowed: true },
    { role: "moderator", permission: "view_admin_panel", allowed: true },
    { role: "moderator", permission: "manage_roles", allowed: false },
    { role: "moderator", permission: "view_audit_log", allowed: false },
    { role: "moderator", permission: "delete_any_theory", allowed: true },
];

describe("ProtectedRoute while the session is still unknown", () => {
    it("holds the route with a loading placeholder", () => {
        // given
        const loading = true;

        // when
        renderProtected({ loading });

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
        expect(screen.queryByText("the golden truth")).not.toBeInTheDocument();
    });

    it("does not send a visitor to the login page before the session has resolved", () => {
        // given
        const loading = true;

        // when
        renderProtected({ loading, user: null });

        // then
        expect(screen.queryByText("sign in first")).not.toBeInTheDocument();
    });

    it("holds the route even for a member whose session is still resolving", () => {
        // given
        const user = makeUser();

        // when
        renderProtected({ loading: true, user });

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
        expect(screen.queryByText("the golden truth")).not.toBeInTheDocument();
    });
});

describe("ProtectedRoute for a signed out visitor", () => {
    it("sends the visitor to the login page", () => {
        // given
        const user = null;

        // when
        renderProtected({ user });

        // then
        expect(screen.getByText("sign in first")).toBeInTheDocument();
    });

    it("never reveals the protected content", () => {
        // given
        const user = null;

        // when
        renderProtected({ user, permission: "view_admin_panel" });

        // then
        expect(screen.queryByText("the golden truth")).not.toBeInTheDocument();
        expect(screen.queryByText("front page")).not.toBeInTheDocument();
    });
});

describe("ProtectedRoute for a banned member", () => {
    it("sends a banned member to the login page", () => {
        // given
        const user = makeUser({ banned: true });

        // when
        renderProtected({ user });

        // then
        expect(screen.getByText("sign in first")).toBeInTheDocument();
        expect(screen.queryByText("the golden truth")).not.toBeInTheDocument();
    });

    it("keeps a banned member of staff out of a route gated on a permission", () => {
        // given
        const user = makeUser({ role: "admin", banned: true });

        // when
        renderProtected({ user, permission: "view_admin_panel" });

        // then
        expect(screen.getByText("sign in first")).toBeInTheDocument();
        expect(screen.queryByText("the golden truth")).not.toBeInTheDocument();
    });

    it("lets a member who is not banned through", () => {
        // given
        const user = makeUser({ banned: false });

        // when
        renderProtected({ user });

        // then
        expect(screen.getByText("the golden truth")).toBeInTheDocument();
    });

    it("still lets a locked member through so they can read the site and message staff", () => {
        // given
        const user = makeUser({ locked: true, lock_reason: "too many red truths" });

        // when
        renderProtected({ user });

        // then
        expect(screen.getByText("the golden truth")).toBeInTheDocument();
    });
});

describe("ProtectedRoute for a signed in member", () => {
    it("shows the protected content when no permission is demanded", () => {
        // given
        const user = makeUser();

        // when
        renderProtected({ user });

        // then
        expect(screen.getByText("the golden truth")).toBeInTheDocument();
    });

    it("shows the protected content to a member of staff when no permission is demanded", () => {
        // given
        const user = makeUser({ role: "moderator" });

        // when
        renderProtected({ user });

        // then
        expect(screen.getByText("the golden truth")).toBeInTheDocument();
    });

    it("sends an ordinary member back to the front page when a permission is demanded", () => {
        // given
        const user = makeUser({ role: undefined });

        // when
        renderProtected({ user, permission: "view_admin_panel" });

        // then
        expect(screen.getByText("front page")).toBeInTheDocument();
        expect(screen.queryByText("the golden truth")).not.toBeInTheDocument();
    });

    it("sends a member with an unknown role back to the front page", () => {
        // given
        const user = makeUser({ role: "witch" as SiteRole });

        // when
        renderProtected({ user, permission: "view_admin_panel" });

        // then
        expect(screen.getByText("front page")).toBeInTheDocument();
    });

    for (const roleCase of roleCases) {
        const outcome = roleCase.allowed ? "lets" : "refuses";
        it(`${outcome} a ${roleCase.role} through a route gated on ${roleCase.permission}`, () => {
            // given
            const user = makeUser({ role: roleCase.role });

            // when
            renderProtected({ user, permission: roleCase.permission });

            // then
            expect(screen.queryByText("the golden truth") !== null).toBe(roleCase.allowed);
            expect(screen.queryByText("front page") !== null).toBe(!roleCase.allowed);
        });
    }
});
