import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { PermissionCatalogueItem, RolePermissionsItem, VanityRolePermissionsItem } from "../../api/endpoints";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { AdminPermissions } from "./AdminPermissions";

const mocks = vi.hoisted(() => ({
    useAdminPermissions: vi.fn(),
    saveRole: vi.fn(),
    saveVanity: vi.fn(),
}));

vi.mock("../../api/queries/admin", () => ({
    useAdminPermissions: mocks.useAdminPermissions,
}));

vi.mock("../../api/mutations/admin", () => ({
    useUpdateRolePermissions: () => ({ mutate: mocks.saveRole, isPending: false }),
    useUpdateVanityRolePermissions: () => ({ mutate: mocks.saveVanity, isPending: false }),
}));

const catalogue: PermissionCatalogueItem[] = [
    { permission: "ban_user", label: "Ban and lock users", vanity_assignable: false },
    { permission: "view_admin_panel", label: "View admin panel", vanity_assignable: false },
    { permission: "use_chatbot", label: "Summon chatbots", vanity_assignable: true },
];

function stubPermissions(
    overrides: {
        roles?: RolePermissionsItem[];
        vanityRoles?: VanityRolePermissionsItem[];
        loading?: boolean;
    } = {},
) {
    mocks.useAdminPermissions.mockReturnValue({
        catalogue,
        roles: overrides.roles ?? [
            { role: "moderator", label: "Moderator", permissions: ["ban_user", "view_admin_panel"] },
        ],
        vanityRoles: overrides.vanityRoles ?? [
            { id: "role-1", label: "Beta Tester", color: "#ff8800", sort_order: 1, permissions: [] },
        ],
        loading: overrides.loading ?? false,
        refresh: vi.fn(),
    });
}

function renderPage(role: "admin" | "moderator" = "admin") {
    return renderWithProviders(<AdminPermissions />, { user: makeUser({ role }) });
}

beforeEach(() => {
    mocks.saveRole.mockReset();
    mocks.saveVanity.mockReset();
    stubPermissions();
});

describe("AdminPermissions", () => {
    it("waits while the permission tables are being fetched", () => {
        // given
        stubPermissions({ loading: true });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("refuses the page to somebody without manage_roles", () => {
        // given
        stubPermissions();

        // when
        renderPage("moderator");

        // then
        expect(screen.getByText(/do not have permission/i)).toBeInTheDocument();
    });

    it("never renders a card for admin or super admin", () => {
        // given
        stubPermissions();

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "Moderator" })).toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: /voyager witch/i })).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: /reality author/i })).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: /super.?admin/i })).not.toBeInTheDocument();
    });

    it("shows the stored moderator grants as switches", () => {
        // given
        stubPermissions();

        // when
        renderPage();

        // then
        expect(screen.getByRole("switch", { name: "Ban and lock users" })).toHaveAttribute("aria-checked", "true");
        expect(screen.getByRole("switch", { name: "Summon chatbots" })).toHaveAttribute("aria-checked", "false");
    });

    it("saves the moderator set with the unticked permission removed", async () => {
        // given
        stubPermissions();
        renderPage();

        // when
        await userEvent.click(screen.getByRole("switch", { name: "Ban and lock users" }));
        await userEvent.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.saveRole).toHaveBeenCalledWith(
            { role: "moderator", permissions: ["view_admin_panel"] },
            expect.anything(),
        );
    });

    it("offers only the assignable subset for a vanity role", async () => {
        // given
        stubPermissions();
        renderPage();

        // when
        await userEvent.selectOptions(screen.getByLabelText("Vanity role"), "role-1");

        // then
        const switches = screen.getAllByRole("switch").map(el => el.getAttribute("aria-label"));
        expect(switches).toContain("Summon chatbots");
        expect(switches.filter(name => name === "Ban and lock users")).toHaveLength(1);
    });

    it("saves a vanity role grant", async () => {
        // given
        stubPermissions();
        renderPage();

        // when
        await userEvent.selectOptions(screen.getByLabelText("Vanity role"), "role-1");
        const vanitySwitches = screen.getAllByRole("switch", { name: "Summon chatbots" });
        await userEvent.click(vanitySwitches[vanitySwitches.length - 1]);
        const saveButtons = screen.getAllByRole("button", { name: "Save" });
        await userEvent.click(saveButtons[saveButtons.length - 1]);

        // then
        expect(mocks.saveVanity).toHaveBeenCalledWith(
            { id: "role-1", permissions: ["use_chatbot"] },
            expect.anything(),
        );
    });

    it("tells the admin when there are no custom vanity roles yet", () => {
        // given
        stubPermissions({ vanityRoles: [] });

        // when
        renderPage();

        // then
        expect(screen.getByText(/no custom vanity roles yet/i)).toBeInTheDocument();
    });
});
