import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { VanityRoleDefinition } from "../../api/endpoints";
import { renderWithProviders } from "../../test-utils/render";
import type { User } from "../../types/api";
import { AdminVanityRoles } from "./AdminVanityRoles";

const mocks = vi.hoisted(() => ({
    useVanityRoles: vi.fn(),
    useVanityRoleUsers: vi.fn(),
    useSearchUsers: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
    assign: vi.fn(),
    unassign: vi.fn(),
}));

vi.mock("../../api/queries/admin", () => ({
    useVanityRoles: mocks.useVanityRoles,
    useVanityRoleUsers: mocks.useVanityRoleUsers,
}));

vi.mock("../../api/queries/misc", () => ({ useSearchUsers: mocks.useSearchUsers }));

vi.mock("../../api/mutations/admin", () => ({
    useCreateVanityRole: () => ({ mutateAsync: mocks.create, isPending: false }),
    useUpdateVanityRole: () => ({ mutateAsync: mocks.update, isPending: false }),
    useDeleteVanityRole: () => ({ mutateAsync: mocks.remove, isPending: false }),
    useAssignVanityRole: () => ({ mutateAsync: mocks.assign, isPending: false }),
    useUnassignVanityRole: () => ({ mutateAsync: mocks.unassign, isPending: false }),
}));

function makeRole(overrides: Partial<VanityRoleDefinition> = {}): VanityRoleDefinition {
    return {
        id: "role-1",
        label: "Beta Tester",
        color: "#ff8800",
        is_system: false,
        sort_order: 3,
        ...overrides,
    };
}

function stubRoles(roles: VanityRoleDefinition[], loading = false) {
    mocks.useVanityRoles.mockReturnValue({ roles, loading, refresh: vi.fn() });
}

function stubAssigned(users: { id: string; username: string; display_name: string; avatar_url: string }[] = []) {
    mocks.useVanityRoleUsers.mockReturnValue({ users, total: users.length, loading: false, refresh: vi.fn() });
}

function stubSearch(users: User[] = []) {
    mocks.useSearchUsers.mockReturnValue({ users, loading: false });
}

beforeEach(() => {
    stubAssigned();
    stubSearch();
    mocks.create.mockResolvedValue(undefined);
    mocks.update.mockResolvedValue(undefined);
    mocks.remove.mockResolvedValue(undefined);
    mocks.assign.mockResolvedValue(undefined);
    mocks.unassign.mockResolvedValue(undefined);
});

