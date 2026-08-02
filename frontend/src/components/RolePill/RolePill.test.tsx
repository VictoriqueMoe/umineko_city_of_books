import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { VanityRoleDefinition } from "../../api/endpoints";
import { renderWithProviders } from "../../test-utils/render";
import { RolePill } from "./RolePill";

const USER_ID = "00000000-0000-0000-0000-000000000001";

function makeVanityRole(overrides: Partial<VanityRoleDefinition> = {}): VanityRoleDefinition {
    return {
        id: "custom",
        label: "Custom",
        color: "#ff0000",
        is_system: false,
        sort_order: 0,
        ...overrides,
    };
}

function pillLabels(): string[] {
    const group = screen.getByLabelText("User roles");
    const labels: string[] = [];
    for (const child of Array.from(group.children)) {
        labels.push(child.textContent ?? "");
    }
    return labels;
}

describe("RolePill", () => {
    it("renders nothing for a plain member with no vanity roles", () => {
        // given
        const role = "user";

        // when
        const { container } = renderWithProviders(<RolePill role={role} userId={USER_ID} />);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("names the site owner as the Reality Author", () => {
        // given
        const role = "super_admin";

        // when
        renderWithProviders(<RolePill role={role} userId={USER_ID} />);

        // then
        expect(screen.getByText("Reality Author")).toHaveAttribute("title", "Site owner - super administrator");
    });

    it("names an administrator as a Voyager Witch", () => {
        // given
        const role = "admin";

        // when
        renderWithProviders(<RolePill role={role} userId={USER_ID} />);

        // then
        expect(screen.getByText("Voyager Witch")).toHaveAttribute("title", "Administrator");
    });

    it("names a moderator as a Witch", () => {
        // given
        const role = "moderator";

        // when
        renderWithProviders(<RolePill role={role} userId={USER_ID} />);

        // then
        expect(screen.getByText("Witch")).toHaveAttribute("title", "Moderator");
    });

    it("labels the group of pills for assistive technology", () => {
        // given
        const role = "admin";

        // when
        renderWithProviders(<RolePill role={role} userId={USER_ID} />);

        // then
        expect(screen.getByLabelText("User roles")).toBeInTheDocument();
    });

    it("shows only the vanity roles assigned to this user", () => {
        // given
        const vanityRoles = [
            makeVanityRole({ id: "mine", label: "Golden Butterfly" }),
            makeVanityRole({ id: "theirs", label: "Stakes of Purgatory" }),
        ];

        // when
        renderWithProviders(<RolePill role="user" userId={USER_ID} />, {
            siteInfo: { vanity_roles: vanityRoles, vanity_role_assignments: { [USER_ID]: ["mine"] } },
        });

        // then
        expect(screen.getByText("Golden Butterfly")).toBeInTheDocument();
        expect(screen.queryByText("Stakes of Purgatory")).not.toBeInTheDocument();
    });

    it("orders vanity pills by their configured sort order", () => {
        // given
        const vanityRoles = [
            makeVanityRole({ id: "third", label: "Third", sort_order: 30 }),
            makeVanityRole({ id: "first", label: "First", sort_order: 10 }),
            makeVanityRole({ id: "second", label: "Second", sort_order: 20 }),
        ];

        // when
        renderWithProviders(<RolePill role="user" userId={USER_ID} />, {
            siteInfo: {
                vanity_roles: vanityRoles,
                vanity_role_assignments: { [USER_ID]: ["third", "first", "second"] },
            },
        });

        // then
        expect(pillLabels()).toEqual(["First", "Second", "Third"]);
    });

    it("puts the staff pill ahead of the vanity pills", () => {
        // given
        const vanityRoles = [makeVanityRole({ id: "mine", label: "Golden Butterfly" })];

        // when
        renderWithProviders(<RolePill role="moderator" userId={USER_ID} />, {
            siteInfo: { vanity_roles: vanityRoles, vanity_role_assignments: { [USER_ID]: ["mine"] } },
        });

        // then
        expect(pillLabels()).toEqual(["Witch", "Golden Butterfly"]);
    });

    it("renders no vanity pills when no user id is supplied", () => {
        // given
        const vanityRoles = [makeVanityRole({ id: "mine", label: "Golden Butterfly" })];

        // when
        const { container } = renderWithProviders(<RolePill role="user" />, {
            siteInfo: { vanity_roles: vanityRoles, vanity_role_assignments: { [USER_ID]: ["mine"] } },
        });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("treats a blank user id as having no vanity roles at all", () => {
        // given
        const vanityRoles = [makeVanityRole({ id: "", label: "Golden Butterfly" })];

        // when
        const { container } = renderWithProviders(<RolePill role="user" userId="" />, {
            siteInfo: { vanity_roles: vanityRoles, vanity_role_assignments: { [USER_ID]: ["mine"] } },
        });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("explains a system vanity role with its built in description", () => {
        // given
        const vanityRoles = [makeVanityRole({ id: "system_witch_hunter", label: "Witch Hunter", is_system: true })];

        // when
        renderWithProviders(<RolePill role="user" userId={USER_ID} />, {
            siteInfo: {
                vanity_roles: vanityRoles,
                vanity_role_assignments: { [USER_ID]: ["system_witch_hunter"] },
            },
        });

        // then
        expect(screen.getByTitle("Solved the Witch Hunter secret")).toHaveTextContent("Witch Hunter");
    });

    it("falls back to the label as the tooltip for a bespoke vanity role", () => {
        // given
        const vanityRoles = [makeVanityRole({ id: "mine", label: "Golden Butterfly" })];

        // when
        renderWithProviders(<RolePill role="user" userId={USER_ID} />, {
            siteInfo: { vanity_roles: vanityRoles, vanity_role_assignments: { [USER_ID]: ["mine"] } },
        });

        // then
        expect(screen.getByText("Golden Butterfly")).toHaveAttribute("title", "Golden Butterfly");
    });

    it("darkens a pale vanity colour so the label stays legible", () => {
        // given
        const vanityRoles = [makeVanityRole({ id: "mine", label: "Golden Butterfly", color: "#ff0000" })];

        // when
        renderWithProviders(<RolePill role="user" userId={USER_ID} />, {
            siteInfo: { vanity_roles: vanityRoles, vanity_role_assignments: { [USER_ID]: ["mine"] } },
        });

        // then
        expect(screen.getByText("Golden Butterfly")).toHaveStyle({
            color: "rgb(214, 0, 0)",
            backgroundColor: "rgba(255, 0, 0, 0.18)",
        });
    });

    it("leaves an already dark vanity colour untouched", () => {
        // given
        const vanityRoles = [makeVanityRole({ id: "mine", label: "Golden Butterfly", color: "#102030" })];

        // when
        renderWithProviders(<RolePill role="user" userId={USER_ID} />, {
            siteInfo: { vanity_roles: vanityRoles, vanity_role_assignments: { [USER_ID]: ["mine"] } },
        });

        // then
        expect(screen.getByText("Golden Butterfly")).toHaveStyle({ color: "rgb(16, 32, 48)" });
    });

    it("swallows the click on a pill that can be expanded on mobile", async () => {
        // given
        const onParentClick = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <div onClick={onParentClick}>
                <RolePill role="admin" userId={USER_ID} compactOnMobile />
            </div>,
        );

        // when
        await user.click(screen.getByText("Voyager Witch"));

        // then
        expect(onParentClick).not.toHaveBeenCalled();
    });

    it("lets a click through when the pill is not collapsible", async () => {
        // given
        const onParentClick = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <div onClick={onParentClick}>
                <RolePill role="admin" userId={USER_ID} />
            </div>,
        );

        // when
        await user.click(screen.getByText("Voyager Witch"));

        // then
        expect(onParentClick).toHaveBeenCalledOnce();
    });
});