describe("AdminVanityRoles", () => {
    it("waits while the roles are being fetched", () => {
        // given
        stubRoles([], true);

        // when
        renderWithProviders(<AdminVanityRoles />);

        // then
        expect(screen.getByText("Loading vanity roles...")).toBeInTheDocument();
    });

    it("says so when no vanity role exists yet", () => {
        // given
        stubRoles([]);

        // when
        renderWithProviders(<AdminVanityRoles />);

        // then
        expect(screen.getByText("No vanity roles yet.")).toBeInTheDocument();
    });

    it("lists a custom role with its colour and offers to delete it", () => {
        // given
        stubRoles([makeRole()]);

        // when
        renderWithProviders(<AdminVanityRoles />);

        // then
        expect(screen.getByText("Custom")).toBeInTheDocument();
        expect(screen.getByText("#ff8800")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    });

    it("refuses to delete a system role", () => {
        // given
        stubRoles([makeRole({ is_system: true, label: "Top Detective" })]);

        // when
        renderWithProviders(<AdminVanityRoles />);

        // then
        expect(screen.getByText("System")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("starts a new role after the last existing sort order", async () => {
        // given
        stubRoles([makeRole({ sort_order: 3 })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminVanityRoles />);

        // when
        await user.click(screen.getByRole("button", { name: "Create Role" }));

        // then
        expect(screen.getByRole("heading", { name: "Create Vanity Role" })).toBeInTheDocument();
        expect(screen.getByRole("spinbutton")).toHaveValue(4);
        expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });

    it("creates the role from the label, colour and order given", async () => {
        // given
        stubRoles([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminVanityRoles />);
        await user.click(screen.getByRole("button", { name: "Create Role" }));
        await user.type(screen.getByPlaceholderText("e.g. Beta Tester"), "Beta Tester");

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.create).toHaveBeenCalledWith({ label: "Beta Tester", color: "#888888", sort_order: 0 });
    });

    it("prefills the form from the role being edited and updates it", async () => {
        // given
        stubRoles([makeRole({ id: "role-9", label: "Beta Tester", color: "#ff8800", sort_order: 3 })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminVanityRoles />);
        await user.click(screen.getByRole("button", { name: "Edit" }));
        expect(screen.getByRole("heading", { name: "Edit Vanity Role" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText("e.g. Beta Tester")).toHaveValue("Beta Tester");

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({
            id: "role-9",
            data: { label: "Beta Tester", color: "#ff8800", sort_order: 3 },
        });
    });

    it("reports why a role could not be saved", async () => {
        // given
        stubRoles([]);
        mocks.create.mockRejectedValue(new Error("that label is taken"));
        const user = userEvent.setup();
        renderWithProviders(<AdminVanityRoles />);
        await user.click(screen.getByRole("button", { name: "Create Role" }));
        await user.type(screen.getByPlaceholderText("e.g. Beta Tester"), "Beta Tester");

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(await screen.findByText("that label is taken")).toBeInTheDocument();
    });

    it("asks before deleting a role", async () => {
        // given
        stubRoles([makeRole()]);
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderWithProviders(<AdminVanityRoles />);

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Delete this vanity role? It will be removed from all users.");
        expect(mocks.remove).not.toHaveBeenCalled();
    });

    it("deletes the role once confirmed", async () => {
        // given
        stubRoles([makeRole({ id: "role-5" })]);
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderWithProviders(<AdminVanityRoles />);

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(mocks.remove).toHaveBeenCalledWith("role-5");
    });

    it("explains that a system role is assigned automatically", async () => {
        // given
        stubRoles([makeRole({ is_system: true, label: "Top Detective" })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminVanityRoles />);

        // when
        await user.click(screen.getByRole("button", { name: "Users" }));

        // then
        expect(screen.getByText(/automatically assigned based on mystery leaderboard scores/)).toBeInTheDocument();
        expect(screen.queryByPlaceholderText("Search users to assign...")).not.toBeInTheDocument();
    });

    it("says when a custom role has nobody assigned", async () => {
        // given
        stubRoles([makeRole()]);
        stubAssigned([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminVanityRoles />);

        // when
        await user.click(screen.getByRole("button", { name: "Users" }));

        // then
        expect(screen.getByText("Assigned (0)")).toBeInTheDocument();
        expect(screen.getByText("No users assigned.")).toBeInTheDocument();
    });

    it("lists everyone already holding the role", async () => {
        // given
        stubRoles([makeRole()]);
        stubAssigned([{ id: "u1", username: "beatrice", display_name: "Beatrice", avatar_url: "" }]);
        const user = userEvent.setup();
        renderWithProviders(<AdminVanityRoles />);

        // when
        await user.click(screen.getByRole("button", { name: "Users" }));

        // then
        expect(screen.getByText("Assigned (1)")).toBeInTheDocument();
        expect(screen.getByText("Beatrice (@beatrice)")).toBeInTheDocument();
    });

    it("only searches for members once at least two letters are typed", async () => {
        // given
        stubRoles([makeRole({ id: "role-2" })]);
        stubSearch([{ id: "u2", username: "battler", display_name: "Battler" }]);
        const user = userEvent.setup();
        renderWithProviders(<AdminVanityRoles />);
        await user.click(screen.getByRole("button", { name: "Users" }));

        // when
        await user.type(screen.getByPlaceholderText("Search users to assign..."), "b");

        // then
        expect(screen.queryByRole("button", { name: "Assign" })).not.toBeInTheDocument();

        await user.type(screen.getByPlaceholderText("Search users to assign..."), "a");
        expect(screen.getByRole("button", { name: "Assign" })).toBeInTheDocument();
    });

    it("assigns the searched member to the role being managed", async () => {
        // given
        stubRoles([makeRole({ id: "role-2" })]);
        stubSearch([{ id: "u2", username: "battler", display_name: "Battler" }]);
        const user = userEvent.setup();
        renderWithProviders(<AdminVanityRoles />);
        await user.click(screen.getByRole("button", { name: "Users" }));
        await user.type(screen.getByPlaceholderText("Search users to assign..."), "ba");

        // when
        await user.click(screen.getByRole("button", { name: "Assign" }));

        // then
        expect(mocks.assign).toHaveBeenCalledWith({ roleId: "role-2", userId: "u2" });
    });

    it("leaves members who already hold the role out of the search results", async () => {
        // given
        stubRoles([makeRole()]);
        stubAssigned([{ id: "u2", username: "battler", display_name: "Battler", avatar_url: "" }]);
        stubSearch([{ id: "u2", username: "battler", display_name: "Battler" }]);
        const user = userEvent.setup();
        renderWithProviders(<AdminVanityRoles />);
        await user.click(screen.getByRole("button", { name: "Users" }));

        // when
        await user.type(screen.getByPlaceholderText("Search users to assign..."), "ba");

        // then
        expect(screen.queryByRole("button", { name: "Assign" })).not.toBeInTheDocument();
    });

    it("takes the role away from an assigned member", async () => {
        // given
        stubRoles([makeRole({ id: "role-7" })]);
        stubAssigned([{ id: "u1", username: "beatrice", display_name: "Beatrice", avatar_url: "" }]);
        const user = userEvent.setup();
        renderWithProviders(<AdminVanityRoles />);
        await user.click(screen.getByRole("button", { name: "Users" }));

        // when
        await user.click(
            within(screen.getByText("Beatrice (@beatrice)").closest("div") as HTMLElement).getByRole("button", {
                name: "Remove",
            }),
        );

        // then
        expect(mocks.unassign).toHaveBeenCalledWith({ roleId: "role-7", userId: "u1" });
    });
});
